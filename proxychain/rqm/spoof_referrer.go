package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// SpoofReferrer modifies the referrer header
// useful if the page can be accessed from a search engine
// or social media site, but not by browsing the website itself
// if url is "", then the referrer header is removed
func SpoofReferrer(url string) proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		if url == "" {
			px.Req.Header.Del("referrer")
			return nil
		}
		px.Req.Header.Set("referrer", url)
		return nil
	}
}

// HideReferrer modifies the referrer header
// so that it is the original referrer, not the proxy
func HideReferrer() proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.Req.Header.Set("referrer", px.URL.String())
		return nil
	}
}
