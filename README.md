# Ari

Ari stands for **A**rray **R**elational **I**nteractive programming environment.

Ari is a set of extensions to the [Goal] programming language with an extensible CLI and dedicated SQL mode.

## Installation

```shell
go install github.com/semperos/ari/cmd/ari@latest
```

Then run `ari` for a REPL or `ari --help` to see CLI options.

```
ari is an interactive environment for array + relational programming.

It embeds the Goal array programming language, with extensions for
working with SQL and HTTP APIs.

Usage:
  ari [flags] [source file]

Flags:
      --config string          ari configuration (default "$HOME/.config/ari/ari-config.yaml")
      --cpu-profile            write CPU profile to file
  -d, --database string        DuckDB database (default: in-memory)
      --debug                  enable detailed debugging output on panic
  -e, --execute string         string of Goal code to execute, last result not printed automatically
  -h, --help                   help for ari
      --history string         history of REPL entries (default "$HOME/.config/ari/ari-history.txt")
  -l, --load stringArray       Goal source files to load on startup
  -m, --mode string            language mode at startup (default "goal")
  -f, --output-format string   evaluation output format (default "goal")
  -p, --println                print final value of the script + newline
  -r, --raw                    raw REPL w/out history or auto-complete
  -v, --version                print version info and exit
```

## Features

- [Goal] is the core language
  - Goal's `lib` files are loaded by default, with prefix matching their file names (see [vendor-goal](vendor-goal) folder in this repo)
- Extensible CLI REPL with:
  - Auto-completion with documentation for:
    - Built-in keywords
    - Built-in syntax aliases (e.g., typing "first" and TAB will show `*` and `¿` in auto-complete results)
    - User-defined globals
  - Runtime configuration:
    - Configure the REPL prompt by setting string values for the `ari.prompt` and `ari.nextprompt` (for multiline input) globals
    - Replace default REPL printing by setting a function value for the `ari.print` global (function receives a single Goal value to print)
    - Configure the output format with `--output-format` or using one of the `)output.` system commands at the REPL. Formats include CSV/TSV, JSON, Markdown, and LaTeX.
  - `ari.p` is bound to the previous result (value from last evaluation at the REPL)
  - Alternatively run `ari` with `--raw` for a simpler, raw REPL that lacks line editing, history, and auto-complete, but is better suited for interaction via an editor like (Neo)Vim, or if you prefer rlwrap or another line editor to the one that ships with ari.
  - `help` based on Goal's, but allows adding help strings when used dyadically (e.g.,`"sql.q"help"Run SQL query"`)
- New Goal functions:
  - `http.` functions for HTTP requests using [Resty]
  - `ratelimit.new` and `ratelimit.take` for rate limiting (leaky bucket algorithm) using [uber-go/ratelimit]
  - `sql.` functions for SQL queries and commands
  - Table-related `csv.tbl` and `json.tbl` to make Goal tables from the output of `csv` and `json` respectively
  - `tt.` test framework
  - `time.` functions for more extensive date/time handling
  - `tui.` functions for basic terminal UI styling (colors, padding/margin, borders)
- Dedicated SQL mode
  - The ari CLI uses DuckDB, but the `github.com/semperos/ari` Go package doesn't directly depend on a specific SQL database driver, so you can BYODB.
  - Activate with `)sql` for read-only, `)sql!` for read/write modes. Execute `)goal` to return to the default Goal mode.
  - Auto-completion of SQL keywords
  - Help entries for SQL keywords (shown during auto-complete, still WIP)
  - Results of the last-run query/command set to the `sql.p` Goal global (for "SQL previous" and mirroring `ari.p`), so you can switch between `)sql` and `)goal` at the REPL to run queries via SQL and do data processing via Goal.

## Projects Using Ari

- [Shortcut API Client](https://github.com/semperos/sc-client-goal)

I began building Ari to replicate the experience described in the [Background](#background) section of this README. That code is not publicly available at this time.

## Examples

### Minimal HTTP Server

```
"localhost:1234"http.serve{say x; ..[status:200;bodystring:"OK"]}
```

### Slack API

Use Slack's API to get all messages in a channel for last 30 days. The first five lines set up the HTTP client specifically for Slack; the second block shows a code comment which can be evaluated to see a list of channels from which to pick the ID you need; the third block of code sets up the starting and ending timestamps with which to call the Slack API and a recursive function to fetch messages; the final line defines an `allmsgs` global with all messages from the given channel for the given time range, and then returns `"ok"` when it completes.

```
url:"https://slack.com/api/"; aj:"application/json"; tk: 'env"SLACK_USER_TOKEN"
hd:..[Accept:aj;"Content-Type":aj;"Authorization":"Bearer $tk"]; hc:http.client[..[Header:hd]]
hp:{[f]{[httpf;path]r: 'httpf[hc;url+path]; 'json r"bodystring"}[f]}
hpp:{[f]{[httpf;path;reqopts]r: 'httpf[hc;url+path;reqopts]; 'json r"bodystring"}[f]}
get:hp[http.get]; getq:hpp[http.get]; post:hpp[http.post]

/ convos:get"conversations.list"; chans:convos"channels"; ^(..name,id)'chans
channel:"<ID HERE>"

day:time.Hour * 24; unow:time.utc time.now@0
oldest:time.unix[time.add[unow;-30 * day]]
latest:time.unix[unow]
msgs:{[acc;channel;oldest;latest] \latest
  hist:post["conversations.history";..[Body:""json..[channel;oldest;latest]]]
  ms:hist"messages"; newlatest:(..ts)@*|ms; acc,:ms
  ?[hist"has_more"
    o[acc;channel;oldest;newlatest]
    acc]}

allmsgs:msgs[();channel;oldest;latest]; "ok"
```

## Development

Ari is implemented in Go and Goal. See the `script` folder for common development operations.

To publish a new version of Ari:

```shell
./script/release vx.y.z
```

### WASM

Use the `./script/build-wasm` script to generate a `./cmd/wasm/goal.wasm` file from the `./cmd/wasm/main.go` entry-point.

Run an HTTP server in the `./cmd/wasm` folder to serve up the `index.html`, which renders a gently adapted version of anaseto's WASM setup in Goal itself.

NB: The JavaScript that controls the user interface in `./cmd/wasm/index.html` is written in Go in the `./cmd/wasm/main.go` file.

See [Go Wiki: WebAssembly](https://go.dev/wiki/WebAssembly) for more information.

## Background

I stumbled into a fairly flexible, powerful setup using Julia and DuckDB to do data analysis.

Details:

- Julia as primary programming language
- Julia Pluto Notebooks as primary programming environment
- Notebook One: Data ETL
  - HTTP: Calling HTTP APIs to fetch JSON data (GitHub, Shortcut)
  - CSV: Transforming fetched JSON data into CSV
  - SQL: (Out of band) Defining SQL tables using SQL schema
  - SQL: (Out of band) Importing CSV into DuckDB using a schema defined in a SQL file
- Notebook Two: Data Analyses
  - SQL: DuckDB tables as source of truth
  - Julia: Wrote trivial utility fn to transform arbitrary SQL query results into Julia DataFrames
    - `DataFrame(DBInterface.execute(conn, sql))`
    - Renders as a well-formatted table in the notebook interface
    - Array-language-like story for interacting with the DataFrame
    - Fully dynamic: type information from the database schema used to dynamically populate the DataFrame with data of appropriate types.
  - Julia: Mustache package for templating functions to build a LaTeX report to house all analyses
  - Julia: Plots package for generating plots as PDF to insert into final LaTeX report
  - Julia: Statistics, StatsBase packages both for aggregates and data to plot, pull from DataFrames
  - Julia: Executed external `latexmk` using Julia's `run` to build LaTeX report

Why move away from this setup?

- Concision and expressive power of array languages.
- A lot of my code was SQL in Julia.

Ari embeds the Goal array programming language. What gaps from my Julia+DuckDB experience need to be filled to use Goal where I used Julia?

- Notebook programming environment
  - Cell dependencies and automatic re-run
    - Nice-to-have
  - Autocomplete
    - Required
  - Built-in language/library help documentation
    - Required
  - Rich rendering
    - Tables: Required
    - Graphics: Nice-to-have
- HTTP Client
  - Nice-to-have (can rely on shelling out to `curl` to start).
- SQL
  - Query results as Goal tables/dictionaries/arrays
    - Required
- Plots
  - Declarative plot definition producing one of PNG/PDF or HTML/JS output
    - Required
    - q tutorials simply leverage JS bridge + Highcharts
    - Julia Plots package has pluggable backends:
      - GR (cross-platform, has C bindings)
      - Plotly (JS)
      - gnuplot (command-line utility)
      - HDF5 file format
    - See GNU Octave plotting
- See [Gonum](https://github.com/gonum)

Goal already has a number of features, so we don't need to fill these gaps to start (more flexible options may be considered in the future):

- Powerful string API
- JSON support
- CSV support

## License

### Ari

Original software in this repository is licensed as follows:

Copyright 2024 Daniel Gregoire

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

### Goal

Source code copied and/or adapted from the [Goal] project has the following license:

Copyright (c) 2022 Yon <anaseto@bardinflor.perso.aquilenet.fr>

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

<!-- Links -->

[Goal]: https://codeberg.org/anaseto/goal
[Resty]: https://github.com/go-resty/resty
[uber-go/ratelimit]: https://github.com/uber-go/ratelimit
