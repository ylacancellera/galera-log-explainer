package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
)

func sedHandler() error {
	toCheck := append(regex.IdentRegexes, regex.SetVerbosity(types.DebugMySQL, regex.ViewsRegexes...)...)
	timeline := createTimeline(CLI.Sed.Paths, toCheck)
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

func sedByName(ni NodeInfo) []string {
	if len(ni.NodeNames) == 0 {
		return nil
	}
	elem := ni.NodeNames[0]
	args := sedSliceWith(ni.NodeUUIDs, elem)
	args = append(args, sedSliceWith(ni.IPs, elem)...)
	return args
}

func sedByIP(ni NodeInfo) []string {
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
