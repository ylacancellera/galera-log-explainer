package main

import (
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

type summary struct {
	list
}

func (s *summary) Run() error {

	toCheck := regex.AllRegexes()

	if CLI.Summary.Format == "svg" {
		// svg text does not handle cli special characters
		utils.SkipColor = true
	}

	timeline := timelineFromPaths(CLI.List.Paths, toCheck, CLI.Since, CLI.Until)
	_ = timeline

	/*switch CLI.List.Format {
	case "cli":
		display.DisplayColumnar(timeline, CLI.List.Verbosity)
		break
	case "svg":
		display.Svg(timeline, CLI.List.Verbosity)
	}
	*/
	return nil
}
