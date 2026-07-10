// Package help provides help text for ari extensions combined with
// Goal's standard language help.
package help

import (
	goalzip "codeberg.org/anaseto/goal/archive/zip"
	goalbase64 "codeberg.org/anaseto/goal/encoding/base64"
	goalhelp "codeberg.org/anaseto/goal/help"
	goalmath "codeberg.org/anaseto/goal/math"
)

// HelpFunc returns a combined help function covering Goal's core language and
// all ari extensions (http, sql, ratelimit). Pass it to
// cmd.Config{Help: help.HelpFunc()}.
func HelpFunc() func(string) string { //nolint:revive // name stutters by design: callers use arihelp.HelpFunc()
	return goalhelp.Wrap(
		extensionHelp(),
		goalmath.HelpFunc(),
		goalzip.HelpFunc(),
		goalbase64.HelpFunc(),
		goalhelp.HelpFunc(),
	)
}

func extensionHelp() func(string) string {
	m := map[string]string{}

	// Override the topics index to include extension sections.
	m[""] = helpTopics

	// Extension section overviews.
	m["http"] = helpHTTP
	m["sql"] = helpSQL
	m["ratelimit"] = helpRateLimit

	// helps
	m["helps"] = `helps s    return help text for topic s as a string (same topics as help)
  t: helps"http.get"    / capture help text for use in a program`

	addHTTPVerbHelp(m)
	addSQLVerbHelp(m)
	addRateLimitVerbHelp(m)

	return func(s string) string { return m[s] }
}

// addHTTPVerbHelp adds the individual http.* verb entries.
func addHTTPVerbHelp(m map[string]string) {
	m["http.get"] = `http.get url                   GET request using default client
http.get[cl;url]               GET request using http.client cl
http.get[cl;url;opts]          GET request with per-request opts dict
Returns: dict with keys "status" (s), "statuscode" (i), "headers" (d),
         "body" (s), "ok" (i)`

	m["http.post"] = `http.post url                  POST request (set Body/ContentType in opts)
http.post[cl;url;opts]         POST with explicit client and opts dict`

	m["http.put"] = `http.put url                   PUT request
http.put[cl;url;opts]          PUT with explicit client and opts dict`

	m["http.patch"] = `http.patch url                 PATCH request
http.patch[cl;url;opts]        PATCH with explicit client and opts dict`

	m["http.delete"] = `http.delete url                DELETE request
http.delete[cl;url;opts]       DELETE with explicit client and opts dict`

	m["http.head"] = `http.head url                  HEAD request (returns headers, no body)
http.head[cl;url;opts]         HEAD with explicit client and opts dict`

	m["http.options"] = `http.options url               OPTIONS request
http.options[cl;url;opts]      OPTIONS with explicit client and opts dict`

	m["http.request"] = `http.request[cl; url]            request with client cl (method defaults to GET)
http.request[cl;url;opts]        request with explicit method, body, headers, etc.
  opts key "Method" sets the HTTP method (default "GET")
  see help"http" for all opts keys
Returns: dict with keys "status" (s), "statuscode" (i), "headers" (d),
         "bodybytes" (byte array), "ok" (i)`

	m["http.client"] = `http.client d    create a reusable http.client configured by options dict d

Client options (keys of d):
  AllowGetMethodPayload  i  allow a body on GET requests (0/1)
  AuthScheme             s  override "Bearer" prefix (default "Bearer")
  AuthToken              s  default Authorization: Bearer token (alias: Token)
  BaseURL                s  prepended to every request URL
  BasicAuth              d  default basic auth; keys: Username, Password
                            (alias: UserInfo)
  Certificate            d  client TLS certificate; keys: CertFile, KeyFile
                            (paths to PEM files)
  CloseConnection        i  close the connection after each request (0/1)
  ContentLength          i  set the Content-Length header (0/1)
  Cookies                d  default cookies; cookie name → value (s)
  Debug                  i  enable resty verbose logging (0/1)
  DebugBodyLimit         i  max body size logged in debug mode (bytes)
  DigestAuth             d  digest auth; keys: Username, Password
  DisableWarn            i  suppress resty warnings (0/1)
  FollowRedirects        i  follow HTTP redirects (0/1, default 1)
  FormData               d  default form data for every request
  GenerateCurlOnDebug    i  log equivalent curl command in debug mode (0/1)
  Header                 d  default headers for every request
  HeaderAuthorizationKey s  override the Authorization header name
  HeaderVerbatim         d  default headers without canonicalisation
  OutputDirectory        s  directory for responses saved via Output
  PathParams             d  default URL path params (URL-encoded)
  Proxy                  s  proxy URL, e.g. "http://proxyserver:8080"
  QueryParam             d  default query parameters for every request
  RateLimitPerSecond     i  max req/s (leaky bucket); applied before each request
  RawPathParams          d  default URL path params (not URL-encoded)
  ResponseBodyLimit      i  max response body size in bytes
  RetryAfterErrorCondition i also retry on 4xx/5xx response status (0/1)
  RetryCount             i  automatic retries on failure
  RetryMaxWaitTimeMilli  i  max retry back-off duration (ms)
  RetryResetReaders      i  reset request readers between retries (0/1)
  RetryWaitTimeMilli     i  initial retry wait time (ms)
  RootCertificate        s  path to a PEM file of trusted root certificates
  RootCertificatePEM     s  PEM content of trusted root certificates
  Scheme                 s  scheme applied to scheme-less request URLs
  TimeoutMilli           i  request timeout in milliseconds
  TLSInsecureSkipVerify  i  skip TLS certificate verification (0/1)
  UnescapeQueryParams    i  unescape (decode) query parameters (0/1)`
}

// addSQLVerbHelp adds the individual sql.* verb entries.
func addSQLVerbHelp(m map[string]string) {
	m["sql.open"] = `sql.open "scheme://dsn"    open a database connection; returns sql.conn or error
  sql.open "sqlite://data.db"
  sql.open "sqlite://:memory:"`

	m["sql.close"] = `sql.close db    close database connection db; returns 1i or error`

	m["sql.q"] = `sql.q[db; "SELECT …"]                  query; returns columnar dict (column name → array)
sql.q[db; "SELECT … WHERE x=?"; args]  parameterised query; args is a Goal array
  Result: t"col" gives column array (AI/AF/AS/AV); nan t"col" marks NULLs`

	m["sql.exec"] = `sql.exec[db; "INSERT …"]                  execute statement; returns exec-result dict
sql.exec[db; "INSERT … VALUES(?)"; args]  parameterised exec
  Result dict keys: "lastInsertId" (i), "rowsAffected" (i)`

	m["sql.tx"] = `sql.tx[db; {[tx] … }]    run lambda in a transaction
  Commits if lambda returns a non-error value; rolls back otherwise.
  tx supports the same sql.q, sql.exec, and sql.tx interface as db.`
}

// addRateLimitVerbHelp adds the individual ratelimit.* verb entries.
func addRateLimitVerbHelp(m map[string]string) {
	m["ratelimit.new"] = `ratelimit.new i    create a leaky-bucket RateLimiter allowing i requests/second
  rl: ratelimit.new 10`

	m["ratelimit.take"] = `ratelimit.take rl    block until the limiter allows the next request; returns 1i
  ratelimit.take rl
  http.get "https://api.example.com/data"`
}

const helpTopics = `TOPICS HELP
Type help TOPIC or h TOPIC where TOPIC is one of:

Goal language:
"s"         syntax
"t"         value types
"v"         verbs (like +*-%,)
"nv"        named verbs (like in, sign)
"a"         adverbs (/\')
"tm"        time handling
"rt"        runtime system
"io"        IO verbs (like say, open, read)
op          where op is a builtin's name (like "+" or "in")
"helps"     helps s — return help text as a string instead of printing it

Extensions:
"http"      HTTP client extension (http.get, http.post, http.client, …)
"sql"       SQL extension (sql.open, sql.q, sql.exec, sql.tx)
"ratelimit" rate limiter extension (ratelimit.new, ratelimit.take)
verb        where verb is an extension verb (like "http.get")

Notations:
        i (integer) n (number) s (string) r (regexp)
        d (dict) t (dict S!Y) h (handle) e (error)
        f (function) F (dyadic function)
        x,y,z (any other) I,N,S,X,Y,A (arrays)
`

const helpHTTP = `HTTP VERBS HELP
Type: http.client (reusable HTTP client with shared config and state)

Named method verbs (url is a string; returns dict with keys "status" (s),
"statuscode" (i), "headers" (d), "body" (s), "ok" (i)):
http.get url                    GET using default client
http.get[cl;url]                GET using http.client cl
http.get[cl;url;opts]           GET with per-request opts dict
http.post / http.put / http.patch / http.delete / http.head / http.options
  – same three calling forms as http.get above

Generic verb (returns the same dict but with "bodybytes" (byte array) instead
of "body"):
http.request[cl; url]           GET using cl (method defaults to "GET")
http.request[cl;url;opts]       opts key "Method" sets the HTTP method

Creating a reusable client:
http.client d    create http.client from options dict d (see help"http.client")

Per-request opts keys:
  AuthScheme          s   Authorization scheme (default "Bearer")
  AuthToken           s   Authorization: <scheme> <token>
  BasicAuth           d   basic auth; keys: Username, Password
  Body                s   raw request body (string or byte array)
  ContentLength       i   set the Content-Length header (0/1)
  ContentType         s   Content-Type header
  Cookies             d   cookies; cookie name → value (s)
  Debug               i   enable resty debug logging for this request (0/1)
  DigestAuth          d   digest auth; keys: Username, Password
  Files               d   multipart file upload; form param → file path
  FormData            d   url-encoded form data (values: s or AS)
  GenerateCurlOnDebug i   log equivalent curl command in debug mode (0/1)
  Header              d   request headers (values: s or AS)
  HeaderVerbatim      d   headers without canonicalisation (values: s or AS)
  Method              s   HTTP method for http.request (default "GET")
  MultipartBoundary   s   custom boundary for multipart requests
  MultipartFormData   d   fields sent as multipart/form-data (values: s)
  Output              s   save response body to this file path ("body" will be
                          empty; relative paths go under OutputDirectory)
  PathParams          d   URL path params (URL-encoded)
  QueryParam          d   URL query parameters (values: s or AS)
  QueryString         s   raw query string, e.g. "a=1&b=2"
  RawPathParams       d   URL path params (not URL-encoded)
  ResponseBodyLimit   i   max response body size in bytes (error if exceeded)
  SRV                 d   resolve host via DNS SRV lookup; keys: Domain
                          (required), Service (optional)
  UnescapeQueryParams i   unescape (decode) query parameters (0/1)

Examples:
  r: http.get "https://example.com"
  r: http.post[cl; "https://api.example.com/items"; ..[Body:body; ContentType:"application/json"]]
  myReq: http.request[myClient;]    / partial application; supply url+opts later
`

const helpSQL = `SQL VERBS HELP
Types: sql.conn (database connection)

sql.open "scheme://dsn"                open connection; returns sql.conn or error
  sql.open "sqlite://data.db"          file-based SQLite database
  sql.open "sqlite://:memory:"         in-memory SQLite database
sql.close db                           close connection; returns 1i or error

sql.q[db; "SELECT …"]                  query; returns columnar dict
sql.q[db; "SELECT … WHERE x=?"; v]    parameterised query; v is a Goal array
sql.exec[db; "INSERT …"]               execute statement; returns exec-result dict
sql.exec[db; "INSERT … VALUES(?)"; v]  parameterised exec
sql.tx[db; {[tx] … }]                  lambda transaction (tx has same interface as db)
  Commits if lambda returns a non-error value; rolls back otherwise.

Query result: dict mapping column names (S) to per-column arrays
  t"col"              column array (AI / AF / AS / AV)
  nan t"col"          boolean array; 1 at each NULL position
  42 nan t"col"       fill NULLs with 42
  "" nan t"col"       fill NULLs with empty string

Exec result dict: "lastInsertId" (i) and "rowsAffected" (i)

NULL maps to 0n (Goal's NaN). Columns with any NULL are returned as AF or AV.

Type mapping:
  SQL NULL / unknown    → 0n
  SQL INTEGER           → I (int64)
  SQL REAL / FLOAT      → N (float64)
  SQL TEXT              → S
  SQL BLOB              → byte array
  SQL BOOLEAN           → I (1 or 0)
  time.Time             → I (Unix microseconds, UTC)
`

const helpRateLimit = `RATELIMIT VERBS HELP
Type: ratelimit.limiter

ratelimit.new i    create a leaky-bucket limiter at i requests/second
ratelimit.take rl  block until the next request slot; returns 1i when unblocked

Uses leaky-bucket algorithm: requests are evenly spaced, no burst capacity.

Example:
  rl: ratelimit.new 10       / allow 10 requests per second
  {ratelimit.take rl; http.get "https://api.example.com/"} each urls

For HTTP clients prefer http.client's RateLimitPerSecond option, which calls
the rate limiter automatically before every request through that client.
`
