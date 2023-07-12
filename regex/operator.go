package regex

import (
	"regexp"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

func init() {
	setType(types.PXCOperatorRegexType, PXCOperatorMap)
}

// Regexes from this type should only be about operator extra logs
// it should not contain Galera logs
// Specifically operators are dumping configuration files, recoveries, script outputs, ...
// only those should be handled here, they are specific to pxc operator but still very insightful
var PXCOperatorMap = types.RegexMap{
	"RegexNodeNameFromEnv": &types.LogRegex{
		Regex:         regexp.MustCompile(". NODE_NAME="),
		InternalRegex: regexp.MustCompile("NODE_NAME=" + regexNodeName),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			nodename := r[internalRegex.SubexpIndex(groupNodeName)]
			nodename, _, _ = strings.Cut(nodename, ".")
			ctx.AddOwnName(nodename)
			return ctx, types.SimpleDisplayer("local name(operator):" + nodename)
		},
		Verbosity: types.DebugMySQL,
	},

	"RegexNodeIPFromEnv": &types.LogRegex{
		Regex:         regexp.MustCompile(". NODE_IP="),
		InternalRegex: regexp.MustCompile("NODE_IP=" + regexNodeIP),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ip := r[internalRegex.SubexpIndex(groupNodeIP)]
			ctx.AddOwnIP(ip)
			return ctx, types.SimpleDisplayer("local ip(operator):" + ip)
		},
		Verbosity: types.DebugMySQL,
	},

	// Why is it not in regular "views" regexes:
	// it could have been useful as an "verbosity=types.Detailed" regexes, very rarely
	// but in context of operators, it is actually a very important information
	"RegexGcacheScan": &types.LogRegex{
		// those "operators" regexes do not have the log prefix added implicitely. It's not strictly needed, but
		// it will help to avoid catching random piece of log out of order
		Regex: regexp.MustCompile("^{\"log\":\".*GCache::RingBuffer initial scan"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer("recovering gcache")
		},
	},
}

/*
{"log":"---- Starting the MySQL server used for post-processing ----\n","file":"/var/lib/mysql/mysqld.post.processing.log"}
{"log":"2023-01-25T11:00:18.305034Z 0 [Warning] [MY-000000] [WSREP] Node is not a cluster node. Disabling pxc_strict_mode\n","file":"/var/lib/mysql/mysqld.post.processing.log"}
{"log":"2023-01-25T11:00:18.305457Z 0 [System] [MY-010116] [Server] /usr/sbin/mysqld (mysqld 8.0.25-15.1) starting as process 17811\n","file":"/var/lib/mysql/mysqld.post.processing.log"}



{"log":"2023-07-10T11:48:30.223629Z 0 [System] [MY-010910] [Server] /usr/sbin/mysqld: Shutdown complete (mysqld 8.0.31-23.2)  Percona XtraDB Cluster (GPL), Release rel23, Revision e6e483f, WSREP version 26.1.4.3
.\n","file":"/var/lib/mysql/wsrep_recovery_verbose.log"}
{"log":"2023-01-25T11:00:31.755792Z 0 [System] [MY-010910] [Server] /usr/sbin/mysqld: Shutdown complete (mysqld 8.0.25-15.1)  Percona XtraDB Cluster (GPL), Release rel15, Revision 8638bb0, WSREP version 26.4.3.\n---- Stopped the MySQL server used for post-processing ----\n","file":"/var/lib/mysql/mysqld.post.processing.log"}

*/
