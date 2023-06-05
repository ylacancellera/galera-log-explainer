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
		Regex: regexp.MustCompile("Aborting"),
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
		Regex: regexp.MustCompile("Recovered position"),
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
/*
2001-01-01T01:01:01.000000Z 0 [System] [MY-010116] [Server] /usr/sbin/mysqld (mysqld 8.0.30-22) starting as process 1


2023-05-09T17:39:19.955040Z 51 [Warning] [MY-000000] [Galera] failed to replay trx: source: fb9d6310-ee8b-11ed-8aee-f7542ad73e53 version: 5 local: 1 flags: 1 conn_id: 48 trx_id: 2696 tstamp: 1683653959142522853; state: EXECUTING:0->REPLICATING:782->CERTIFYING:3509->APPLYING:3748->COMMITTING:1343->COMMITTED:-1
2023-05-09T17:39:19.955085Z 51 [Warning] [MY-000000] [Galera] Invalid state in replay for trx source: fb9d6310-ee8b-11ed-8aee-f7542ad73e53 version: 5 local: 1 flags: 1 conn_id: 48 trx_id: 2696 tstamp: 1683653959142522853; state: EXECUTING:0->REPLICATING:782->CERTIFYING:3509->APPLYING:3748->COMMITTING:1343->COMMITTED:-1 (FATAL)
         at galera/src/replicator_smm.cpp:replay_trx():1247


		 2023-05-28T21:18:23.118262-05:00 0 [Note] [MY-000000] [Galera] STATE EXCHANGE: got state msg: <cluster uuid> from 2 (node2)

2001-01-01T01:01:01.000000Z 0 [ERROR] [MY-000000] [Galera] gcs/src/gcs_group.cpp:group_post_state_exchange():431: Reversing history: 312312 -> 20121, this member has applied 12345 more events than the primary component.Data loss is possible. Must abort.

*/
