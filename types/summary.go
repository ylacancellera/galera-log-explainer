package types

type GroupedEvent struct {
	Base LogInfo

	// string key describes nodes
	Proofs map[string]LogInfo

	SubGroups []GroupedEvent
}
