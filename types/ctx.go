package types

import (
	"fmt"
)

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

// AddOwnName propagates a name into the translation maps using the trusted node's known own hashes and ips
func (ctx *LogCtx) AddOwnName(name string) {
	ctx.OwnNames = append(ctx.OwnNames, name)
	for _, hash := range ctx.OwnHashes {
		ctx.HashToNodeName[hash] = name
	}
	for _, ip := range ctx.OwnIPs {
		ctx.IPToNodeName[ip] = name
	}
	fmt.Println("inside ", ctx.HashToNodeName)
}

// AddOwnHash propagates a hash into the translation maps
func (ctx *LogCtx) AddOwnHash(hash string) {
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

	ctx.OwnIPs = append(ctx.OwnIPs, ip)
	for _, hash := range ctx.OwnHashes {
		ctx.HashToIP[hash] = ip
	}
	for _, name := range ctx.OwnNames {
		ctx.IPToNodeName[ip] = name
	}
}
