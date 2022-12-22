package utils

import (
	"fmt"
	"strings"

	"github.com/ylacancellera/galera-log-explainer/types"
)

// Color is given its own type for safe function signatures
type Color string

// Color codes interpretted by the terminal
// NOTE: all codes must be of the same length or they will throw off the field alignment of tabwriter
const (
	ResetText         Color = "\x1b[0000m"
	BrightText              = "\x1b[0001m"
	RedText                 = "\x1b[0031m"
	GreenText               = "\x1b[0032m"
	YellowText              = "\x1b[0033m"
	BlueText                = "\x1b[0034m"
	MagentaText             = "\x1b[0035m"
	CyanText                = "\x1b[0036m"
	WhiteText               = "\x1b[0037m"
	DefaultText             = "\x1b[0039m"
	BrightRedText           = "\x1b[1;31m"
	BrightGreenText         = "\x1b[1;32m"
	BrightYellowText        = "\x1b[1;33m"
	BrightBlueText          = "\x1b[1;34m"
	BrightMagentaText       = "\x1b[1;35m"
	BrightCyanText          = "\x1b[1;36m"
	BrightWhiteText         = "\x1b[1;37m"
)

var SkipColor bool

// Color implements the Stringer interface for interoperability with string
func (c *Color) String() string {
	return string(*c)
}

func Paint(color Color, value string) string {
	if SkipColor {
		return value
	}
	return fmt.Sprintf("%v%v%v", color, value, ResetText)
}

func ColorForState(text, state string) string {

	switch state {
	case "DONOR", "JOINER", "DESYNCED":
		return Paint(YellowText, text)
	case "SYNCED":
		return Paint(GreenText, text)
	case "CLOSED", "NON-PRIMARY":
		return Paint(RedText, text)
	default:
		return text
	}
}

func SliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// StringsReplaceReversed is similar to strings.Replace, but replacing the
// right-most elements instead of left-most
func StringsReplaceReversed(s, old, new string, n int) string {

	s2 := s
	stop := len(s)

	for i := 0; i < n; i++ {
		stop = strings.LastIndex(s[:stop], old)

		s2 = (s[:stop]) + new + s2[stop+len(old):]
	}
	return s2
}

func DisplayLocalNodeSimplestForm(ctx types.LogCtx) string {
	if len(ctx.SourceNodeIP) > 0 {
		return DisplayNodeSimplestForm(ctx.SourceNodeIP[len(ctx.SourceNodeIP)-1], ctx)
	}
	return ctx.FilePath
}

func DisplayNodeSimplestForm(ip string, ctx types.LogCtx) string {
	if nodename, ok := ctx.IPToNodeName[ip]; ok {
		return nodename
	}

	for hash, storedip := range ctx.HashToIP {
		if ip == storedip {
			if nodename, ok := ctx.HashToNodeName[hash]; ok {
				return nodename
			}
		}
	}
	if hostname, ok := ctx.IPToHostname[ip]; ok {
		return hostname
	}
	return ip
}

func MergeContextsInfo(ctxs map[string]types.LogCtx) map[string]types.LogCtx {
	if len(ctxs) == 1 {
		return ctxs
	}
	//base := ctxs[]
	//ctxs = ctxs[1:]
	for i, base := range ctxs {
		for j, ctx := range ctxs {
			if i == j {
				continue
			}
			for hash, ip := range ctx.HashToIP {
				base.HashToIP[hash] = ip
			}
			for hash, nodename := range ctx.HashToNodeName {
				base.HashToNodeName[hash] = nodename
			}
			for ip, hostname := range ctx.IPToHostname {
				base.IPToHostname[ip] = hostname
			}
			for ip, nodename := range ctx.IPToNodeName {
				base.IPToNodeName[ip] = nodename
			}
			for ip, method := range ctx.IPToMethod {
				base.IPToMethod[ip] = method
			}
		}
	}
	/*
		for _, ctx := range ctxs {
			ctx.HashToIP = base.HashToIP
			ctx.HashToNodeName = base.HashToNodeName
			ctx.IPToHostname = base.IPToHostname
			ctx.IPToNodeName = base.IPToNodeName
			ctx.IPToMethod = base.IPToMethod
		}*/
	return ctxs
}
