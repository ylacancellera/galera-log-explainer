package regex

import (
	"regexp"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

var SSTRegexes = []LogRegex{RegexSSTRequestSuccess, RegexSSTResourceUnavailable, RegexSSTComplete, RegexSSTError, RegexISTReceived, RegexSSTCancellation, RegexSSTProceeding, RegexISTSender, RegexSSTStreamingTo}

var (
	RegexSSTRequestSuccess = LogRegex{
		Regex:         regexp.MustCompile("requested state transfer.*Selected"),
		internalRegex: regexp.MustCompile("Member .* \\(" + regexNodeName + "\\) requested state transfer.*Selected .* \\(" + regexNodeName2 + "\\)\\("),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			joiner := r[internalRegex.SubexpIndex(groupNodeName)]
			donor := r[internalRegex.SubexpIndex(groupNodeName2)]

			if utils.SliceContains(ctx.OwnNames, joiner) {
				ctx.ResyncedFromNode = donor
				return ctx, types.SimpleDisplayer(donor + utils.Paint(utils.GreenText, " accepted to resync local node"))
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.ResyncingNode = joiner
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "local node accepted to resync ") + joiner)
			}

			return ctx, types.SimpleDisplayer(donor + utils.Paint(utils.GreenText, " accepted to resync ") + joiner)
		},
		Verbosity: types.Detailed,
	}

	RegexSSTResourceUnavailable = LogRegex{
		Regex:         regexp.MustCompile("requested state transfer.*Resource temporarily unavailable"),
		internalRegex: regexp.MustCompile("Member .* \\(" + regexNodeName + "\\) requested state transfer"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
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
	}

	RegexSSTComplete = LogRegex{
		Regex:         regexp.MustCompile("State transfer to.*complete"),
		internalRegex: regexp.MustCompile("\\(" + regexNodeName + "\\): State transfer.*\\(" + regexNodeName2 + "\\) complete"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			donor := r[internalRegex.SubexpIndex(groupNodeName)]
			joiner := r[internalRegex.SubexpIndex(groupNodeName2)]
			if utils.SliceContains(ctx.OwnNames, joiner) {
				ctx.ResyncedFromNode = ""
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "finished resyncing from ") + donor)
			}
			if utils.SliceContains(ctx.OwnNames, donor) {
				ctx.ResyncingNode = ""
				return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "finished sending SST to ") + joiner)
			}

			return ctx, types.SimpleDisplayer(joiner + utils.Paint(utils.GreenText, " finished resyncing from ") + donor)
		},
	}

	RegexSSTError = LogRegex{
		Regex: regexp.MustCompile("Process completed with error: wsrep_sst"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "SST error"))
		},
	}

	RegexSSTCancellation = LogRegex{
		Regex: regexp.MustCompile("Initiating SST cancellation"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {

			return ctx, types.SimpleDisplayer(utils.Paint(utils.RedText, "Former SST cancelled"))
		},
	}

	RegexSSTProceeding = LogRegex{
		Regex: regexp.MustCompile("Proceeding with SST"),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			ctx.State = "JOINER"

			return ctx, types.SimpleDisplayer(utils.Paint(utils.YellowText, "Receiving SST"))
		},
	}

	RegexSSTStreamingTo = LogRegex{
		Regex:         regexp.MustCompile("Streaming the backup to"),
		internalRegex: regexp.MustCompile("Streaming the backup to joiner at " + regexNodeIP),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
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
	}

	RegexISTReceived = LogRegex{
		Regex: regexp.MustCompile("IST received"),

		// the UUID here is not from a node, it's a cluster state UUID, this is only used to ensure it's correctly parsed
		internalRegex: regexp.MustCompile("IST received: " + regexNodeHash4Dash + ":" + regexSeqno),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
			r, err := internalRegexSubmatch(internalRegex, log)
			if err != nil {
				return ctx, nil
			}

			seqno := r[internalRegex.SubexpIndex(groupSeqno)]
			return ctx, types.SimpleDisplayer(utils.Paint(utils.GreenText, "IST received") + "(seqno:" + seqno + ")")
		},
	}

	RegexISTSender = LogRegex{
		Regex: regexp.MustCompile("IST sender starting"),

		internalRegex: regexp.MustCompile("IST sender starting to serve " + regexNodeIPMethod + " sending [0-9]+-" + regexSeqno),
		handler: func(internalRegex *regexp.Regexp, ctx types.LogCtx, log string) (types.LogCtx, types.LogDisplayer) {
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
	}
)

/*
var (
REGEX_IST_UNAVAILABLE="Failed to prepare for incremental state transfer"
REGEX_SST_BYPASS="\(Bypassing state dump\|IST sender starting\|IST received\)"
)
*/
