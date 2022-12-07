package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	// regular tabwriter do not work with color, this is a forked versions that ignores color special characters
	"github.com/Ladicle/tabwriter"
)

// iterateNode is used to search the source node(s) that contains the next chronological events
// it returns a slice in case 2 nodes have their next event precisely at the same time, which
// happens a lot on some versions
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

// DisplayColumnar is the main function to print
// It will print header and footers, and dequeue the timeline chronologically
func DisplayColumnar(timeline Timeline) {
	var (
		lastDate   time.Time
		lastLayout string
		args       []string
	)
	// to hold the current context for each node
	keys, currentContext := initKeysContext(timeline)
	lastContext := map[string]LogCtx{}

	w := tabwriter.NewWriter(os.Stdout, 8, 8, 3, ' ', tabwriter.AlignRight)
	defer w.Flush()

	// header
	fmt.Fprintln(w, headerNodes(keys))
	fmt.Fprintln(w, headerFilePath(keys, currentContext))
	fmt.Fprintln(w, headerHostname(keys, currentContext))
	fmt.Fprintln(w, separator(keys))

	// as long as there is a next event to print
	for nextNodes, nextDate := iterateNode(timeline); len(nextNodes) != 0; nextNodes, nextDate = iterateNode(timeline) {

		// Date column
		dateCol, tmpLastLayout := dateBlock(nextDate, lastDate, timeline[nextNodes[0]][0].DateLayout, lastLayout)
		args = []string{dateCol}
		displayedValue := 0

		// node values
		for _, node := range keys {

			if !sliceContains(nextNodes, node) {
				// if there are no events, having a | is needed for tabwriter
				// A few color can also help highlighting how the node is doing
				args = append(args, defaultColumnValue("| ", currentContext[node].State))
				continue
			}
			nl := timeline[node][0]
			lastContext[node] = currentContext[node]
			currentContext[node] = nl.Ctx

			// dequeue the events
			if len(timeline[node]) > 0 {
				timeline[node] = timeline[node][1:]
			}

			if CLI.List.Verbosity > nl.Verbosity {
				args = append(args, nl.Msg)
				displayedValue++
			} else {
				args = append(args, defaultColumnValue("| ", nl.Ctx.State))
			}
		}

		if sep := fileTransitionSeparator(keys, lastContext, currentContext); sep != "" {
			fmt.Fprintln(w, sep)
		}

		// If line is not filled with default placeholder values
		if displayedValue == 0 {
			continue

		}

		// Print tabwriter line
		_, err := fmt.Fprintln(w, strings.Join(args, "\t")+"\t")
		if err != nil {
			log.Println("Failed to write a line", err)
		}

		// a tmp value was stored because we could not know if it was going to be displayed yet
		lastLayout = tmpLastLayout
		lastDate = nextDate

	}

	// footer
	// only having a header is not fast enough to read when there are too many lines
	fmt.Fprintln(w, separator(keys))
	fmt.Fprintln(w, headerNodes(keys))
	fmt.Fprintln(w, headerFilePath(keys, currentContext))
	fmt.Fprintln(w, headerHostname(keys, currentContext))
}

func initKeysContext(timeline Timeline) ([]string, map[string]LogCtx) {
	currentContext := map[string]LogCtx{}

	// keys will be used to access the timeline map with an ordered manner
	// without this, we would not print on the correct column as the order of a map is guaranteed to be random each time
	keys := make([]string, 0, len(timeline))
	for node := range timeline {
		keys = append(keys, node)
		if len(timeline[node]) > 0 {
			currentContext[node] = timeline[node][0].Ctx
		} else {
			// Avoid crashing, but not ideal: we could have a better default Ctx with filepath at least
			currentContext[node] = LogCtx{}
		}
	}
	sort.Strings(keys)
	return keys, currentContext
}

// defaultColumnValue is displayed if the node did not have an event for a line
func defaultColumnValue(placeholder, state string) string {

	switch state {
	case "DONOR", "JOINER", "DESYNCED":
		return Paint(YellowText, placeholder)
	case "SYNCED":
		return Paint(GreenText, placeholder)
	case "CLOSED":
		return Paint(RedText, placeholder)
	default:
		return placeholder
	}
}

var timeBlocks = []struct {
	layout   string
	duration time.Duration
}{
	{
		layout:   ".000000Z",
		duration: time.Second,
	},
	{
		layout:   "05.000000Z",
		duration: time.Minute,
	},
	{
		layout:   "04:05.000000Z",
		duration: time.Hour,
	},
}

func dateBlock(nextDate, lastDate time.Time, layout, lastLayout string) (string, string) {
	if !CLI.List.GroupByTime {
		return nextDate.Format(layout), layout
	}
	// To avoid having a complete datetime everytime, we partially print some dates to make them looked "grouped"
	// It highlights that some events happened during the same second/minute/hour
	//
	// comparing last layout and current to avoid having a date shorter than the last one, it would create unreadable cascades of partial dates
	for _, tb := range timeBlocks {
		if nextDate.Truncate(tb.duration).Equal(lastDate.Truncate(tb.duration)) && len(lastLayout) >= len(tb.layout) {
			return nextDate.Format(tb.layout), tb.layout
		}
	}
	// Taking the first next event to log for the date format
	// It could be troublesome if some nodes do not have the same one (mysql versions, different timezone) but it's good enough for now.
	// nextNodes[0] is always supposed to exist, else we would not have anything to print anymore, same for timeline[nextNodes[0]][0] which is the next log to print for the nextnode
	return nextDate.Format(layout), layout
}

func printMetadata(timeline Timeline) {
	ip2hash := make(map[string][]string)
	for _, nodetl := range timeline {
		for hash, ip := range nodetl[len(nodetl)-1].Ctx.HashToIP {
			ip2hash[ip] = append(ip2hash[ip], hash)
		}
		//fmt.Println(nodetl[len(nodetl)-1].Ctx.HashToIP)
		//fmt.Println(nodetl[len(nodetl)-1].Ctx.IPToHostname)
	}
	for ip, hash := range ip2hash {
		fmt.Println(ip+": ", strings.Join(hash, ", "), "\n")
	}
}

func separator(keys []string) string {
	return " \t" + strings.Repeat(" \t", len(keys))
}

func headerNodes(keys []string) string {
	return "DATE\t" + strings.Join(keys, "\t") + "\t"
}

func headerFilePath(keys []string, ctxs map[string]LogCtx) string {
	header := " \t"
	for _, node := range keys {
		if ctx, ok := ctxs[node]; ok {
			header += ctx.FilePath + "\t"
		} else {
			header += " \t"
		}
	}
	return header
}

func headerHostname(keys []string, ctxs map[string]LogCtx) string {
	header := " \t"
	for _, node := range keys {
		if ctx, ok := ctxs[node]; ok {
			header += ctx.IPToHostname[ctx.SourceNodeIP] + "\t"
		} else {
			header += " \t"
		}
	}
	return header
}

func fileTransitionSeparator(keys []string, oldctxs, ctxs map[string]LogCtx) string {
	sep1 := " \t"
	sep2 := " \t"
	sep3 := " \t"
	found := false
	for _, node := range keys {
		ctx, ok1 := ctxs[node]
		oldctx, ok2 := oldctxs[node]
		if ok1 && ok2 && ctx.FilePath != oldctx.FilePath {
			sep1 += Paint(BrightText, oldctx.FilePath) + "\t"
			sep2 += Paint(BrightText, " V ") + "\t"
			sep3 += Paint(BrightText, ctx.FilePath) + "\t"
			found = true
		} else {
			sep1 += " \t"
			sep2 += " \t"
			sep3 += " \t"
		}
	}
	if !found {
		return ""
	}
	return sep1 + "\n" + sep2 + "\n" + sep3

}
