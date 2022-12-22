package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

var IdentRegexes = []LogRegex{RegexSourceNode, RegexBaseHost, RegexMember, RegexOwnUUID}

var (
	// sourceNode is to identify from which node this log was taken
	RegexSourceNode = LogRegex{
		Regex:         regexp.MustCompile("(local endpoint for a connection, blacklisting address)|(points to own listening address, blacklisting)"),
		internalRegex: regexp.MustCompile("\\(" + regexNodeHash + ", '.+'\\).+" + regexNodeIPMethod),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.SourceNodeIP = append(ctx.SourceNodeIP, ip)
			ctx.HashToIP[r[internalRegex.SubexpIndex(groupNodeHash)]] = ip
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
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			ctx.SourceNodeIP = append(ctx.SourceNodeIP, r[internalRegex.SubexpIndex(groupNodeIP)])
			return ctx, types.SimpleDisplayer(ctx.SourceNodeIP[len(ctx.SourceNodeIP)-1] + " is local")
		},
		Verbosity: types.DebugMySQL,
	}

	//        0: 015702fc-32f5-11ed-a4ca-267f97316394, node-1
	//	      1: 08dd5580-32f7-11ed-a9eb-af5e3d01519e, garb
	RegexMember = LogRegex{
		Regex:         regexp.MustCompile("[0-9]: " + regexNodeHash4Dash + ", " + regexNodeName),
		internalRegex: regexp.MustCompile("[0-9]: " + regexNodeHash4Dash + ", " + regexNodeName),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

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
			r := internalRegex.FindAllStringSubmatch(log, -1)[0]

			hash := r[internalRegex.SubexpIndex(groupNodeHash)]
			splitted := strings.Split(hash, "-")
			shorthash := splitted[0] + "-" + splitted[3]

			ctx.OwnHashes = append(ctx.OwnHashes, shorthash)
			for _, ip := range ctx.SourceNodeIP {
				ctx.HashToIP[shorthash] = ip
			}

			return ctx, types.SimpleDisplayer(shorthash + " is local")
		},
		Verbosity: types.DebugMySQL,
	}
)
