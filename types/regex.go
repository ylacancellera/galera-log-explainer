package types

import (
	"regexp"

	"github.com/rs/zerolog"
)

type LogRegex struct {
	Regex         *regexp.Regexp // to send to grep, should be as simple as possible but without collisions
	InternalRegex *regexp.Regexp // for internal usage in handler func
	Type          RegexType
	logger        zerolog.Logger // wip

	// Taking into arguments the current context and log line, returning an updated context and a closure to get the msg to display
	// Why a closure: to later inject an updated context instead of the current partial context, to ensure hash/ip/nodenames are known
	Handler   func(*regexp.Regexp, LogCtx, string) (LogCtx, LogDisplayer)
	Verbosity Verbosity // To be able to hide details from summaries
}

func (l *LogRegex) Handle(ctx LogCtx, line string) (LogCtx, LogDisplayer) {
	return l.Handler(l.InternalRegex, ctx, line)
}

type RegexType string

var (
	EventsRegexType RegexType = "events"
	SSTRegexType    RegexType = "sst"
	ViewsRegexType  RegexType = "views"
	IdentRegexType  RegexType = "identity"
	StatesRegexType RegexType = "states"
)

type RegexMap map[string]*LogRegex

func (r RegexMap) Merge(r2 RegexMap) RegexMap {
	for key, value := range r2 {
		r[key] = value
	}
	return r
}
