package main

import (
	"bufio"
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
		Paths      []string `arg:"" name:"path" help:"paths of the log to use"`
		ListStates bool
		ListViews  bool
		ListSST    bool
	} `cmd:""`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "list <path>":
		toCheck := []LogRegex{RegexSourceNode}
		if CLI.List.ListStates {
			toCheck = append(toCheck, RegexShift)
		}
		if CLI.List.ListViews {
			toCheck = append(toCheck, []LogRegex{RegexNodeEstablied, RegexNodeJoined, RegexNodeLeft}...)
		}

		timeline := make(Timeline)
		_ = timeline
		for _, path := range CLI.List.Paths {
			node, localTimeline, err := search(path, toCheck...)
			if err != nil {
				panic(err)
			}

			timeline[node] = localTimeline
		}

		DisplayColumnar(timeline)
		break
	default:
		panic(ctx.Command())
	}

}

//type LocalTimeline map[int64][]LogInfo
type LocalTimeline []LogInfo
type Timeline map[string]LocalTimeline

type LogInfo struct {
	Date time.Time
	Msg  string
	Log  string
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

	regexToSendSlice := []string{}
	for _, regex := range regexes {
		regexToSendSlice = append(regexToSendSlice, regex.Regex)
	}
	grepRegex := "(" + strings.Join(regexToSendSlice, "|") + ")"

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

		for s.Scan() {
			line = s.Text()
			toDisplay = line
			var t time.Time

		SearchDate:
			for _, layout := range DateLayouts {
				t, err = time.Parse(layout, line[:len(layout)])
				if err == nil {
					break SearchDate
				}
			}

			for _, regex := range regexes {
				r := regexp.MustCompile(regex.Regex)
				if !r.Match([]byte(line)) {
					continue
				}
				if regex.Handler != nil {
					ctx, toDisplay = regex.Handler(ctx, line)
				}
				if !regex.SkipPrint {
					lt = append(lt, LogInfo{
						Date: t,
						Log:  line,
						Msg:  toDisplay,
					})
				}
			}
		}
		wg.Done()
	}()

	wg.Wait()
	return ctx.SourceNodeIP, lt, nil
}
