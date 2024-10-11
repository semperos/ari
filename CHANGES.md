# v0.1.2 2024-10-11

- Removed Ari's `glob` and `abspath` now that Goal has them.
- Made `help` use Goal's updated help, while also adding a dyadic arity that allows adding/overriding help entries in Goal code.
- Added `rtnames` which is a dictionary, the values of which are all globals, keywords, and syntax characters defined in the Goal environment
- Added `ac` function (mnemonic "auto-complete") which relies on `rtnames` to match names based on glob (if arg is string) or regex (if arg is regex).
- `pp.tbl` and `pp.dict` which invoke `fmt.tbl` and `fmt.dict` with default values (all rows/columns; `"%.2f"` float format)
- `time.utc` to set location of a time value to UTC
- `time.format` to format a time with a given format. Format constants from Go's `time` package are already defined.
- `time.date` to create a time object given the year, month, day, hour, minute, second, nanosecond, and location. Accepts 1 to 8 arguments, defaulting to 0 values and UTC for those omitted.
- `time.*` functions to extract parts of a time object
- `time.loadlocation` to create a location object given a canonical IANA name (and given the system has IANA data)
- `time.fixedzone` to create an ad hoc location object given a name and offset-in-seconds-from-UTC
- `time.locationstring` to get a string representation of a location object
- `url.encode` monad to escape either a path (if arg is string) or query parameters (if arg is dictionary)
- `http.serve` dyad to run an HTTP server. See implementation for details.
- Bug Fix: Raw REPL would not properly ignore comment content. Fix ported from anaseto's fix in the Goal repo [here](https://codeberg.org/anaseto/goal/commit/ec3e8a97179fd6ff8bfe035504cf0a9b506312c).
- Upgrades to go-resty and go-duckdb dependencies.

# v0.1.1 2024-09-13

- Goal version from commit [fa948e4ad6cb7c9d5d8d1b0d6b95e5128d600087](https://codeberg.org/anaseto/goal/commit/fa948e4ad6cb7c9d5d8d1b0d6b95e5128d600087)
- Test framework. See usage in the `testing/` folder and the test scripts in the `script/` folder.
- Raw REPL. The default REPL provides a rich editing experience with history, auto-complete, etc.
  However, that rich REPL is not ideal for evaluating code that is sent from an editor to the REPL,
  since it is both slow and doesn't allow for more than one complete multi-line expression at a time.
  Running with `ari -r` provides a much simpler REPL similar to Goal's which is more performant and
  allows for an arbitrary amount of input.
- Output format supported in Goal and SQL modes. See `ari --help` or `)output.` at the REPL for options.
  Initial output formats supported are CSV/TSV, Markdown, LaTeX, and JSON (both compact and indented).
- `glob` and `abspath` functions, which wrap Go's `path/filepath.Glob` and `path/filepath.Abs`
  functions respectively.
- `FILE` is bound to the absolute path to the current source file when invoking `ari` for one-off execution of a script.
  TBD whether this will remain absolute path or be changed to relative.
- `csv.tbl` and `json.tbl` functions for creating Goal tables from output of `csv` and `json` respectively.
  JSON payload is expected to be a JSON array of JSON objects with common keys.
- `tui.style`, `tui.color`, and `tui.render` functions for creating colorized/stylized output at the terminal.

# v0.1.0 2024-08-26

- Initial release
- Goal language with HTTP, SQL, and nascent date/time extensions.
- Custom CLI with auto-completion and feature-rich line editing with bubbline library.
- Starting to use ari for load-bearing scripting and data analysis, so it's time to cut an initial release.
