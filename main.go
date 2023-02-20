package main

import (
	"bufio"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var CLI struct {
	NoColor bool
	Since   *time.Time `help:"Only list events after this date, you can copy-paste a date from mysql error log"`
	Until   *time.Time `help:"Only list events before this date, you can copy-paste a date from mysql error log"`

	List  list  `cmd:""`
	Whois whois `cmd:""`
	Sed   sed   `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("galera-log-explainer"),
		kong.Description("An utility to transform Galera logs in a readable version"),
		kong.UsageOnError(),
	)

	utils.SkipColor = CLI.NoColor
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

// timelineFromPaths takes every path, search them using a list of regexes
// and organize them in a timeline that will be ready to aggregate or read
func timelineFromPaths(paths []string, toCheck []regex.LogRegex, since, until *time.Time) types.Timeline {
	timeline := make(types.Timeline)

	for _, path := range paths {

		extr := newExtractor(path, toCheck, since, until)

		node, localTimeline, err := extr.search()
		if err != nil {
			log.Println(err)
		}

		if t, ok := timeline[node]; ok {
			localTimeline = types.MergeTimeline(t, localTimeline)
		}
		timeline[node] = localTimeline
	}
	return timeline
}

type extractor struct {
	regexes      []regex.LogRegex
	path         string
	since, until *time.Time
}

func newExtractor(path string, toCheck []regex.LogRegex, since, until *time.Time) extractor {
	return extractor{regexes: toCheck, path: path, since: since, until: until}
}

func (e *extractor) grepArgument() string {

	regexToSendSlice := []string{}
	for _, regex := range e.regexes {
		regexToSendSlice = append(regexToSendSlice, regex.Regex.String())
	}
	var grepRegex string
	if e.since != nil || e.until != nil {
		grepRegex += "(" + regex.BetweenDateRegex(e.since, e.until) + "|" + regex.NoDatesRegex() + ").*"
	}
	return "(" + strings.Join(regexToSendSlice, "|") + ")"
}

// search is the main function to search what we want in a file
func (e *extractor) search() (string, types.LocalTimeline, error) {

	// A first pass is done, with every regexes we want compiled in a single one.
	grepRegex := e.grepArgument()

	// Regular grep is actually used
	// There are no great alternatives, even less as golang libraries. grep itself do not have great alternatives: they are less performant for common use-cases, or are not easily portable, or are costlier to execute.
	// grep is everywhere, grep is good enough, it even enable to use the stdout pipe.
	// The usual bottleneck with grep is that it is single-threaded, but we actually benefit from a sequential scan here as we will rely on the log order. Being sequential also ensure this program is light enough to run without too much impacts
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

	// If we found anything
	if len(lt) > 0 {
		// identify the node with the easiest to read information
		return types.DisplayLocalNodeSimplestForm(lt[len(lt)-1].Ctx), lt, nil
	}
	return filepath.Base(e.path), lt, nil
}

func (e *extractor) iterateOnResults(s *bufio.Scanner) ([]types.LogInfo, error) {

	var (
		line         string
		li           types.LogInfo
		lt           []types.LogInfo
		recentEnough bool
		err          error
	)
	ctx := types.NewLogCtx()
	ctx.FilePath = filepath.Base(e.path)

	for s.Scan() {
		line = s.Text()

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

		ctx, li, err = e.infoAndContextFromLine(ctx, line, date)
		if err != nil {
			return nil, err // even though we could actually tolerate errors
		}
		lt = append(lt, li)

	}
	return lt, nil
}

func (e *extractor) infoAndContextFromLine(ctx types.LogCtx, log string, date types.Date) (types.LogCtx, types.LogInfo, error) {

	// We have to find again what regex worked to get this log line
	for _, regex := range e.regexes {
		if !regex.Regex.MatchString(log) {
			continue
		}
		updatedCtx, displayer := regex.Handle(ctx, log)
		li := types.LogInfo{
			Date:      date,
			Log:       log,
			Msg:       displayer,
			Ctx:       updatedCtx,
			RegexType: regex.Type,
			Verbosity: regex.Verbosity,
		}
		return updatedCtx, li, nil
	}

	return types.LogCtx{}, types.LogInfo{}, errors.Errorf("Could not find regex again: %s", log)
}
