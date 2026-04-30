package http_test

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"time"

	"codeberg.org/anaseto/goal"
	goalhttp "github.com/semperos/ari/http"
	goalratelimit "github.com/semperos/ari/ratelimit"
)

// ---------------------------------------------------------------------------
// Helpers: Goal context
// ---------------------------------------------------------------------------

func newCtx(t *testing.T) *goal.Context {
	t.Helper()
	ctx := goal.NewContext()
	goalhttp.Import(ctx, "")
	return ctx
}

// eval evaluates Goal source, fails on error or panic, returns the result.
func eval(t *testing.T, ctx *goal.Context, src string) goal.V {
	t.Helper()
	v, err := ctx.Eval(src)
	if err != nil {
		t.Fatalf("eval %q: %v", src, err)
	}
	if v.IsPanic() {
		t.Fatalf("eval %q: panic: %s", src, v.Sprint(ctx, true))
	}
	return v
}

// evalPanic evaluates Goal source and asserts a panic is returned.
// Returns the panic message so callers can inspect it.
func evalPanic(t *testing.T, ctx *goal.Context, src string) string {
	t.Helper()
	v, err := ctx.Eval(src)
	if err != nil {
		return err.Error()
	}
	if !v.IsPanic() {
		t.Fatalf("eval %q: expected panic, got %s", src, v.Sprint(ctx, true))
	}
	return v.Sprint(ctx, true)
}

// ---------------------------------------------------------------------------
// Helpers: value extraction
// ---------------------------------------------------------------------------

func mustDict(t *testing.T, ctx *goal.Context, v goal.V) *goal.D {
	t.Helper()
	d, ok := v.BV().(*goal.D)
	if !ok {
		t.Fatalf("expected dict, got %q: %s", v.Type(), v.Sprint(ctx, true))
	}
	return d
}

// dictField retrieves a key from a Goal dict that uses AS keys.
func dictField(t *testing.T, d *goal.D, key string) goal.V {
	t.Helper()
	kas, ok := d.KeyArray().(*goal.AS)
	if !ok {
		t.Fatalf("dict keys: expected AS, got %q", d.KeyArray().Type())
	}
	for i, k := range kas.Slice {
		if k != key {
			continue
		}
		switch va := d.ValueArray().(type) {
		case *goal.AV:
			return va.Slice[i]
		case *goal.AS:
			return goal.NewS(va.Slice[i])
		case *goal.AI:
			return goal.NewI(va.Slice[i])
		case *goal.AF:
			return goal.NewF(va.Slice[i])
		default:
			t.Fatalf("dict values: unexpected type %q", d.ValueArray().Type())
		}
	}
	t.Fatalf("key %q not found in dict", key)
	return goal.V{}
}

func mustI(t *testing.T, v goal.V) int64 {
	t.Helper()
	if !v.IsI() {
		t.Fatalf("expected integer, got %q: %v", v.Type(), v)
	}
	return v.I()
}

func mustS(t *testing.T, ctx *goal.Context, v goal.V) string {
	t.Helper()
	s, ok := v.BV().(goal.S)
	if !ok {
		t.Fatalf("expected string, got %q: %s", v.Type(), v.Sprint(ctx, true))
	}
	return string(s)
}

// ---------------------------------------------------------------------------
// Helpers: opts dict construction
//
// Building opts dicts in Go directly is cleaner than fighting Goal's
// dict-literal syntax inside format strings.  The pattern mirrors how
// sql_test.go works: assemble Go values, assign them to Goal globals,
// then reference those globals in evaluated Goal code.
// ---------------------------------------------------------------------------

// opts1 builds a single-entry opts dict: {"key": value}.
func opts1(key string, value goal.V) goal.V {
	return goal.NewD(
		goal.NewAS([]string{key}),
		goal.NewAV([]goal.V{value}),
	)
}

// opts2 builds a two-entry opts dict.
func opts2(k1 string, v1 goal.V, k2 string, v2 goal.V) goal.V {
	return goal.NewD(
		goal.NewAS([]string{k1, k2}),
		goal.NewAV([]goal.V{v1, v2}),
	)
}

// strDict builds a Goal dict with AS keys and AS values from alternating
// key, value string pairs: strDict("a","1","b","2") → "a""b"!"1""2".
func strDict(kvs ...string) goal.V {
	if len(kvs)%2 != 0 {
		panic("strDict: odd number of arguments")
	}
	keys := make([]string, 0, len(kvs)/2)
	vals := make([]string, 0, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		keys = append(keys, kvs[i])
		vals = append(vals, kvs[i+1])
	}
	return goal.NewD(goal.NewAS(keys), goal.NewAS(vals))
}

// ---------------------------------------------------------------------------
// Helpers: test HTTP server
// ---------------------------------------------------------------------------

// captured holds details of the most recent request received by a test server.
type captured struct {
	method  string
	path    string
	rawURL  string // full URL path including query
	query   map[string]string
	headers http.Header
	body    string
}

// newServer starts a local httptest.Server that records each incoming request
// into a *captured and returns the given status+body.
// The server is shut down automatically via t.Cleanup.
func newServer(t *testing.T, status int, responseBody string) (*httptest.Server, *captured) {
	t.Helper()
	capt := &captured{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		capt.method = r.Method
		capt.path = r.URL.Path
		capt.rawURL = r.URL.RequestURI()
		capt.headers = r.Header.Clone()
		capt.body = string(raw)
		capt.query = make(map[string]string)
		for k, vs := range r.URL.Query() {
			if len(vs) > 0 {
				capt.query[k] = vs[0]
			}
		}
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(status)
		fmt.Fprint(w, responseBody)
	}))
	t.Cleanup(ts.Close)
	return ts, capt
}

// ---------------------------------------------------------------------------
// TestResponseDict – verify all five response fields are present and typed
// correctly for a successful 200 response.
// ---------------------------------------------------------------------------

func TestResponseDict(t *testing.T) {
	ts, _ := newServer(t, 200, "hello")
	ctx := newCtx(t)

	v := eval(t, ctx, fmt.Sprintf("http.get[%q]", ts.URL))
	d := mustDict(t, ctx, v)

	// status – string like "200 OK"
	status := mustS(t, ctx, dictField(t, d, "status"))
	if !strings.HasPrefix(status, "200") {
		t.Errorf("status: expected prefix 200, got %q", status)
	}

	// statuscode – integer 200
	code := mustI(t, dictField(t, d, "statuscode"))
	if code != 200 {
		t.Errorf("statuscode: expected 200, got %d", code)
	}

	// body – the response body
	body := mustS(t, ctx, dictField(t, d, "body"))
	if body != "hello" {
		t.Errorf("body: expected %q, got %q", "hello", body)
	}

	// ok – 1i for 2xx
	if mustI(t, dictField(t, d, "ok")) != 1 {
		t.Errorf("ok: expected 1 for 200 response")
	}

	// headers – must be a dict
	mustDict(t, ctx, dictField(t, d, "headers"))
}

// ---------------------------------------------------------------------------
// TestOkField – ok is 0i for non-2xx responses.
// ---------------------------------------------------------------------------

func TestOkField(t *testing.T) {
	for _, tc := range []struct {
		status int
		wantOk int64
	}{
		{200, 1},
		{201, 1},
		{204, 1},
		{301, 0},
		{400, 0},
		{404, 0},
		{500, 0},
	} {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			ts, _ := newServer(t, tc.status, "")
			ctx := newCtx(t)
			v := eval(t, ctx, fmt.Sprintf("http.get[%q]", ts.URL))
			d := mustDict(t, ctx, v)
			got := mustI(t, dictField(t, d, "ok"))
			if got != tc.wantOk {
				t.Errorf("status %d: ok=%d, want %d", tc.status, got, tc.wantOk)
			}
			code := mustI(t, dictField(t, d, "statuscode"))
			if code != int64(tc.status) {
				t.Errorf("status %d: statuscode=%d, want %d", tc.status, code, tc.status)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHTTPMethods – each named verb sends the right HTTP method.
// ---------------------------------------------------------------------------

func TestHTTPMethods(t *testing.T) {
	for _, tc := range []struct{ verb, want string }{
		{"http.delete", "DELETE"},
		{"http.get", "GET"},
		{"http.head", "HEAD"},
		{"http.options", "OPTIONS"},
		{"http.patch", "PATCH"},
		{"http.post", "POST"},
		{"http.put", "PUT"},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			ts, capt := newServer(t, 200, "")
			ctx := newCtx(t)
			eval(t, ctx, fmt.Sprintf("%s %q", tc.verb, ts.URL))
			if capt.method != tc.want {
				t.Errorf("%s: server got method %q, want %q", tc.verb, capt.method, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestQueryParam – QueryParam dict entries arrive as URL query parameters.
// ---------------------------------------------------------------------------

func TestQueryParam(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	qp := strDict("format", "json", "page", "2")
	ctx.AssignGlobal("opts", opts1("QueryParam", qp))

	eval(t, ctx, fmt.Sprintf("http.get[%q;opts]", ts.URL))

	if capt.query["format"] != "json" {
		t.Errorf("QueryParam format: got %q, want %q", capt.query["format"], "json")
	}
	if capt.query["page"] != "2" {
		t.Errorf("QueryParam page: got %q, want %q", capt.query["page"], "2")
	}
}

// ---------------------------------------------------------------------------
// TestBody – Body string arrives as the request body for POST.
// ---------------------------------------------------------------------------

func TestBody(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", opts1("Body", goal.NewS(`{"x":1}`)))

	eval(t, ctx, fmt.Sprintf("http.post[%q;opts]", ts.URL))

	if capt.body != `{"x":1}` {
		t.Errorf("Body: got %q, want %q", capt.body, `{"x":1}`)
	}
}

// ---------------------------------------------------------------------------
// TestContentType – ContentType shorthand sets the Content-Type header.
// ---------------------------------------------------------------------------

func TestContentType(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", opts2(
		"Body", goal.NewS("{}"),
		"ContentType", goal.NewS("application/json"),
	))

	eval(t, ctx, fmt.Sprintf("http.post[%q;opts]", ts.URL))

	got := capt.headers.Get("Content-Type")
	if got != "application/json" {
		t.Errorf("ContentType: got %q, want %q", got, "application/json")
	}
}

// ---------------------------------------------------------------------------
// TestHeader – custom headers arrive at the server.
// ---------------------------------------------------------------------------

func TestHeader(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", opts1("Header", strDict("X-Custom", "my-value")))

	eval(t, ctx, fmt.Sprintf("http.get[%q;opts]", ts.URL))

	if capt.headers.Get("X-Custom") != "my-value" {
		t.Errorf("Header: got %q, want %q", capt.headers.Get("X-Custom"), "my-value")
	}
}

// ---------------------------------------------------------------------------
// TestAuthToken – AuthToken sets Authorization: Bearer <token>.
// ---------------------------------------------------------------------------

func TestAuthToken(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", opts1("AuthToken", goal.NewS("my-secret-token")))

	eval(t, ctx, fmt.Sprintf("http.get[%q;opts]", ts.URL))

	got := capt.headers.Get("Authorization")
	want := "Bearer my-secret-token"
	if got != want {
		t.Errorf("AuthToken: got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// TestBasicAuth – BasicAuth sets the Authorization: Basic header.
// ---------------------------------------------------------------------------

func TestBasicAuth(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", opts1("BasicAuth", strDict("Username", "alice", "Password", "secret")))

	eval(t, ctx, fmt.Sprintf("http.get[%q;opts]", ts.URL))

	authHeader := capt.headers.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Fatalf("BasicAuth: expected Authorization: Basic …, got %q", authHeader)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		t.Fatalf("BasicAuth: base64 decode failed: %v", err)
	}
	if string(decoded) != "alice:secret" {
		t.Errorf("BasicAuth: decoded credentials %q, want %q", string(decoded), "alice:secret")
	}
}

// ---------------------------------------------------------------------------
// TestPathParams – PathParams substitutes {token} placeholders in the URL.
// ---------------------------------------------------------------------------

func TestPathParams(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	urlTemplate := ts.URL + "/users/{id}/posts/{pid}"
	ctx.AssignGlobal("opts", opts1("PathParams", strDict("id", "42", "pid", "7")))

	eval(t, ctx, fmt.Sprintf("http.get[%q;opts]", urlTemplate))

	if capt.path != "/users/42/posts/7" {
		t.Errorf("PathParams: server got path %q, want %q", capt.path, "/users/42/posts/7")
	}
}

// ---------------------------------------------------------------------------
// TestDyadicBracket – the bracket form `http.get[url;opts]` passes both
// the URL and options dict correctly.
// ---------------------------------------------------------------------------

func TestDyadicBracket(t *testing.T) {
	ts, capt := newServer(t, 200, "body-text")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", opts1("QueryParam", strDict("x", "1")))
	ctx.AssignGlobal("url", goal.NewS(ts.URL))

	v := eval(t, ctx, "http.get[url;opts]")
	d := mustDict(t, ctx, v)

	if mustI(t, dictField(t, d, "ok")) != 1 {
		t.Errorf("dyadic bracket: expected ok=1")
	}
	if mustS(t, ctx, dictField(t, d, "body")) != "body-text" {
		t.Errorf("dyadic bracket: unexpected body")
	}
	if capt.query["x"] != "1" {
		t.Errorf("dyadic bracket: QueryParam x not received")
	}
}

// ---------------------------------------------------------------------------
// TestPartialApplication – http.get[url;] binds the URL and returns a monad
// that accepts an opts dict, enabling reuse across multiple calls.
// ---------------------------------------------------------------------------

func TestPartialApplication(t *testing.T) {
	ts, capt := newServer(t, 200, "ok")
	ctx := newCtx(t)

	ctx.AssignGlobal("url", goal.NewS(ts.URL))

	// Create a partially-applied function with the URL bound.
	getURL := eval(t, ctx, "http.get[url;]")
	if getURL.IsPanic() {
		t.Fatalf("partial application returned panic: %s", getURL.Sprint(ctx, true))
	}
	if !getURL.IsCallable() {
		t.Fatalf("partial application: expected callable, got %q", getURL.Type())
	}

	// Call the projection with different opts — first call with QueryParam q=1.
	ctx.AssignGlobal("getUrl", getURL)
	ctx.AssignGlobal("opts1", opts1("QueryParam", strDict("q", "first")))
	eval(t, ctx, "getUrl opts1")
	if capt.query["q"] != "first" {
		t.Errorf("partial application call 1: q=%q, want %q", capt.query["q"], "first")
	}

	// Second call with different params, same URL.
	ctx.AssignGlobal("opts2", opts1("QueryParam", strDict("q", "second")))
	eval(t, ctx, "getUrl opts2")
	if capt.query["q"] != "second" {
		t.Errorf("partial application call 2: q=%q, want %q", capt.query["q"], "second")
	}
}

// ---------------------------------------------------------------------------
// TestHttpRequest – http.request with an explicit client, URL, and opts.
// The response dict uses "bodybytes" (AB) rather than "body" (S).
// ---------------------------------------------------------------------------

func TestHttpRequest(t *testing.T) {
	ts, capt := newServer(t, 201, "created")
	ctx := newCtx(t)

	// Build a client with a base URL.
	clientOpts := goal.NewD(
		goal.NewAS([]string{"BaseURL"}),
		goal.NewAV([]goal.V{goal.NewS(ts.URL)}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")

	ctx.AssignGlobal("client", client)
	ctx.AssignGlobal("reqOpts", opts2(
		"Method", goal.NewS("POST"),
		"Body", goal.NewS(`{"name":"test"}`),
	))

	v := eval(t, ctx, `http.request[client;"/items";reqOpts]`)
	d := mustDict(t, ctx, v)

	if mustI(t, dictField(t, d, "statuscode")) != 201 {
		t.Errorf("http.request: expected statuscode 201")
	}
	if capt.method != "POST" {
		t.Errorf("http.request: server got method %q, want POST", capt.method)
	}
	if capt.body != `{"name":"test"}` {
		t.Errorf("http.request: body %q, want %q", capt.body, `{"name":"test"}`)
	}

	// bodybytes must be an AB (byte array), not a string.
	bb := dictField(t, d, "bodybytes")
	ab, ok := bb.BV().(*goal.AB)
	if !ok {
		t.Fatalf("http.request bodybytes: expected AB, got %q", bb.Type())
	}
	if string(ab.Slice) != "created" {
		t.Errorf("http.request bodybytes: got %q, want %q", string(ab.Slice), "created")
	}

	// "body" key must not be present.
	kas, _ := d.KeyArray().(*goal.AS)
	for _, k := range kas.Slice {
		if k == "body" {
			t.Errorf("http.request response dict must not contain \"body\", only \"bodybytes\"")
		}
	}
}

// ---------------------------------------------------------------------------
// TestHttpRequestBracket – `http.request[client;url]` does a GET with no opts.
// ---------------------------------------------------------------------------

func TestHttpRequestBracket(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	clientOpts := goal.NewD(
		goal.NewAS([]string{"BaseURL"}),
		goal.NewAV([]goal.V{goal.NewS(ts.URL)}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")
	ctx.AssignGlobal("client", client)

	eval(t, ctx, `http.request[client;"/ping"]`)

	if capt.method != "GET" {
		t.Errorf("http.request bracket: method %q, want GET", capt.method)
	}
	if capt.path != "/ping" {
		t.Errorf("http.request bracket: path %q, want /ping", capt.path)
	}
}

// ---------------------------------------------------------------------------
// TestHttpRequestProjection – http.request[client;] binds the client,
// producing a dyad that accepts [url; opts].
// ---------------------------------------------------------------------------

func TestHttpRequestProjection(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	clientOpts := goal.NewD(
		goal.NewAS([]string{"BaseURL"}),
		goal.NewAV([]goal.V{goal.NewS(ts.URL)}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")
	ctx.AssignGlobal("client", client)

	// Bind the client; leave url+opts free.
	myReq := eval(t, ctx, "http.request[client;]")
	if !myReq.IsCallable() {
		t.Fatalf("http.request projection: expected callable, got %q", myReq.Type())
	}
	ctx.AssignGlobal("myReq", myReq)

	ctx.AssignGlobal("reqOpts", opts1("Method", goal.NewS("DELETE")))
	eval(t, ctx, `myReq["/items/99";reqOpts]`)

	if capt.method != "DELETE" {
		t.Errorf("projection call: method %q, want DELETE", capt.method)
	}
	if capt.path != "/items/99" {
		t.Errorf("projection call: path %q, want /items/99", capt.path)
	}
}

// ---------------------------------------------------------------------------
// TestClientToken – http.client Token sets Authorization: Bearer on all reqs.
// ---------------------------------------------------------------------------

func TestClientToken(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	clientOpts := goal.NewD(
		goal.NewAS([]string{"Token"}),
		goal.NewAV([]goal.V{goal.NewS("shared-token")}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")
	ctx.AssignGlobal("client", client)

	// Use http.request so we can supply the explicit client.
	ctx.AssignGlobal("reqOpts", opts1("Method", goal.NewS("GET")))
	eval(t, ctx, fmt.Sprintf(`http.request[client;%q;reqOpts]`, ts.URL))

	got := capt.headers.Get("Authorization")
	if got != "Bearer shared-token" {
		t.Errorf("client Token: Authorization header %q, want %q", got, "Bearer shared-token")
	}
}

// ---------------------------------------------------------------------------
// TestClientBaseURL – http.client BaseURL is prepended to relative URLs in
// http.request.
// ---------------------------------------------------------------------------

func TestClientBaseURL(t *testing.T) {
	ts, capt := newServer(t, 200, "")
	ctx := newCtx(t)

	clientOpts := goal.NewD(
		goal.NewAS([]string{"BaseURL"}),
		goal.NewAV([]goal.V{goal.NewS(ts.URL)}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")
	ctx.AssignGlobal("client", client)

	ctx.AssignGlobal("reqOpts", opts1("Method", goal.NewS("GET")))
	eval(t, ctx, `http.request[client;"/api/v1/status";reqOpts]`)

	if capt.path != "/api/v1/status" {
		t.Errorf("BaseURL: path %q, want /api/v1/status", capt.path)
	}
}

// ---------------------------------------------------------------------------
// TestResponseHeaders – response headers are returned as a dict.
// ---------------------------------------------------------------------------

func TestResponseHeaders(t *testing.T) {
	ts, _ := newServer(t, 200, "")
	ctx := newCtx(t)

	v := eval(t, ctx, fmt.Sprintf("http.get[%q]", ts.URL))
	d := mustDict(t, ctx, v)
	hd := mustDict(t, ctx, dictField(t, d, "headers"))

	// newServer always sets X-Test-Header: test-value.
	// Headers are canonically capitalised by net/http.
	kas, ok := hd.KeyArray().(*goal.AS)
	if !ok {
		t.Fatalf("response headers: keys must be AS, got %q", hd.KeyArray().Type())
	}
	found := false
	for _, k := range kas.Slice {
		if k == "X-Test-Header" {
			found = true
		}
	}
	if !found {
		t.Errorf("response headers: X-Test-Header not present; keys = %v", kas.Slice)
	}
}

// ---------------------------------------------------------------------------
// Error cases: wrong argument types produce panics.
// ---------------------------------------------------------------------------

func TestWrongArgTypes(t *testing.T) {
	ctx := newCtx(t)

	// URL must be a string.
	msg := evalPanic(t, ctx, "http.get[42]")
	if !strings.Contains(msg, "expected string URL") {
		t.Errorf("wrong URL type: unexpected panic message: %q", msg)
	}

	// opts must be a dict.
	ctx.AssignGlobal("url", goal.NewS("http://localhost"))
	msg = evalPanic(t, ctx, `http.get[url;"not-a-dict"]`)
	if !strings.Contains(msg, "expected dict") {
		t.Errorf("wrong opts type: unexpected panic message: %q", msg)
	}

	// http.client arg must be a dict.
	msg = evalPanic(t, ctx, "http.client[99]")
	if !strings.Contains(msg, "expected dict") {
		t.Errorf("http.client non-dict: unexpected panic message: %q", msg)
	}

	// http.request client must be http.client or dict.
	msg = evalPanic(t, ctx, `http.request["not-a-client";"http://x";(!"")!()]`)
	if !strings.Contains(msg, "client argument") {
		t.Errorf("http.request bad client: unexpected panic message: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// Error cases: unsupported option keys produce panics.
// ---------------------------------------------------------------------------

func TestUnsupportedClientOption(t *testing.T) {
	ctx := newCtx(t)

	ctx.AssignGlobal("badOpts", goal.NewD(
		goal.NewAS([]string{"Unsupported"}),
		goal.NewAV([]goal.V{goal.NewI(1)}),
	))
	msg := evalPanic(t, ctx, "http.client[badOpts]")
	if !strings.Contains(msg, "unsupported option") {
		t.Errorf("unsupported client option: unexpected panic message: %q", msg)
	}
}

func TestUnsupportedRequestOption(t *testing.T) {
	ts, _ := newServer(t, 200, "")
	ctx := newCtx(t)

	ctx.AssignGlobal("opts", goal.NewD(
		goal.NewAS([]string{"Bogus"}),
		goal.NewAV([]goal.V{goal.NewI(1)}),
	))
	msg := evalPanic(t, ctx, fmt.Sprintf("http.get[%q;opts]", ts.URL))
	if !strings.Contains(msg, "unsupported request option") {
		t.Errorf("unsupported request option: unexpected panic message: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// Rate-limiting tests
//
// These tests exercise both the ratelimit.* verbs directly and the
// RateLimitPerSecond http.client option, which wires a leaky-bucket limiter
// into every request made through the client.
//
// Timing assertions use a rate of 5 req/sec (200 ms gap between tokens).
// Three takes/requests consume two inter-token gaps → ≥ 400 ms actual; the
// tests assert ≥ 300 ms to give a 25 % safety margin against scheduler jitter
// without making the suite excessively slow.
// ---------------------------------------------------------------------------

// newRLCtx creates a Goal context with both http and ratelimit verbs imported.
func newRLCtx(t *testing.T) *goal.Context {
	t.Helper()
	ctx := goal.NewContext()
	goalhttp.Import(ctx, "")
	goalratelimit.Import(ctx, "")
	return ctx
}

// TestRateLimitNewTake verifies that ratelimit.new returns a limiter value and
// that ratelimit.take returns 1i.
func TestRateLimitNewTake(t *testing.T) {
	ctx := newRLCtx(t)

	// A fast limiter so the test doesn't block meaningfully.
	rl := eval(t, ctx, "ratelimit.new 100")
	ctx.AssignGlobal("rl", rl)

	// ratelimit.take must return 1i.
	result := eval(t, ctx, "ratelimit.take rl")
	if !result.IsI() || result.I() != 1 {
		t.Errorf("ratelimit.take: expected 1i, got %s", result.Sprint(ctx, true))
	}
}

// TestRateLimitTakeSpacing verifies that the leaky-bucket limiter actually
// spaces successive calls.  At 5 req/sec the token interval is 200 ms, so
// three takes should block for at least 300 ms total.
func TestRateLimitTakeSpacing(t *testing.T) {
	ctx := newRLCtx(t)
	eval(t, ctx, "rl: ratelimit.new 5")

	start := time.Now()
	eval(t, ctx, "ratelimit.take rl")
	eval(t, ctx, "ratelimit.take rl")
	eval(t, ctx, "ratelimit.take rl")
	elapsed := time.Since(start)

	const wantMin = 300 * time.Millisecond
	if elapsed < wantMin {
		t.Errorf("ratelimit (5/s): 3 takes took %v, want >= %v", elapsed, wantMin)
	}
}

// TestRateLimitClientOption verifies that an http.client created with
// RateLimitPerSecond automatically rate-limits named-method verb requests
// (http.get, http.post, …).  Three requests through a 5/sec client must take
// at least 300 ms.
func TestRateLimitClientOption(t *testing.T) {
	ts, _ := newServer(t, 200, "ok")
	ctx := newRLCtx(t)

	clientOpts := goal.NewD(
		goal.NewAS([]string{"RateLimitPerSecond"}),
		goal.NewAV([]goal.V{goal.NewI(5)}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")
	ctx.AssignGlobal("client", client)
	ctx.AssignGlobal("url", goal.NewS(ts.URL))

	start := time.Now()
	eval(t, ctx, "http.get[client;url]")
	eval(t, ctx, "http.get[client;url]")
	eval(t, ctx, "http.get[client;url]")
	elapsed := time.Since(start)

	const wantMin = 300 * time.Millisecond
	if elapsed < wantMin {
		t.Errorf("http.client RateLimitPerSecond 5 (named verb): 3 requests took %v, want >= %v", elapsed, wantMin)
	}
}

// TestRateLimitClientViaRequest verifies that RateLimitPerSecond also applies
// when requests are made through http.request (the explicit-client path).
func TestRateLimitClientViaRequest(t *testing.T) {
	ts, _ := newServer(t, 200, "ok")
	ctx := newRLCtx(t)

	clientOpts := goal.NewD(
		goal.NewAS([]string{"BaseURL", "RateLimitPerSecond"}),
		goal.NewAV([]goal.V{goal.NewS(ts.URL), goal.NewI(5)}),
	)
	ctx.AssignGlobal("clientOpts", clientOpts)
	client := eval(t, ctx, "http.client[clientOpts]")
	ctx.AssignGlobal("client", client)

	start := time.Now()
	eval(t, ctx, `http.request[client;"/"]`)
	eval(t, ctx, `http.request[client;"/"]`)
	eval(t, ctx, `http.request[client;"/"]`)
	elapsed := time.Since(start)

	const wantMin = 300 * time.Millisecond
	if elapsed < wantMin {
		t.Errorf("http.client RateLimitPerSecond 5 (http.request): 3 requests took %v, want >= %v", elapsed, wantMin)
	}
}

// TestRateLimitErrors verifies that invalid arguments produce panics with
// informative messages, covering both the ratelimit.* verbs and the
// RateLimitPerSecond client option.
func TestRateLimitErrors(t *testing.T) {
	ctx := newRLCtx(t)

	// ratelimit.new with zero rate.
	msg := evalPanic(t, ctx, "ratelimit.new 0")
	if !strings.Contains(msg, "positive integer") {
		t.Errorf("ratelimit.new 0: unexpected message: %q", msg)
	}

	// ratelimit.new with non-integer (string).
	msg = evalPanic(t, ctx, `ratelimit.new "ten"`)
	if !strings.Contains(msg, "expected integer") {
		t.Errorf("ratelimit.new string: unexpected message: %q", msg)
	}

	// ratelimit.take with a non-limiter value.
	msg = evalPanic(t, ctx, "ratelimit.take 42")
	if !strings.Contains(msg, "expected ratelimit.limiter") {
		t.Errorf("ratelimit.take non-limiter: unexpected message: %q", msg)
	}

	// http.client RateLimitPerSecond=0 must panic.
	ctx.AssignGlobal("zeroOpts", goal.NewD(
		goal.NewAS([]string{"RateLimitPerSecond"}),
		goal.NewAV([]goal.V{goal.NewI(0)}),
	))
	msg = evalPanic(t, ctx, "http.client[zeroOpts]")
	if !strings.Contains(msg, "positive integer") {
		t.Errorf("http.client RateLimitPerSecond=0: unexpected message: %q", msg)
	}

	// http.client RateLimitPerSecond with a non-integer value must panic.
	ctx.AssignGlobal("strOpts", goal.NewD(
		goal.NewAS([]string{"RateLimitPerSecond"}),
		goal.NewAV([]goal.V{goal.NewS("fast")}),
	))
	msg = evalPanic(t, ctx, "http.client[strOpts]")
	if !strings.Contains(msg, "must be an integer") {
		t.Errorf("http.client RateLimitPerSecond string: unexpected message: %q", msg)
	}
}
