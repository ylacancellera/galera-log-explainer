package regex

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

func init() {
	setType(types.IdentRegexType, IdentsMap)
}

var IdentsMap = types.RegexMap{
	// sourceNode is to identify from which node this log was taken
	"RegexSourceNode": &types.LogRegex{
		Regex:         regexp.MustCompile("(local endpoint for a connection, blacklisting address)|(points to own listening address, blacklisting)"),
		InternalRegex: regexp.MustCompile("\\(" + regexNodeHash + ", '.+'\\).+" + regexNodeIPMethod),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.AddOwnIP(ip)
			return ctx, types.SimpleDisplayer(ip + " is local")
		},
		Verbosity: types.DebugMySQL,
	},

	// 2022-12-18T01:03:17.950545Z 0 [Note] [MY-000000] [Galera] Passing config to GCS: base_dir = /var/lib/mysql/; base_host = 127.0.0.1;
	"RegexBaseHost": &types.LogRegex{
		Regex:         regexp.MustCompile("base_host"),
		InternalRegex: regexp.MustCompile("base_host = " + regexNodeIP),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.AddOwnIP(ip)
			return ctx, types.SimpleDisplayer(ctx.OwnIPs[len(ctx.OwnIPs)-1] + " is local")
		},
		Verbosity: types.DebugMySQL,
	},

	//        0: 015702fc-32f5-11ed-a4ca-267f97316394, node-1
	//	      1: 08dd5580-32f7-11ed-a9eb-af5e3d01519e, garb
	// TODO: store indexes to later search for them using SST infos and STATES EXCHANGES logs. Could be unsafe if galera do not log indexes in time though
	"RegexMember": &types.LogRegex{
		Regex:         regexp.MustCompile("[0-9]: [a-z0-9]+-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]+, [a-zA-Z0-9-_]+"),
		InternalRegex: regexp.MustCompile(regexIdx + ": " + regexNodeHash4Dash + ", " + regexNodeName),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			idx := r[internalRegex.SubexpIndex(groupIdx)]
			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			nodename := r[internalRegex.SubexpIndex(groupNodeName)]

			// nodenames are truncated after 32 characters ...
			if len(nodename) == 31 {
				return ctx, nil
			}
			splitted := strings.Split(hash, "-")
			shorthash := splitted[0] + "-" + splitted[3]
			ctx.HashToNodeName[shorthash] = nodename

			//fmt.Println(shorthash, nodename)
			//	}
			if ctx.MyIdx == idx && ctx.State == "PRIMARY" {
				ctx.AddOwnHash(shorthash)
				fmt.Println("regexmember")
				ctx.AddOwnName(nodename)
			}

			return ctx, types.SimpleDisplayer(shorthash + " is " + nodename)
		},
		Verbosity: types.DebugMySQL,
	},

	// My UUID: 6938f4ae-32f4-11ed-be8d-8a0f53f88872
	"RegexOwnUUID": &types.LogRegex{
		Regex:         regexp.MustCompile("My UUID"),
		InternalRegex: regexp.MustCompile("My UUID: " + regexNodeHash4Dash),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			splitted := strings.Split(hash, "-")
			shorthash := splitted[0] + "-" + splitted[3]

			ctx.AddOwnHash(shorthash)

			return ctx, types.SimpleDisplayer(shorthash + " is local")
		},
		Verbosity: types.DebugMySQL,
	},

	// 2023-01-06T06:59:26.527748Z 0 [Note] WSREP: (9509c194, 'tcp://0.0.0.0:4567') turning message relay requesting on, nonlive peers:
	"RegexOwnUUIDFromMessageRelay": &types.LogRegex{
		Regex:         regexp.MustCompile("turning message relay requesting"),
		InternalRegex: regexp.MustCompile("\\(" + regexNodeHash + ", '" + regexNodeIPMethod + "'\\)"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			ctx.AddOwnHash(hash)

			return ctx, types.SimpleDisplayer(hash + " is local")
		},
		Verbosity: types.DebugMySQL,
	},

	// 2023-01-06T07:05:35.693861Z 0 [Note] WSREP: New COMPONENT: primary = yes, bootstrap = no, my_idx = 0, memb_num = 2
	"RegexMyIDXFromComponent": &types.LogRegex{
		Regex:         regexp.MustCompile("New COMPONENT:"),
		InternalRegex: regexp.MustCompile("New COMPONENT:.*my_idx = " + regexIdx),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			idx := r[internalRegex.SubexpIndex(groupIdx)]
			ctx.MyIdx = idx
			fmt.Println("myidx: " + idx)
			fmt.Println(log)
			return ctx, types.SimpleDisplayer("my_idx=" + idx)
		},
		Verbosity: types.DebugMySQL,
	},

	"RegexOwnNameFromStateExchange": &types.LogRegex{
		Regex:         regexp.MustCompile("STATE EXCHANGE: got state msg"),
		InternalRegex: regexp.MustCompile("STATE EXCHANGE:.* from " + regexIdx + " \\(" + regexNodeName + "\\)"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			idx := r[internalRegex.SubexpIndex(groupIdx)]
			name := r[internalRegex.SubexpIndex(groupNodeName)]
			if idx != ctx.MyIdx {
				return ctx, types.SimpleDisplayer("name(" + name + ") from unknown idx")
			}

			if ctx.State == "NON-PRIMARY" {
				return ctx, types.SimpleDisplayer("name(" + name + ") can't be trusted as it's non-primary")
			}

			if name != "cluster1-pxc-0" {
				fmt.Println("regexownnamefromstate")
				fmt.Println(log)
			}
			ctx.AddOwnName(name)
			return ctx, types.SimpleDisplayer("local name:" + name)
		},
		Verbosity: types.DebugMySQL,
	},
}

func init() {
	// 2023-01-06T07:05:34.035959Z 0 [Note] WSREP: (9509c194, 'tcp://0.0.0.0:4567') connection established to 838ebd6d tcp://ip:4567
	IdentsMap["RegexOwnUUIDFromEstablished"] = &types.LogRegex{
		Regex:         regexp.MustCompile("connection established to"),
		InternalRegex: IdentsMap["RegexOwnUUIDFromMessageRelay"].InternalRegex,
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return IdentsMap["RegexOwnUUIDFromMessageRelay"].Handler(internalRegex, ctx, log)
		},
		Verbosity: types.DebugMySQL,
	}

	IdentsMap["RegexOwnIndexFromView"] = &types.LogRegex{
		Regex:         regexp.MustCompile("own_index:"),
		InternalRegex: regexp.MustCompile("own_index: " + regexIdx),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return IdentsMap["RegexMyIDXFromComponent"].Handler(internalRegex, ctx, log)
		},
		Verbosity: types.DebugMySQL,
	}

	// 2023-01-06T07:05:35.698869Z 7 [Note] WSREP: New cluster view: global state: 00000000-0000-0000-0000-000000000000:0, view# 10: Primary, number of nodes: 2, my index: 0, protocol version 3
	// WARN: my index seems to always be 0 on this log on certain version. It had broken some nodenames
	// Curently disabled, not present in identsRegexes slice
	IdentsMap["RegexMyIDXFromClusterView"] = &types.LogRegex{
		Regex:         regexp.MustCompile("New cluster view:"),
		InternalRegex: regexp.MustCompile("New cluster view:.*my index: -?" + regexIdx + ","),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return IdentsMap["RegexMyIDXFromComponent"].Handler(internalRegex, ctx, log)
		},
		Verbosity: types.DebugMySQL,
	}
}
