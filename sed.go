package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
)

type sed struct {
	Paths []string `arg:"" name:"paths" help:"paths of the log to use"`
	ByIP  bool     `help:"Replace by IP instead of name"`
}

func (s *sed) Help() string {
	return `sed translates a log, replacing node UUID, IPS, names with either name or IP everywhere. By default it replaces by name.

Use like so:
	cat node1.log | galera-log-explainer sed *.log | less
	galera-log-explainer sed *.log < node1.log | less

You can also simply call the command to get a generated sed command to review and apply yourself
	galera-log-explainer sed *.log`
}

func (s *sed) Run() error {
	regex.SetVerbosity(types.DebugMySQL, regex.ViewsMap)
	toCheck := regex.IdentsMap.Merge(regex.ViewsMap)
	timeline, err := timelineFromPaths(s.Paths, toCheck, CLI.Since, CLI.Until)
	if err != nil {
		return errors.Wrap(err, "Found nothing worth replacing")
	}
	ctxs := timeline.GetLatestUpdatedContextsByNodes()

	args := []string{}
	for key, ctx := range ctxs {

		tosearchs := []string{key}
		tosearchs = append(tosearchs, ctx.OwnHashes...)
		tosearchs = append(tosearchs, ctx.OwnIPs...)
		tosearchs = append(tosearchs, ctx.OwnNames...)
		for _, tosearch := range tosearchs {
			ni := whoIs(ctxs, tosearch)

			switch {
			case CLI.Sed.ByIP:
				args = append(args, sedByIP(ni)...)
			default:
				args = append(args, sedByName(ni)...)
			}
		}

	}
	if len(args) == 0 {
		return errors.New("Could not find informations to replace")
	}

	fstat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if fstat.Size() == 0 {
		fmt.Println("No files found in stdin, returning the sed command instead:")
		fmt.Println("sed", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command("sed", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func sedByName(ni types.NodeInfo) []string {
	if len(ni.NodeNames) == 0 {
		return nil
	}
	elem := ni.NodeNames[0]
	args := sedSliceWith(ni.NodeUUIDs, elem)
	args = append(args, sedSliceWith(ni.IPs, elem)...)
	return args
}

func sedByIP(ni types.NodeInfo) []string {
	if len(ni.IPs) == 0 {
		return nil
	}
	elem := ni.IPs[0]
	args := sedSliceWith(ni.NodeUUIDs, elem)
	args = append(args, sedSliceWith(ni.NodeNames, elem)...)
	return args
}

func sedSliceWith(elems []string, replace string) []string {
	args := []string{}
	for _, elem := range elems {
		args = append(args, "-e")
		args = append(args, "s/"+elem+"/"+replace+"/g")
	}
	return args
}
