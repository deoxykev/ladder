package rsm // ReSponseModifers

import (
	"ladder/proxychain"
)

// ModifyResponseHeader modifies response headers from the upstream server
// if value is "", then the response header is deleted.
func ModifyResponseHeader(key string, value string) proxychain.ResMod {
	return func(px *proxychain.ProxyChain) error {
		if value == "" {
			px.Ctx.Response().Header.Del(key)
			return nil
		}
		px.Ctx.Response().Header.Set(key, value)
		return nil
	}
}

// DeleteResponseHeader removes response headers from the upstream server
func DeleteResponseHeader(key string) proxychain.ResMod {
	return func(px *proxychain.ProxyChain) error {
		px.Ctx.Response().Header.Del(key)
		return nil
	}
}
