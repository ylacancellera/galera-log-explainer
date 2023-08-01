package regex

import (
	"regexp"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func init() {
	setType(types.StatesRegexType, StatesMap)
}

var (
	shiftFunc = func(submatches map[string]string, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

		ctx.SetState(submatches["state2"])
		log = utils.PaintForState(submatches["state1"], submatches["state1"]) + " -> " + utils.PaintForState(submatches["state2"], submatches["state2"])

		return ctx, types.SimpleDisplayer(log)
	}
	shiftRegex = regexp.MustCompile("(?P<state1>[A-Z]+) -> (?P<state2>[A-Z]+)")
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
		Handler: func(submatches map[string]string, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			var displayer types.LogDisplayer
			ctx, displayer = shiftFunc(submatches, ctx, log)

			return ctx, types.SimpleDisplayer("(restored)" + displayer(ctx))
		},
	},
	// 2022-09-22T20:01:32.505660Z 0 [Note] [MY-000000] [Galera] Restored state OPEN -> SYNCED (13361114)
}

// 2023-06-15T12:20:51.880330+03:00 2 [Note] [MY-000000] [WSREP] Server status change connected -> joiner
