package rqm // ReQuestModifier

import (
	"ladder/proxychain"
	"net/url"
)

const googleCacheUrl string = "https://webcache.googleusercontent.com/search?q=cache:"

// RequestGoogleCache modifies a ProxyChain's URL to request its Google Cache version.
func RequestGoogleCache() proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		encodedURL := url.QueryEscape(px.URL.String())
		newURL, err := url.Parse(googleCacheUrl + encodedURL)
		if err != nil {
			return err
		}
		px.URL = newURL
		return nil
	}
}
