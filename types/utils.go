package types

func DisplayLocalNodeSimplestForm(ctx LogCtx) string {
	if len(ctx.SourceNodeIPs) > 0 {
		return DisplayNodeSimplestForm(ctx.SourceNodeIPs[len(ctx.SourceNodeIPs)-1], ctx)
	}
	return ctx.FilePath
}

func DisplayNodeSimplestForm(ip string, ctx LogCtx) string {
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

func MergeContextsInfo(ctxs map[string]LogCtx) map[string]LogCtx {
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
	return ctxs
}
