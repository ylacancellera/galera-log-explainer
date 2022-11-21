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

// Color is given its own type for safe function signatures
type Color string

// Color codes interpretted by the terminal
// NOTE: all codes must be of the same length or they will throw off the field alignment of tabwriter
const (
	ResetText         Color = "\x1b[0000m"
	BrightText              = "\x1b[0001m"
	RedText                 = "\x1b[0031m"
	GreenText               = "\x1b[0032m"
	YellowText              = "\x1b[0033m"
	BlueText                = "\x1b[0034m"
	MagentaText             = "\x1b[0035m"
	CyanText                = "\x1b[0036m"
	WhiteText               = "\x1b[0037m"
	DefaultText             = "\x1b[0039m"
	BrightRedText           = "\x1b[1;31m"
	BrightGreenText         = "\x1b[1;32m"
	BrightYellowText        = "\x1b[1;33m"
	BrightBlueText          = "\x1b[1;34m"
	BrightMagentaText       = "\x1b[1;35m"
	BrightCyanText          = "\x1b[1;36m"
	BrightWhiteText         = "\x1b[1;37m"
)

// Color implements the Stringer interface for interoperability with string
func (c *Color) String() string {
	return fmt.Sprintf("%v", c)
}

func Paint(color Color, value string) string {
	if CLI.NoColor {
		return value
	}
	return fmt.Sprintf("%v%v%v", color, value, ResetText)
}

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

func DisplayColumnar(timeline Timeline) {
	var (
		lastDate time.Time
		args     []string
	)
	// to hold the current context for each node
	currentContext := map[string]LogCtx{}

	w := tabwriter.NewWriter(os.Stdout, 8, 8, 3, ' ', tabwriter.AlignRight)
	defer w.Flush()

	// keys will be used to access the timeline map with an ordered manner
	// without this, we would not print on the correct column as the order of a map is guaranteed to be random each time
	keys := make([]string, 0, len(timeline))
	for node := range timeline {
		keys = append(keys, node)
	}
	sort.Strings(keys)

	// header
	header, separator := headerAndSeparator(keys, timeline)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, separator)

	// as long as there is a next event to print
	for nextNodes, nextDate := iterateNode(timeline); len(nextNodes) != 0; nextNodes, nextDate = iterateNode(timeline) {

		// To avoid having a complete datetime everytime, we partially print some dates to make them looked "grouped"
		// It highlights that some events happened during the same second
		if nextDate.Truncate(time.Second).Equal(lastDate.Truncate(time.Second)) {
			args = []string{nextDate.Format(".000000Z")}
		} else {
			// Taking the first next event to log for the date format
			// It could be troublesome if some nodes do not have the same one (mysql versions, different timezone) but it's good enough for now.
			// nextNodes[0] is always supposed to exist, else we would not have anything to print anymore, same for timeline[nextNodes[0]][0] which is the next log to print for the nextnode
			args = []string{nextDate.Format(timeline[nextNodes[0]][0].DateLayout)}
		}

	MakeLine:
		for _, node := range keys {

			for _, nextNode := range nextNodes {

				if node == nextNode {
					nl := timeline[nextNode][0]
					currentContext[nextNode] = nl.Ctx
					args = append(args, nl.Msg)

					// dequeue the events
					if len(timeline[nextNode]) > 0 {
						timeline[nextNode] = timeline[nextNode][1:]

					}
					// we found something to print for this node
					continue MakeLine
				}
			}

			// if there are no events, having a | is needed for tabwriter
			// A few color can also help highlighting how the node is doing
			switch currentContext[node].State {
			case "DONOR", "JOINER", "DESYNCED":
				args = append(args, Paint(YellowText, "| "))
			case "SYNCED":
				args = append(args, Paint(GreenText, "| "))
			case "CLOSED":
				args = append(args, Paint(RedText, "| "))
			default:
				args = append(args, "| ")
			}

		}
		_, err := fmt.Fprintln(w, strings.Join(args, "\t")+"\t")
		if err != nil {
			log.Println("Failed to write a line", err)
		}

		lastDate = nextDate
	}

	// footer
	// only having a header is not fast enough to read when there are too many lines
	fmt.Fprintln(w, separator)
	fmt.Fprintln(w, header)

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

func headerAndSeparator(keys []string, timeline Timeline) (string, string) {
	separator := " \t" + strings.Repeat(" \t", len(keys))
	header := "DATE\t" + strings.Join(keys, "\t") + "\t" + "\n \t"
	for _, node := range keys {
		if len(timeline[node]) > 0 {
			header += timeline[node][0].Ctx.FilePath + "\t"
		} else {
			header += " \t"
		}
	}
	header += "\n \t"
	for _, node := range keys {
		if len(timeline[node]) > 0 {
			header += timeline[node][0].Ctx.IPToHostname[timeline[node][0].Ctx.SourceNodeIP] + "\t"
		} else {
			header += " \t"
		}
	}
	return header, separator
}
