# Ari

Ari stands for **A**rray **R**elational **I**nteractive programming environment.

Ari is a set of extensions to the [Goal] programming language that includes SQL support (SQLite and DuckDB) and an HTTP client with rate limiting support.

This is my personal daily driver for scripting and data analysis, even in the age of coding agents. Parts of this code base have been developed using coding agents and LLMs.

## Installation

```shell
go install github.com/semperos/ari/cmd/ari@latest
```

## Practical Usage

- [Examples in this repo](examples)
- [Shortcut API Client](https://github.com/semperos/sc-client-goal)
- [Personal Anthology](https://goalprogramming.info/personal-anthology.html)

## Development

Ari is implemented in Go and Goal. See the `scripts` folder for common development operations.

To publish a new version of Ari:

```shell
./scripts/release vx.y.z
```

## SQL

The `sql` package exposes a uniform set of verbs for any supported database. Open a connection with a URI whose scheme selects the driver:

```
db: sql.open "sqlite://:memory:"   / SQLite in-memory (pure Go, no CGo)
db: sql.open "sqlite://data.db"    / SQLite file
db: sql.open "duckdb://"           / DuckDB in-memory
db: sql.open "duckdb:///data.db"   / DuckDB file
```

| Verb | Form | Description |
|---|---|---|
| `sql.open` | `sql.open uri` | Open a connection; returns `sql.conn` |
| `sql.close` | `sql.close db` | Close a connection; returns `1i` |
| `sql.q` | `db sql.q "SELECT ..."` | Query; returns columnar dict |
| `sql.q` | `sql.q[db; "SELECT ... WHERE x=?"; args]` | Parameterised query |
| `sql.exec` | `db sql.exec "INSERT ..."` | Execute statement; returns exec dict |
| `sql.exec` | `sql.exec[db; "INSERT ... VALUES(?)"; args]` | Parameterised exec |
| `sql.tx` | `db sql.tx {[tx] ...}` | Lambda-scoped transaction |

Query results are columnar dicts mapping column name strings to typed arrays (`AI`, `AF`, `AS`, or `AV`). SQL `NULL` maps to Goal's `0n` (float NaN).

> **Note:** DuckDB requires CGo. SQLite uses a pure-Go implementation ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)) and has no CGo dependency.

## Background

In 2024 I stumbled into a flexible, powerful setup using Julia and DuckDB to do data analysis. Read [BACKGROUND.md](BACKGROUND.md) for more details.

## License

### Ari

Original software in this repository is licensed as follows:

Copyright 2024–2026 Daniel Gregoire

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

### Rye

Source code copied and/or adapted from the [Rye] project has the following license:

Copyright 2019-2024 Janko Metelko

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

<!-- Links -->

[Goal]: https://codeberg.org/anaseto/goal
[Rye]: https://github.com/refaktor/rye
