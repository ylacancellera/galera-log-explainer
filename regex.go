package main

import (
	"regexp"
	"strings"
)

type Verbosity int

const (
	Info Verbosity = iota
	// Detailed is having every suspect/warn
	Detailed
	// DebugMySQL only includes finding that are usually not relevant to show but useful to create the log context (eg: how we found the local address)
	DebugMySQL
	Debug
)

// 5.5 date : 151027  6:02:49
// 5.6 date : 2019-07-17 07:16:37
//5.7 date : 2019-07-17T15:16:37.123456Z
//5.7 date : 2019-07-17T15:16:37.123456+01:00
// 10.3 date: 2019-07-15  7:32:25
var DateLayouts = []string{
	"2006-01-02T15:04:05.000000Z",      // 5.7
	"2006-01-02T15:04:05.000000-07:00", // 5.7
	"060102 15:04:05",                  // 5.5
	"2006-01-02 15:04:05",              // 5.6
	"2006-01-02  15:04:05",             // 10.3
}

/*
REGEX_DAYS="\([0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}\|[0-9]\{6\}\)"
REGEX_HOURS=".[0-9]:[0-9]\{2\}:[0-9]\{2\}\(\.[0-9]\{6\}"
REGEX_DATE="$REGEX_DAYS.$REGEX_HOURS\(Z\|+[0-9]\{2\}:[0-9]\{2\}\)\|\.[0-9]\{3\}\|\)"
SYSLOG_DATE="\(Jan\|Feb\|Mar\|Apr\|May\|Jun\|Jul\|Aug\|Sep\|Oct\|Nov\|Dec\) \( \|[0-9]\)[0-9] [0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}"
REGEX_LOG_PREFIX="$REGEX_DATE \?[0-9]* "
*/

type LogRegex struct {
	Regex *regexp.Regexp

	// Taking into arguments the current context and log line, returning an updated context and a message to display
	Handler   func(LogCtx, string) (LogCtx, string)
	Verbosity Verbosity
	SkipPrint bool
}

// SilenceRegex accepts any LogRegex and set SkipPrint to avoid having it displayed.
// Some can be useful to construct context, but we can choose not to display them
func SilenceRegex(regexes ...LogRegex) []LogRegex {
	silenced := []LogRegex{}
	for _, regex := range regexes {
		regex.SkipPrint = true
		silenced = append(silenced, regex)
	}
	return silenced
}

// Grouped LogRegex per functions
var (
	StatesRegexes = []LogRegex{RegexShift, RegexRestoredState}
	ViewsRegexes  = []LogRegex{RegexNodeEstablished, RegexNodeJoined, RegexNodeLeft, RegexNodeSuspect, RegexNodeChangedIdentity, RegexWsrepUnsafeBootstrap, RegexWsrepConsistenctyCompromised, RegexWsrepNonPrimary}
	EventsRegexes = []LogRegex{RegexShutdownComplete, RegexShutdownSignal, RegexTerminated, RegexWsrepLoad, RegexWsrepRecovery, RegexUnknownConf, RegexBindAddressAlreadyUsed, RegexAssertionFailure}
)

// general buidling block wsrep regexes
// It's later used to identify subgroups easier
var (
	groupMethod        = "ssltcp"
	groupNodeIP        = "nodeip"
	groupNodeHash      = "nodehash"
	regexNodeHash      = "(?P<nodehash>.+)"
	regexNodeHash4Dash = "(?P<nodehash>[a-z0-9]+-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]+)" // eg ed97c863-d5c9-11ec-8ab7-671bbd2d70ef
	regexNodeHash1Dash = "(?P<nodehash>[a-z0-9]+-[a-z0-9]{4})"                                   // eg ed97c863-8ab7
	regexNodeIPMethod  = "(?P<ssltcp>.+)://(?P<nodeip>[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}):[0-9]{1,6}"
)

var (
	// sourceNode is to identify from which node this log was taken
	regexSourceNodeHandler = regexp.MustCompile("\\(" + regexNodeHash + ", '.+'\\).+" + regexNodeIPMethod)
	RegexSourceNode        = LogRegex{
		Regex: regexp.MustCompile("(local endpoint for a connection, blacklisting address)|(points to own listening address, blacklisting)"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexSourceNodeHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.SourceNodeIP = r[regexSourceNodeHandler.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[regexSourceNodeHandler.SubexpIndex(groupNodeHash)]] = ctx.SourceNodeIP
			return ctx, ctx.SourceNodeIP + " is local"
		},
		Verbosity: DebugMySQL,
	}
)

var (
	regexShiftHandler = regexp.MustCompile("[A-Z]+ -> [A-Z]+")
	shiftFunc         = func(ctx LogCtx, log string) (LogCtx, string) {
		log = regexShiftHandler.FindString(log)

		splitted := strings.Split(log, " -> ")
		ctx.State = splitted[1]

		log = strings.Replace(log, "DONOR", Paint(YellowText, "DONOR"), -1)
		log = strings.Replace(log, "DESYNCED", Paint(YellowText, "DESYNCED"), -1)
		log = strings.Replace(log, "JOINER", Paint(YellowText, "JOINER"), -1)
		log = strings.Replace(log, " SYNCED", Paint(GreenText, " SYNCED"), -1)
		log = strings.Replace(log, "JOINED", Paint(GreenText, "JOINED"), -1)
		log = strings.Replace(log, "CLOSED", Paint(RedText, "CLOSED"), -1)

		return ctx, log
	}
	RegexShift = LogRegex{
		Regex:   regexp.MustCompile("Shifting"),
		Handler: shiftFunc,
	}
	// 2022-07-18T11:20:52.125141Z 0 [Note] [MY-000000] [Galera] Shifting CLOSED -> OPEN (TO: 0)

	RegexRestoredState = LogRegex{
		Regex: regexp.MustCompile("Restored state"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx, log = shiftFunc(ctx, log)

			return ctx, "(restored)" + log
		},
	}
	// 2022-09-22T20:01:32.505660Z 0 [Note] [MY-000000] [Galera] Restored state OPEN -> SYNCED (13361114)
)

// "galera views" regexes
var (
	regexNodeEstablishedHandler = regexSourceNodeHandler
	RegexNodeEstablished        = LogRegex{
		Regex: regexp.MustCompile("connection established"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeEstablishedHandler.FindAllStringSubmatch(log, -1)[0]

			ip := r[regexNodeEstablishedHandler.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[regexNodeEstablishedHandler.SubexpIndex(groupNodeHash)]] = ip
			if ip == ctx.SourceNodeIP {
				return ctx, ""
			}
			return ctx, ip + " established"
		},
	}

	regexNodeJoinedHandler = regexp.MustCompile("declaring " + regexNodeHash + " at " + regexNodeIPMethod)
	RegexNodeJoined        = LogRegex{
		Regex: regexp.MustCompile("declaring .* stable"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeJoinedHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.HashToIP[r[regexNodeJoinedHandler.SubexpIndex(groupNodeHash)]] = r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)]
			ctx.IPToMethod[r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)]] = r[regexNodeJoinedHandler.SubexpIndex(groupMethod)]
			return ctx, r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)] + Paint(GreenText, " has joined")
		},
	}

	regexNodeLeftHandler = regexp.MustCompile("forgetting" + regexNodeHash + "\\(" + regexNodeIPMethod)
	RegexNodeLeft        = LogRegex{
		Regex: regexp.MustCompile("forgetting"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeLeftHandler.FindAllStringSubmatch(log, -1)[0]

			return ctx, r[regexNodeLeftHandler.SubexpIndex(groupNodeIP)] + Paint(RedText, " has left")
		},
	}

	regexNodeSuspectHandler = regexp.MustCompile("suspecting node: " + regexNodeHash)
	RegexNodeSuspect        = LogRegex{
		Regex: regexp.MustCompile("suspecting node"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeSuspectHandler.FindAllStringSubmatch(log, -1)[0]

			hash := r[regexNodeSuspectHandler.SubexpIndex(groupNodeHash)]
			ip, ok := ctx.HashToIP[hash]
			if ok {
				return ctx, ip + Paint(YellowText, " suspected to be down")
			}
			return ctx, hash + Paint(YellowText, " suspected to be down")
		},
		Verbosity: Detailed,
	}

	regexNodeChangedIdentityHandler = regexp.MustCompile("remote endpoint " + regexNodeIPMethod + " changed identity " + regexNodeHash + " -> " + strings.Replace(regexNodeHash, groupNodeHash, groupNodeHash+"2", -1))
	RegexNodeChangedIdentity        = LogRegex{
		Regex: regexp.MustCompile("remote endpoint.*changed identity"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {

			r := regexNodeChangedIdentityHandler.FindAllStringSubmatch(log, -1)[0]

			hash := r[regexNodeChangedIdentityHandler.SubexpIndex(groupNodeHash)]
			ip, ok := ctx.HashToIP[hash]
			if !ok && regexp.MustCompile(regexNodeHash4Dash).MatchString(hash) {
				splitted := strings.Split(hash, "-")
				ip, ok = ctx.HashToIP[splitted[0]+"-"+splitted[3]]

				// there could have additional corner case to discover yet
				if !ok {
					return ctx, hash + Paint(YellowText, " changed identity ")
				}
			}
			hash2 := r[regexNodeChangedIdentityHandler.SubexpIndex(groupNodeHash+"2")]
			ctx.HashToIP[hash2] = ip
			return ctx, ip + Paint(YellowText, " changed identity ")
		},
		Verbosity: Detailed,
	}

	RegexWsrepUnsafeBootstrap = LogRegex{
		Regex: regexp.MustCompile("ERROR.*not be safe to bootstrap"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "not safe to bootstrap")
		},
	}
	RegexWsrepConsistenctyCompromised = LogRegex{
		Regex: regexp.MustCompile(".ode consistency compromi.ed"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "consistencty compromised")
		},
	}
	RegexWsrepNonPrimary = LogRegex{
		Regex: regexp.MustCompile("failed to reach primary view"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			return ctx, Paint(RedText, "non primary")
		},
	}
)

/*
var (
	"SELF-LEAVE."
	"2022-10-29T12:00:34.449023Z 0 [Note] WSREP: Found saved state: 8e862473-455e-11e8-a0ca-3fcd8faf3209:-1, safe_to_bootstrap: 0"
	REGEX_NEW_VIEW          = "New cluster view"
	REGEX_NODE_INACTIVE     = "declaring inactive"
	REGEX_NODE_TIMEOUT      = "timed out, no messages seen in"
	REGEX_INCONSISTENT_VIEW = "node uuid:.*is inconsistent to restored view"
)
*/

/*
var (
REGEX_SST_REQ="requested state transfer"
REGEX_SST_TRANSFER_TO="State transfer to.*complete"
REGEX_SST_TRANSFER_FROM="State transfer from.*complete"
REGEX_SST_SYNCED="Member.*synced with group"
REGEX_SST_ERROR_PROCESS="Process completed with error: wsrep_sst"
REGEX_IST_UNAVAILABLE="Failed to prepare for incremental state transfer"
REGEX_SST_BYPASS="\(Bypassing state dump\|IST sender starting\|IST received\)"
REGEX_IST="\( IST \| ist \)"
REGEX_SST_METHOD="\(wsrep_sst_common\|wsrep_sst_rsync\|wsrep_sst_mysqldump\|wsrep_sst_xtrabackup-v2\|wsrep_sst_xtrabackup\)"

REGEX_SST_ERRORS="ERROR.*\($REGEX_SST_ERROR_PROCESS\|innobackupex\|xtrabackup\|$REGEX_IST\)"
REGEX_SST_NOTES="\(Note\|Warning\). WSREP.*\($REGEX_SST_REQ\|$REGEX_SST_TRANSFER_TO\|$REGEX_SST_TRANSFER_FROM\|$REGEX_SST_SYNCED\)"

REGEX_SST_COMPILED="\($REGEX_SST_ERRORS\|$REGEX_SST_NOTES\|$REGEX_SST_METHOD\|$REGEX_IST_UNAVAILABLE\|$REGEX_SST_BYPASS\)"

)
*/
var (
	RegexShutdownComplete = LogRegex{
		Regex: regexp.MustCompile("mysqld: Shutdown complete"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "shutdown complete")
		},
	}
	RegexTerminated = LogRegex{
		Regex: regexp.MustCompile("mysqld: Terminated"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "terminated")
		},
	}
	RegexShutdownSignal = LogRegex{
		Regex: regexp.MustCompile("Normal|Received shutdown"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "received shutdown")
		},
	}
	RegexAborting = LogRegex{
		Regex: regexp.MustCompile("[ERROR] Aborting"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "ABORTING")
		},
	}

	regexWsrepLoadNone = regexp.MustCompile("none")
	RegexWsrepLoad     = LogRegex{
		Regex: regexp.MustCompile("wsrep_load\\(\\): loading provider library"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "OPEN"
			if regexWsrepLoadNone.MatchString(log) {
				return ctx, Paint(GreenText, "started(standalone)")
			}
			return ctx, Paint(GreenText, "started(cluster)")
		},
	}
	RegexWsrepRecovery = LogRegex{
		//  INFO: WSREP: Recovered position 00000000-0000-0000-0000-000000000000:-1
		Regex: regexp.MustCompile("WSREP: Recovered position"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "RECOVERY"

			return ctx, "wsrep recovery"
		},
	}

	RegexUnknownConf = LogRegex{
		Regex: regexp.MustCompile("unknown variable"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			split := strings.Split(log, "'")
			v := "?"
			if len(split) > 0 {
				v = split[1]
			}
			return ctx, Paint(YellowText, "unknown variable") + ": " + v
		},
	}

	RegexAssertionFailure = LogRegex{
		Regex: regexp.MustCompile("Assertion failure"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "ASSERTION FAILURE")
		},
	}
	RegexBindAddressAlreadyUsed = LogRegex{
		Regex: regexp.MustCompile("asio error .bind: Address already in use"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			ctx.State = "CLOSED"

			return ctx, Paint(RedText, "bind address already used")
		},
	}
)
