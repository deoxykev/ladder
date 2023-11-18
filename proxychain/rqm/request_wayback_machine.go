package rqm // ReQuestModifier

import (
	"ladder/proxychain"
	"net/url"
)

const waybackUrl string = "https://web.archive.org/web/"

// RequestWaybackMachine modifies a ProxyChain's URL to request the wayback machine (archive.org) version.
func RequestWaybackMachine() proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.URL.RawQuery = ""
		newURLString := waybackUrl + px.URL.String()
		newURL, err := url.Parse(newURLString)
		if err != nil {
			return err
		}
		px.URL = newURL
		return nil
	}
}
