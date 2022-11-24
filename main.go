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
		Paths                  []string  `arg:"" name:"paths" help:"paths of the log to use"`
		Verbosity              Verbosity `default:"1"`
		PrintMetadata          bool
		SkipStateColoredColumn bool
		ListStates             bool
		ListViews              bool
		ListEvents             bool
		ListSST                bool
	} `cmd:""`
	Metadata struct {
		Paths []string `arg:"" name:"paths" help:"paths of the log to use"`
	} `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "list <paths>":
		// RegexSourceNode is always needed: we would not be able to identify the node where the file come from
		toCheck := []LogRegex{RegexSourceNode}
		if CLI.List.ListStates {
			toCheck = append(toCheck, StatesRegexes...)
		} else if !CLI.List.SkipStateColoredColumn {
			toCheck = append(toCheck, SilenceRegex(StatesRegexes...)...)
		}
		if CLI.List.ListViews {
			toCheck = append(toCheck, ViewsRegexes...)
		}
		if CLI.List.ListEvents {
			toCheck = append(toCheck, EventsRegexes...)
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
	DateLayout string // Per LogInfo and not global, because it could be useful in case a major version upgrade happened sometime
	Msg        string // what to show
	Log        string // the raw log
	Ctx        LogCtx // the context is copied for each logInfo, so that it is easier to handle some info (current state), and this is also interesting how it evolved
}

// LogCtx is a context for a given file.
// It used to keep track of what is going on at each new event.
type LogCtx struct {
	FilePath         string
	SourceNodeIP     string
	State            string
	ResyncingNode    string
	ResyncedFromNode string
	HashToIP         map[string]string
	IPToHostname     map[string]string
	IPToMethod       map[string]string
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
	grepRegex := "(" + strings.Join(regexToSendSlice, "|") + ")"

	// Regular grep is actually used
	// There are no great alternatives, even less as golang libraries. grep itself do not have great alternatives: they are less performant for common use-cases, or are not easily portable, or are costlier to execute.
	// grep is everywhere, grep is good enough, it even enable to use the stdout pipe.
	// The usual bottleneck with grep is that it is single-threaded, but we actually benefit from a sequential scan here as we will rely on the log order. Being sequential also ensure this program is light enough to run without too much impacts
	cmd := exec.Command("grep", "-P", grepRegex, path)
	out, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to search in %s", path)
	}

	// grep treatment
	s := bufio.NewScanner(out)
	var (
		line      string
		toDisplay string
	)
	lt := []LogInfo{}
	ctx := LogCtx{FilePath: path, HashToIP: map[string]string{}, IPToHostname: map[string]string{}, IPToMethod: map[string]string{}}

	// Scan for each grep results
	for s.Scan() {
		line = s.Text()
		toDisplay = line
		t, dateLayout := searchDateFromLog(line)

		// We have to find again what regex worked to get this log line
		for _, regex := range regexes {
			if !regex.Regex.Match([]byte(line)) {
				continue
			}
			if regex.Handler == nil {
				continue
			}
			ctx, toDisplay = regex.Handler(ctx, line)
			if CLI.List.Verbosity < regex.Verbosity || regex.SkipPrint {
				continue
			}
			lt = append(lt, LogInfo{
				Date:       t,
				DateLayout: dateLayout,
				Log:        line,
				Msg:        toDisplay,
				Ctx:        ctx,
			})
		}
	}

	source := ctx.SourceNodeIP
	if source == "" {
		source = path
	}
	return source, lt, nil
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
