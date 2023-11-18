package rqm // ReQuestModifier

import (
	"ladder/proxychain"
)

// MasqueradeAsGoogleBot modifies user agent and x-forwarded for
// to appear to be a Google Bot
func MasqueradeAsGoogleBot() proxychain.ReqMod {
	const botUA string = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; Googlebot/2.1; http://www.google.com/bot.html) Chrome/79.0.3945.120 Safari/537.36"
	const botIP string = "66.249.78.8" // TODO: create a random ip pool from https://developers.google.com/static/search/apis/ipranges/googlebot.json
	return masqueradeAsTrustedBot(botUA, botIP)
}

// MasqueradeAsBingBot modifies user agent and x-forwarded for
// to appear to be a Bing Bot
func MasqueradeAsBingBot() proxychain.ReqMod {
	const botUA string = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm) Chrome/79.0.3945.120 Safari/537.36"
	const botIP string = "13.66.144.9" // https://www.bing.com/toolbox/bingbot.json
	return masqueradeAsTrustedBot(botUA, botIP)
}

func masqueradeAsTrustedBot(botUA string, botIP string) proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.AddReqMods(
			SpoofUserAgent(botUA),
			SpoofXForwardedFor(botIP),
			SpoofReferrer(""),
		)
		return nil
	}
}
