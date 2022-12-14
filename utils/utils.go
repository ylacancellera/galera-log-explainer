package utils

import (
	"fmt"
	"strings"
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
