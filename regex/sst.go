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
				ctx.SST.ResyncedFromNode = donor
				return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " will resync local node"))
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.SST.ResyncingNode = joiner
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
				ctx.SST.ResyncedFromNode = ""
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "finished resyncing from ") + displayDonor)
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.SST.ResyncingNode = ""
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "finished sending SST to ") + displayJoiner)
			}

			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " synced ") + displayJoiner)
		},
	},

	// some weird ones:
	// 2022-12-24T03:27:41.966118Z 0 [Note] WSREP: 0.0 (name): State transfer to -1.-1 (left the group) complete.
	"RegexSSTCompleteUnknown": &types.LogRegex{
		Regex:         regexp.MustCompile("State transfer to.*left the group.*complete"),
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

	"RegexSSTFailedUnknown": &types.LogRegex{
		Regex:         regexp.MustCompile("State transfer to.*left the group.*failed"),
		InternalRegex: regexp.MustCompile("\\(" + regexNodeName + "\\): State transfer.*\\(left the group\\) failed"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			donor := r[internalRegex.SubexpIndex(groupNodeName)]
			displayDonor := types.ShortNodeName(donor)
			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.RedText, " failed to sync ??(node left)"))
		},
	},

	"RegexSSTStateTransferFailed": &types.LogRegex{
		Regex:         regexp.MustCompile("State transfer to.*failed:"),
		InternalRegex: regexp.MustCompile("\\(" + regexNodeName + "\\): State transfer.*\\(" + regexNodeName2 + "\\) failed"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			donor := r[internalRegex.SubexpIndex(groupNodeName)]
			joiner := r[internalRegex.SubexpIndex(groupNodeName2)]
			displayDonor := types.ShortNodeName(donor)
			displayJoiner := types.ShortNodeName(joiner)
			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.RedText, " failed to sync ") + displayJoiner)
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
			ctx.SST.Type = "SST"

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
			if ctx.SST.ResyncingNode == "" { // we should already have something at this point
				ctx.SST.ResyncingNode = node
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

	"RegexFailedToPrepareIST": &types.LogRegex{
		Regex: regexp.MustCompile("Failed to prepare for incremental state transfer"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.SST.Type = "SST"
			return ctx, types.SimpleDisplayer("IST is not applicable")
		},
	},

	// could not find production examples yet, but it did exist in older version there also was "Bypassing state dump"
	"RegexBypassSST": &types.LogRegex{
		Regex: regexp.MustCompile("Bypassing SST"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.SST.Type = "IST"
			return ctx, types.SimpleDisplayer("IST will be used")
		},
	},

	"RegexSocatConnRefused": &types.LogRegex{
		Regex: regexp.MustCompile("E connect.*Connection refused"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "socat: connection refused"))
		},
	},

	// 2023-05-12T02:52:33.767132Z 0 [Note] [MY-000000] [WSREP-SST] Preparing the backup at /var/lib/mysql/sst-xb-tmpdir
	"RegexPreparingBackup": &types.LogRegex{
		Regex: regexp.MustCompile("Preparing the backup at"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer("preparing SST backup")
		},
		Verbosity: types.Detailed,
	},

	"RegexTimeoutReceivingFirstData": &types.LogRegex{
		Regex: regexp.MustCompile("Possible timeout in receving first data from donor in gtid/keyring stage"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "timeout from donor in gtid/keyring stage"))
		},
	},

	"RegexWillNeverReceive": &types.LogRegex{
		Regex: regexp.MustCompile("Will never receive state. Need to abort"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "will never receive SST, aborting"))
		},
	},

	"RegexISTFailed": &types.LogRegex{
		Regex:         regexp.MustCompile("async IST sender failed to serve"),
		InternalRegex: regexp.MustCompile("IST sender failed to serve " + regexNodeIPMethod + ".*\\((?P<error>.*)\\)"),
		Handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			node := r[internalRegex.SubexpIndex(groupNodeIP)]
			istError := r[internalRegex.SubexpIndex("error")]

			return ctx, func(ctx types.LogCtx) string {
				return "IST to " + types.DisplayNodeSimplestForm(ctx, node) + utils.Paint(utils.RedText, " failed: ") + istError
			}
		},
	},
}

/*

2023-06-07T02:42:29.734960-06:00 0 [ERROR] WSREP: sst sent called when not SST donor, state SYNCED
2023-06-07T02:42:00.234711-06:00 0 [Warning] WSREP: Protocol violation. JOIN message sender 0.0 (node1) is not in state transfer (SYNCED). Message ignored.

)
*/
