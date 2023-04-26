package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var CLI struct {
	NoColor   bool
	Since     *time.Time      `help:"Only list events after this date, you can copy-paste a date from mysql error log"`
	Until     *time.Time      `help:"Only list events before this date, you can copy-paste a date from mysql error log"`
	Verbosity types.Verbosity `default:"1" help:"0: Info, 1: Detailed, 2: DebugMySQL (every mysql info the tool used), 3: Debug (internal tool debug)"`
	K8s       bool            `default:"false" help:"Analyze logs from Percona PXC operator. Off by default because it negatively impacts performance for non-k8s setups" name:"k8s"`

	List    list    `cmd:""`
	Whois   whois   `cmd:""`
	Sed     sed     `cmd:""`
	Summary summary `cmd:""`
	Ctx     ctx     `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("galera-log-explainer"),
		kong.Description("An utility to transform Galera logs in a readable version"),
		kong.UsageOnError(),
	)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if CLI.Verbosity == types.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	utils.SkipColor = CLI.NoColor
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

// timelineFromPaths takes every path, search them using a list of regexes
// and organize them in a timeline that will be ready to aggregate or read
func timelineFromPaths(paths []string, toCheck types.RegexMap, since, until *time.Time) (types.Timeline, error) {
	timeline := make(types.Timeline)
	found := false

	for _, path := range paths {

		extr := newExtractor(path, toCheck, since, until)

		node, localTimeline, err := extr.search()
		if err != nil {
			extr.logger.Warn().Err(err).Msg("Search failed")
			continue
		}
		found = true

		if t, ok := timeline[node]; ok {
			localTimeline = types.MergeTimeline(t, localTimeline)
		}
		timeline[node] = localTimeline
	}
	if !found {
		return nil, errors.New("Could not find data")
	}
	return timeline, nil
}

type extractor struct {
	regexes      types.RegexMap
	path         string
	since, until *time.Time
	logger       zerolog.Logger
}

func newExtractor(path string, toCheck types.RegexMap, since, until *time.Time) extractor {
	e := extractor{regexes: toCheck, path: path, since: since, until: until}
	e.logger = log.With().Str("component", "extractor").Str("path", e.path).Logger()
	if since != nil {
		e.logger = e.logger.With().Time("since", *e.since).Logger()
	}
	if until != nil {
		e.logger = e.logger.With().Time("until", *e.until).Logger()
	}

	return e
}

func (e *extractor) grepArgument() string {

	regexToSendSlice := []string{}
	for _, regex := range e.regexes {
		regexToSendSlice = append(regexToSendSlice, regex.Regex.String())
	}
	grepRegex := ""
	if CLI.K8s {
		grepRegex = "({\"log\":\")?" // to ignore k8s log prefixes. Works without it, but it helps to correctly filter date
	}
	if e.since != nil || e.until != nil {
		grepRegex += "(" + regex.BetweenDateRegex(e.since, e.until) + "|" + regex.NoDatesRegex() + ").*"
	}
	return grepRegex + "(" + strings.Join(regexToSendSlice, "|") + ")"
}

// search is the main function to search what we want in a file
func (e *extractor) search() (string, types.LocalTimeline, error) {

	// A first pass is done, with every regexes we want compiled in a single one.
	grepRegex := e.grepArgument()
	e.logger.Debug().Str("grepArg", grepRegex).Msg("")

	/*
		Regular grep is actually used

		There are no great alternatives, even less as golang libraries.
		grep itself do not have great alternatives: they are less performant for common use-cases, or are not easily portable, or are costlier to execute.
		grep is everywhere, grep is good enough, it even enable to use the stdout pipe.

		The usual bottleneck with grep is that it is single-threaded, but we actually benefit
		from a sequential scan here as we will rely on the log order.

		Also, being sequential also ensure this program is light enough to run without too much impacts
		It also helps to be transparent and not provide an obscure tool that work as a blackbox
	*/
	cmd := exec.Command("grep", "-P", grepRegex, e.path)
	out, _ := cmd.StdoutPipe()
	defer out.Close()
	err := cmd.Start()
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to search in %s", e.path)
	}

	// grep treatment
	s := bufio.NewScanner(out)

	lt, err := e.iterateOnResults(s)
	if err != nil {
		e.logger.Warn().Err(err).Msg("Failed to iterate on results")
	}

	// If we found anything
	if len(lt) == 0 {
		// identify the node with the easiest to read information
		return "", nil, errors.New("Found nothing")
	}

	// Why not just return the file path:
	// so that we are able to merge files that belong to the same nodes
	// we wouldn't want them to be shown as from different nodes
	return types.DisplayLocalNodeSimplestForm(lt[len(lt)-1].Ctx), lt, nil
}

func (e *extractor) sanitizeLine(s string) string {
	if len(s) > 0 && s[0] == '\t' {
		return s[1:]
	}
	return s
}

func (e *extractor) iterateOnResults(s *bufio.Scanner) ([]types.LogInfo, error) {

	var (
		line         string
		lt           []types.LogInfo
		recentEnough bool
		displayer    types.LogDisplayer
	)
	ctx := types.NewLogCtx()
	ctx.FilePath = filepath.Base(e.path)

	for s.Scan() {
		line = e.sanitizeLine(s.Text())

		date := types.NewDate(regex.SearchDateFromLog(line))

		// If it's recentEnough, it means we already validated a log: every next logs necessarily happened later
		// this is useful because not every logs have a date attached, and some without date are very useful
		if !recentEnough && e.since != nil && e.since.After(date.Time) {
			continue
		}
		if e.until != nil && e.until.Before(date.Time) {
			return lt, nil
		}
		recentEnough = true

		// We have to find again what regex worked to get this log line
		for key, regex := range e.regexes {
			if !regex.Regex.MatchString(line) {
				continue
			}
			ctx, displayer = regex.Handle(ctx, line)
			lt = append(lt, types.LogInfo{
				Date:      date,
				Log:       line,
				Msg:       displayer,
				Ctx:       ctx,
				RegexType: regex.Type,
				RegexUsed: key,
				Verbosity: regex.Verbosity,
			})
		}

	}
	return lt, nil
}
