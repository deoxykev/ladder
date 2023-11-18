package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// SpoofUserAgent modifies the user agent
func SpoofUserAgent(ua string) proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.Req.Header.Set("user-agent", ua)
		return nil
	}
}
