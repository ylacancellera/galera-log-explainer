package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var EventsRegexes = []LogRegex{RegexShutdownComplete, RegexShutdownSignal, RegexTerminated, RegexWsrepLoad, RegexWsrepRecovery, RegexUnknownConf, RegexBindAddressAlreadyUsed, RegexAssertionFailure}

func init() {
	EventsRegexes = setType(types.EventsRegexType, EventsRegexes...)
}

var (
	RegexShutdownComplete = LogRegex{
		Regex: regexp.MustCompile("mysqld: Shutdown complete"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "shutdown complete"))
		},
	}
	RegexTerminated = LogRegex{
		Regex: regexp.MustCompile("mysqld: Terminated"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "terminated"))
		},
	}
	RegexShutdownSignal = LogRegex{
		Regex: regexp.MustCompile("Normal|Received shutdown"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "received shutdown"))
		},
	}
	RegexAborting = LogRegex{
		Regex: regexp.MustCompile("[ERROR] Aborting"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "ABORTING"))
		},
	}

	regexWsrepLoadNone = regexp.MustCompile("none")
	RegexWsrepLoad     = LogRegex{
		Regex: regexp.MustCompile("wsrep_load\\(\\): loading provider library"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "OPEN"
			if regexWsrepLoadNone.MatchString(log) {
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "started(standalone)"))
			}
			return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "started(cluster)"))
		},
	}
	RegexWsrepRecovery = LogRegex{
		//  INFO: WSREP: Recovered position 00000000-0000-0000-0000-000000000000:-1
		Regex: regexp.MustCompile("WSREP: Recovered position"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "RECOVERY"

			return ctx, types.SimpleDisplayer("wsrep recovery")
		},
	}

	RegexUnknownConf = LogRegex{
		Regex: regexp.MustCompile("unknown variable"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			split := strings.Split(log, "'")
			v := "?"
			if len(split) > 0 {
				v = split[1]
			}
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "unknown variable") + ": " + v)
		},
	}

	RegexAssertionFailure = LogRegex{
		Regex: regexp.MustCompile("Assertion failure"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "ASSERTION FAILURE"))
		},
	}
	RegexBindAddressAlreadyUsed = LogRegex{
		Regex: regexp.MustCompile("asio error .bind: Address already in use"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "bind address already used"))
		},
	}
)
