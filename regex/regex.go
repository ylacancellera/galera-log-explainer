package regex

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

type LogRegex struct {
	Regex         *regexp.Regexp
	internalRegex *regexp.Regexp
	Type          types.RegexType

	// Taking into arguments the current context and log line, returning an updated context and a closure to get the msg to display
	// Why a closure: to later inject an updated context instead of the current partial context, to ensure hash/ip/nodenames are known
	handler   func(*regexp.Regexp, types.LogCtx, string) (types.LogCtx, types.LogDisplayer)
	Verbosity types.Verbosity
}

func (l *LogRegex) Handle(ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
	return l.handler(l.internalRegex, ctx, log)
}

func internalRegexSubmatch(regex *regexp.Regexp, log string) ([]string, error) {
	slice := regex.FindStringSubmatch(log)
	if len(slice) == 0 {
		return nil, errors.New(fmt.Sprintf("Could not find submatch from log \"%s\" using pattern \"%s\"", log, regex.String()))
	}
	return slice, nil
}

func setType(t types.RegexType, regexes ...LogRegex) []LogRegex {
	rs := regexes[:0]
	for _, regex := range regexes {
		regex.Type = t
		rs = append(rs, regex)
	}
	return rs
}

// SetVerbosity accepts any LogRegex
// Some can be useful to construct context, but we can choose not to display them
func SetVerbosity(verbosity types.Verbosity, regexes ...LogRegex) []LogRegex {
	rs := regexes[:0]
	for _, regex := range regexes {
		regex.Verbosity = verbosity
		rs = append(rs, regex)
	}
	return rs
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

func IsNodeUUID(s string) bool {
	b, _ := regexp.MatchString(regexNodeHash4Dash, s)
	return b
}
