# v "next"

- `pp.tbl` and `pp.dict` which invoke `fmt.tbl` and `fmt.dict` with default values (all rows/columns; `"%.2f"` float format)
- `time.utc` to set location of a time value to UTC
- `time.format` to format a time with a given format. Format constants from Go's `time` package are already defined.
- Bug Fix: Raw REPL would not properly ignore comment content. Fix ported from anaseto's fix in the Goal repo [here](https://codeberg.org/anaseto/goal/commit/ec3e8a97179fd6ff8bfe035504cf0a9b506312c).

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
