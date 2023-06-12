package regex

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

func internalRegexSubmatch(regex *regexp.Regexp, log string) ([]string, error) {
	slice := regex.FindStringSubmatch(log)
	if len(slice) == 0 {
		return nil, errors.New(fmt.Sprintf("Could not find submatch from log \"%s\" using pattern \"%s\"", log, regex.String()))
	}
	return slice, nil
}

func setType(t types.RegexType, regexes types.RegexMap) {
	for _, regex := range regexes {
		regex.Type = t
	}
	return
}

// SetVerbosity accepts any LogRegex
// Some can be useful to construct context, but we can choose not to display them
func SetVerbosity(verbosity types.Verbosity, regexes types.RegexMap) {
	for _, regex := range regexes {
		regex.Verbosity = verbosity
	}
	return
}

func AllRegexes() types.RegexMap {
	IdentsMap.Merge(ViewsMap).Merge(SSTMap).Merge(EventsMap).Merge(StatesMap)
	return IdentsMap
}

// general building block wsrep regexes
// It's later used to identify subgroups easier
var (
	groupMethod        = "ssltcp"
	groupNodeIP        = "nodeip"
	groupNodeHash      = "nodehash"
	groupNodeName      = "nodename"
	groupNodeName2     = "nodename2"
	groupIdx           = "idx"
	groupSeqno         = "seqno"
	groupMembers       = "members"
	groupVersion       = "version"
	regexMembers       = "(?P<" + groupMembers + ">[0-9]{1,2})"
	regexNodeHash      = "(?P<" + groupNodeHash + ">[a-zA-Z0-9-_]+)"
	regexNodeName      = "(?P<" + groupNodeName + ">[a-zA-Z0-9-_]+)"
	regexNodeName2     = strings.Replace(regexNodeName, groupNodeName, groupNodeName2, 1)
	regexNodeHash4Dash = "(?P<" + groupNodeHash + ">[a-z0-9]+-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]+)" // eg ed97c863-d5c9-11ec-8ab7-671bbd2d70ef
	regexNodeHash1Dash = "(?P<" + groupNodeHash + ">[a-z0-9]+-[a-z0-9]{4})"                                   // eg ed97c863-8ab7
	regexSeqno         = "(?P<" + groupSeqno + ">[0-9]+)"
	regexNodeIP        = "(?P<" + groupNodeIP + ">[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3})"
	regexNodeIPMethod  = "(?P<" + groupMethod + ">.+)://" + regexNodeIP + ":[0-9]{1,6}"
	regexIdx           = "(?P<" + groupIdx + ">-?[0-9]{1,2})"
	regexVersion       = "(?P<" + groupVersion + ">(5|8|10|11)\\.[0-9]\\.[0-9]{1,2})"
)

func IsNodeUUID(s string) bool {
	b, _ := regexp.MatchString(regexNodeHash4Dash, s)
	return b
}
