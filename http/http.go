// Package http provides Goal bindings for making HTTP requests.
//
// Import registers http.* verbs into a Goal context. Call it as:
//
//	http.Import(ctx, "")
//
// which registers globals prefixed with "http." (e.g. http.client, http.get).
//
// # Named method verbs (http.get, http.post, …)
//
// Each named verb accepts an optional explicit client and optional per-request
// opts dict:
//
//	http.get url                     – GET url, default client, no opts
//	http.get[client;url]             – GET url, explicit client, no opts
//	http.get[client;url;opts]        – GET url, explicit client, opts dict
//
// The default client is shared and lazily initialised. Passing an explicit
// http.client value uses that client instead; this is the intended hook for
// attaching per-client state such as rate limiters.
//
// Available named-method verbs:
//
//	http.delete   http.get    http.head    http.options
//	http.patch    http.post   http.put
//
// # http.request — explicit-client generic verb
//
// http.request takes an explicit client, a URL, and an options dict.
// The HTTP method is supplied via the "Method" key in opts (default "GET").
//
//	client http.request url              – GET url using client (no opts)
//	http.request[client;url;opts]        – request with client, url, opts
//
// Partially applying the client produces a reusable per-client request verb:
//
//	myReq: http.request[myClient;]       – bind client; supply url+opts later
//	myReq["https://api.example.com/items";(!"Method";"Body")!("POST";json)]
//
// # http.client — create a reusable client
//
//	http.client d   – create an http.client value configured by dict d
//
// # Per-request options (keys of the opts dict for named verbs and http.request)
//
//	AuthToken    s  – sets Authorization: Bearer <token>
//	BasicAuth    d  – basic auth for this request; keys: Username, Password
//	Body         s  – raw request body string
//	ContentType  s  – shorthand for the Content-Type header
//	Debug        i  – enable resty debug logging for this request (0/1)
//	FormData     d  – form data (url-encoded); values: s or AS
//	Header       d  – request headers; values: s or AS
//	PathParams   d  – URL path params (URL-encoded); values must be strings
//	QueryParam   d  – URL query parameters; values: s or AS
//	RawPathParams d – URL path params (not URL-encoded); values must be strings
//
// http.request opts also accepts:
//
//	Method       s  – HTTP method: "GET", "POST", "PUT", etc. (default "GET")
//
// # http.client options (keys of the dict passed to http.client)
//
//	AllowGetMethodPayload  i  – allow a body on GET requests (0/1)
//	AuthScheme             s  – auth header scheme prefix (default "Bearer")
//	BaseURL                s  – base URL prepended to every request URL
//	Debug                  i  – enable verbose resty debug logging (0/1)
//	DisableWarn            i  – suppress resty warning messages (0/1)
//	FollowRedirects        i  – follow HTTP redirects (0/1, default 1)
//	FormData               d  – default form data for every request
//	Header                 d  – default headers for every request
//	HeaderAuthorizationKey s  – override the Authorization header name
//	PathParams             d  – default URL path params (URL-encoded)
//	QueryParam             d  – default query parameters for every request
//	RawPathParams          d  – default URL path params (not URL-encoded)
//	RetryCount             i  – number of automatic retries on failure
//	RetryMaxWaitTimeMilli  i  – maximum retry back-off duration (ms)
//	RetryResetReaders      i  – reset request readers between retries (0/1)
//	RateLimitPerSecond     i  – max requests per second (leaky bucket); the
//	                           limiter is called automatically before each
//	                           request made through this client
//	RetryWaitTimeMilli     i  – initial retry wait time (ms)
//	TimeoutMilli           i  – request timeout in milliseconds
//	TLSInsecureSkipVerify  i  – skip TLS certificate verification (0/1)
//	Token                  s  – bearer token (sets the Authorization header)
//	UserInfo               d  – basic auth; required keys: Username, Password
//
// # Response dict
//
// Named method verbs (http.get, http.post, …) return:
//
//	t"status"     – string e.g. "200 OK"
//	t"statuscode" – integer e.g. 200
//	t"headers"    – dict: header name (s) → string array of values (AS)
//	t"body"       – response body as a string
//	t"ok"         – 1i if status is 2xx, else 0i
//
// http.request returns the same dict but with "bodybytes" (AB) instead of
// "body" (s), so callers that cannot safely decode the bytes as UTF-8 get
// them raw and can convert or inspect as needed.
package http

import (
	"crypto/tls"
	"fmt"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/anaseto/goal"
	"github.com/go-resty/resty/v2"
	uber "go.uber.org/ratelimit"
)

// ---------------------------------------------------------------------------
// BV wrapper: http.client
// ---------------------------------------------------------------------------

// Client wraps a *resty.Client as a Goal boxed value.
// limiter is non-nil when RateLimitPerSecond was set on the client; it is
// called automatically before every request made through this client.
type Client struct {
	c       *resty.Client
	limiter uber.Limiter
}

func (cl *Client) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	base := cl.c.BaseURL
	if base == "" {
		return append(dst, "http.client"...)
	}
	return append(dst, fmt.Sprintf("http.client[%q]", base)...)
}

func (cl *Client) Matches(y goal.BV) bool {
	yv, ok := y.(*Client)
	return ok && cl.c == yv.c
}

// LessT falls back to type-name ordering; there is no meaningful ordering for
// opaque client handles.
func (cl *Client) LessT(y goal.BV) bool { return cl.Type() < y.Type() }

func (cl *Client) Type() string { return "http.client" }

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

// Import registers all http.* verbs into ctx.
// When pfx is "" the globals are named "http.client", "http.get", etc.
// When pfx is non-empty it is prepended with a dot separator.
func Import(ctx *goal.Context, pfx string) {
	ctx.RegisterExtension("http", "")
	if pfx != "" {
		pfx += "."
	}

	reg := func(name string, f goal.VariadicFunc, dyad bool) {
		fullname := pfx + name
		var v goal.V
		if dyad {
			v = ctx.RegisterDyad("."+fullname, f)
		} else {
			v = ctx.RegisterMonad("."+fullname, f)
		}
		ctx.AssignGlobal(fullname, v)
	}

	// Shared default client for the named-method verbs. Lazily initialised so
	// that programs that never make HTTP requests pay no cost.
	var defaultClient *Client
	getDefault := func() *Client {
		if defaultClient == nil {
			defaultClient = &Client{c: resty.New()}
		}
		return defaultClient
	}

	// http.client — registered as dyad so bracket form works freely;
	// the implementation only accepts one argument (the options dict).
	reg("http.client", vfClientFn(), true)

	// Named method verbs — signature [url; opts], url is the first/left arg.
	// Registered as dyads so `url http.get opts` infix works.
	for _, method := range []string{"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"} {
		m := method
		reg("http."+strings.ToLower(m), vfNamedMethod(getDefault, m), true)
	}

	// http.request — explicit client, url, and opts.
	// Registered as dyad so `client http.request url` infix works.
	reg("http.request", vfRequest(), true)
}

// ---------------------------------------------------------------------------
// http.client verb
// ---------------------------------------------------------------------------

func vfClientFn() goal.VariadicFunc {
	return func(_ *goal.Context, args []goal.V) goal.V {
		if len(args) != 1 {
			return goal.Panicf("http.client d : expected 1 argument, got %d", len(args))
		}
		d, ok := args[0].BV().(*goal.D)
		if !ok {
			return goal.Panicf("http.client d : expected dict, got %q", args[0].Type())
		}
		cl, err := newClient(d)
		if err != nil {
			return goal.NewPanicError(err)
		}
		return goal.NewV(cl)
	}
}

// ---------------------------------------------------------------------------
// Named method verbs: http.get, http.post, …
//
// Signature: [url; opts]
// args layout by arity:
//
//	1 arg:  http.get[url]
//	          args[0] = url
//	2 args: http.get[client;url]   args[1] = client (*Client), args[0] = url
//	        http.get[url;opts]     args[1] = url (string),    args[0] = opts
//	3 args: http.get[client;url;opts]
//	          args[2] = client, args[1] = url, args[0] = opts
//
// The 2-arg cases are distinguished by the type of args[1]: *Client vs string.
// Opts are never in the first (leftmost) position, making this unambiguous.
// ---------------------------------------------------------------------------

func vfNamedMethod(getDefault func() *Client, upper string) goal.VariadicFunc {
	lower := strings.ToLower(upper)
	return func(_ *goal.Context, args []goal.V) goal.V {
		switch len(args) {
		case 1:
			return namedMethodExec(getDefault(), args[0], nil, lower, upper)
		case 2:
			if cl, ok := args[1].BV().(*Client); ok {
				// http.get[client;url]
				return namedMethodExec(cl, args[0], nil, lower, upper)
			}
			// http.get[url;opts]
			optsD, ok := args[0].BV().(*goal.D)
			if !ok {
				return goal.Panicf("http.%s[url;opts] : expected dict as second argument, got %q", lower, args[0].Type())
			}
			return namedMethodExec(getDefault(), args[1], optsD, lower, upper)
		case 3:
			// http.get[client;url;opts]
			cl, err := clientFromV(args[2], lower)
			if err != nil {
				return goal.NewPanicError(err)
			}
			optsD, ok := args[0].BV().(*goal.D)
			if !ok {
				return goal.Panicf("http.%s[client;url;opts] : expected dict as third argument, got %q", lower, args[0].Type())
			}
			return namedMethodExec(cl, args[1], optsD, lower, upper)
		default:
			return goal.Panicf("http.%s : expected 1, 2, or 3 arguments, got %d", lower, len(args))
		}
	}
}

// namedMethodExec executes the request against cl for the given url value,
// optionally augmenting the request with opts (nil means no per-request opts).
// If the client has a rate limiter configured it is taken before the request.
func namedMethodExec(cl *Client, urlV goal.V, opts *goal.D, lower, upper string) goal.V {
	if cl.limiter != nil {
		cl.limiter.Take()
	}
	urlS, ok := urlV.BV().(goal.S)
	if !ok {
		return goal.Panicf("http.%s : expected string URL, got %q", lower, urlV.Type())
	}
	req := cl.c.R()
	if opts != nil {
		if err := augmentRequest(req, opts, lower); err != nil {
			return goal.NewPanicError(err)
		}
	}
	resp, err := req.Execute(upper, string(urlS))
	if err != nil {
		return goal.Errorf("http.%s: %v", lower, err)
	}
	return responseDict(resp)
}

// ---------------------------------------------------------------------------
// http.request — explicit-client generic verb
//
// Signature: [client; url; opts]
//   - 2 args (dyadic):  client http.request url
//     args[1] = client (first/left), args[0] = url (second/right)
//   - 3 args (bracket): http.request[client;url;opts]
//     args[2] = client, args[1] = url, args[0] = opts
//
// The HTTP method is supplied as opts["Method"] (default "GET").
//
// Partial application http.request[myClient;] binds the client, producing
// a dyad that accepts [url; opts].
// ---------------------------------------------------------------------------

func vfRequest() goal.VariadicFunc {
	return func(_ *goal.Context, args []goal.V) goal.V {
		switch len(args) {
		case 2:
			return requestDyadic(args)
		case 3:
			return requestTriadic(args)
		default:
			return goal.Panicf("http.request : expected 2 or 3 arguments, got %d", len(args))
		}
	}
}

// requestDyadic handles:  client http.request url  (GET, no opts)
// args[1] = client (first/left), args[0] = url (second/right).
func requestDyadic(args []goal.V) goal.V {
	cl, err := clientFromV(args[1], "request")
	if err != nil {
		return goal.NewPanicError(err)
	}
	if cl.limiter != nil {
		cl.limiter.Take()
	}
	urlS, ok := args[0].BV().(goal.S)
	if !ok {
		return goal.Panicf("client http.request url : expected string URL, got %q", args[0].Type())
	}
	resp, gErr := cl.c.R().Execute("GET", string(urlS))
	if gErr != nil {
		return goal.Errorf("http.request: %v", gErr)
	}
	return responseDictBytes(resp)
}

// requestTriadic handles:  http.request[client;url;opts]
// args[2] = client, args[1] = url, args[0] = opts.
func requestTriadic(args []goal.V) goal.V {
	cl, err := clientFromV(args[2], "request")
	if err != nil {
		return goal.NewPanicError(err)
	}
	if cl.limiter != nil {
		cl.limiter.Take()
	}
	urlS, ok := args[1].BV().(goal.S)
	if !ok {
		return goal.Panicf("http.request[client;url;opts] : expected string URL as second argument, got %q", args[1].Type())
	}
	optsD, ok := args[0].BV().(*goal.D)
	if !ok {
		return goal.Panicf("http.request[client;url;opts] : expected dict as third argument, got %q", args[0].Type())
	}
	method, req, err := requestFromOpts(cl.c, optsD)
	if err != nil {
		return goal.NewPanicError(err)
	}
	resp, gErr := req.Execute(method, string(urlS))
	if gErr != nil {
		return goal.Errorf("http.request: %v", gErr)
	}
	return responseDictBytes(resp)
}

// requestFromOpts extracts the "Method" key (defaulting to "GET") from the
// opts dict and augments a fresh resty.Request with all remaining keys.
func requestFromOpts(cl *resty.Client, d *goal.D) (string, *resty.Request, error) {
	method := "GET"
	req := cl.R()
	if d.Len() == 0 {
		return method, req, nil
	}
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return "", nil, fmt.Errorf("http.request : opts keys must be strings, got %q", d.KeyArray().Type())
	}
	for i, k := range kas.Slice {
		v := d.ValueArray().At(i)
		if k == "Method" {
			s, sErr := stringArg(v, "Method")
			if sErr != nil {
				return "", nil, sErr
			}
			method = strings.ToUpper(s)
		} else {
			if aErr := applyRequestOption(req, k, v, "request"); aErr != nil {
				return "", nil, aErr
			}
		}
	}
	return method, req, nil
}

// ---------------------------------------------------------------------------
// Client construction
// ---------------------------------------------------------------------------

// clientFromV extracts or constructs a *Client from a Goal value.
// Accepts an http.client BV or an options dict (for one-shot clients).
func clientFromV(x goal.V, verb string) (*Client, error) {
	switch v := x.BV().(type) {
	case *Client:
		return v, nil
	case *goal.D:
		return newClient(v)
	default:
		return nil, fmt.Errorf("http.%s : client argument must be an http.client or an options dict, got %q", verb, x.Type())
	}
}

// newClient builds a *Client from a Goal dict of client options.
func newClient(d *goal.D) (*Client, error) {
	cl := &Client{c: resty.New()}
	if d.Len() == 0 {
		return cl, nil
	}
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return nil, fmt.Errorf("http.client : option keys must be strings, got %q", d.KeyArray().Type())
	}
	for i, k := range kas.Slice {
		if err := applyClientOption(cl, k, d.ValueArray().At(i)); err != nil {
			return nil, err
		}
	}
	return cl, nil
}

func applyClientOption(cl *Client, key string, v goal.V) error { //nolint:gocognit,gocyclo,cyclop,funlen,lll // exhaustive option switch
	switch key {
	case "AllowGetMethodPayload":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		cl.c.AllowGetMethodPayload = b

	case "AuthScheme":
		s, err := stringArg(v, key)
		if err != nil {
			return err
		}
		cl.c.AuthScheme = s

	case "BaseURL":
		s, err := stringArg(v, key)
		if err != nil {
			return err
		}
		cl.c.BaseURL = s

	case "Debug":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		cl.c.Debug = b

	case "DisableWarn":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		cl.c.DisableWarn = b

	case "FollowRedirects":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		if !b {
			cl.c.SetRedirectPolicy(resty.NoRedirectPolicy())
		}

	case "FormData":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		uv, err := toURLValues(d, key)
		if err != nil {
			return err
		}
		cl.c.FormData = uv

	case "Header":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		h, err := toHTTPHeader(d, key)
		if err != nil {
			return err
		}
		cl.c.Header = h

	case "HeaderAuthorizationKey":
		s, err := stringArg(v, key)
		if err != nil {
			return err
		}
		cl.c.HeaderAuthorizationKey = s

	case "PathParams":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		m, err := stringStringMap(d, key)
		if err != nil {
			return err
		}
		cl.c.PathParams = m

	case "QueryParam":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		uv, err := toURLValues(d, key)
		if err != nil {
			return err
		}
		cl.c.QueryParam = uv

	case "RateLimitPerSecond":
		n, err := intArg(v, key)
		if err != nil {
			return err
		}
		if n <= 0 {
			return fmt.Errorf("http.client : RateLimitPerSecond must be a positive integer, got %d", n)
		}
		cl.limiter = uber.New(n, uber.WithoutSlack)

	case "RawPathParams":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		m, err := stringStringMap(d, key)
		if err != nil {
			return err
		}
		cl.c.RawPathParams = m

	case "RetryCount":
		n, err := intArg(v, key)
		if err != nil {
			return err
		}
		cl.c.RetryCount = n

	case "RetryMaxWaitTimeMilli":
		n, err := intArg(v, key)
		if err != nil {
			return err
		}
		cl.c.RetryMaxWaitTime = time.Duration(n) * time.Millisecond

	case "RetryResetReaders":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		cl.c.RetryResetReaders = b

	case "RetryWaitTimeMilli":
		n, err := intArg(v, key)
		if err != nil {
			return err
		}
		cl.c.RetryWaitTime = time.Duration(n) * time.Millisecond

	case "TimeoutMilli":
		n, err := intArg(v, key)
		if err != nil {
			return err
		}
		cl.c.SetTimeout(time.Duration(n) * time.Millisecond)

	case "TLSInsecureSkipVerify":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		cl.c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: b}) //nolint:gosec,lll // G402: InsecureSkipVerify is user-controlled

	case "Token":
		s, err := stringArg(v, key)
		if err != nil {
			return err
		}
		cl.c.Token = s

	case "UserInfo":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		username, password, err := usernamePassword(d, key)
		if err != nil {
			return err
		}
		cl.c.UserInfo = &resty.User{Username: username, Password: password}

	default:
		return fmt.Errorf("http.client : unsupported option %q", key)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Per-request option application
// ---------------------------------------------------------------------------

func augmentRequest(req *resty.Request, d *goal.D, verb string) error {
	if d.Len() == 0 {
		return nil
	}
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return fmt.Errorf("http.%s : opts keys must be strings, got %q", verb, d.KeyArray().Type())
	}
	for i, k := range kas.Slice {
		if err := applyRequestOption(req, k, d.ValueArray().At(i), verb); err != nil {
			return err
		}
	}
	return nil
}

func applyRequestOption(req *resty.Request, key string, v goal.V, verb string) error { //nolint:gocognit,funlen
	switch key {
	case "AuthToken":
		s, err := stringArg(v, key)
		if err != nil {
			return err
		}
		req.SetAuthToken(s)

	case "BasicAuth":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		username, password, err := usernamePassword(d, key)
		if err != nil {
			return err
		}
		req.SetBasicAuth(username, password)

	case "Body":
		switch bv := v.BV().(type) {
		case goal.S:
			req.SetBody(string(bv))
		case *goal.AB:
			req.SetBody(bv.Slice)
		default:
			return fmt.Errorf("http.%s \"Body\" must be a string or byte array, got %q", verb, v.Type())
		}

	case "ContentType":
		s, err := stringArg(v, key)
		if err != nil {
			return err
		}
		req.SetHeader("Content-Type", s)

	case "Debug":
		b, err := boolArg(v, key)
		if err != nil {
			return err
		}
		req.Debug = b

	case "FormData":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		uv, err := toURLValues(d, key)
		if err != nil {
			return err
		}
		req.FormData = uv

	case "Header":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		h, err := toHTTPHeader(d, key)
		if err != nil {
			return err
		}
		req.Header = h

	case "PathParams":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		m, err := stringStringMap(d, key)
		if err != nil {
			return err
		}
		req.PathParams = m

	case "QueryParam":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		uv, err := toURLValues(d, key)
		if err != nil {
			return err
		}
		req.QueryParam = uv

	case "RawPathParams":
		d, err := dictArg(v, key)
		if err != nil {
			return err
		}
		m, err := stringStringMap(d, key)
		if err != nil {
			return err
		}
		req.RawPathParams = m

	default:
		return fmt.Errorf("http.%s : unsupported request option %q", verb, key)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Response dict construction
// ---------------------------------------------------------------------------

func responseHeaders(resp *resty.Response) goal.V {
	raw := resp.Header()
	keys := make([]string, 0, len(raw))
	vals := make([]goal.V, 0, len(raw))
	for k, vs := range raw {
		keys = append(keys, k)
		vals = append(vals, goal.NewAS(vs))
	}
	return goal.NewD(goal.NewAS(keys), goal.NewAV(vals))
}

func responseOk(resp *resty.Response) goal.V {
	if resp.IsSuccess() {
		return goal.NewI(1)
	}
	return goal.NewI(0)
}

// responseDict is used by the named method verbs (http.get, http.post, …).
// The body is returned as a Goal string under the key "body".
func responseDict(resp *resty.Response) goal.V {
	ks := goal.NewAS([]string{"status", "statuscode", "headers", "body", "ok"})
	vs := goal.NewAV([]goal.V{
		goal.NewS(resp.Status()),
		goal.NewI(int64(resp.StatusCode())),
		responseHeaders(resp),
		goal.NewS(resp.String()),
		responseOk(resp),
	})
	return goal.NewD(ks, vs)
}

// responseDictBytes is used by http.request.
// The body is returned as a Goal byte array (AB) under the key "bodybytes",
// preserving the raw bytes for callers that need them verbatim.
func responseDictBytes(resp *resty.Response) goal.V {
	ks := goal.NewAS([]string{"status", "statuscode", "headers", "bodybytes", "ok"})
	vs := goal.NewAV([]goal.V{
		goal.NewS(resp.Status()),
		goal.NewI(int64(resp.StatusCode())),
		responseHeaders(resp),
		goal.NewAB(resp.Body()),
		responseOk(resp),
	})
	return goal.NewD(ks, vs)
}

// ---------------------------------------------------------------------------
// Type-extraction helpers
// ---------------------------------------------------------------------------

// boolArg extracts a Go bool using Goal's truthiness rules (0/0i = false,
// everything else = true). Returns an error for values that are neither
// (unreachable in current Goal, kept as a safety net).
func boolArg(v goal.V, key string) (bool, error) {
	switch {
	case v.IsTrue():
		return true, nil
	case v.IsFalse():
		return false, nil
	default:
		return false, fmt.Errorf("http option %q must be 0 or 1, got %q: %v", key, v.Type(), v)
	}
}

// stringArg extracts a Go string from a Goal S value.
func stringArg(v goal.V, key string) (string, error) {
	s, ok := v.BV().(goal.S)
	if !ok {
		return "", fmt.Errorf("http option %q must be a string, got %q: %v", key, v.Type(), v)
	}
	return string(s), nil
}

// intArg extracts a Go int from a Goal integer value.
func intArg(v goal.V, key string) (int, error) {
	if !v.IsI() {
		return 0, fmt.Errorf("http option %q must be an integer, got %q: %v", key, v.Type(), v)
	}
	return int(v.I()), nil
}

// dictArg extracts a *goal.D from a Goal V.
func dictArg(v goal.V, key string) (*goal.D, error) {
	d, ok := v.BV().(*goal.D)
	if !ok {
		return nil, fmt.Errorf("http option %q must be a dict, got %q: %v", key, v.Type(), v)
	}
	return d, nil
}

// stringStringMap converts a Goal dict with AS keys and AS values to
// map[string]string. Both key and value arrays must be *goal.AS.
func stringStringMap(d *goal.D, key string) (map[string]string, error) {
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return nil, fmt.Errorf("http option %q: dict keys must be strings, got %q", key, d.KeyArray().Type())
	}
	vas, ok := d.ValueArray().(*goal.AS)
	if !ok {
		return nil, fmt.Errorf("http option %q: dict values must be strings, got %q", key, d.ValueArray().Type())
	}
	m := make(map[string]string, len(kas.Slice))
	for i, k := range kas.Slice {
		m[k] = vas.Slice[i]
	}
	return m, nil
}

// toURLValues converts a Goal dict with AS keys and S-or-AS values to
// url.Values. Each key maps to a single string or an array of strings.
func toURLValues(d *goal.D, key string) (url.Values, error) {
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return nil, fmt.Errorf("http option %q: dict keys must be strings, got %q", key, d.KeyArray().Type())
	}
	uv := make(url.Values, len(kas.Slice))
	for i, k := range kas.Slice {
		val := d.ValueArray().At(i)
		switch hv := val.BV().(type) {
		case goal.S:
			uv.Add(k, string(hv))
		case *goal.AS:
			for _, s := range hv.Slice {
				uv.Add(k, s)
			}
		default:
			return nil, fmt.Errorf("http option %q: values must be strings or string arrays, got %q", key, val.Type())
		}
	}
	return uv, nil
}

// toHTTPHeader converts a Goal dict with AS keys and S-or-AS values to
// http.Header. Each key maps to a single string or an array of strings.
func toHTTPHeader(d *goal.D, key string) (nethttp.Header, error) {
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return nil, fmt.Errorf("http option %q: dict keys must be strings, got %q", key, d.KeyArray().Type())
	}
	h := make(nethttp.Header, len(kas.Slice))
	for i, k := range kas.Slice {
		val := d.ValueArray().At(i)
		switch hv := val.BV().(type) {
		case goal.S:
			h.Add(k, string(hv))
		case *goal.AS:
			for _, s := range hv.Slice {
				h.Add(k, s)
			}
		default:
			return nil, fmt.Errorf("http option %q: values must be strings or string arrays, got %q", key, val.Type())
		}
	}
	return h, nil
}

// usernamePassword extracts "Username" and "Password" string fields from a
// Goal dict. Used for both client-level UserInfo and per-request BasicAuth.
func usernamePassword(d *goal.D, key string) (string, string, error) {
	var username, password string
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return "", "", fmt.Errorf("http option %q: dict keys must be strings, got %q", key, d.KeyArray().Type())
	}
	for i, k := range kas.Slice {
		val := d.ValueArray().At(i)
		s, ok := val.BV().(goal.S)
		if !ok {
			return "", "", fmt.Errorf("http option %q: values must be strings, got %q for key %q", key, val.Type(), k)
		}
		switch k {
		case "Username":
			username = string(s)
		case "Password":
			password = string(s)
		default:
			return "", "", fmt.Errorf("http option %q: unsupported key %q (want \"Username\" and \"Password\")", key, k)
		}
	}
	return username, password, nil
}
