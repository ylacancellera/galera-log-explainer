package main

import (
	"encoding/json"

	"github.com/ylacancellera/galera-log-explainer/regex"
	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

type NodeInfo struct {
	Input     string   `json:"input"`
	IPs       []string `json:"IPs"`
	NodeNames []string `json:"nodeNames"`
	Hostname  string   `json:"hostname"`
	NodeUUIDs []string `json:"nodeUUIDs:"`
}

func WhoIs(ctxs map[string]types.LogCtx, search string) string {
	ni := NodeInfo{Input: search}
	if regex.IsNodeUUID(search) {
		search = utils.UUIDToShortUUID(search)
	}
	for _, ctx := range ctxs {
		if utils.SliceContains(ctx.OwnNames, search) || utils.SliceContains(ctx.OwnHashes, search) || utils.SliceContains(ctx.OwnIPs, search) {
			ni.NodeNames = ctx.OwnNames
			ni.NodeUUIDs = ctx.OwnHashes
			ni.IPs = ctx.OwnIPs
			ni.Hostname = ctx.OwnHostname()
		}
	}
	json, err := json.MarshalIndent(ni, "", "\t")
	if err != nil {
		return ""
	}
	return string(json)
}
