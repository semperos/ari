# Ari

Ari stands for **A**rray **R**elational **I**nteractive programming environment.

Ari takes the [Goal] programming language, wraps it in a custom CLI and provides new functions for the features listed below.

## Features

- [Goal] is the core language
- Extensible CLI REPL with:
  - Auto-completion for:
    - Built-in keywords
    - Built-in syntax aliases (e.g., typing "first" and TAB will show `*` and `¿` in auto-complete results)
    - User-defined globals
  - Runtime configuration:
    - Configure the REPL prompt by setting string values for the `ari.prompt` and `ari.nextprompt` (for multiline input) globals
    - Replace default REPL printing by setting a function value for the `ari.print` global (function receives a single Goal value to print)
  - `ari.p` as previous result (value from last evaluation)
- Extensible help system
  - `help"help"` for an overview
  - `help"TOPIC"` similar to Goal's CLI help
  - `help rx/search/` to use regular expression matching to find help entries that match
  - `"myvar" help "My Var Help"` to extend the help system with new documentation
  - `help""` returns entire help dictionary (including non-Goal help for the SQL mode; use `(help"")"goal"` for just the Goal help)
  - Auto-complete shows abbreviated help for each result.
- New Goal functions:
  - `http.` functions for HTTP requests using [Resty]
  - `sql.` functions for SQL queries and commands
- Dedicated SQL mode for DuckDB
  - Activate with `)sql` for read-only, `)sql!` for read/write modes. Execute `)goal` to return to the default Goal mode.
  - Auto-completion of SQL keywords
  - Help entries for SQL keywords (shown during auto-complete, still WIP)
  - Results of last-run query/command set to the `sql.t` Goal global, so you can switch between `)sql` and `)goal` at the REPL to run queries via SQL and do data processing via Goal.

## To Do

Non-exhaustive list:

- TODO: Test coverage.
- TODO: Allow specifying a database as an argument to `)sql` and `)sql!` system commands.
- TODO: Support plots/charts (consider https://github.com/wcharczuk/go-chart)
- TODO: `tui.` functions in CLI mode using https://github.com/charmbracelet/lipgloss (already a transitive dependency) for colored output, etc.

## Ideas

- Support Prolog https://github.com/ichiban/prolog?tab=readme-ov-file
  - River Crossing or Learn Datalog Today as examples
- API Explorer (possibly a project that relies on Ari, but not a part of Ari)
  - Stateful "current working directory" concept within entities of an API
  - Contextual command/function invocations (e.g., "ls" inside an epic lists stories/issues)
  - Formatted table summaries; raw JSON payload details available.
  - Auto-complete based on API schema and entities

## Examples

I am using ari to build an API client environment for [Shortcut](https://shortcut.com). The under-major-construction code for that can be found [in this GitHub Gist](https://gist.github.com/semperos/daba47a3665c89794a3613cfdb0a2d6c).

I am also using ari to build out an alternative to the Julia setup described in the Background section of this README, but that code is not available at this time.

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

Why move away from this slick setup?

- Concision and expressive power of array languages.
- A lot of my code was SQL in Julia.

Ari currently embeds the Goal array programming language. What gaps from my Julia+DuckDB experience need to be filled to use Goal where I used Julia?

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

Goal already has a number of features that mean we don't need to fill these gaps to start (more flexible options may be considered in the future):

- Powerful string API
- JSON support
- CSV support

<!-- Links -->

[Goal]: https://codeberg.org/anaseto/goal
[Resty]: https://github.com/go-resty/resty
