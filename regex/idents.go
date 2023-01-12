package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

var IdentRegexes = []LogRegex{RegexSourceNode, RegexBaseHost, RegexMember, RegexOwnUUID, RegexMyIDXFromComponent, RegexOwnNameFromStateExchange, RegexOwnUUIDFromEstablished, RegexOwnUUIDFromMessageRelay}

var (
	// sourceNode is to identify from which node this log was taken
	RegexSourceNode = LogRegex{
		Regex:         regexp.MustCompile("(local endpoint for a connection, blacklisting address)|(points to own listening address, blacklisting)"),
		internalRegex: regexp.MustCompile("\\(" + regexNodeHash + ", '.+'\\).+" + regexNodeIPMethod),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.AddOwnIP(ip)
			return ctx, types.SimpleDisplayer(ip + " is local")
		},
		Verbosity: types.DebugMySQL,
	}
)

var (

	// 2022-12-18T01:03:17.950545Z 0 [Note] [MY-000000] [Galera] Passing config to GCS: base_dir = /var/lib/mysql/; base_host = 127.0.0.1;
	RegexBaseHost = LogRegex{
		Regex:         regexp.MustCompile("base_host"),
		internalRegex: regexp.MustCompile("base_host = " + regexNodeIP),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.AddOwnIP(ip)
			return ctx, types.SimpleDisplayer(ctx.OwnIPs[len(ctx.OwnIPs)-1] + " is local")
		},
		Verbosity: types.DebugMySQL,
	}

	//        0: 015702fc-32f5-11ed-a4ca-267f97316394, node-1
	//	      1: 08dd5580-32f7-11ed-a9eb-af5e3d01519e, garb
	RegexMember = LogRegex{
		Regex:         regexp.MustCompile("[0-9]: " + regexNodeHash4Dash + ", " + regexNodeName),
		internalRegex: regexp.MustCompile("[0-9]: " + regexNodeHash4Dash + ", " + regexNodeName),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			nodename := r[internalRegex.SubexpIndex(groupNodeName)]
			splitted := strings.Split(hash, "-")
			shorthash := splitted[0] + "-" + splitted[3]
			ctx.HashToNodeName[shorthash] = nodename

			return ctx, types.SimpleDisplayer(shorthash + " is " + nodename)
		},
		Verbosity: types.DebugMySQL,
	}

	// My UUID: 6938f4ae-32f4-11ed-be8d-8a0f53f88872
	RegexOwnUUID = LogRegex{
		Regex:         regexp.MustCompile("My UUID"),
		internalRegex: regexp.MustCompile("My UUID: " + regexNodeHash4Dash),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
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
	}

	// 2023-01-06T06:59:26.527748Z 0 [Note] WSREP: (9509c194, 'tcp://0.0.0.0:4567') turning message relay requesting on, nonlive peers:
	RegexOwnUUIDFromMessageRelay = LogRegex{
		Regex:         regexp.MustCompile("turning message relay requesting"),
		internalRegex: regexp.MustCompile("\\(" + regexNodeHash + ", '" + regexNodeIPMethod + "'\\)"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			ctx.AddOwnHash(hash)

			return ctx, types.SimpleDisplayer(hash + " is local")
		},
		Verbosity: types.DebugMySQL,
	}

	// 2023-01-06T07:05:34.035959Z 0 [Note] WSREP: (9509c194, 'tcp://0.0.0.0:4567') connection established to 838ebd6d tcp://ip:4567
	RegexOwnUUIDFromEstablished = LogRegex{
		Regex:         regexp.MustCompile("connection established to"),
		internalRegex: RegexOwnUUIDFromMessageRelay.internalRegex,
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return RegexOwnUUIDFromMessageRelay.handler(internalRegex, ctx, log)
		},
		Verbosity: types.DebugMySQL,
	}

	// 2023-01-06T07:05:35.693861Z 0 [Note] WSREP: New COMPONENT: primary = yes, bootstrap = no, my_idx = 0, memb_num = 2
	RegexMyIDXFromComponent = LogRegex{
		Regex:         regexp.MustCompile("New COMPONENT:"),
		internalRegex: regexp.MustCompile("New COMPONENT:.*my_idx = -?" + regexMyIdx),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			idx := r[internalRegex.SubexpIndex(groupMyIdx)]
			ctx.MyIdx = idx
			return ctx, types.SimpleDisplayer("my_idx=" + idx)
		},
		Verbosity: types.DebugMySQL,
	}

	// 2023-01-06T07:05:35.698869Z 7 [Note] WSREP: New cluster view: global state: 00000000-0000-0000-0000-000000000000:0, view# 10: Primary, number of nodes: 2, my index: 0, protocol version 3
	// WARN: my index seems to always be 0 on this log on certain version. It had broken some nodenames
	// Curently disabled, not present in identsRegexes slice
	RegexMyIDXFromClusterView = LogRegex{
		Regex:         regexp.MustCompile("New cluster view:"),
		internalRegex: regexp.MustCompile("New cluster view:.*my index: -?" + regexMyIdx + ","),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return RegexMyIDXFromComponent.handler(internalRegex, ctx, log)
		},
		Verbosity: types.DebugMySQL,
	}

	RegexOwnNameFromStateExchange = LogRegex{
		Regex:         regexp.MustCompile("STATE EXCHANGE: got state msg"),
		internalRegex: regexp.MustCompile("STATE EXCHANGE:.* from " + regexMyIdx + " \\(" + regexNodeName + "\\)"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			idx := r[internalRegex.SubexpIndex(groupMyIdx)]
			name := r[internalRegex.SubexpIndex(groupNodeName)]
			if idx != ctx.MyIdx {
				return ctx, types.SimpleDisplayer("name from unknown idx")
			}
			ctx.AddOwnName(name)
			return ctx, types.SimpleDisplayer("local name:" + name)
		},
		Verbosity: types.DebugMySQL,
	}
)
