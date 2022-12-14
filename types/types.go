package types

import (
	"time"
)

type Verbosity int

const (
	Info Verbosity = iota
	// Detailed is having every suspect/warn
	Detailed
	// DebugMySQL only includes finding that are usually not relevant to show but useful to create the log context (eg: how we found the local address)
	DebugMySQL
	Debug
)

// LogInfo is to store a single event in log. This is something that should be displayed ultimately, this is what we want when we launch this tool
type LogInfo struct {
	Date      Date
	Msg       LogDisplayer // what to show
	Log       string       // the raw log
	Ctx       LogCtx       // the context is copied for each logInfo, so that it is easier to handle some info (current state), and this is also interesting to check how it evolved
	Verbosity Verbosity
}

type Date struct {
	Time        time.Time
	DisplayTime string
	Layout      string
}

func NewDate(t time.Time, layout string) Date {
	return Date{
		Time:        t,
		Layout:      layout,
		DisplayTime: t.Format(layout),
	}
}

// LogDisplayer is the handler to generate messages thanks to a context
// The context in parameters should be as updated as possible
type LogDisplayer func(LogCtx) string

// SimpleDisplayer satisfies LogDisplayer and ignores any context received
func SimpleDisplayer(s string) LogDisplayer {
	return func(_ LogCtx) string { return s }
}

// It should be kept already sorted by timestamp
type LocalTimeline []LogInfo

// "string" key is a node IP
type Timeline map[string]LocalTimeline

// MergeTimeline is helpful when log files are split by date, it can be useful to be able to merge content
// a "timeline" come from a log file. Log files that came from some node should not never have overlapping dates
func MergeTimeline(t1, t2 LocalTimeline) LocalTimeline {
	if len(t1) == 0 {
		return t2
	}
	if len(t2) == 0 {
		return t1
	}

	// if t2 is an updated version of t1, or t1 an updated of t2, or t1=t2
	if t1[0].Date.Time.Equal(t2[0].Date.Time) {
		if t1[len(t1)-1].Date.Time.Before(t2[len(t2)-1].Date.Time) {
			return t2
		}
		return t1
	}
	if t1[0].Date.Time.Before(t2[0].Date.Time) {
		return append(t1, t2...)
	}
	return append(t2, t1...)
}
