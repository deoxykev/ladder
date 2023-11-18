package rqm // ReQuestModifier

import (
	"ladder/proxychain"
	"regexp"
)

func ModifyDomainWithRegex(match regexp.Regexp, replacement string) proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.URL.Host = match.ReplaceAllString(px.URL.Host, replacement)
		return nil
	}
}
