package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func init() {
	setType(types.StatesRegexType, StatesMap)
}

var (
	shiftFunc = func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
		log = internalRegex.FindString(log)

		splitted := strings.Split(log, " -> ")
		ctx.State = splitted[1]
		log = utils.PaintForState(splitted[0], splitted[0]) + " -> " + utils.PaintForState(splitted[1], splitted[1])

		return ctx, types.SimpleDisplayer(log)
	}
	shiftRegex = regexp.MustCompile("[A-Z]+ -> [A-Z]+")
)

var StatesMap = types.RegexMap{
	"RegexShift": &types.LogRegex{
		Regex:         regexp.MustCompile("Shifting"),
		InternalRegex: shiftRegex,
		Handler:       shiftFunc,
	},
	// 2022-07-18T11:20:52.125141Z 0 [Note] [MY-000000] [Galera] Shifting CLOSED -> OPEN (TO: 0)

	"RegexRestoredState": &types.LogRegex{
		Regex:         regexp.MustCompile("Restored state"),
		InternalRegex: shiftRegex,
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			var displayer types.LogDisplayer
			ctx, displayer = shiftFunc(internalRegex, ctx, log)

			return ctx, types.SimpleDisplayer("(restored)" + displayer(ctx))
		},
	},
	// 2022-09-22T20:01:32.505660Z 0 [Note] [MY-000000] [Galera] Restored state OPEN -> SYNCED (13361114)
}

// 2023-06-15T12:20:51.880330+03:00 2 [Note] [MY-000000] [WSREP] Server status change connected -> joiner
