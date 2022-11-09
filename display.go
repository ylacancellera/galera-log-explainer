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
	Reset             Color = "\x1b[0000m"
	Bright                  = "\x1b[0001m"
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
	return fmt.Sprintf("%v%v%v", color, value, Reset)
}

// iterateNode is used to search the node that contains the next events
// as they stored in chronological order
// this is a slice in case 2 nodes have an event precisely at the same time
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
		if timeline[node][0].Msg == "" {
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
	// without this, we would not print on the correct column as the order of a map is guaranteed to be random each time
	keys := make([]string, 0, len(timeline))
	for node := range timeline {
		keys = append(keys, node)
	}
	sort.Strings(keys)

	// header
	separator := " \t" + strings.Repeat(" \t", len(keys))
	header := "DATE\t" + strings.Join(keys, "\t") + "\t" + "\n \t"
	for _, node := range keys {
		if len(timeline[node]) > 0 {
			header += timeline[node][0].Ctx.FilePath + "\t"
		} else {
			header += " \t"
		}
	}
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, separator)

	// as long as there is a next event to print
	for nextNodes, nextDate := iterateNode(timeline); len(nextNodes) != 0; nextNodes, nextDate = iterateNode(timeline) {

		// to avoid having a complete datetime everytime. It highlights that some events happened during the same second
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

			// if there are no events
			args = append(args, "| ")

		}
		_, err := fmt.Fprintln(w, strings.Join(args, "\t")+"\t")
		if err != nil {
			log.Println("Failed to write a line", err)
		}

		lastDate = nextDate
	}

	// footer
	fmt.Fprintln(w, separator)
	fmt.Fprintln(w, header)
}
