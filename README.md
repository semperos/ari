# Ari

Ari stands for **A**rray **R**elational **I**nteractive programming environment.

Ari is a set of extensions to the [Goal] programming language with an extensible CLI and dedicated SQL mode.

## Installation

```shell
go install github.com/semperos/ari/cmd/ari@latest
```

Then run `ari` for a REPL or `ari --help` to see CLI options.

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
  - `sql.` functions for SQL queries and commands
  - `tt.` test framework
  - `csv.tbl` and `json.tbl` to make Goal tables from the output of `csv` and `json` respectively
  - `time.` functions for more extensive date/time handling
  - `tui.` functions for basic terminal UI styling (colors, padding/margin, borders)
- Dedicated SQL mode
  - The ari CLI uses DuckDB, but the `github.com/semperos/ari` Go package doesn't directly depend on a specific SQL database driver, so you can BYODB.
  - Activate with `)sql` for read-only, `)sql!` for read/write modes. Execute `)goal` to return to the default Goal mode.
  - Auto-completion of SQL keywords
  - Help entries for SQL keywords (shown during auto-complete, still WIP)
  - Results of the last-run query/command set to the `sql.p` Goal global (for "SQL previous" and mirroring `ari.p`), so you can switch between `)sql` and `)goal` at the REPL to run queries via SQL and do data processing via Goal.

## To Do

See [CHANGES.md](CHANGES.md) for recent changes.

Non-exhaustive list:

- IN PROGRESS: Test coverage
- IN PROGRESS: Consider which of ari's functions should be pervasive; review Goal's approach and implement
- TODO: Consider which of ari's pervasive functions should operate on dictionaries
- TODO: BUG When using `-l` flag, loaded files aren't completely evaluated. Evaluation stops partway through and it's not immediately clear why.
- TODO: BUG `json.tbl` produces improper zero values when keys are missing. Where possible, column lists should be of uniform type and not generic if the data allows for it.
- TODO: Make version of `fmt.tbl` that allows setting maximum column width.
- TODO: Functions to conveniently populate SQL tables with Goal values.
- TODO: User commands (as found in [APL](https://aplwiki.com/wiki/User_command)), executable from Goal or SQL modes

I plan to support the above items. The following are stretch goals, nice-to-have's, or thoughts for further consideration:

- TODO: Use custom table functions via replacement scan to query Goal tables from DuckDB.
- TODO: Looser auto-complete, not just prefix-based
- TODO: `)help` for cross-mode help
- TODO: System functions to switch between rich and raw REPL `)repl.rich` and `)repl.raw`
- IN PROGRESS: `tui.` functions in CLI mode using https://github.com/charmbracelet/lipgloss (already a transitive dependency) for colored output, etc.
- TODO: Implement a subset of [q](https://code.kx.com/q/) functions to extend what Goal already has.
- Specific user commands:
  - TODO: Toggle pretty-printing (TBD what this means; can use JSON output at this point for possibly more familiar output)
  - TODO: Toggle paging at the REPL (as found in [PicoLisp](https://picolisp.com/wiki/?home))
  - TODO: Toggle colored output

## Examples

I began building Ari to replicate the experience described in the [Background](#background) section of this README. That code is not publicly available at this time.

I am also using Ari to build an API client environment for the [Shortcut](https://shortcut.com) [REST API](https://developer.shortcut.com/api/rest/v3). The under-major-construction code for that can be found [in this GitHub Gist](https://gist.github.com/semperos/daba47a3665c89794a3613cfdb0a2d6c).

## Development

Ari is implemented in Go and Goal. See the `script` folder for common development operations.

To publish a new version of Ari:

```shell
./script/release vx.y.z
```

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
