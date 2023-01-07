package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var ViewsRegexes = []LogRegex{RegexNodeEstablished, RegexNodeJoined, RegexNodeLeft, RegexNodeSuspect, RegexNodeChangedIdentity, RegexWsrepUnsafeBootstrap, RegexWsrepConsistenctyCompromised, RegexWsrepNonPrimary, RegexNewComponent}

// "galera views" regexes
var (
	RegexNodeEstablished = LogRegex{
		Regex:         regexp.MustCompile("connection established"),
		internalRegex: regexp.MustCompile("established to " + regexNodeHash + " " + regexNodeIPMethod),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[internalRegex.SubexpIndex(groupNodeHash)]] = ip
			if utils.SliceContains(ctx.OwnIPs, ip) {
				return ctx, nil
			}
			return ctx, func(ctx types.LogCtx) string { return types.DisplayNodeSimplestForm(ctx, ip) + " established" }
		},
		Verbosity: types.Detailed,
	}

	RegexNodeJoined = LogRegex{
		Regex:         regexp.MustCompile("declaring .* stable"),
		internalRegex: regexp.MustCompile("declaring " + regexNodeHash + " at " + regexNodeIPMethod),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[internalRegex.SubexpIndex(groupNodeHash)]] = ip
			ctx.IPToMethod[ip] = r[internalRegex.SubexpIndex(groupMethod)]
			return ctx, func(ctx types.LogCtx) string {
				return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.GreenText, " has joined")
			}
		},
	}

	RegexNodeLeft = LogRegex{
		Regex:         regexp.MustCompile("forgetting"),
		internalRegex: regexp.MustCompile("forgetting" + regexNodeHash + "\\(" + regexNodeIPMethod),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			return ctx, func(ctx types.LogCtx) string {
				return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.RedText, " has left")
			}
		},
	}

	// New COMPONENT: primary = yes, bootstrap = no, my_idx = 1, memb_num = 5
	RegexNewComponent = LogRegex{
		Regex:         regexp.MustCompile("New COMPONENT:"),
		internalRegex: regexp.MustCompile("New COMPONENT: primary = (?P<primary>.+), bootstrap = (?P<bootstrap>.*), my_idx = .*, memb_num = (?P<memb_num>[0-9]{1,2})"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			primary := r[internalRegex.SubexpIndex("primary")] == "yes"
			memb_num := r[internalRegex.SubexpIndex("memb_num")]
			bootstrap := r[internalRegex.SubexpIndex("bootstrap")] == "yes"

			if primary {
				msg := utils.Paint(utils.GreenText, "PRIMARY") + "(n=" + memb_num + ")"
				if bootstrap {
					msg += " ,bootstrap=yes"
				}
				return ctx, types.SimpleDisplayer(msg)
			}

			// We stores nonprim as state, but not PRIMARY because we should find DONOR/JOINER/SYNCED/DESYNCED when it is primary
			// and we do not want to override these as they have more value
			ctx.State = "NON-PRIMARY"
			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "NON-PRIMARY") + "(n=" + memb_num + ")")
		},
	}

	RegexNodeSuspect = LogRegex{
		Regex:         regexp.MustCompile("suspecting node"),
		internalRegex: regexp.MustCompile("suspecting node: " + regexNodeHash),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			ip, ok := ctx.HashToIP[hash]
			if ok {
				return ctx, func(ctx types.LogCtx) string {
					return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.YellowText, " suspected to be down")
				}
			}
			return ctx, types.SimpleDisplayer(hash + utils.Paint(utils.YellowText, " suspected to be down"))
		},
		Verbosity: types.Detailed,
	}

	RegexNodeChangedIdentity = LogRegex{
		Regex:         regexp.MustCompile("remote endpoint.*changed identity"),
		internalRegex: regexp.MustCompile("remote endpoint " + regexNodeIPMethod + " changed identity " + regexNodeHash + " -> " + strings.Replace(regexNodeHash, groupNodeHash, groupNodeHash+"2", -1)),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			ip, ok := ctx.HashToIP[hash]
			if !ok && regexp.MustCompile(regexNodeHash4Dash).MatchString(hash) {
				splitted := strings.Split(hash, "-")
				ip, ok = ctx.HashToIP[splitted[0]+"-"+splitted[3]]

				// there could have additional corner case to discover yet
				if !ok {
					return ctx, types.SimpleDisplayer(hash + utils.Paint(utils.YellowText, " changed identity "))
				}
			}
			hash2 := r[internalRegex.SubexpIndex(groupNodeHash+"2")]
			ctx.HashToIP[hash2] = ip
			return ctx, func(ctx types.LogCtx) string {
				return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.YellowText, " changed identity ")
			}
		},
		Verbosity: types.Detailed,
	}

	RegexWsrepUnsafeBootstrap = LogRegex{
		Regex: regexp.MustCompile("ERROR.*not be safe to bootstrap"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "not safe to bootstrap"))
		},
	}
	RegexWsrepConsistenctyCompromised = LogRegex{
		Regex: regexp.MustCompile(".ode consistency compromi.ed"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "consistency compromised"))
		},
	}
	RegexWsrepNonPrimary = LogRegex{
		Regex: regexp.MustCompile("failed to reach primary view"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "non primary"))
		},
	}
)

/*
var (
	"SELF-LEAVE."
	"2022-10-29T12:00:34.449023Z 0 [Note] WSREP: Found saved state: 8e862473-455e-11e8-a0ca-3fcd8faf3209:-1, safe_to_bootstrap: 0"
	REGEX_NODE_INACTIVE     = "declaring inactive"
	REGEX_NODE_TIMEOUT      = "timed out, no messages seen in"
	REGEX_INCONSISTENT_VIEW = "node uuid:.*is inconsistent to restored view"
)
*/

/*

2023-01-05T03:49:55.653891Z 0 [Note] WSREP: gcomm: bootstrapping new group 'group'


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
