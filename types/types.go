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

// LogCtx is a context for a given file.
// It used to keep track of what is going on at each new event.
type LogCtx struct {
	FilePath         string
	SourceNodeIP     []string
	State            string
	ResyncingNode    string
	ResyncedFromNode string
	OwnHashes        []string
	HashToIP         map[string]string
	HashToNodeName   map[string]string
	IPToHostname     map[string]string
	IPToMethod       map[string]string
	IPToNodeName     map[string]string
}

func NewLogCtx() LogCtx {
	return LogCtx{HashToIP: map[string]string{}, IPToHostname: map[string]string{}, IPToMethod: map[string]string{}, IPToNodeName: map[string]string{}, HashToNodeName: map[string]string{}}
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
	if t1[0].Date.Time.Before(t2[0].Date.Time) {
		return append(t1, t2...)
	}
	return append(t2, t1...)
}
