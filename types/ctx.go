package types

import "github.com/ylacancellera/galera-log-explainer/utils"

// LogCtx is a context for a given file.
// It used to keep track of what is going on at each new event.
type LogCtx struct {
	FilePath         string
	OwnIPs           []string
	OwnHashes        []string
	OwnNames         []string
	State            string
	ResyncingNode    string
	ResyncedFromNode string
	MyIdx            string
	HashToIP         map[string]string
	HashToNodeName   map[string]string
	IPToHostname     map[string]string
	IPToMethod       map[string]string
	IPToNodeName     map[string]string
}

func NewLogCtx() LogCtx {
	return LogCtx{HashToIP: map[string]string{}, IPToHostname: map[string]string{}, IPToMethod: map[string]string{}, IPToNodeName: map[string]string{}, HashToNodeName: map[string]string{}}
}

func (ctx *LogCtx) OwnHostname() string {
	for _, ip := range ctx.OwnIPs {
		if hn, ok := ctx.IPToHostname[ip]; ok {
			return hn
		}
	}
	for _, hash := range ctx.OwnHashes {
		if hn, ok := ctx.IPToHostname[ctx.HashToIP[hash]]; ok {
			return hn
		}
	}
	return ""
}

func (ctx *LogCtx) HashesFromIP(ip string) []string {
	hashes := []string{}
	for hash, ip2 := range ctx.HashToIP {
		if ip == ip2 {
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func (ctx *LogCtx) HashesFromNodeName(nodename string) []string {
	hashes := []string{}
	for hash, nodename2 := range ctx.HashToNodeName {
		if nodename == nodename2 {
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func (ctx *LogCtx) IPsFromNodeName(nodename string) []string {
	ips := []string{}
	for ip, nodename2 := range ctx.IPToNodeName {
		if nodename == nodename2 {
			ips = append(ips, ip)
		}
	}
	return ips
}

func (ctx *LogCtx) AllNodeNames() []string {
	nodenames := ctx.OwnNames
	for _, nn := range ctx.HashToNodeName {
		if !utils.SliceContains(nodenames, nn) {
			nodenames = append(nodenames, nn)
		}
	}
	for _, nn := range ctx.IPToNodeName {
		if !utils.SliceContains(nodenames, nn) {
			nodenames = append(nodenames, nn)
		}
	}
	return nodenames
}

// AddOwnName propagates a name into the translation maps using the trusted node's known own hashes and ips
func (ctx *LogCtx) AddOwnName(name string) {
	if utils.SliceContains(ctx.OwnNames, name) {
		return
	}
	ctx.OwnNames = append(ctx.OwnNames, name)
	for _, hash := range ctx.OwnHashes {
		ctx.HashToNodeName[hash] = name
	}
	for _, ip := range ctx.OwnIPs {
		ctx.IPToNodeName[ip] = name
	}
}

// AddOwnHash propagates a hash into the translation maps
func (ctx *LogCtx) AddOwnHash(hash string) {
	if utils.SliceContains(ctx.OwnHashes, hash) {
		return
	}
	ctx.OwnHashes = append(ctx.OwnHashes, hash)

	for _, ip := range ctx.OwnIPs {
		ctx.HashToIP[hash] = ip
	}
	for _, name := range ctx.OwnNames {
		ctx.HashToNodeName[hash] = name
	}
}

// AddOwnIP propagates a ip into the translation maps
func (ctx *LogCtx) AddOwnIP(ip string) {
	if utils.SliceContains(ctx.OwnIPs, ip) {
		return
	}
	ctx.OwnIPs = append(ctx.OwnIPs, ip)
	for _, hash := range ctx.OwnHashes {
		ctx.HashToIP[hash] = ip
	}
	for _, name := range ctx.OwnNames {
		ctx.IPToNodeName[ip] = name
	}
}

func (base *LogCtx) MergeMapsWith(ctxs []LogCtx) {
	for _, ctx := range ctxs {
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
