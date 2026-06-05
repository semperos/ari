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

	// -----------------------------------------------------------------------
	// http individual verb entries
	// -----------------------------------------------------------------------

	m["http.get"] = `http.get url                   GET request using default client
http.get[cl;url]               GET request using http.client cl
http.get[cl;url;opts]          GET request with per-request opts dict
Returns: dict with keys "status" (i), "body" (s), "header" (d)`

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
  see help"http" for all opts keys`

	m["http.client"] = `http.client d    create a reusable http.client configured by options dict d

Client options (keys of d):
  BaseURL                s  prepended to every request URL
  AuthToken              s  default Authorization: Bearer token
  AuthScheme             s  override "Bearer" prefix (default "Bearer")
  BasicAuth              d  default basic auth; keys: Username, Password
  Debug                  i  enable resty verbose logging (0/1)
  DisableWarn            i  suppress resty warnings (0/1)
  FollowRedirects        i  follow HTTP redirects (0/1, default 1)
  FormData               d  default form data for every request
  Header                 d  default headers for every request
  HeaderAuthorizationKey s  override the Authorization header name
  PathParams             d  default URL path params (URL-encoded)
  QueryParam             d  default query parameters for every request
  RawPathParams          d  default URL path params (not URL-encoded)
  RateLimitPerSecond     i  max req/s (leaky bucket); applied before each request
  RetryCount             i  automatic retries on failure
  RetryMaxWaitTimeMilli  i  max retry back-off duration (ms)
  RetryResetReaders      i  reset request readers between retries (0/1)`

	// -----------------------------------------------------------------------
	// sql individual verb entries
	// -----------------------------------------------------------------------

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

	// -----------------------------------------------------------------------
	// ratelimit individual verb entries
	// -----------------------------------------------------------------------

	m["ratelimit.new"] = `ratelimit.new i    create a leaky-bucket RateLimiter allowing i requests/second
  rl: ratelimit.new 10`

	m["ratelimit.take"] = `ratelimit.take rl    block until the limiter allows the next request; returns 1i
  ratelimit.take rl
  http.get "https://api.example.com/data"`

	return func(s string) string { return m[s] }
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

Named method verbs (url is a string; returns dict with "status" (i), "body" (s), "header" (d)):
http.get url                    GET using default client
http.get[cl;url]                GET using http.client cl
http.get[cl;url;opts]           GET with per-request opts dict
http.post / http.put / http.patch / http.delete / http.head / http.options
  – same three calling forms as http.get above

Generic verb:
http.request[cl; url]           GET using cl (method defaults to "GET")
http.request[cl;url;opts]       opts key "Method" sets the HTTP method

Creating a reusable client:
http.client d    create http.client from options dict d (see help"http.client")

Per-request opts keys:
  AuthToken       s   Authorization: Bearer <token>
  BasicAuth       d   basic auth; keys: Username, Password
  Body            s   raw request body
  ContentType     s   Content-Type header
  Debug           i   enable resty debug logging for this request (0/1)
  FormData        d   url-encoded form data (values: s or AS)
  Header          d   request headers (values: s or AS)
  PathParams      d   URL path params (URL-encoded)
  QueryParam      d   URL query parameters (values: s or AS)
  RawPathParams   d   URL path params (not URL-encoded)
  Method          s   HTTP method for http.request (default "GET")

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
