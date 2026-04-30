// driver_sqlite.go registers the modernc.org/sqlite driver under the "sqlite"
// URI scheme. The blank import triggers the driver's init() function, which
// calls database/sql.Register("sqlite", ...).
//
// To add another backend, create a similar file (e.g. driver_duckdb.go) and
// append its scheme/driver-name pair to driverSchemes in sql.go.

package sql

import _ "modernc.org/sqlite" // registers the sqlite driver via its init() function
