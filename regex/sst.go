package regex

import (
	"regexp"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func init() {
	setType(types.SSTRegexType, SSTMap)
}

var SSTMap = types.RegexMap{
	// TODO: requested state from unknown node
	"RegexSSTRequestSuccess": &types.LogRegex{
		Regex:         regexp.MustCompile("requested state transfer.*Selected"),
		InternalRegex: regexp.MustCompile("Member .* \\(" + regexNodeName + "\\) requested state transfer.*Selected .* \\(" + regexNodeName2 + "\\)\\("),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			joiner := r[internalRegex.SubexpIndex(groupNodeName)]
			donor := r[internalRegex.SubexpIndex(groupNodeName2)]
			displayJoiner := types.ShortNodeName(joiner)
			displayDonor := types.ShortNodeName(donor)
			if utils.SliceContains(ctx.OwnNames, joiner) {
				ctx.ResyncedFromNode = donor
				return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " will resync local node"))
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.ResyncingNode = joiner
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "local node will resync ") + displayJoiner)
			}

			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " will resync ") + displayJoiner)
		},
		Verbosity: types.Detailed,
	},

	"RegexSSTResourceUnavailable": &types.LogRegex{
		Regex:         regexp.MustCompile("requested state transfer.*Resource temporarily unavailable"),
		InternalRegex: regexp.MustCompile("Member .* \\(" + regexNodeName + "\\) requested state transfer"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			joiner := r[internalRegex.SubexpIndex(groupNodeName)]
			if utils.SliceContains(ctx.OwnNames, joiner) {

				return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "cannot find donor"))
			}

			return ctx, types.SimpleDisplayer(joiner + utils.Paint(utils.YellowText, " cannot find donor"))
		},
		Verbosity: types.Detailed,
	},

	// 2022-12-24T03:28:22.444125Z 0 [Note] WSREP: 0.0 (name): State transfer to 2.0 (name2) complete.
	"RegexSSTComplete": &types.LogRegex{
		Regex:         regexp.MustCompile("State transfer to.*complete"),
		InternalRegex: regexp.MustCompile("\\(" + regexNodeName + "\\): State transfer.*\\(" + regexNodeName2 + "\\) complete"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			donor := r[internalRegex.SubexpIndex(groupNodeName)]
			joiner := r[internalRegex.SubexpIndex(groupNodeName2)]
			displayJoiner := types.ShortNodeName(joiner)
			displayDonor := types.ShortNodeName(donor)
			if utils.SliceContains(ctx.OwnNames, joiner) {
				ctx.ResyncedFromNode = ""
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "finished resyncing from ") + displayDonor)
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.ResyncingNode = ""
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "finished sending SST to ") + displayJoiner)
			}

			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " synced ") + displayJoiner)
		},
	},

	// some weird ones:
	// 2022-12-24T03:27:41.966118Z 0 [Note] WSREP: 0.0 (name): State transfer to -1.-1 (left the group) complete.
	"RegexSSTCompleteUnknown": &types.LogRegex{
		Regex:         regexp.MustCompile("State transfer to.*complete"),
		InternalRegex: regexp.MustCompile("\\(" + regexNodeName + "\\): State transfer.*\\(left the group\\) complete"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			donor := r[internalRegex.SubexpIndex(groupNodeName)]
			displayDonor := types.ShortNodeName(donor)
			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.RedText, " synced ??(node left)"))
		},
	},

	"RegexSSTError": &types.LogRegex{
		Regex: regexp.MustCompile("Process completed with error: wsrep_sst"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "SST error"))
		},
	},

	"RegexSSTCancellation": &types.LogRegex{
		Regex: regexp.MustCompile("Initiating SST cancellation"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "Former SST cancelled"))
		},
	},

	"RegexSSTProceeding": &types.LogRegex{
		Regex: regexp.MustCompile("Proceeding with SST"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "JOINER"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "Receiving SST"))
		},
	},

	"RegexSSTStreamingTo": &types.LogRegex{
		Regex:         regexp.MustCompile("Streaming the backup to"),
		InternalRegex: regexp.MustCompile("Streaming the backup to joiner at " + regexNodeIP),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			ctx.State = "DONOR"
			node := r[internalRegex.SubexpIndex(groupNodeIP)]
			if ctx.ResyncingNode == "" { // we should already have something at this point
				ctx.ResyncingNode = node
			}

			return ctx, func(ctx types.LogCtx) string {
				return utils.Paint(utils.YellowText, "SST to ") + types.DisplayNodeSimplestForm(ctx, node)
			}
		},
	},

	"RegexISTReceived": &types.LogRegex{
		Regex: regexp.MustCompile("IST received"),

		// the UUID here is not from a node, it's a cluster state UUID, this is only used to ensure it's correctly parsed
		InternalRegex: regexp.MustCompile("IST received: " + regexNodeHash4Dash + ":" + regexSeqno),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			seqno := r[internalRegex.SubexpIndex(groupSeqno)]
			return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "IST received") + "(seqno:" + seqno + ")")
		},
	},

	"RegexISTSender": &types.LogRegex{
		Regex: regexp.MustCompile("IST sender starting"),

		InternalRegex: regexp.MustCompile("IST sender starting to serve " + regexNodeIPMethod + " sending [0-9]+-" + regexSeqno),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			seqno := r[internalRegex.SubexpIndex(groupSeqno)]
			node := r[internalRegex.SubexpIndex(groupNodeIP)]

			return ctx, func(ctx types.LogCtx) string {
				return utils.Paint(utils.YellowText, "IST to ") + types.DisplayNodeSimplestForm(ctx, node) + "(seqno:" + seqno + ")"
			}
		},
	},
}

/*
var (
REGEX_IST_UNAVAILABLE="Failed to prepare for incremental state transfer"
REGEX_SST_BYPASS="\(Bypassing state dump\|IST sender starting\|IST received\)"
        2023-06-09T06:42:36.266382Z WSREP_SST: [INFO] Bypassing SST. Can work it through IST
2023/06/09 06:43:06 socat[23579] E connect(62, AF=2 172.17.0.20:4444, 16): Connection refused
2022-11-29T23:34:51.820069-05:00 0 [Warning] [MY-000000] [Galera] 0.1 (node): State transfer to -1.-1 (left the group) failed: -111 (Connection refused)


2023-03-20 17:26:27 140666176771840 [Warning] WSREP: 1.0 (node1): State transfer to 0.0 (node2) failed: -255 (Unknown error 255)

2023-03-20 17:26:27 140666176771840 [ERROR] WSREP: gcs/src/gcs_group.cpp:gcs_group_handle_join_msg():736: Will never receive state. Need to abort.

2023-05-12T02:52:33.767132Z 0 [Note] [MY-000000] [WSREP-SST] Preparing the backup at /var/lib/mysql/sst-xb-tmpdir

2023-06-07T02:42:29.734960-06:00 0 [ERROR] WSREP: sst sent called when not SST donor, state SYNCED
2023-06-07T02:42:00.234711-06:00 0 [Warning] WSREP: Protocol violation. JOIN message sender 0.0 (node1) is not in state transfer (SYNCED). Message ignored.

        2023-06-09T06:41:43.752403Z WSREP_SST: [ERROR] Possible timeout in receving first data from donor in gtid/keyring stage

		2023-06-09T06:41:43.807462Z 0 [ERROR] WSREP: async IST sender failed to serve tcp://172.17.0.2:4568: ist send failed: asio.system:104', asio error 'write: Connection reset by peer': 104 (Connection reset by peer)


)
*/
