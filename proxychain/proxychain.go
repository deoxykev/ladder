package proxychain

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"ladder/pkg/ruleset"

	"github.com/gofiber/fiber/v2"
)

var defaultClient *http.Client

func DefaultClient() {
	defaultClient = &http.Client{
		Timeout: 15,
	}
}

/*
ProxyChain manages the process of forwarding an HTTP request to an upstream server,
applying request and response modifications along the way.

  - It accepts incoming HTTP requests (as a Fiber *ctx), and applies
    request modifiers (ReqMods) and response modifiers (ResMods) before passing the
    upstream response back to the client.

  - ProxyChains can be reused to avoid memory allocations.

---

# EXAMPLE

```

import (

	"ladder/internal/proxychain/rqm"
	"ladder/internal/proxychain/rsm"
	"ladder/internal/proxychain"

)

proxychain.NewProxyChain().

	SetCtx(ctx).
	SetReqMods(
		rqm.NoCookie(),
		rqm.NoCors()
	).
	SetResMods(
		rsm.InjectJs("alert('hello world')")
	).
	Execute()

```

client              ladder service          upstream
┌─────────┐    ┌────────────────────────┐    ┌─────────┐
│         │GET │                        │    │         │
│         ├────┼───► ProxyChain         │    │         │
│         │    │       │                │    │         │
│         │    │       ▼                │    │         │
│         │    │     apply              │    │         │
│         │    │     ReqMods            │    │         │
│         │    │       │                │    │         │
│         │    │       ▼                │    │         │
│         │    │     send        GET    │    │         │
│         │    │     Request ───────────┼─►  │         │
│         │    │                        │    │         │
│         │    │                 200 OK │    │         │
│         │    │       ┌────────────────┼─   │         │
│         │    │       ▼                │    │         │
│         │    │     apply              │    │         │
│         │    │     ResMods		    │    │         │
│         │    │       │                │    │         │
│         │◄───┼───────┘                │    │         │
│         │    │ 200 OK                 │    │         │
│         │    │                        │    │         │
└─────────┘    └────────────────────────┘    └─────────┘
*/
type ProxyChain struct {
	Ctx        *fiber.Ctx
	URL        *url.URL
	Client     *http.Client
	Req        *http.Request
	Resp       *http.Response
	Body       []byte
	reqMods    []ReqMod
	resMods    []ResMod
	ruleset    *ruleset.RuleSet
	verbose    bool
	_abort_err error
}

// a ProxyStrategy is a pre-built proxychain with purpose-built defaults
type ProxyStrategy ProxyChain

// A ReqMod is a function that should operate on the
// ProxyChain Req or Client field, using the fiber ctx as needed.
type ReqMod func(*ProxyChain) error

// A ResMod is a function that should operate on the
// ProxyChain Res (http result) & Body (buffered http response body) field
type ResMod func(*ProxyChain) error

// SetReqMods sets the ProxyChain's request modifers
// the modifier will not fire until ProxyChain.Execute() is run.
func (p *ProxyChain) SetReqMods(reqMods ...ReqMod) *ProxyChain {
	p.reqMods = reqMods
	return p
}

// AddReqMods sets the ProxyChain's request modifers
// the modifier will not fire until ProxyChain.Execute() is run.
func (p *ProxyChain) AddReqMods(reqMods ...ReqMod) *ProxyChain {
	p.reqMods = append(p.reqMods, reqMods...)
	return p
}

// SetResMods sets the ProxyChain's response modifers
// the modifier will not fire until ProxyChain.Execute() is run.
func (p *ProxyChain) SetResMods(resMods ...ResMod) *ProxyChain {
	p.resMods = resMods
	return p
}

// AddResMods adds to the ProxyChain's response modifers
// the modifier will not fire until ProxyChain.Execute() is run.
func (p *ProxyChain) AddResMods(resMods ...ResMod) *ProxyChain {
	p.resMods = append(p.resMods, resMods...)
	return p
}

// Adds a ruleset to ProxyChain
func (p *ProxyChain) AddRuleset(rs *ruleset.RuleSet) *ProxyChain {
	p.ruleset = rs
	// TODO: add _applyRuleset method
	return p
}

// _execute sends the request for the ProxyChain and returns the raw body only
// the caller is responsible for returning a response back to the requestor
// the caller is also responsible for calling p._reset() when they are done with the body
func (p *ProxyChain) _execute() (*[]byte, error) {
	p._validate_ctx_is_set()
	if p._abort_err != nil {
		return nil, p._abort_err
	}
	if p.Ctx == nil {
		return nil, errors.New("request ctx not set. Use ProxyChain.SetCtx()")
	}
	if p.URL.Scheme == "" {
		return nil, errors.New("request url not set or invalid. Check ProxyChain ReqMods for issues")
	}

	// Apply ReqMods
	for _, reqMod := range p.reqMods {
		err := reqMod(p)
		if err != nil {
			return nil, p.abort(err)
		}
	}

	// Send Request Upstream
	resp, err := p.Client.Do(p.Req)
	if err != nil {
		return nil, p.abort(err)
	}
	p.Resp = resp

	// Buffer response into memory
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, p.abort(err)
	}
	p.Body = body
	defer resp.Body.Close()

	// Apply ResponseModifiers
	for _, resMod := range p.resMods {
		err := resMod(p)
		if err != nil {
			return nil, p.abort(err)
		}
	}

	return &p.Body, nil
}

// Execute sends the request for the ProxyChain and returns the request to the sender
// and resets the fields so that the ProxyChain can be reused.
// if any step in the ProxyChain fails, the request will abort and a 500 error will
// be returned to the client
func (p *ProxyChain) Execute() error {
	defer p._reset()
	body, err := p._execute()
	if err != nil {
		return err
	}
	// Return request back to client
	return p.Ctx.Send(*body)
}

// ExecuteAPIContent sends the request for the ProxyChain and returns the response body as
// a structured API response to the client
// if any step in the ProxyChain fails, the request will abort and a 500 error will
// be returned to the client
func (p *ProxyChain) ExecuteAPIContent() error {
	defer p._reset()
	body, err := p._execute()
	if err != nil {
		return err
	}
	// TODO: implement reader API
	// Return request back to client
	return p.Ctx.Send(*body)
}

// extractUrl extracts a URL from the request ctx. If the URL in the request
// is a relative path, it reconstructs the full URL using the referer header.
func (p *ProxyChain) extractUrl() error {
	// try to extract url-encoded
	reqUrl, err := url.QueryUnescape(p.Ctx.Params("*"))
	if err != nil {
		// fallback
		reqUrl = p.Ctx.Params("*")
	}

	// Extract the actual path from req ctx
	urlQuery, err := url.Parse(reqUrl)
	if err != nil {
		return fmt.Errorf("error parsing request URL '%s': %v", reqUrl, err)
	}

	isRelativePath := urlQuery.Scheme == ""
	// default behavior
	if !isRelativePath {
		// eg: https://localhost:8080/https://realsite.com/images/foobar.jpg -> https://realsite.com/images/foobar.jpg
		p.URL = urlQuery
		return nil
	}

	// eg: https://localhost:8080/images/foobar.jpg -> https://realsite.com/images/foobar.jpg
	// Parse the referer URL from the request header.
	refererUrl, err := url.Parse(p.Ctx.Get("referer"))
	if err != nil || refererUrl.Host == "" {
		return fmt.Errorf("error parsing referer URL from req: '%s': %v", reqUrl, err)
	}

	// Extract the real url from referer path
	realUrl, err := url.Parse(strings.TrimPrefix(refererUrl.Path, "/"))
	if err != nil {
		return fmt.Errorf("error parsing real URL from referer '%s': %v", refererUrl.Path, err)
	}

	// reconstruct the full URL using the referer's scheme, host, and the relative path / queries
	p.URL = &url.URL{
		Scheme:   realUrl.Scheme,
		Host:     realUrl.Host,
		Path:     urlQuery.Path,
		RawQuery: urlQuery.RawQuery,
	}

	isRelativePath = urlQuery.Scheme == ""
	if isRelativePath {
		return fmt.Errorf("ProxyChain failed to extract url from relative path: '%s'", reqUrl)
	}

	if p.verbose {
		log.Printf("modified relative URL: '%s' -> '%s'", reqUrl, p.URL.String())
	}
	return nil
}

// SetCtx takes the request ctx from the client
// for the modifiers and execute function to use.
// it must be set everytime a new request comes through
// if the upstream request url cannot be extracted from the ctx,
// a 500 error will be sent back to the client
func (p *ProxyChain) SetCtx(ctx *fiber.Ctx) *ProxyChain {
	p.Ctx = ctx
	err := p.extractUrl()
	if err != nil {
		p._abort_err = p.abort(err)
	}
	return p
}

func (p *ProxyChain) _validate_ctx_is_set() {
	if p.Ctx != nil {
		return
	}
	err := errors.New("proxyChain was called without setting a fiber Ctx. Use ProxyChain.SetCtx()")
	p._abort_err = p.abort(err)
}

// SetClient sets a new upstream http client transport
// useful for modifying TLS
func (p *ProxyChain) SetClient(httpClient *http.Client) *ProxyChain {
	p.Client = httpClient
	return p
}

// SetVerbose changes the logging behavior to print
// the modification steps and applied rulesets for debugging
func (p *ProxyChain) SetVerbose() *ProxyChain {
	p.verbose = true
	return p
}

// abort proxychain and return 500 error to client
// this will prevent Execute from firing and reset the state
// returns the initial error enriched with context
func (p *ProxyChain) abort(err error) error {
	defer p._reset()
	p._abort_err = err
	p.Ctx.Response().SetStatusCode(500)
	e := fmt.Errorf("ProxyChain error for '%s': %s", p.URL.String(), err.Error())
	p.Ctx.SendString(e.Error())
	log.Println(e.Error())
	return e
}

// internal function to reset state of ProxyChain for reuse
func (p *ProxyChain) _reset() {
	p._abort_err = nil
	p.Body = nil
	p.Req = nil
	p.Resp = nil
	p.Ctx = nil
	p.URL = nil
}

// NewProxyChain initializes a new ProxyChain
func NewProxyChain() *ProxyChain {
	px := new(ProxyChain)
	px.Client = defaultClient
	return px
}
