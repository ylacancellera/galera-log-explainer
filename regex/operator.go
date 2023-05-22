package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

func init() {
	setType(types.PXCOperatorRegexType, PXCOperatorMap)
}

// Regexes from this type should only be about operator extra logs
// it should not contain Galera logs
// Specifically operators are dumping configuration files, recoveries, script outputs, ...
// only those should be handled here, they are specific to pxc operator but still very insightful
var PXCOperatorMap = types.RegexMap{
	"RegexNodeNameFromEnv": &types.LogRegex{
		Regex:         regexp.MustCompile(". NODE_NAME="),
		InternalRegex: regexp.MustCompile("NODE_NAME=" + regexNodeName),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			nodename := r[internalRegex.SubexpIndex(groupNodeName)]
			nodename, _, _ = strings.Cut(nodename, ".")
			ctx.AddOwnName(nodename)
			return ctx, types.SimpleDisplayer("local name(operator):" + nodename)
		},
		Verbosity: types.DebugMySQL,
	},

	"RegexNodeIPFromEnv": &types.LogRegex{
		Regex:         regexp.MustCompile(". NODE_IP="),
		InternalRegex: regexp.MustCompile("NODE_IP=" + regexNodeIP),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.AddOwnIP(ip)
			return ctx, types.SimpleDisplayer("local ip(operator):" + ip)
		},
		Verbosity: types.DebugMySQL,
	},
}
