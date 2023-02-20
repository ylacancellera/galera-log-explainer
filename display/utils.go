package display

import (
	"math"
	"time"

	"github.com/ylacancellera/galera-log-explainer/types"
)

// iterateNode is used to search the source node(s) that contains the next chronological events
// it returns a slice in case 2 nodes have their next event precisely at the same time, which
// happens a lot on some versions
func iterateNode(timeline types.Timeline) []string {
	var (
		nextDate  time.Time
		nextNodes []string
	)
	nextDate = time.Unix(math.MaxInt32, 0)
	for node := range timeline {
		if len(timeline[node]) == 0 {
			continue
		}
		curDate := timeline[node][0].Date.Time
		if curDate.Before(nextDate) {
			nextDate = curDate
			nextNodes = []string{node}
		} else if curDate.Equal(nextDate) {
			nextNodes = append(nextNodes, node)
		}
	}
	return nextNodes
}
