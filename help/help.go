// Package help provides help text for ari extensions combined with
// Goal's standard language help.
package help

import (
	goalhelp "codeberg.org/anaseto/goal/help"
)

// HelpFunc returns a combined help function covering Goal's core language and
// all ari extensions (fyne, http, sql, ratelimit). Pass it to
// cmd.Config{Help: help.HelpFunc()}.
func HelpFunc() func(string) string { //nolint:revive // name stutters by design: callers use arihelp.HelpFunc()
	return goalhelp.Wrap(extensionHelp(), goalhelp.HelpFunc())
}

func extensionHelp() func(string) string { //nolint:funlen
	m := map[string]string{}

	// Override the topics index to include extension sections.
	m[""] = helpTopics

	// Extension section overviews.
	m["fyne"] = helpFyne
	m["http"] = helpHTTP
	m["sql"] = helpSQL
	m["ratelimit"] = helpRateLimit

	// -----------------------------------------------------------------------
	// fyne individual verb entries
	// -----------------------------------------------------------------------

	m["fyne.app"] = `fyne.app s        create Fyne application; s is the app ID string
fyne.app 0        create app with no explicit ID`

	m["fyne.window"] = `fyne.window a              create window (no title) from fyne.app a
fyne.window["t"; a]        create window with title t from fyne.app a`

	m["fyne.run"] = `fyne.run w     ShowAndRun window w — shows the window and blocks until it is closed`

	m["fyne.setcontent"] = `fyne.setcontent[w; widget]    set window w content to widget; returns w`

	m["fyne.settitle"] = `fyne.settitle[w; s]    set window w title to string s; returns w`

	m["fyne.resize"] = `fyne.resize[w; (width;height)]    resize window w to float dimensions; returns w`

	m["fyne.title"] = `fyne.title w    return the title string of window w`

	m["fyne.label"] = `fyne.label s    create a Label widget displaying string s`

	m["fyne.entry"] = `fyne.entry s    create an Entry widget with placeholder text s (use "" for none)`

	m["fyne.password"] = `fyne.password s    create a password Entry widget with placeholder text s`

	m["fyne.multiline"] = `fyne.multiline s    create a multiline Entry widget with placeholder text s`

	m["fyne.progress"] = `fyne.progress n    create a ProgressBar set to value n (0.0..1.0)`

	m["fyne.separator"] = `fyne.separator 0    create a horizontal Separator widget`

	m["fyne.spacer"] = `fyne.spacer 0    create a layout Spacer (expands in box/toolbar layouts)`

	m["fyne.button"] = `fyne.button s             create Button with label s and no callback
fyne.button["l"; f]       create Button with label l; f is called with 0i on tap`

	m["fyne.check"] = `fyne.check s              create Check with label s and no callback
fyne.check["l"; f]        create Check with label l; f is called with 1i (checked) or 0i`

	m["fyne.slider"] = `fyne.slider (lo;hi)           create Slider with range [lo,hi] and no callback
fyne.slider[(lo;hi); f]      create Slider; f is called with the current float value on change`

	m["fyne.select"] = `fyne.select opts             create Select widget from string array opts; no callback
fyne.select[opts; f]         create Select; f is called with the selected string on change`

	m["fyne.text"] = `fyne.text w    return text string from Label, Entry, or Select widget w`

	m["fyne.settext"] = `fyne.settext[w; s]    set text of Label, Entry, or Button widget w to s; returns w`

	m["fyne.value"] = `fyne.value w    return numeric value from Slider or ProgressBar (float), or Check (0i/1i)`

	m["fyne.setvalue"] = `fyne.setvalue[w; n]    set value of Slider or ProgressBar widget w to n; returns w`

	m["fyne.enable"] = `fyne.enable w    enable widget w (if it supports Enable/Disable); returns w`

	m["fyne.disable"] = `fyne.disable w    disable widget w; returns w`

	m["fyne.show"] = `fyne.show w    show widget w; returns w`

	m["fyne.hide"] = `fyne.hide w    hide widget w; returns w`

	m["fyne.refresh"] = `fyne.refresh w    refresh widget w (redraw); returns w`

	m["fyne.vbox"] = `fyne.vbox widgets    create a VBox container from an array of fyne.widget values`

	m["fyne.hbox"] = `fyne.hbox widgets    create an HBox container from an array of fyne.widget values`

	m["fyne.scroll"] = `fyne.scroll w    wrap widget w in a ScrollContainer`

	m["fyne.padded"] = `fyne.padded w    wrap widget w in a Padded container`

	m["fyne.center"] = `fyne.center w    wrap widget w in a Center container`

	m["fyne.split"] = `fyne.split["h"; (w1;w2)]    create a horizontal HSplitContainer
fyne.split["v"; (w1;w2)]    create a vertical VSplitContainer`

	m["fyne.border"] = `fyne.border[top;bottom;left;right;rest…]    Border container; pass 0 for unused slots`

	m["fyne.tabs"] = `fyne.tabs[("label";widget);…]    AppTabs container with one tab per pair`

	m["fyne.form"] = `fyne.form[("label";widget);…]    Form widget with one labelled row per pair`

	m["fyne.toolbar"] = `fyne.toolbar[action;…]    Toolbar built from fyne.action values`

	m["fyne.action"] = `fyne.action[icon;f]    ToolbarAction with Fyne Resource icon and callback f`

	m["fyne.do"] = `fyne.do f    run Goal function f on the Fyne main event thread (safe from goroutines)`

	m["fyne.table"] = `fyne.table[rows;cols;cellFn;headerFn]    Table widget
  rows, cols  – integers
  cellFn      – f called with (row;col) → fyne.widget cell content
  headerFn    – f called with col → fyne.widget header (or 0 for none)`

	m["fyne.showinfo"] = `fyne.showinfo["title"; ("msg";win)]    show information dialog on window win`

	m["fyne.showerr"] = `fyne.showerr["title"; ("msg";win)]    show error dialog on window win`

	m["fyne.confirm"] = `fyne.confirm["title"; ("msg";f;win)]    show confirm dialog; f called with 1i (yes) or 0i (no)`

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

Extensions:
"fyne"      Fyne GUI extension (fyne.app, fyne.window, fyne.button, …)
"http"      HTTP client extension (http.get, http.post, http.client, …)
"sql"       SQL extension (sql.open, sql.q, sql.exec, sql.tx)
"ratelimit" rate limiter extension (ratelimit.new, ratelimit.take)
verb        where verb is an extension verb (like "fyne.app" or "http.get")

Notations:
        i (integer) n (number) s (string) r (regexp)
        d (dict) t (dict S!Y) h (handle) e (error)
        f (function) F (dyadic function)
        x,y,z (any other) I,N,S,X,Y,A (arrays)
`

const helpFyne = `FYNE GUI VERBS HELP
Types: fyne.app (application), fyne.window (window), fyne.widget (canvas object)

Application and window:
fyne.app s                    create app; s is an app-ID string (or 0 for no ID)
fyne.window["t"; a]           create window with title t from fyne.app a
fyne.window a                 create window (no title) from fyne.app a
fyne.run w                    ShowAndRun window w (blocks until closed)
fyne.setcontent[w; c]         set window content; returns w
fyne.settitle[w; s]           set window title; returns w
fyne.resize[w; (width;height)] resize window to float dimensions; returns w
fyne.title w                  get title string of window w

Basic widgets (all return fyne.widget):
fyne.label s            Label widget
fyne.entry s            Entry widget (s is placeholder; "" for none)
fyne.password s         password Entry widget
fyne.multiline s        multiline Entry widget
fyne.progress n         ProgressBar at value n (0.0..1.0)
fyne.separator 0        horizontal Separator
fyne.spacer 0           layout Spacer (expands in box/toolbar layouts)

Interactive widgets (callbacks receive one Goal value):
fyne.button["l"; f]          Button (f called with 0i on tap)
fyne.check["l"; f]           Check (f called with 1i checked / 0i unchecked)
fyne.slider[(lo;hi); f]      Slider (f called with current float value)
fyne.select[opts; f]         Select (f called with selected string; opts is AS)

Accessing and updating widget state:
fyne.text w                  get text string from Label, Entry, or Select
fyne.settext[w; s]           set text of Label, Entry, or Button; returns w
fyne.value w                 get value from Slider/ProgressBar (n) or Check (i)
fyne.setvalue[w; n]          set value of Slider or ProgressBar; returns w
fyne.enable w                enable widget; returns w
fyne.disable w               disable widget; returns w
fyne.show w                  show widget; returns w
fyne.hide w                  hide widget; returns w
fyne.refresh w               redraw widget; returns w

Containers and layout:
fyne.vbox widgets                   VBox container from array of fyne.widget
fyne.hbox widgets                   HBox container from array of fyne.widget
fyne.scroll w                       ScrollContainer wrapping w
fyne.padded w                       Padded container wrapping w
fyne.center w                       Center container wrapping w
fyne.split["h"; (w1;w2)]            HSplit container
fyne.split["v"; (w1;w2)]            VSplit container
fyne.border[t;b;l;r;…]             Border container (pass 0 for unused slots)
fyne.tabs[("l";w);…]               AppTabs (one tab per label/widget pair)
fyne.form[("l";w);…]               Form widget (one labelled row per pair)
fyne.toolbar[act;…]                Toolbar from fyne.action values
fyne.action[icon;f]                ToolbarAction (icon is a Fyne Resource widget)
fyne.table[rows;cols;cellFn;headerFn]  Table widget

Threading:
fyne.do f               run f on the Fyne main event thread (safe from goroutines)

Dialogs:
fyne.showinfo["t"; ("msg";w)]     information dialog on window w
fyne.showerr["t"; ("msg";w)]      error dialog on window w
fyne.confirm["t"; ("msg";f;w)]    confirm dialog (f called with 1i/0i)
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
