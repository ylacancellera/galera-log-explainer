package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

func iterateNode(timeline Timeline) ([]string, time.Time) {
	var (
		nextDate  time.Time
		nextNodes []string
	)
	nextDate = time.Unix(math.MaxInt32, 0)
	for node := range timeline {
		if len(timeline[node]) == 0 {
			continue
		}
		curDate := timeline[node][0].Date
		if curDate.Before(nextDate) {
			nextDate = curDate
			nextNodes = []string{node}
		} else if curDate.Equal(nextDate) {
			nextNodes = append(nextNodes, node)
		}
	}
	return nextNodes, nextDate
}

func DisplayColumnar(timeline Timeline) {
	var (
		lastDate time.Time
		args     []string
	)

	w := tabwriter.NewWriter(os.Stdout, 8, 8, 3, ' ', tabwriter.AlignRight)
	defer w.Flush()

	// keys will be used to access the timeline map with an ordered manner
	keys := make([]string, 0, len(timeline))
	for node := range timeline {
		keys = append(keys, node)
	}
	sort.Strings(keys)

	fmt.Fprintln(w, "DATE\t"+strings.Join(keys, "\t")+"\t")
	fmt.Fprintln(w, " \t"+strings.Repeat(" \t", len(keys)))

	for nextNodes, nextDate := iterateNode(timeline); len(nextNodes) != 0; nextNodes, nextDate = iterateNode(timeline) {

		if nextDate.Truncate(time.Second).Equal(lastDate.Truncate(time.Second)) {
			args = []string{nextDate.Format(".000000Z")}
		} else {
			args = []string{nextDate.Format("2006-01-02 15:04:05.000000Z")}
		}

	MakeLine:
		for _, node := range keys {

			for _, nextNode := range nextNodes {

				if node == nextNode {
					nl := timeline[nextNode][0]
					args = append(args, nl.Msg)
					if len(timeline[nextNode]) > 0 {
						timeline[nextNode] = timeline[nextNode][1:]

					}
					continue MakeLine
				}
			}
			args = append(args, "| ")

		}
		_, err := fmt.Fprintln(w, strings.Join(args, "\t")+"\t")
		if err != nil {
			panic(err)
		}

		lastDate = nextDate
		_ = lastDate

	}
}
