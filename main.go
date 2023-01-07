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
	"github.com/ylacancellera/galera-log-explainer/display"
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var CLI struct {
	NoColor bool
	List    struct {
		Paths                  []string        `arg:"" name:"paths" help:"paths of the log to use"`
		Verbosity              types.Verbosity `default:"1" help:"0: Info, 1: Detailed, 2: DebugMySQL (every mysql info the tool used), 3: Debug (internal tool debug)"`
		SkipStateColoredColumn bool            `help:"avoid having the placeholder colored with mysql state, which is guessed using several regexes that will not be displayed"`
		States                 bool            `help:"List WSREP state changes(SYNCED, DONOR, ...)"`
		Views                  bool            `help:"List how Galera views evolved (who joined, who left)"`
		Events                 bool            `help:"List generic mysql events (start, shutdown, assertion failures)"`
		SST                    bool            `help:"List Galera synchronization event"`
		Since                  *time.Time      `help:"Only list events after this date, you can copy-paste a date from mysql error log"`
		Until                  *time.Time      `help:"Only list events before this date, you can copy-paste a date from mysql error log"`
	} `cmd:""`
	Metadata struct {
		Paths []string `arg:"" name:"paths" help:"paths of the log to use"`
	} `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI)

	utils.SkipColor = CLI.NoColor
	switch ctx.Command() {
	case "list <paths>":
		// IdentRegexes is always needed: we would not be able to identify the node where the file come from
		toCheck := regex.IdentRegexes
		if CLI.List.States {
			toCheck = append(toCheck, regex.StatesRegexes...)
		} else if !CLI.List.SkipStateColoredColumn {
			toCheck = append(toCheck, regex.SetVerbosity(types.DebugMySQL, regex.StatesRegexes...)...)
		}
		if CLI.List.Views {
			toCheck = append(toCheck, regex.ViewsRegexes...)
		}
		if CLI.List.Events {
			toCheck = append(toCheck, regex.EventsRegexes...)
		} else if !CLI.List.SkipStateColoredColumn {
			toCheck = append(toCheck, regex.SetVerbosity(types.DebugMySQL, regex.EventsRegexes...)...)
		}
		timeline := createTimeline(CLI.List.Paths, toCheck)
		display.DisplayColumnar(timeline, CLI.List.Verbosity)

	case "metadata <paths>":
		toCheck := append(append(regex.IdentRegexes, regex.StatesRegexes...), regex.ViewsRegexes...)
		timeline := createTimeline(CLI.Metadata.Paths, toCheck)
		display.PrintMetadata(timeline)

	default:
		log.Fatal("Command not known:", ctx.Command())
	}
}

func createTimeline(paths []string, toCheck []regex.LogRegex) types.Timeline {
	timeline := make(types.Timeline)

	for _, path := range paths {
		node, localTimeline, err := search(path, toCheck...)
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

// search is the main function to search what we want in a file
func search(path string, regexes ...regex.LogRegex) (string, types.LocalTimeline, error) {

	// A first pass is done, with every regexes we want compiled in a single one.
	regexToSendSlice := []string{}
	for _, regex := range regexes {
		regexToSendSlice = append(regexToSendSlice, regex.Regex.String())
	}
	var grepRegex string
	if CLI.List.Since != nil || CLI.List.Until != nil {
		grepRegex += "(" + regex.BetweenDateRegex(CLI.List.Since, CLI.List.Until) + "|" + regex.NoDatesRegex() + ").*"
	}
	grepRegex += "(" + strings.Join(regexToSendSlice, "|") + ")"

	// Regular grep is actually used
	// There are no great alternatives, even less as golang libraries. grep itself do not have great alternatives: they are less performant for common use-cases, or are not easily portable, or are costlier to execute.
	// grep is everywhere, grep is good enough, it even enable to use the stdout pipe.
	// The usual bottleneck with grep is that it is single-threaded, but we actually benefit from a sequential scan here as we will rely on the log order. Being sequential also ensure this program is light enough to run without too much impacts
	cmd := exec.Command("grep", "-P", grepRegex, path)
	out, _ := cmd.StdoutPipe()
	defer out.Close()
	err := cmd.Start()
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to search in %s", path)
	}

	// grep treatment
	s := bufio.NewScanner(out)
	var (
		line         string
		displayer    types.LogDisplayer
		recentEnough bool
	)
	ctx := types.NewLogCtx()
	ctx.FilePath = filepath.Base(path)
	lt := []types.LogInfo{}

	// Scan for each grep results
SCAN:
	for s.Scan() {
		line = s.Text()
		date := types.NewDate(regex.SearchDateFromLog(line))

		// If it's recentEnough, it means we already validated a log: every next logs necessarily happened later
		// this is useful because not every logs have a date attached, and some without date are very useful
		if CLI.List.Since != nil && !recentEnough && CLI.List.Since.After(date.Time) {
			continue
		}
		if CLI.List.Until != nil && CLI.List.Until.Before(date.Time) {
			break SCAN
		}
		recentEnough = true

		// We have to find again what regex worked to get this log line
		for _, regex := range regexes {
			if !regex.Regex.MatchString(line) {
				continue
			}
			ctx, displayer = regex.Handle(ctx, line)
			lt = append(lt, types.LogInfo{
				Date:      date,
				Log:       line,
				Msg:       displayer,
				Ctx:       ctx,
				Verbosity: regex.Verbosity,
			})
		}
	}

	if len(lt) > 0 {
		return types.DisplayLocalNodeSimplestForm(lt[len(lt)-1].Ctx), lt, nil
	}
	return filepath.Base(path), lt, nil
}
