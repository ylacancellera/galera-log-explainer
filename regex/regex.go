package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

type LogRegex struct {
	Regex         *regexp.Regexp
	internalRegex *regexp.Regexp

	// Taking into arguments the current context and log line, returning an updated context and a handler to get the msg to display
	// The message is not a string, but a function taking a context to return a string
	// this is to be able to display information using the latest updated context containing most hash/ip/nodenames information
	handler   func(*regexp.Regexp, types.LogCtx, string) (types.LogCtx, types.LogDisplayer)
	Verbosity types.Verbosity
}

func (l *LogRegex) Handle(ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
	return l.handler(l.internalRegex, ctx, log)
}

// SetVerbosity accepts any LogRegex
// Some can be useful to construct context, but we can choose not to display them
func SetVerbosity(verbosity types.Verbosity, regexes ...LogRegex) []LogRegex {
	silenced := []LogRegex{}
	for _, regex := range regexes {
		regex.Verbosity = verbosity
		silenced = append(silenced, regex)
	}
	return silenced
}

// general buidling block wsrep regexes
// It's later used to identify subgroups easier
var (
	groupMethod        = "ssltcp"
	groupNodeIP        = "nodeip"
	groupNodeHash      = "nodehash"
	groupNodeName      = "nodename"
	groupNodeName2     = "nodename2"
	groupMyIdx         = "myidx"
	groupSeqno         = "seqno"
	regexNodeHash      = "(?P<" + groupNodeHash + ">.+)"
	regexNodeName      = "(?P<" + groupNodeName + ">.+)"
	regexNodeName2     = strings.Replace(regexNodeName, groupNodeName, groupNodeName2, 1)
	regexNodeHash4Dash = "(?P<" + groupNodeHash + ">[a-z0-9]+-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]+)" // eg ed97c863-d5c9-11ec-8ab7-671bbd2d70ef
	regexNodeHash1Dash = "(?P<" + groupNodeHash + ">[a-z0-9]+-[a-z0-9]{4})"                                   // eg ed97c863-8ab7
	regexSeqno         = "(?P<" + groupSeqno + ">[0-9]+)"
	regexNodeIP        = "(?P<" + groupNodeIP + ">[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3})"
	regexNodeIPMethod  = "(?P<" + groupMethod + ">.+)://" + regexNodeIP + ":[0-9]{1,6}"
	regexMyIdx         = "(?P<" + groupMyIdx + ">[0-9]{1,2})"
)
