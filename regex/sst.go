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
				return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " accepted to resync local node"))
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.ResyncingNode = joiner
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "local node accepted to resync ") + displayJoiner)
			}

			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " accepted to resync ") + displayJoiner)
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

			// some weird ones:
			// 2022-12-24T03:27:41.966118Z 0 [Note] WSREP: 0.0 (name): State transfer to -1.-1 (left the group) complete.
			if displayJoiner == "left the group" {
				return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.RedText, " synced ??(node left)"))
			}
			return ctx, types.SimpleDisplayer(displayDonor + utils.Paint(utils.GreenText, " synced ") + displayJoiner)
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

2023-03-20 17:26:27 140666176771840 [Warning] WSREP: 1.0 (node1): State transfer to 0.0 (node2) failed: -255 (Unknown error 255)

2023-03-20 17:26:27 140666176771840 [ERROR] WSREP: gcs/src/gcs_group.cpp:gcs_group_handle_join_msg():736: Will never receive state. Need to abort.


)
*/
