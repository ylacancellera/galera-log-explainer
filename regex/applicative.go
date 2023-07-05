package regex

import (
	"regexp"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func init() {
	setType(types.ApplicativeRegexType, ApplicativeMap)
}

var ApplicativeMap = types.RegexMap{

	"RegexDesync": &types.LogRegex{
		Regex: regexp.MustCompile("desyncs itself from group"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.Desynced = true
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "desyncs itself from group"))
		},
	},

	"RegexResync": &types.LogRegex{
		Regex: regexp.MustCompile("resyncs itself to group"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.Desynced = false
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "resyncs itself to group"))
		},
	},
}
