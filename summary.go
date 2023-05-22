package main

import (
	"fmt"
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

	timeline, err := timelineFromPaths(s.Paths, toCheck, CLI.Since, CLI.Until)
	if err != nil {
		return errors.Wrap(err, "Could not get summary")
	}
	groupedEvent := groupEvents(timeline, "RegexNewComponent")
	fmt.Println("base")
	fmt.Println(groupedEvent.Base.Msg(groupedEvent.Base.Ctx))
	fmt.Println()
	for node, m := range groupedEvent.Proofs {
		fmt.Println("proof from", node)
		fmt.Println(m.Log)
		fmt.Println()
	}
	//_ = groupedEvents

	groups := []types.GroupedEvent{}
	_ = groups
	return nil
}

func groupEvents(timeline types.Timeline, groupWith string) types.GroupedEvent {

	group := types.GroupedEvent{Base: types.LogInfo{Date: types.Date{Time: time.Date(2100, time.January, 1, 1, 1, 1, 1, time.UTC)}}, Proofs: map[string]types.LogInfo{}}

	for node := range timeline {
		stack := groupEventsFromLocalTimeline(timeline[node], groupWith)
		if group.Base.Date.Time.After(stack[len(stack)-1].Date.Time) {
			group.Base = stack[len(stack)-1]
		}
		group.Proofs[node] = stack[len(stack)-1]
	}
	return group
}

func groupEventsFromLocalTimeline(lt types.LocalTimeline, groupWith string) []types.LogInfo {

	for i, li := range lt {
		if li.RegexUsed == groupWith {
			return lt[:i+1]
		}
	}
	return []types.LogInfo{}
}
