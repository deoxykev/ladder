package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// SpoofXForwardedFor modifies the X-Forwarded-For header
// in some cases, a forward proxy may interpret this as the source IP
func SpoofXForwardedFor(ip string) proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.Req.Header.Set("X-FORWARDED-FOR", ip)
		return nil
	}
}
