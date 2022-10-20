package main

import (
	"bufio"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
)

var CLI struct {
	List struct {
		Paths      []string `arg:"" name:"paths" help:"paths of the log to use"`
		ListStates bool
		ListViews  bool
		ListSST    bool
	} `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "list <paths>":
		toCheck := listingChecks()
		timeline := make(Timeline)

		for _, path := range CLI.List.Paths {
			node, localTimeline, err := search(path, toCheck...)
			if err != nil {
				log.Println(err)
			}

			// TODO: merge timelines if the nodes already exists
			timeline[node] = localTimeline
		}

		DisplayColumnar(timeline)
	default:
		log.Fatal("Command not known:", ctx.Command())
	}
}

func listingChecks() []LogRegex {
	toCheck := []LogRegex{RegexSourceNode}
	if CLI.List.ListStates {
		toCheck = append(toCheck, RegexShift)
	}
	if CLI.List.ListViews {
		toCheck = append(toCheck, []LogRegex{RegexNodeEstablied, RegexNodeJoined, RegexNodeLeft}...)
	}
	return toCheck
}

// It should be kept already sorted by timestamp
type LocalTimeline []LogInfo

// "string" key is a node IP
type Timeline map[string]LocalTimeline

type LogInfo struct {
	Date time.Time
	Msg  string // what to show
	Log  string // the raw log
}

type LogCtx struct {
	SourceNodeIP     string
	IsStarted        bool
	IsInRecovery     bool
	ResyncingNode    string
	ResyncedFromNode string
	HashToIP         map[string]string
	IPToHostname     map[string]string
	IPToMethod       map[string]string
}

func search(path string, regexes ...LogRegex) (string, LocalTimeline, error) {
	lt := []LogInfo{}
	ctx := LogCtx{HashToIP: map[string]string{}, IPToHostname: map[string]string{}, IPToMethod: map[string]string{}}

	// A first pass is done, with every regexes we want compiled. We will iterate on this one later
	regexToSendSlice := []string{}
	for _, regex := range regexes {
		regexToSendSlice = append(regexToSendSlice, regex.Regex)
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s := bufio.NewScanner(out)
		var (
			line      string
			toDisplay string
		)

	SequentialScan:
		for s.Scan() {
			line = s.Text()
			toDisplay = line
			t := searchDateFromLog(line)

			for _, regex := range regexes {
				r := regexp.MustCompile(regex.Regex)
				if !r.Match([]byte(line)) {
					continue
				}
				if regex.Handler != nil {
					ctx, toDisplay = regex.Handler(ctx, line)
				}
				if regex.SkipPrint {
					continue SequentialScan
				}
				lt = append(lt, LogInfo{
					Date: t,
					Log:  line,
					Msg:  toDisplay,
				})
			}
		}
		wg.Done()
	}()

	wg.Wait()
	return ctx.SourceNodeIP, lt, nil
}

func searchDateFromLog(log string) time.Time {
	for _, layout := range DateLayouts {
		t, err := time.Parse(layout, log[:len(layout)])
		if err == nil {
			return t
		}
	}
	return time.Time{}
}
