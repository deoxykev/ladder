package rqm // ReQuestModifier

import (
	"ladder/proxychain"
	"regexp"
)

func ModifyPathWithRegex(match regexp.Regexp, replacement string) proxychain.ReqMod {
	return func(px *proxychain.ProxyChain) error {
		px.URL.Path = match.ReplaceAllString(px.URL.Path, replacement)
		return nil
	}
}
