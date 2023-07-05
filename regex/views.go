package regex

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func init() {
	setType(types.ViewsRegexType, ViewsMap)
}

// "galera views" regexes
var ViewsMap = types.RegexMap{
	"RegexNodeEstablished": &types.LogRegex{
		Regex:         regexp.MustCompile("connection established"),
		InternalRegex: regexp.MustCompile("established to " + regexNodeHash + " " + regexNodeIPMethod),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[internalRegex.SubexpIndex(groupNodeHash)]] = ip
			if utils.SliceContains(ctx.OwnIPs, ip) {
				return ctx, nil
			}
			return ctx, func(ctx types.LogCtx) string { return types.DisplayNodeSimplestForm(ctx, ip) + " established" }
		},
		Verbosity: types.Detailed,
	},

	"RegexNodeJoined": &types.LogRegex{
		Regex:         regexp.MustCompile("declaring .* stable"),
		InternalRegex: regexp.MustCompile("declaring " + regexNodeHash + " at " + regexNodeIPMethod),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.HashToIP[r[internalRegex.SubexpIndex(groupNodeHash)]] = ip
			ctx.IPToMethod[ip] = r[internalRegex.SubexpIndex(groupMethod)]
			return ctx, func(ctx types.LogCtx) string {
				return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.GreenText, " joined")
			}
		},
	},

	"RegexNodeLeft": &types.LogRegex{
		Regex:         regexp.MustCompile("forgetting"),
		InternalRegex: regexp.MustCompile("forgetting " + regexNodeHash + " \\(" + regexNodeIPMethod),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			return ctx, func(ctx types.LogCtx) string {
				return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.RedText, " left")
			}
		},
	},

	// New COMPONENT: primary = yes, bootstrap = no, my_idx = 1, memb_num = 5
	"RegexNewComponent": &types.LogRegex{
		Regex:         regexp.MustCompile("New COMPONENT:"),
		InternalRegex: regexp.MustCompile("New COMPONENT: primary = (?P<primary>.+), bootstrap = (?P<bootstrap>.*), my_idx = .*, memb_num = (?P<memb_num>[0-9]{1,2})"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			primary := r[internalRegex.SubexpIndex("primary")] == "yes"
			membNum := r[internalRegex.SubexpIndex("memb_num")]
			bootstrap := r[internalRegex.SubexpIndex("bootstrap")] == "yes"
			memberCount, err := strconv.Atoi(membNum)
			if err != nil {
				return ctx, nil
			}

			ctx.MemberCount = memberCount
			if primary {
				// we don't always store PRIMARY because we could have found DONOR/JOINER/SYNCED/DESYNCED just earlier
				// and we do not want to override these as they have more value
				if ctx.State == "CLOSED" || ctx.State == "NON-PRIMARY" || ctx.State == "OPEN" || ctx.State == "RECOVERY" || ctx.State == "" {
					ctx.State = "PRIMARY"
				}
				msg := utils.Paint(utils.GreenText, "PRIMARY") + "(n=" + membNum + ")"
				if bootstrap {
					msg += ",bootstrap"
				}
				return ctx, types.SimpleDisplayer(msg)
			}

			ctx.State = "NON-PRIMARY"
			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "NON-PRIMARY") + "(n=" + membNum + ")")
		},
	},

	"RegexNodeSuspect": &types.LogRegex{
		Regex:         regexp.MustCompile("suspecting node"),
		InternalRegex: regexp.MustCompile("suspecting node: " + regexNodeHash),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

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
	},

	"RegexNodeChangedIdentity": &types.LogRegex{
		Regex:         regexp.MustCompile("remote endpoint.*changed identity"),
		InternalRegex: regexp.MustCompile("remote endpoint " + regexNodeIPMethod + " changed identity " + regexNodeHash + " -> " + strings.Replace(regexNodeHash, groupNodeHash, groupNodeHash+"2", -1)),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			hash2 := r[internalRegex.SubexpIndex(groupNodeHash+"2")]
			ip, ok := ctx.HashToIP[hash]
			if !ok && regexp.MustCompile(regexNodeHash4Dash).MatchString(hash) {
				ip, ok = ctx.HashToIP[utils.UUIDToShortUUID(hash)]

				// there could have additional corner case to discover yet
				if !ok {
					return ctx, types.SimpleDisplayer(hash + utils.Paint(utils.YellowText, " changed identity"))
				}
				hash2 = utils.UUIDToShortUUID(hash2)
			}
			ctx.HashToIP[hash2] = ip
			return ctx, func(ctx types.LogCtx) string {
				return types.DisplayNodeSimplestForm(ctx, ip) + utils.Paint(utils.YellowText, " changed identity")
			}
		},
		Verbosity: types.Detailed,
	},

	"RegexWsrepUnsafeBootstrap": &types.LogRegex{
		Regex: regexp.MustCompile("ERROR.*not be safe to bootstrap"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "not safe to bootstrap"))
		},
	},
	"RegexWsrepConsistenctyCompromised": &types.LogRegex{
		Regex: regexp.MustCompile(".ode consistency compromi.ed"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "CLOSED"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "consistency compromised"))
		},
	},
	"RegexWsrepNonPrimary": &types.LogRegex{
		Regex: regexp.MustCompile("failed to reach primary view"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer("received " + utils.Paint(utils.RedText, "non primary"))
		},
	},

	"RegexBootstrap": &types.LogRegex{
		Regex: regexp.MustCompile("gcomm: bootstrapping new group"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "bootstrapping"))
		},
	},

	"RegexSafeToBoostrapSet": &types.LogRegex{
		Regex: regexp.MustCompile("safe_to_bootstrap: 1"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "safe_to_bootstrap: 1"))
		},
	},
	"RegexNoGrastate": &types.LogRegex{
		Regex: regexp.MustCompile("Could not open state file for reading.*grastate.dat"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "no grastate.dat file"))
		},
		Verbosity: types.Detailed,
	},
	"RegexBootstrapingDefaultState": &types.LogRegex{
		Regex: regexp.MustCompile("Bootstraping with default state"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "bootstrapping(empty grastate)"))
		},
	},
}

/*
var (
	"SELF-LEAVE."
	REGEX_NODE_INACTIVE     = "declaring inactive"
	REGEX_NODE_TIMEOUT      = "timed out, no messages seen in"
	REGEX_INCONSISTENT_VIEW = "node uuid:.*is inconsistent to restored view"
)
*/

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

2022-11-29T23:34:51.820009-05:00 0 [Warning] [MY-000000] [Galera] Could not find peer: c0ff4085-5ad7-11ed-8b74-cfeec74147fe

2022-12-07  1:00:06 0 [Note] WSREP: Member 0.0 (node) synced with group.


2021-03-25T21:58:13.570928Z 0 [Warning] WSREP: no nodes coming from prim view, prim not possible
2021-03-25T21:58:13.855983Z 0 [Warning] WSREP: Quorum: No node with complete state:



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

 Transport endpoint is not connected


 2023-03-31T08:05:57.964535Z 0 [Note] WSREP: handshake failed, my group: '<group>', peer group: '<bad group>'

 2023-04-04T22:35:23.487304Z 0 [Warning] [MY-000000] [Galera] Handshake failed: tlsv1 alert decrypt error

 2023-04-16T19:35:06.875877Z 0 [Warning] [MY-000000] [Galera] Action message in non-primary configuration from member 0

{"log":"2023-06-10T04:50:46.835491Z 0 [Note] [MY-000000] [Galera] going to give up, state dump for diagnosis:\nevs::proto(evs::proto(6d0345f5-bcc0, GATHER, view_id(REG,02e369be-8363,1046)), GATHER) {\ncurrent_view=Current view of cluster as seen by this node\nview (view_id(REG,02e369be-8363,1046)\nmemb {\n\t02e369be-8363,0\n\t49761f3d-bd34,0\n\t6d0345f5-bcc0,0\n\tb05443d1-96bf,0\n\tb05443d1-96c0,0\n\t}\njoined {\n\t}\nleft {\n\t}\npartitioned {\n\t}\n),\ninput_map=evs::input_map: {aru_seq=461,safe_seq=461,node_index=node: {idx=0,range=[462,461],safe_seq=461} node: {idx=1,range=[462,461],safe_seq=461} node: {idx=2,range=[462,461],safe_seq=461} node: {idx=3,range=[462,461],safe_seq=461} node: {idx=4,range=[462,461],safe_seq=461} },\nfifo_seq=221418422,\nlast_sent=461,\nknown:\n","file":"/var/lib/mysql/mysqld-error.log"}


*/
