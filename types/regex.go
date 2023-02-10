package types

type RegexType string

var (
	EventsRegexType RegexType = "events"
	SSTRegexType    RegexType = "sst"
	ViewsRegexType  RegexType = "views"
	IdentRegexType  RegexType = "identity"
	StatesRegexType RegexType = "states"
)
