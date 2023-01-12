package main

import (
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

func whoIs(ctxs map[string]types.LogCtx, search string) NodeInfo {
	ni := NodeInfo{Input: search}
	if regex.IsNodeUUID(search) {
		search = utils.UUIDToShortUUID(search)
	}
	var (
		ips       []string
		hashes    []string
		nodenames []string
	)

	for _, ctx := range ctxs {
		if utils.SliceContains(ctx.OwnNames, search) || utils.SliceContains(ctx.OwnHashes, search) || utils.SliceContains(ctx.OwnIPs, search) {
			ni.NodeNames = ctx.OwnNames
			ni.NodeUUIDs = ctx.OwnHashes
			ni.IPs = ctx.OwnIPs
			ni.Hostname = ctx.OwnHostname()
			return ni
		}

		if nodename, ok := ctx.HashToNodeName[search]; ok {
			nodenames = utils.SliceMergeDeduplicate(nodenames, []string{nodename})
			hashes = utils.SliceMergeDeduplicate(hashes, []string{search})
		}

		if ip, ok := ctx.HashToIP[search]; ok {
			ips = utils.SliceMergeDeduplicate(ips, []string{ip})
			hashes = utils.SliceMergeDeduplicate(hashes, []string{search})

		} else if nodename, ok := ctx.IPToNodeName[search]; ok {
			nodenames = utils.SliceMergeDeduplicate(nodenames, []string{nodename})
			ips = utils.SliceMergeDeduplicate(ips, []string{search})

		} else if utils.SliceContains(ctx.AllNodeNames(), search) {
			nodenames = []string{search}
		}

		for _, nodename := range nodenames {
			hashes = utils.SliceMergeDeduplicate(hashes, ctx.HashesFromNodeName(nodename))
			ips = utils.SliceMergeDeduplicate(ips, ctx.IPsFromNodeName(nodename))
		}

		for _, ip := range ips {
			hashes = utils.SliceMergeDeduplicate(hashes, ctx.HashesFromIP(ip))
			nodename, ok := ctx.IPToNodeName[ip]
			if ok {
				nodenames = utils.SliceMergeDeduplicate(nodenames, []string{nodename})
			}
		}
		for _, hash := range hashes {
			nodenames = utils.SliceMergeDeduplicate(nodenames, []string{ctx.HashToNodeName[hash]})
		}
	}
	ni.NodeNames = nodenames
	ni.NodeUUIDs = hashes
	ni.IPs = ips
	return ni
}
