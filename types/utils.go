package types

// DisplayLocalNodeSimplestForm is used to identify a node timeline.
// This has impacts on how logs are identified in the global timelines,
// It will also impacts how logs are merged if we have multiple logs per nodes
//
// In order of preference: wsrep_node_name (or galera "node" name), hostname, ip, filepath
func DisplayLocalNodeSimplestForm(ctx LogCtx) string {
	if len(ctx.OwnNames) > 0 {
		return ctx.OwnNames[len(ctx.OwnNames)-1]
	}
	if len(ctx.OwnIPs) > 0 {
		return DisplayNodeSimplestForm(ctx, ctx.OwnIPs[len(ctx.OwnIPs)-1])
	}
	if len(ctx.OwnHashes) > 0 {
		if name, ok := ctx.HashToNodeName[ctx.OwnHashes[0]]; ok {
			return name
		}
		if ip, ok := ctx.HashToIP[ctx.OwnHashes[0]]; ok {
			return DisplayNodeSimplestForm(ctx, ip)
		}
	}
	return ctx.FilePath
}

// DisplayNodeSimplestForm is useful to get the most easily to read string for a given IP
// This only has impacts on display
// In order of preference: wsrep_node_name (or galera "node" name), hostname, ip
func DisplayNodeSimplestForm(ctx LogCtx, ip string) string {
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
