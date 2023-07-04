package types

import (
	"fmt"
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
	Date            *Date
	displayer       LogDisplayer // what to show
	Log             string       // the raw log
	RegexType       RegexType
	RegexUsed       string
	Ctx             LogCtx // the context is copied for each logInfo, so that it is easier to handle some info (current state), and this is also interesting to check how it evolved
	Verbosity       Verbosity
	RepetitionCount int
}

func NewLogInfo(date *Date, displayer LogDisplayer, log string, regex *LogRegex, regexkey string, ctx LogCtx) LogInfo {
	return LogInfo{
		Date:      date,
		Log:       log,
		displayer: displayer,
		Ctx:       ctx,
		RegexType: regex.Type,
		RegexUsed: regexkey,
		Verbosity: regex.Verbosity,
	}
}

func (li *LogInfo) Msg(ctx LogCtx) string {
	if li.displayer == nil {
		return ""
	}
	msg := ""
	if li.RepetitionCount > 0 {
		msg += fmt.Sprintf("(repeated x%d)", li.RepetitionCount)
	}
	msg += li.displayer(ctx)
	return msg
}

func (current *LogInfo) IsDuplicatedEvent(base, previous LogInfo) bool {
	return base.RegexUsed == previous.RegexUsed &&
		base.displayer(base.Ctx) == previous.displayer(previous.Ctx) &&
		previous.RegexUsed == current.RegexUsed &&
		previous.displayer(previous.Ctx) == current.displayer(current.Ctx)
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
