package main

import (
	"github.com/pkg/errors"
	"github.com/ylacancellera/galera-log-explainer/display"
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

type list struct {
	// Paths is duplicated because it could not work as variadic with kong cli if I set it as CLI object
	Paths                  []string `arg:"" name:"paths" help:"paths of the log to use"`
	Format                 string   `help:"Types of output format" enum:"cli,svg" default:"cli"`
	SkipStateColoredColumn bool     `help:"avoid having the placeholder colored with mysql state, which is guessed using several regexes that will not be displayed"`
	All                    bool     `help:"List everything" xor:"states,views,events,sst"`
	States                 bool     `help:"List WSREP state changes(SYNCED, DONOR, ...)" xor:"states"`
	Views                  bool     `help:"List how Galera views evolved (who joined, who left)" xor:"views"`
	Events                 bool     `help:"List generic mysql events (start, shutdown, assertion failures)" xor:"events"`
	SST                    bool     `help:"List Galera synchronization event" xor:"sst"`
}

func (l *list) Help() string {
	return `List events for each nodes

Usage:
	galera-log-explainer list --all <list of files>
	galera-log-explainer list --all *.log
	galera-log-explainer list --sst --views --states <list of files>
	galera-log-explainer list --events --views *.log
	`
}

func (l *list) Run() error {

	if !(l.All || l.Events || l.States || l.SST || l.Views) {
		return errors.New("Please select a type of logs to search: --all, or any parameters from: --sst --views --events --states")
	}

	toCheck := l.regexesToUse()
	if CLI.List.Format == "svg" {
		// svg text does not handle cli special characters
		utils.SkipColor = true
	}

	timeline, err := timelineFromPaths(CLI.List.Paths, toCheck, CLI.Since, CLI.Until)
	if err != nil {
		return errors.Wrap(err, "Could not list events")
	}

	switch CLI.List.Format {
	case "cli":
		display.TimelineCLI(timeline, CLI.Verbosity)
		break
	case "svg":
		display.TimelineSVG(timeline, CLI.Verbosity)
	}

	return nil
}

func (l *list) regexesToUse() types.RegexMap {

	// IdentRegexes is always needed: we would not be able to identify the node where the file come from
	toCheck := regex.IdentsMap
	if CLI.List.States || CLI.List.All {
		toCheck.Merge(regex.StatesMap)
	} else if !CLI.List.SkipStateColoredColumn {
		regex.SetVerbosity(types.DebugMySQL, regex.StatesMap)
		toCheck.Merge(regex.StatesMap)
	}
	if CLI.List.Views || CLI.List.All {
		toCheck.Merge(regex.ViewsMap)
	}
	if CLI.List.SST || CLI.List.All {
		toCheck.Merge(regex.SSTMap)
	}
	if CLI.List.Events || CLI.List.All {
		toCheck.Merge(regex.EventsMap)
	} else if !CLI.List.SkipStateColoredColumn {
		regex.SetVerbosity(types.DebugMySQL, regex.EventsMap)
		toCheck.Merge(regex.EventsMap)
	}
	return toCheck
}
