package types

type NodeInfo struct {
	Input     string   `json:"input"`
	IPs       []string `json:"IPs"`
	NodeNames []string `json:"nodeNames"`
	Hostname  string   `json:"hostname"`
	NodeUUIDs []string `json:"nodeUUIDs:"`
}
