package main

import (
	"regexp"
)

type Criticity int

const (
	Info Criticity = iota
	Warn
	Error
)

// 5.5 date : 151027  6:02:49
// 5.6 date : 2019-07-17 07:16:37
//5.7 date : 2019-07-17T15:16:37.123456Z
//5.7 date : 2019-07-17T15:16:37.123456+01:00
// 10.3 date: 2019-07-15  7:32:25
var DateLayouts = []string{
	"2006-01-02T15:04:05.000000Z", // 5.7
	"060102 15:04:05",             // 5.5
	"2006-01-02 15:04:05",         // 5.6
	"2006-01-02  15:04:05",        // 10.3
}

/*
REGEX_DAYS="\([0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}\|[0-9]\{6\}\)"
REGEX_HOURS=".[0-9]:[0-9]\{2\}:[0-9]\{2\}\(\.[0-9]\{6\}"
REGEX_DATE="$REGEX_DAYS.$REGEX_HOURS\(Z\|+[0-9]\{2\}:[0-9]\{2\}\)\|\.[0-9]\{3\}\|\)"
SYSLOG_DATE="\(Jan\|Feb\|Mar\|Apr\|May\|Jun\|Jul\|Aug\|Sep\|Oct\|Nov\|Dec\) \( \|[0-9]\)[0-9] [0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}"
REGEX_LOG_PREFIX="$REGEX_DATE \?[0-9]* "
*/

type LogRegex struct {
	// to use with regular grep
	Regex     string
	Criticity Criticity
	UpdateCtx func(LogCtx, string) LogCtx
	Msg       func(string) string
	SkipPrint bool
}

var simpleMsg = func(s string) string { return s }

var (
	RegexSourceNode = LogRegex{
		Regex: "local endpoint for a connection, blacklisting address",
		UpdateCtx: func(ctx LogCtx, log string) LogCtx {
			r := regexp.MustCompile("\\((?P<nodehash>.+), '.+'\\).*//(?P<nodeip>[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}):[0-9]{1,6}")
			r2 := r.FindAllStringSubmatch(log, -1)[0]

			ctx.SourceNodeIP = r2[2]
			ctx.HashToIP[ctx.SourceNodeIP] = r2[1]
			return ctx
		},
		SkipPrint: true,
	}
	RegexShift LogRegex = LogRegex{
		Regex:     "Shifting",
		Criticity: Info,
		Msg: func(s string) string {
			r := regexp.MustCompile("[A-Z]+ -> [A-Z]+")
			return r.FindString(s)
		},
	}
)

/*
var (
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
