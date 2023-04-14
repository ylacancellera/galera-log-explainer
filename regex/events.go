package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func init() {
	setType(types.EventsRegexType, EventsMap)
}

var EventsMap = types.RegexMap{
	"RegexShutdownComplete": &types.LogRegex{
		Regex: regexp.MustCompile("mysqld: Shutdown complete"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "shutdown complete"))
		},
	},
	"RegexTerminated": &types.LogRegex{
		Regex: regexp.MustCompile("mysqld: Terminated"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "terminated"))
		},
	},
	"RegexShutdownSignal": &types.LogRegex{
		Regex: regexp.MustCompile("Normal|Received shutdown"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "received shutdown"))
		},
	},
	"RegexAborting": &types.LogRegex{
		Regex: regexp.MustCompile("[ERROR] Aborting"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "ABORTING"))
		},
	},

	"RegexWsrepLoad": &types.LogRegex{
		Regex: regexp.MustCompile("wsrep_load\\(\\): loading provider library"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "OPEN"
			if regexWsrepLoadNone.MatchString(log) {
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "started(standalone)"))
			}
			return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "started(cluster)"))
		},
	},
	"RegexWsrepRecovery": &types.LogRegex{
		//  INFO: WSREP: Recovered position 00000000-0000-0000-0000-000000000000:-1
		Regex: regexp.MustCompile("WSREP: Recovered position"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "RECOVERY"

			return ctx, types.SimpleDisplayer("wsrep recovery")
		},
	},

	"RegexUnknownConf": &types.LogRegex{
		Regex: regexp.MustCompile("unknown variable"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			split := strings.Split(log, "'")
			v := "?"
			if len(split) > 0 {
				v = split[1]
			}
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "unknown variable") + ": " + v)
		},
	},

	"RegexAssertionFailure": &types.LogRegex{
		Regex: regexp.MustCompile("Assertion failure"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "ASSERTION FAILURE"))
		},
	},
	"RegexBindAddressAlreadyUsed": &types.LogRegex{
		Regex: regexp.MustCompile("asio error .bind: Address already in use"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "bind address already used"))
		},
	},
}
var regexWsrepLoadNone = regexp.MustCompile("none")

// mysqld got signal 6/11
