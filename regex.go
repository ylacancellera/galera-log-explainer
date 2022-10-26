package main

import (
	"regexp"
	"strings"
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
	SkipPrint bool
}

// general buidling block wsrep regexes
var (
	groupMethod       = "ssltcp"
	groupNodeIP       = "nodeip"
	groupNodeHash     = "nodehash"
	regexNodeHash     = "(?P<nodehash>.+)"
	regexNodeIPMethod = "(?P<ssltcp>.+)://(?P<nodeip>[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}):[0-9]{1,6}"
)

var (
	// sourceNode is to identify from which node this log was taken
	regexSourceNodeHandler = regexp.MustCompile("\\(" + regexNodeHash + ", '.+'\\).+" + regexNodeIPMethod)
	RegexSourceNode        = LogRegex{
		Regex: regexp.MustCompile("local endpoint for a connection, blacklisting address"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexSourceNodeHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.SourceNodeIP = r[regexSourceNodeHandler.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[ctx.SourceNodeIP] = r[regexSourceNodeHandler.SubexpIndex(groupNodeHash)]
			return ctx, ""
		},
		SkipPrint: true,
	}

	regexShiftHandler          = regexp.MustCompile("[A-Z]+ -> [A-Z]+")
	RegexShift        LogRegex = LogRegex{
		Regex: regexp.MustCompile("Shifting"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			log = regexShiftHandler.FindString(log)
			log = strings.Replace(log, "DONOR", Paint(YellowText, "DONOR"), -1)
			log = strings.Replace(log, "DESYNCED", Paint(YellowText, "DESYNCED"), -1)
			log = strings.Replace(log, "JOINER", Paint(YellowText, "JOINER"), -1)
			log = strings.Replace(log, " SYNCED", Paint(GreenText, " SYNCED"), -1)
			log = strings.Replace(log, "JOINED", Paint(GreenText, "JOINED"), -1)
			log = strings.Replace(log, "CLOSED", Paint(RedText, "CLOSED"), -1)
			return ctx, log
		},
	}

	regexNodeEstablishedHandler          = regexSourceNodeHandler
	RegexNodeEstablished        LogRegex = LogRegex{
		Regex: regexp.MustCompile("connection established"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeEstablishedHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.HashToIP[r[regexNodeEstablishedHandler.SubexpIndex(groupNodeHash)]] = ctx.SourceNodeIP
			return ctx, r[regexNodeEstablishedHandler.SubexpIndex(groupNodeIP)] + " established"
		},
	}

	regexNodeJoinedHandler          = regexp.MustCompile(regexNodeHash + " at " + regexNodeIPMethod)
	RegexNodeJoined        LogRegex = LogRegex{
		Regex: regexp.MustCompile("declaring .* stable"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeJoinedHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.HashToIP[r[regexNodeJoinedHandler.SubexpIndex(groupNodeHash)]] = r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)]
			ctx.IPToMethod[r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)]] = r[regexNodeJoinedHandler.SubexpIndex(groupMethod)]
			return ctx, r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)] + Paint(GreenText, " has joined")
		},
	}

	regexNodeLeftHandler          = regexp.MustCompile("forgetting" + regexNodeHash + "\\(" + regexNodeIPMethod)
	RegexNodeLeft        LogRegex = LogRegex{
		Regex: regexp.MustCompile("forgetting"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeLeftHandler.FindAllStringSubmatch(log, -1)[0]

			return ctx, r[regexNodeLeftHandler.SubexpIndex(groupNodeIP)] + Paint(RedText, " has left")
		},
	}
)

/*
var (
	"SELF-LEAVE."
	REGEX_NEW_VIEW          = "New cluster view"
	REGEX_NODE_LEFT         = "forgetting"
	REGEX_NODE_ESTABLISHED  = "connection established"
	REGEX_NODE_SUSPECT      = "suspecting node"
	REGEX_NODE_INACTIVE     = "declaring inactive"
	REGEX_NODE_JOINED       = "declaring .* stable"
	REGEX_NODE_TIMEOUT      = "timed out, no messages seen in"
	REGEX_INCONSISTENT_VIEW = "node uuid:.*is inconsistent to restored view"
	REGEX_IDENTITY_CHANGES  = "remote endpoint.*changed identity.*"
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

var (
REGEX_SHUT_COMPLETE="mysqld: Shutdown complete"
REGEX_TERMINATED="mysqld: Terminated"
REGEX_SHUT="\($REGEX_SHUT_COMPLETE\|$REGEX_TERMINATED\)"
REGEX_STARTED="wsrep_load(): loading provider library"
REGEX_STARTED_STANDALONE="$REGEX_STARTED 'none'"
REGEX_RECOVER="dump/restore during wsrep recovery"
REGEX_NORMAL_SHUT="\(Normal\|Received\) shutdown"
REGEX_ABORTING="[ERROR] Aborting"
REGEX_ERROR_CONF="unknown variable"
REGEX_ERROR_BOOTSTRAP="ERROR.*not be safe to bootstrap"
REGEX_ERROR_NONP="failed to reach primary view"
REGEX_ERROR_ASSERT="Assertion failure"
REGEX_ERROR_CONSISTENCY=".ode consistency compromi.ed"
REGEX_ERROR_GALERA_BIND="asio error .bind: Address already in use"
)
*/
