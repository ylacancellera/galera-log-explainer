package types

import (
	"encoding/json"
	"regexp"
)

type LogRegex struct {
	Regex         *regexp.Regexp // to send to grep, should be as simple as possible but without collisions
	InternalRegex *regexp.Regexp // for internal usage in handler func
	Type          RegexType

	// Taking into arguments the current context and log line, returning an updated context and a closure to get the msg to display
	// Why a closure: to later inject an updated context instead of the current partial context, to ensure hash/ip/nodenames are known
	Handler   func(*regexp.Regexp, LogCtx, string) (LogCtx, LogDisplayer)
	Verbosity Verbosity // To be able to hide details from summaries
}

func (l *LogRegex) Handle(ctx LogCtx, line string) (LogCtx, LogDisplayer) {
	return l.Handler(l.InternalRegex, ctx, line)
}

func (l *LogRegex) MarshalJSON() ([]byte, error) {
	out := &struct {
		Regex         string    `json:"regex"`
		InternalRegex string    `json:"internalRegex"`
		Type          RegexType `json:"type"`
		Verbosity     Verbosity `json:"verbosity"`
	}{
		Type:      l.Type,
		Verbosity: l.Verbosity,
	}
	if l.Regex != nil {
		out.Regex = l.Regex.String()
	}
	if l.InternalRegex != nil {
		out.InternalRegex = l.InternalRegex.String()
	}

	return json.Marshal(out)
}

type RegexType string

var (
	EventsRegexType      RegexType = "events"
	SSTRegexType         RegexType = "sst"
	ViewsRegexType       RegexType = "views"
	IdentRegexType       RegexType = "identity"
	StatesRegexType      RegexType = "states"
	PXCOperatorRegexType RegexType = "pxc-operator"
)

type RegexMap map[string]*LogRegex

func (r RegexMap) Merge(r2 RegexMap) RegexMap {
	for key, value := range r2 {
		r[key] = value
	}
	return r
}

func (r RegexMap) Compile() []string {

	arr := []string{}
	for _, regex := range r {
		arr = append(arr, regex.Regex.String())
	}
	return arr
}
