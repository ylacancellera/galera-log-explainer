package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var StatesRegexes = []LogRegex{RegexShift, RegexRestoredState}

var (
	shiftFunc = func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
		log = internalRegex.FindString(log)

		splitted := strings.Split(log, " -> ")
		ctx.State = splitted[1]
		log = utils.ColorForState(splitted[0], splitted[0]) + " -> " + utils.ColorForState(splitted[1], splitted[1])

		return ctx, types.SimpleDisplayer(log)
	}
	RegexShift = LogRegex{
		Regex:         regexp.MustCompile("Shifting"),
		internalRegex: regexp.MustCompile("[A-Z]+ -> [A-Z]+"),
		handler:       shiftFunc,
	}
	// 2022-07-18T11:20:52.125141Z 0 [Note] [MY-000000] [Galera] Shifting CLOSED -> OPEN (TO: 0)

	RegexRestoredState = LogRegex{
		Regex:         regexp.MustCompile("Restored state"),
		internalRegex: RegexShift.internalRegex,
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			var displayer types.LogDisplayer
			ctx, displayer = shiftFunc(internalRegex, ctx, log)

			return ctx, types.SimpleDisplayer("(restored)" + displayer(ctx))
		},
	}
	// 2022-09-22T20:01:32.505660Z 0 [Note] [MY-000000] [Galera] Restored state OPEN -> SYNCED (13361114)
)
