// driver_duckdb.go registers the github.com/marcboeker/go-duckdb driver under
// the "duckdb" URI scheme. The blank import triggers the driver's init()
// function, which calls database/sql.Register("duckdb", ...).
//
// Usage:
//
//	db: sql.open "duckdb://"          – in-memory database
//	db: sql.open "duckdb:///data.db"  – file-based database

package sql

import _ "github.com/marcboeker/go-duckdb" // registers the duckdb driver via its init() function
