package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// BetweenDateRegex generate a regex to filter mysql error log dates to just get
// events between 2 dates
// Currently limited to filter by day to produce "short" regexes. Finer events will be filtered later in code
// Trying to filter hours, minutes using regexes would produce regexes even harder to read
// while not really adding huge benefit as we do not expect so many events of interets
func BetweenDateRegex(since, until *time.Time) string {
	/*
		"2006-01-02
		"2006-01-0[3-9]
		"2006-01-[1-9][0-9]
		"2006-0[2-9]-[0-9]{2}
		"2006-[1-9][0-9]-[0-9]{2}
		"200[7-9]-[0-9]{2}-[0-9]{2}
		"20[1-9][0-9]-[0-9]{2}-[0-9]{2}
	*/
	regexConstructor := []struct {
		unit      int
		unitToStr string
	}{
		{
			unit:      since.Day(),
			unitToStr: fmt.Sprintf("%02d", since.Day()),
		},
		{
			unit:      int(since.Month()),
			unitToStr: fmt.Sprintf("%02d", since.Month()),
		},
		{
			unit:      since.Year(),
			unitToStr: fmt.Sprintf("%d", since.Year())[2:],
		},
	}
	s := ""
	for _, layout := range []string{"2006-01-02", "060102"} {
		// base complete date
		lastTransformed := since.Format(layout)
		s += "|^" + lastTransformed

		for _, construct := range regexConstructor {
			if construct.unit != 9 {
				s += "|^" + StringsReplaceReversed(lastTransformed, construct.unitToStr, string(construct.unitToStr[0])+"["+strconv.Itoa(construct.unit%10+1)+"-9]", 1)
			}
			// %1000 here is to cover the transformation of 2022 => 22
			s += "|^" + StringsReplaceReversed(lastTransformed, construct.unitToStr, "["+strconv.Itoa((construct.unit%1000/10)+1)+"-9][0-9]", 1)

			lastTransformed = StringsReplaceReversed(lastTransformed, construct.unitToStr, "[0-9][0-9]", 1)

		}
	}
	s += ")"
	return "(" + s[1:]
}

/*
SYSLOG_DATE="\(Jan\|Feb\|Mar\|Apr\|May\|Jun\|Jul\|Aug\|Sep\|Oct\|Nov\|Dec\) \( \|[0-9]\)[0-9] [0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}"
REGEX_LOG_PREFIX="$REGEX_DATE \?[0-9]* "
*/

type LogRegex struct {
	Regex *regexp.Regexp

	// Taking into arguments the current context and log line, returning an updated context and a message to display
	Handler   func(LogCtx, string) (LogCtx, string)
	Verbosity Verbosity
}

// SetVerbosity accepts any LogRegex and set
// Some can be useful to construct context, but we can choose not to display them
func SetVerbosity(verbosity Verbosity, regexes ...LogRegex) []LogRegex {
	silenced := []LogRegex{}
	for _, regex := range regexes {
		regex.Verbosity = verbosity
		silenced = append(silenced, regex)
	}
	return silenced
}

// Grouped LogRegex per functions
var (
	IdentRegexes  = []LogRegex{RegexSourceNode, RegexBaseHost}
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
	groupNodeName      = "nodename"
	regexNodeHash      = "(?P<" + groupNodeHash + ">.+)"
	regexNodeName      = "(?P<" + groupNodeName + ">.+)"
	regexNodeHash4Dash = "(?P<" + groupNodeHash + ">[a-z0-9]+-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]+)" // eg ed97c863-d5c9-11ec-8ab7-671bbd2d70ef
	regexNodeHash1Dash = "(?P<" + groupNodeHash + ">[a-z0-9]+-[a-z0-9]{4})"                                   // eg ed97c863-8ab7
	regexNodeIP        = "(?P<" + groupNodeIP + ">[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3})"
	regexNodeIPMethod  = "(?P<" + groupMethod + ">.+)://" + regexNodeIP + ":[0-9]{1,6}"
)

var (
	// sourceNode is to identify from which node this log was taken
	regexSourceNodeHandler = regexp.MustCompile("\\(" + regexNodeHash + ", '.+'\\).+" + regexNodeIPMethod)
	RegexSourceNode        = LogRegex{
		Regex: regexp.MustCompile("(local endpoint for a connection, blacklisting address)|(points to own listening address, blacklisting)"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexSourceNodeHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.SourceNodeIP = append(ctx.SourceNodeIP, r[regexSourceNodeHandler.SubexpIndex(groupNodeIP)])
			ctx.HashToIP[r[regexSourceNodeHandler.SubexpIndex(groupNodeHash)]] = ctx.SourceNodeIP[len(ctx.SourceNodeIP)-1]
			return ctx, ctx.SourceNodeIP[len(ctx.SourceNodeIP)-1] + " is local"
		},
		Verbosity: DebugMySQL,
	}
	regexBaseHostHandler = regexp.MustCompile("base_host = " + regexNodeIP)
	RegexBaseHost        = LogRegex{
		Regex: regexp.MustCompile("base_host"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexBaseHostHandler.FindAllStringSubmatch(log, -1)[0]

			ctx.SourceNodeIP = append(ctx.SourceNodeIP, r[regexBaseHostHandler.SubexpIndex(groupNodeIP)])
			return ctx, ctx.SourceNodeIP[len(ctx.SourceNodeIP)-1] + " is local"
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
		log = ColorForState(splitted[0], splitted[0]) + " -> " + ColorForState(splitted[1], splitted[1])

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
			if sliceContains(ctx.SourceNodeIP, ip) {
				return ctx, ""
			}
			return ctx, DisplayNodeSimplestForm(ip, ctx) + " established"
		},
	}

	regexNodeJoinedHandler = regexp.MustCompile("declaring " + regexNodeHash + " at " + regexNodeIPMethod)
	RegexNodeJoined        = LogRegex{
		Regex: regexp.MustCompile("declaring .* stable"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeJoinedHandler.FindAllStringSubmatch(log, -1)[0]

			ip := r[regexNodeJoinedHandler.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[regexNodeJoinedHandler.SubexpIndex(groupNodeHash)]] = ip
			ctx.IPToMethod[ip] = r[regexNodeJoinedHandler.SubexpIndex(groupMethod)]
			return ctx, DisplayNodeSimplestForm(ip, ctx) + Paint(GreenText, " has joined")
		},
	}

	regexNodeLeftHandler = regexp.MustCompile("forgetting" + regexNodeHash + "\\(" + regexNodeIPMethod)
	RegexNodeLeft        = LogRegex{
		Regex: regexp.MustCompile("forgetting"),
		Handler: func(ctx LogCtx, log string) (LogCtx, string) {
			r := regexNodeLeftHandler.FindAllStringSubmatch(log, -1)[0]

			ip := r[regexNodeLeftHandler.SubexpIndex(groupNodeIP)]
			return ctx, DisplayNodeSimplestForm(ip, ctx) + Paint(RedText, " has left")
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
				return ctx, DisplayNodeSimplestForm(ip, ctx) + Paint(YellowText, " suspected to be down")
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
			return ctx, DisplayNodeSimplestForm(ip, ctx) + Paint(YellowText, " changed identity ")
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
2022-11-29T23:31:36.971883-05:00 0 [Note] [MY-000000] [WSREP] Initiating SST cancellation


REGEX_SST_ERRORS="ERROR.*\($REGEX_SST_ERROR_PROCESS\|innobackupex\|xtrabackup\|$REGEX_IST\)"
REGEX_SST_NOTES="\(Note\|Warning\). WSREP.*\($REGEX_SST_REQ\|$REGEX_SST_TRANSFER_TO\|$REGEX_SST_TRANSFER_FROM\|$REGEX_SST_SYNCED\)"
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

/*

2022-11-25T17:05:00.693591-05:00 468765 [Warning] [MY-000000] [WSREP] Toggling wsrep_on to OFF will affect sql_log_bin. Check manual for more details

2022-11-21T20:59:04.893186-05:00 0 [Note] [MY-000000] [Galera] Member 2(node1) initiates vote on 9214cd54-5acd-11ed-8489-f7f024f872b4:5405,ad544d173db06c24:  <error>, Error_code: 1304;
2022-11-21T20:59:04.893287-05:00 0 [Note] [MY-000000] [Galera] Votes over 9214cd54-5acd-11ed-8489-f7f024f872b4:5405:
   ad544d173db06c24:   1/3
Waiting for more votes.
2022-11-21T20:59:04.893345-05:00 12 [Note] [MY-000000] [Galera] Got vote request for seqno 9214cd54-5acd-11ed-8489-f7f024f872b4:5405
2022-11-21T20:59:04.894150-05:00 0 [Note] [MY-000000] [Galera] Member 1(node1) initiates vote on 9214cd54-5acd-11ed-8489-f7f024f872b4:5405,ad544d173db06c24:  <error>,  Error_code: 1304;
2022-11-21T20:59:04.894178-05:00 0 [Note] [MY-000000] [Galera] Votes over 9214cd54-5acd-11ed-8489-f7f024f872b4:5405:
   ad544d173db06c24:   2/3
Winner: ad544d173db06c24
2022-11-21T20:59:04.898114-05:00 10 [ERROR] [MY-010584] [Repl] Slave SQL: Error '<query>', Error_code: MY-001304
2022-11-21T20:59:04.898180-05:00 10 [Warning] [MY-000000] [WSREP] Event 1 Query apply failed: 1, seqno 5405

04:31:38 UTC - mysqld got signal 6 ;

2022-11-29T23:34:51.820009-05:00 0 [Warning] [MY-000000] [Galera] Could not find peer: c0ff4085-5ad7-11ed-8b74-cfeec74147fe
2022-11-29T23:34:51.820069-05:00 0 [Warning] [MY-000000] [Galera] 0.1 (node): State transfer to -1.-1 (left the group) failed: -111 (Connection refused)

2022-12-07  1:00:03 0 [Note] WSREP: Member 0.0 (node) desyncs itself from group
2022-12-07  1:00:06 0 [Note] WSREP: Member 0.0 (node) resyncs itself to group.
2022-12-07  1:00:06 0 [Note] WSREP: Member 0.0 (node) synced with group.


2021-03-25T21:58:08.570748Z 0 [Warning] WSREP: no nodes coming from prim view, prim not possible
2021-03-25T21:58:13.570928Z 0 [Warning] WSREP: no nodes coming from prim view, prim not possible
2021-03-25T21:58:13.855983Z 0 [Warning] WSREP: Quorum: No node with complete state:

2021-03-25T21:58:02.322381Z 0 [Warning] WSREP: No persistent state found. Bootstraping with default state


2021-04-22T08:01:05.000581Z 0 [Warning] WSREP: Failed to report last committed 66328091, -110 (Connection timed out)


input_map=evs::input_map: {aru_seq=8,safe_seq=8,node_index=node: {idx=0,range=[9,8],safe_seq=8} node: {idx=1,range=[9,8],safe_seq=8} },
fifo_seq=4829086170,
last_sent=8,
known:
17a2e064 at tcp://ip:4567
{o=0,s=1,i=0,fs=-1,}
470a6438 at tcp://ip:4567
{o=1,s=0,i=0,fs=4829091361,jm=
{v=0,t=4,ut=255,o=1,s=8,sr=-1,as=8,f=4,src=470a6438,srcvid=view_id(REG,470a6438,24),insvid=view_id(UNKNOWN,00000000,0),ru=00000000,r=[-1,-1],fs=4829091361,nl=(
        17a2e064, {o=0,s=1,e=0,ls=-1,vid=view_id(REG,00000000,0),ss=-1,ir=[-1,-1],}
        470a6438, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,470a6438,24),ss=8,ir=[9,8],}
        6548cf50, {o=1,s=1,e=0,ls=-1,vid=view_id(REG,17a2e064,24),ss=12,ir=[13,12],}
        8b0c0f77, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,470a6438,24),ss=8,ir=[9,8],}
        d4397932, {o=0,s=1,e=0,ls=-1,vid=view_id(REG,00000000,0),ss=-1,ir=[-1,-1],}
)
},
}
6548cf50 at tcp://ip:4567
{o=1,s=1,i=0,fs=-1,jm=
{v=0,t=4,ut=255,o=1,s=12,sr=-1,as=12,f=4,src=6548cf50,srcvid=view_id(REG,17a2e064,24),insvid=view_id(UNKNOWN,00000000,0),ru=00000000,r=[-1,-1],fs=4829165031,nl=(
        17a2e064, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,17a2e064,24),ss=12,ir=[13,12],}
        470a6438, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,00000000,0),ss=-1,ir=[-1,-1],}
        6548cf50, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,17a2e064,24),ss=12,ir=[13,12],}
        8b0c0f77, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,00000000,0),ss=-1,ir=[-1,-1],}
        d4397932, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,17a2e064,24),ss=12,ir=[13,12],}
)
},
}
8b0c0f77 at
{o=1,s=0,i=0,fs=-1,jm=
{v=0,t=4,ut=255,o=1,s=8,sr=-1,as=8,f=0,src=8b0c0f77,srcvid=view_id(REG,470a6438,24),insvid=view_id(UNKNOWN,00000000,0),ru=00000000,r=[-1,-1],fs=4829086170,nl=(
        17a2e064, {o=0,s=1,e=0,ls=-1,vid=view_id(REG,00000000,0),ss=-1,ir=[-1,-1],}
        470a6438, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,470a6438,24),ss=8,ir=[9,8],}
        6548cf50, {o=1,s=1,e=0,ls=-1,vid=view_id(REG,17a2e064,24),ss=12,ir=[13,12],}
        8b0c0f77, {o=1,s=0,e=0,ls=-1,vid=view_id(REG,470a6438,24),ss=8,ir=[9,8],}
        d4397932, {o=0,s=1,e=0,ls=-1,vid=view_id(REG,00000000,0),ss=-1,ir=[-1,-1],}
)
},
}
d4397932 at tcp://ip:4567
{o=0,s=1,i=0,fs=4685894552,}
 }

*/
