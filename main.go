package main

import (
	"bufio"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
)

var CLI struct {
	NoColor bool
	List    struct {
		Paths                  []string   `arg:"" name:"paths" help:"paths of the log to use"`
		Verbosity              Verbosity  `default:"1" help:"0: Info, 1: Detailed, 2: DebugMySQL (every mysql info the tool used), 3: Debug (internal tool debug)"`
		SkipStateColoredColumn bool       `help:"avoid having the placeholder colored with mysql state, which is guessed using several regexes that will not be displayed"`
		ListStates             bool       `help:"List WSREP state changes(SYNCED, DONOR, ...)"`
		ListViews              bool       `help:"List how Galera views evolved (who joined, who left)"`
		ListEvents             bool       `help:"List generic mysql events (start, shutdown, assertion failures)"`
		ListSST                bool       `help:"List Galera synchronization event"`
		GroupByTime            bool       `default:"false" help:"Avoid printing complete date to highlight which events happened close to each others. eg: if two events happened the same minute, only show the seconds part (unstable, only works with UTC rfc3339 micro format, as in 2006-01-02T15:04:05.000000Z)"`
		Since                  *time.Time `help:"Only list events after this date, you can copy-paste a date from mysql error log"`
		Until                  *time.Time `help:"Only list events before this date, you can copy-paste a date from mysql error log"`
	} `cmd:""`
	Metadata struct {
		Paths []string `arg:"" name:"paths" help:"paths of the log to use"`
	} `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "list <paths>":
		// IdentRegexes is always needed: we would not be able to identify the node where the file come from
		toCheck := IdentRegexes
		if CLI.List.ListStates {
			toCheck = append(toCheck, StatesRegexes...)
		} else if !CLI.List.SkipStateColoredColumn {
			toCheck = append(toCheck, SetVerbosity(DebugMySQL, StatesRegexes...)...)
		}
		if CLI.List.ListViews {
			toCheck = append(toCheck, ViewsRegexes...)
		}
		if CLI.List.ListEvents {
			toCheck = append(toCheck, EventsRegexes...)
		} else if !CLI.List.SkipStateColoredColumn {
			toCheck = append(toCheck, SetVerbosity(DebugMySQL, EventsRegexes...)...)
		}
		timeline := createTimeline(CLI.List.Paths, toCheck)
		DisplayColumnar(timeline)

	case "metadata <paths>":
		toCheck := append(append([]LogRegex{RegexSourceNode}, StatesRegexes...), ViewsRegexes...)
		timeline := createTimeline(CLI.Metadata.Paths, toCheck)
		printMetadata(timeline)

	default:
		log.Fatal("Command not known:", ctx.Command())
	}
}

// It should be kept already sorted by timestamp
type LocalTimeline []LogInfo

// "string" key is a node IP
type Timeline map[string]LocalTimeline

// LogInfo is to store a single event in log. This is something that should be displayed ultimately, this is what we want when we launch this tool
type LogInfo struct {
	Date       time.Time
	DateLayout string       // Per LogInfo and not global, because it could be useful in case a major version upgrade happened
	Msg        LogDisplayer // what to show
	Log        string       // the raw log
	Ctx        LogCtx       // the context is copied for each logInfo, so that it is easier to handle some info (current state), and this is also interesting to check how it evolved
	Verbosity  Verbosity
}

// LogCtx is a context for a given file.
// It used to keep track of what is going on at each new event.
type LogCtx struct {
	FilePath         string
	SourceNodeIP     []string
	State            string
	ResyncingNode    string
	ResyncedFromNode string
	OwnHashes        []string
	HashToIP         map[string]string
	HashToNodeName   map[string]string
	IPToHostname     map[string]string
	IPToMethod       map[string]string
	IPToNodeName     map[string]string
}

func newLogCtx() LogCtx {
	return LogCtx{HashToIP: map[string]string{}, IPToHostname: map[string]string{}, IPToMethod: map[string]string{}, IPToNodeName: map[string]string{}, HashToNodeName: map[string]string{}}
}

func createTimeline(paths []string, toCheck []LogRegex) Timeline {
	timeline := make(Timeline)

	for _, path := range paths {
		node, localTimeline, err := search(path, toCheck...)
		if err != nil {
			log.Println(err)
		}

		if t, ok := timeline[node]; ok {
			localTimeline = mergeTimeline(t, localTimeline)
		}
		timeline[node] = localTimeline
	}
	return timeline
}

// mergeTimeline is helpful when log files are split by date, it can be useful to be able to merge content
// a "timeline" come from a log file. Log files that came from some node should not never have overlapping dates
func mergeTimeline(t1, t2 LocalTimeline) LocalTimeline {
	if len(t1) == 0 {
		return t2
	}
	if len(t2) == 0 {
		return t1
	}
	if t1[0].Date.Before(t2[0].Date) {
		return append(t1, t2...)
	}
	return append(t2, t1...)
}

// search is the main function to search what we want in a file
func search(path string, regexes ...LogRegex) (string, LocalTimeline, error) {

	// A first pass is done, with every regexes we want compiled in a single one.
	regexToSendSlice := []string{}
	for _, regex := range regexes {
		regexToSendSlice = append(regexToSendSlice, regex.Regex.String())
	}
	var grepRegex string
	if CLI.List.Since != nil || CLI.List.Until != nil {
		grepRegex += "(" + BetweenDateRegex(CLI.List.Since, CLI.List.Until) + "|" + NoDatesRegex() + ").*"
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
		displayer    LogDisplayer
		recentEnough bool
	)
	ctx := newLogCtx()
	ctx.FilePath = path
	lt := []LogInfo{}

	// Scan for each grep results
SCAN:
	for s.Scan() {
		line = s.Text()
		t, dateLayout := searchDateFromLog(line)

		// If it's recentEnough, it means we already validated a log: every next logs necessarily happened later
		// this is useful because not every logs have a date attached, and some without date are very useful
		if CLI.List.Since != nil && !recentEnough && CLI.List.Since.After(t) {
			continue
		}
		if CLI.List.Until != nil && CLI.List.Until.Before(t) {
			break SCAN
		}
		recentEnough = true

		// We have to find again what regex worked to get this log line
		for _, regex := range regexes {
			if !regex.Regex.MatchString(line) {
				continue
			}
			if regex.Handler == nil {
				continue
			}
			ctx, displayer = regex.Handler(ctx, line)
			lt = append(lt, LogInfo{
				Date:       t,
				DateLayout: dateLayout,
				Log:        line,
				Msg:        displayer,
				Ctx:        ctx,
				Verbosity:  regex.Verbosity,
			})
		}
	}

	if len(lt) > 0 {
		return DisplayLocalNodeSimplestForm(lt[len(lt)-1].Ctx), lt, nil
	}
	return path, lt, nil
}

func searchDateFromLog(log string) (time.Time, string) {
	for _, layout := range DateLayouts {
		t, err := time.Parse(layout, log[:len(layout)])
		if err == nil {
			return t, layout
		}
	}
	return time.Time{}, ""
}
