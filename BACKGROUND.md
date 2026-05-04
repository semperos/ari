# Ari Background

In 2024 I stumbled into a flexible, powerful setup using Julia and DuckDB to do data analysis:

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
- A lot of my code was SQL via Julia.

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


