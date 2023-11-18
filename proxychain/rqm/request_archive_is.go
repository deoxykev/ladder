package rqm

import (
	"ladder/proxychain"
	"net/url"
)

const archivistUrl string = "https://archive.is/latest/"

// RequestArchiveIs modifies a ProxyChain's URL to request an archived version from archive.is
func RequestArchiveIs() proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.URL.RawQuery = ""
		newURLString := archivistUrl + px.URL.String()
		newURL, err := url.Parse(newURLString)
		if err != nil {
			return err
		}

		// archivist seems to sabotage requests from cloudflare's DNS
		// bypass this just in case
		px.AddReqMods(ResolveWithGoogleDoH())

		px.URL = newURL
		return nil
	}
}
