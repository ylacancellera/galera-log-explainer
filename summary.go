package main

import (
	"time"

	"github.com/pkg/errors"
	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
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
		return errors.New("unimplemented")
	}

	timeline, _ := timelineFromPaths(CLI.List.Paths, toCheck, CLI.Since, CLI.Until)
	_ = timeline
	//groupedEvents := GroupEventsFromTimeline(timeline, types.ViewsRegexType)
	//_ = groupedEvents

	groups := []types.GroupedEvent{}
	_ = groups
	return nil
}

func groupEvents(timeline types.Timeline, groupWith types.RegexType) {

	group := types.GroupedEvent{Base: types.LogInfo{Date: types.Date{Time: time.Date(2100, time.January, 1, 1, 1, 1, 1, time.UTC)}}}

	for node := range timeline {
		stack := groupEventsFromLocalTimeline(timeline[node], groupWith)
		if group.Base.Date.Time.After(stack[len(stack)-1].Date.Time) {
			group.Base = stack[len(stack)-1]
		}
		group.Proofs[node] = stack[len(stack)-1]
	}
}

func groupEventsFromLocalTimeline(lt types.LocalTimeline, groupWith types.RegexType) []types.LogInfo {

	for i, li := range lt {
		if li.RegexType == groupWith {
			return lt[:i]
		}
	}
	return []types.LogInfo{}
}
