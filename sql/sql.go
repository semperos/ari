// Package sql provides Goal bindings for SQL databases.
//
// Import registers the sql.* verbs into a Goal context. Call it as:
//
//	sql.Import(ctx, "")
//
// which registers globals prefixed with "sql." (e.g. sql.open, sql.q).
//
// # Connection model
//
// Connections are opened by URI and stored as sql.conn boxed values. The URI
// scheme selects the driver; the remainder is the data source name (DSN):
//
//	db: sql.open "sqlite://data.db"
//	db: sql.open "sqlite://:memory:"
//
// # Verb summary
//
// Monads:
//
//	sql.open  "scheme://dsn"  – open a connection; returns sql.conn or error
//	sql.close db              – close a connection; returns 1i or error
//
// Dyads:
//
//	db sql.q    "SELECT ..."              – query; returns columnar dict
//	db sql.q   ["SELECT ... WHERE x=?"; args]  – parameterised query
//	db sql.exec "INSERT ..."             – execute statement; returns exec dict
//	db sql.exec["INSERT ... VALUES(?)"; args]  – parameterised exec
//	db sql.tx  {[tx] ... }              – lambda-scoped transaction
//
// # QueryResult dict
//
// sql.q returns a dict mapping column name strings (AS) to per-column arrays:
//
//	t"name"        – column array (AI | AF | AS | AV)
//	#t"name"       – row count
//	nan t"age"     – boolean array; 1 at each NULL position
//
// Column arrays are specialised:
//   - All integers, no NULLs        → AI
//   - All floats, no NULLs          → AF
//   - All strings, no NULLs         → AS
//   - Integers + NULLs              → AF (Goal normalises I+NaN → AF)
//   - Floats + NULLs                → AF (NaN stays in AF)
//   - Strings + NULLs               → AV (S+NaN cannot be further normalised)
//   - Mixed types, BLOBs, or others → AV (NULL slots hold 0n in all cases)
//
// # ExecResult dict
//
// sql.exec returns a dict with two integer keys:
//
//	r"lastInsertId"  – row ID of most recent INSERT, or 0
//	r"rowsAffected"  – rows changed by the statement, or 0
//
// # NULL handling
//
// SQL NULL maps to Goal's float NaN (0n), the universal null marker.
// Any column containing at least one NULL is returned as AV.
//
//	nan col          – boolean array; 1 at each NULL (0n) position
//	0 nan col        – fill NULLs with 0
//	"" nan col       – fill NULLs with ""
//
// # Type mapping
//
//	SQL NULL              → 0n  (float NaN)
//	SQL INTEGER / int64   → I   (int64)
//	SQL INTEGER / int32   → I   (int64, widened)
//	SQL REAL / FLOAT      → F   (float64)
//	SQL TEXT              → S   (string)
//	SQL BLOB              → AB  (byte array)
//	SQL BOOLEAN true      → I 1
//	SQL BOOLEAN false     → I 0
//	time.Time             → I   (Unix microseconds since 1970-01-01 UTC)
//
// # Transactions
//
// sql.tx runs a lambda with a transaction object. The transaction commits if
// the lambda returns a non-error value; it rolls back otherwise:
//
//	r: db sql.tx {[tx]
//	    tx sql.exec["INSERT INTO t (x) VALUES (?)" ; ,42]
//	}
//
// The tx value passed to the lambda accepts sql.q and sql.exec identically to
// a sql.conn. Nested transactions are not supported.
//
// # Registered drivers
//
// The sqlite URI scheme is registered by importing this package (via the
// blank import of modernc.org/sqlite in the driver file). Adding a backend
// requires registering its Go database/sql driver and mapping its URI scheme
// in the open verb; no other verb changes.
package sql

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	goal "codeberg.org/anaseto/goal"
)

// ---------------------------------------------------------------------------
// BV wrapper types
// ---------------------------------------------------------------------------

// Conn wraps a *sql.DB as a Goal boxed value (sql.conn).
type Conn struct {
	db     *stdsql.DB
	driver string
	dsn    string
	closed bool
}

func (c *Conn) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("sql.conn[%s:%s]", c.driver, c.dsn)...)
}
func (c *Conn) Matches(y goal.BV) bool { yv, ok := y.(*Conn); return ok && c == yv }
func (c *Conn) Type() string           { return "sql.conn" }

// GoalTx wraps a *sql.Tx as a Goal boxed value (sql.tx).
type GoalTx struct {
	tx   *stdsql.Tx
	done bool
}

func (t *GoalTx) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, "sql.tx"...)
}
func (t *GoalTx) Matches(y goal.BV) bool { yv, ok := y.(*GoalTx); return ok && t == yv }
func (t *GoalTx) Type() string           { return "sql.tx" }

// ---------------------------------------------------------------------------
// querier: common interface for *sql.DB and *sql.Tx
// ---------------------------------------------------------------------------

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*stdsql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (stdsql.Result, error)
}

// toQuerier extracts a querier and reports whether the underlying conn is open.
// Returns (nil, "", false) if v is not a sql.conn or sql.tx.
func toQuerier(v goal.V) (querier, string, bool) {
	switch bv := v.BV().(type) {
	case *Conn:
		return bv.db, "sql.conn", !bv.closed
	case *GoalTx:
		return bv.tx, "sql.tx", !bv.done
	}
	return nil, "", false
}

// ---------------------------------------------------------------------------
// URI parsing
// ---------------------------------------------------------------------------

// driverSchemes maps URI schemes to database/sql driver names.
// New backends register here; no verb logic changes.
var driverSchemes = map[string]string{ //nolint:gochecknoglobals // package-level registry initialised once at startup
	"sqlite": "sqlite",
}

// parseURI splits "scheme://dsn" into (scheme, dsn).
// For URIs without "://", the whole string is treated as a file path with
// scheme "sqlite" for backwards compatibility.
func parseURI(uri string) (string, string, error) {
	idx := strings.Index(uri, "://")
	if idx < 0 {
		return "", "", fmt.Errorf("sql.open: URI %q missing scheme (expected e.g. \"sqlite://data.db\")", uri)
	}
	scheme := uri[:idx]
	dsn := uri[idx+3:]
	return scheme, dsn, nil
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

// Import registers all sql.* verbs into ctx. When pfx is empty the globals
// are named "sql.open", "sql.q", etc. When pfx is non-empty it is prepended
// with a dot, so Import(ctx,"db") yields "db.sql.open", etc.
func Import(ctx *goal.Context, pfx string) {
	ctx.RegisterExtension("sql", "")
	if pfx != "" {
		pfx += "."
	}

	reg := func(name string, f goal.VariadicFunc, dyad bool) {
		fullname := pfx + name
		var v goal.V
		if dyad {
			v = ctx.RegisterDyad("."+fullname, f)
		} else {
			v = ctx.RegisterMonad("."+fullname, f)
		}
		ctx.AssignGlobal(fullname, v)
	}

	// monads
	reg("sql.open", vfOpen, false)
	reg("sql.close", vfClose, false)

	// dyads (also accept bracket notation with extra args)
	reg("sql.q", vfQuery, true)
	reg("sql.exec", vfExec, true)
	reg("sql.tx", wrapCtxTx(ctx, vfTx), true)
}

// wrapCtxTx injects the Goal context into sql.tx's closure (needed to call
// the user-supplied lambda).
func wrapCtxTx(ctx *goal.Context, f func(*goal.Context, []goal.V) goal.V) goal.VariadicFunc {
	return func(_ *goal.Context, args []goal.V) goal.V { return f(ctx, args) }
}

// ---------------------------------------------------------------------------
// sql.open  (monad: sql.open "scheme://dsn")
// ---------------------------------------------------------------------------

// vfOpen opens a database connection from a URI.
//
// Usage:
//
//	db: sql.open "sqlite://data.db"
//	db: sql.open "sqlite://:memory:"
func vfOpen(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("sql.open uri : expected 1 argument, got %d", len(args))
	}
	s, ok := args[0].BV().(goal.S)
	if !ok {
		return goal.Panicf("sql.open uri : expected string URI, got %q", args[0].Type())
	}
	uri := string(s)

	scheme, dsn, err := parseURI(uri)
	if err != nil {
		return goal.Panicf("%v", err)
	}

	driverName, ok := driverSchemes[scheme]
	if !ok {
		known := make([]string, 0, len(driverSchemes))
		for k := range driverSchemes {
			known = append(known, k)
		}
		return goal.Panicf("sql.open: unknown URI scheme %q (registered: %s)", scheme, strings.Join(known, ", "))
	}

	db, err := stdsql.Open(driverName, dsn)
	if err != nil {
		return goal.Panicf("sql.open %q: %v", uri, err)
	}
	// Ping to surface connection errors immediately.
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return goal.Panicf("sql.open %q: %v", uri, err)
	}

	return goal.NewV(&Conn{db: db, driver: scheme, dsn: dsn})
}

// ---------------------------------------------------------------------------
// sql.close  (monad: sql.close db)
// ---------------------------------------------------------------------------

// vfClose closes a database connection.
//
// Usage:
//
//	sql.close db
func vfClose(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("sql.close conn : expected 1 argument, got %d", len(args))
	}
	c, ok := args[0].BV().(*Conn)
	if !ok {
		return goal.Panicf("sql.close conn : expected sql.conn, got %q", args[0].Type())
	}
	if c.closed {
		return goal.Panicf("sql.close conn : connection is already closed")
	}
	if err := c.db.Close(); err != nil {
		return goal.Panicf("sql.close conn : %v", err)
	}
	c.closed = true
	return goal.NewI(1)
}

// ---------------------------------------------------------------------------
// sql.q  (dyad: conn sql.q "query"  or  conn sql.q["query";args])
// ---------------------------------------------------------------------------

// vfQuery executes a SELECT and returns a columnar dict.
//
// Dyadic (no parameters):
//
//	t: db sql.q "SELECT * FROM users"
//
// Bracket (parameterised):
//
//	t: db sql.q["SELECT * FROM users WHERE age > ?" ; ,25]
//	t: db sql.q["SELECT * FROM t WHERE a=? AND b=?" ; (1;"x")]
//
// conn accepts either sql.conn or sql.tx.
func vfQuery(_ *goal.Context, args []goal.V) goal.V {
	conn, query, params, err := parseConnQueryArgs("sql.q", args)
	if err != nil {
		return goal.Panicf("%v", err)
	}
	sqlArgs, err := goalToSQLArgs(params)
	if err != nil {
		return goal.Panicf("sql.q: %v", err)
	}

	rows, err := conn.QueryContext(context.Background(), query, sqlArgs...)
	if err != nil {
		return goal.Panicf("sql.q %q: %v", query, err)
	}
	defer rows.Close()

	result, err := scanRows(rows)
	if err != nil {
		return goal.Panicf("sql.q %q: %v", query, err)
	}
	return result
}

// ---------------------------------------------------------------------------
// sql.exec  (dyad: conn sql.exec "stmt"  or  conn sql.exec["stmt";args])
// ---------------------------------------------------------------------------

// vfExec executes a non-SELECT statement and returns an exec-result dict.
//
// Dyadic (no parameters):
//
//	r: db sql.exec "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)"
//
// Bracket (parameterised):
//
//	r: db sql.exec["INSERT INTO users (name, age) VALUES (?, ?)" ; ("Alice";30)]
//
// conn accepts either sql.conn or sql.tx.
func vfExec(_ *goal.Context, args []goal.V) goal.V {
	conn, query, params, err := parseConnQueryArgs("sql.exec", args)
	if err != nil {
		return goal.Panicf("%v", err)
	}
	sqlArgs, err := goalToSQLArgs(params)
	if err != nil {
		return goal.Panicf("sql.exec: %v", err)
	}

	res, err := conn.ExecContext(context.Background(), query, sqlArgs...)
	if err != nil {
		return goal.Panicf("sql.exec %q: %v", query, err)
	}

	lastID, _ := res.LastInsertId()
	rowsAff, _ := res.RowsAffected()

	keys := goal.NewAS([]string{"lastInsertId", "rowsAffected"})
	vals := goal.NewAI([]int64{lastID, rowsAff})
	return goal.NewD(keys, vals)
}

// ---------------------------------------------------------------------------
// sql.tx  (dyad: conn sql.tx {[tx] ...})
// ---------------------------------------------------------------------------

// vfTx runs a lambda inside a database transaction.
//
// The lambda receives a sql.tx value that accepts sql.q and sql.exec
// identically to a sql.conn. The transaction commits if the lambda returns a
// non-error Goal value and rolls back if it returns a Goal error value.
//
// Usage:
//
//	r: db sql.tx {[tx]
//	    tx sql.exec["INSERT INTO orders (item) VALUES (?)" ; ,"widget"]
//	}
func vfTx(ctx *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("conn sql.tx fn : expected 2 arguments, got %d", len(args))
	}
	// args[0] = fn (right), args[1] = conn (left)
	fn := args[0]
	if !fn.IsFunction() {
		return goal.Panicf("conn sql.tx fn : expected function in right arg, got %q", fn.Type())
	}

	c, ok := args[1].BV().(*Conn)
	if !ok {
		if _, isT := args[1].BV().(*GoalTx); isT {
			return goal.Panicf("conn sql.tx fn : nested transactions are not supported")
		}
		return goal.Panicf("conn sql.tx fn : expected sql.conn in left arg, got %q", args[1].Type())
	}
	if c.closed {
		return goal.Panicf("conn sql.tx fn : connection is closed")
	}

	tx, err := c.db.BeginTx(context.Background(), nil)
	if err != nil {
		return goal.Panicf("conn sql.tx fn : begin: %v", err)
	}

	txVal := goal.NewV(&GoalTx{tx: tx})
	result := fn.ApplyAt(ctx, txVal)

	gtx := txVal.BV().(*GoalTx) //nolint:errcheck // type is guaranteed: txVal was just created as *GoalTx above
	gtx.done = true

	if result.IsPanic() {
		_ = tx.Rollback()
		return result
	}
	if err := tx.Commit(); err != nil {
		return goal.Panicf("conn sql.tx fn : commit: %v", err)
	}
	return result
}

// ---------------------------------------------------------------------------
// Argument parsing helpers
// ---------------------------------------------------------------------------

// parseConnQueryArgs handles the two calling conventions shared by sql.q and
// sql.exec:
//
//	len==2: conn verb "query"                   → (conn, query, noParams)
//	len==3: conn verb["query" ; params]         → (conn, query, params)
func parseConnQueryArgs(verb string, args []goal.V) (querier, string, goal.V, error) {
	var connV, queryV, paramsV goal.V
	switch len(args) {
	case 2:
		// dyadic: conn verb "query"
		// args[0] = query (right), args[1] = conn (left)
		connV = args[1]
		queryV = args[0]
		paramsV = goal.V{} // zero value = no params
	case 3:
		// bracket: conn verb["query" ; params]
		// args[0] = params (last bracket arg)
		// args[1] = query  (first bracket arg)
		// args[2] = conn   (implicit left)
		connV = args[2]
		queryV = args[1]
		paramsV = args[0]
	default:
		return nil, "", goal.V{}, fmt.Errorf("%s : expected 2 or 3 arguments, got %d", verb, len(args))
	}

	q, _, isOpen := toQuerier(connV)
	if q == nil {
		return nil, "", goal.V{}, fmt.Errorf("%s : expected sql.conn or sql.tx in left arg, got %q", verb, connV.Type())
	}
	if !isOpen {
		return nil, "", goal.V{}, fmt.Errorf("%s : connection or transaction is closed", verb)
	}

	qs, ok := queryV.BV().(goal.S)
	if !ok {
		return nil, "", goal.V{}, fmt.Errorf("%s : expected string query, got %q", verb, queryV.Type())
	}
	return q, string(qs), paramsV, nil
}

// ---------------------------------------------------------------------------
// Goal → SQL argument conversion
// ---------------------------------------------------------------------------

// goalToSQLArgs converts a Goal value representing query parameters into a
// []any slice suitable for database/sql.
//
// An array (AI, AF, AS, AV) yields one argument per element.
// A zero V (no params supplied) yields an empty slice.
// A scalar value yields a single-element slice.
func goalToSQLArgs(v goal.V) ([]any, error) {
	if v == (goal.V{}) {
		return nil, nil
	}
	switch xv := v.BV().(type) {
	case *goal.AI:
		out := make([]any, len(xv.Slice))
		for i, x := range xv.Slice {
			out[i] = x
		}
		return out, nil
	case *goal.AF:
		out := make([]any, len(xv.Slice))
		for i, x := range xv.Slice {
			out[i] = x
		}
		return out, nil
	case *goal.AS:
		out := make([]any, len(xv.Slice))
		for i, x := range xv.Slice {
			out[i] = x
		}
		return out, nil
	case *goal.AV:
		out := make([]any, len(xv.Slice))
		for i, el := range xv.Slice {
			sa, err := goalScalarToSQL(el)
			if err != nil {
				return nil, fmt.Errorf("param[%d]: %w", i, err)
			}
			out[i] = sa
		}
		return out, nil
	case *goal.AB:
		out := make([]any, len(xv.Slice))
		for i, b := range xv.Slice {
			out[i] = b
		}
		return out, nil
	}
	// Scalar
	sa, err := goalScalarToSQL(v)
	if err != nil {
		return nil, err
	}
	return []any{sa}, nil
}

// goalScalarToSQL converts a single Goal value to a SQL-compatible any.
func goalScalarToSQL(v goal.V) (any, error) {
	if v.IsI() {
		return v.I(), nil
	}
	if v.IsF() {
		f := v.F()
		if math.IsNaN(f) {
			return nil, nil //nolint:nilnil // nil value represents SQL NULL; nil error means no failure
		}
		return f, nil
	}
	if s, ok := v.BV().(goal.S); ok {
		return string(s), nil
	}
	if ab, ok := v.BV().(*goal.AB); ok {
		return ab.Slice, nil
	}
	return nil, fmt.Errorf("unsupported Goal type %q as SQL parameter", v.Type())
}

// ---------------------------------------------------------------------------
// Row scanning and column array construction
// ---------------------------------------------------------------------------

// scanRows scans all rows from *sql.Rows and returns a QueryResult dict.
func scanRows(rows *stdsql.Rows) (goal.V, error) {
	cols, err := rows.Columns()
	if err != nil {
		return goal.V{}, err
	}
	n := len(cols)

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return goal.V{}, err
	}

	// Accumulate raw driver values per column.
	colRaw := make([][]any, n)
	for i := range colRaw {
		colRaw[i] = make([]any, 0, 16)
	}

	scanBuf := make([]any, n)
	ptrs := make([]any, n)
	for i := range ptrs {
		ptrs[i] = &scanBuf[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return goal.V{}, err
		}
		for i, v := range scanBuf {
			colRaw[i] = append(colRaw[i], v)
		}
	}
	if err := rows.Err(); err != nil {
		return goal.V{}, err
	}

	// Build per-column Goal arrays.
	colArrays := make([]goal.V, n)
	for i, raw := range colRaw {
		if len(raw) == 0 {
			colArrays[i] = emptyColumnArray(colTypes[i])
		} else {
			colArrays[i] = buildColumn(raw)
		}
	}

	// Build result dict: AS(colnames) → AV(arrays).
	keys := goal.NewAS(cols)
	vals := goal.NewAV(colArrays)
	return goal.NewD(keys, vals), nil
}

// emptyColumnArray returns an appropriately-typed empty array for a column
// with zero rows, using database type name metadata.
func emptyColumnArray(ct *stdsql.ColumnType) goal.V {
	dbType := strings.ToUpper(ct.DatabaseTypeName())
	switch {
	case strings.Contains(dbType, "INT"):
		return goal.NewAI([]int64{})
	case strings.Contains(dbType, "REAL") ||
		strings.Contains(dbType, "FLOAT") ||
		strings.Contains(dbType, "DOUBLE") ||
		strings.Contains(dbType, "NUMERIC") ||
		strings.Contains(dbType, "DECIMAL"):
		return goal.NewAF([]float64{})
	case strings.Contains(dbType, "TEXT") ||
		strings.Contains(dbType, "CHAR") ||
		strings.Contains(dbType, "CLOB"):
		return goal.NewAS([]string{})
	case strings.Contains(dbType, "BLOB") || dbType == "":
		// SQLite columns without an explicit type affinity, or BLOBs.
		return goal.NewAV([]goal.V{})
	default:
		return goal.NewAV([]goal.V{})
	}
}

// buildColumn converts a slice of raw driver values (from a single column) to
// a specialised Goal array according to the SqlToGoalTypeMapping invariant.
func buildColumn(raw []any) goal.V { //nolint:funlen
	// First pass: map each value to a Goal V, detect NULLs and mixed types.
	vals := make([]goal.V, len(raw))
	hasNull := false
	allInt := true
	allFloat := true
	allStr := true
	allBytes := true

	for i, v := range raw {
		gv := sqlValueToGoal(v)
		vals[i] = gv
		if v == nil {
			hasNull = true
			allInt = false
			allFloat = false
			allStr = false
			allBytes = false
			continue
		}
		switch v.(type) {
		case int64, int32, int16, int8, int, uint64, uint32, uint16, uint8, uint, bool:
			allFloat = false
			allStr = false
			allBytes = false
		case float64, float32:
			allInt = false
			allStr = false
			allBytes = false
		case string:
			allInt = false
			allFloat = false
			allBytes = false
		case []byte:
			allInt = false
			allFloat = false
			allStr = false
		case time.Time:
			// time.Time maps to int64 (Unix microseconds).
			allFloat = false
			allStr = false
			allBytes = false
		default:
			allInt = false
			allFloat = false
			allStr = false
			allBytes = false
		}
	}

	if hasNull {
		return goal.NewAV(vals)
	}

	_ = allBytes // AV is already the fallback

	if allInt {
		ints := make([]int64, len(vals))
		for i, gv := range vals {
			ints[i] = gv.I()
		}
		return goal.NewAI(ints)
	}
	if allFloat {
		floats := make([]float64, len(vals))
		for i, gv := range vals {
			floats[i] = gv.F()
		}
		return goal.NewAF(floats)
	}
	if allStr {
		strs := make([]string, len(vals))
		for i, gv := range vals {
			strs[i] = string(gv.BV().(goal.S)) //nolint:errcheck // type is guaranteed: allStr was verified in first pass
		}
		return goal.NewAS(strs)
	}

	// Mixed types (e.g. untyped SQLite columns) → AV.
	return goal.NewAV(vals)
}

// ---------------------------------------------------------------------------
// SQL → Goal value conversion
// ---------------------------------------------------------------------------

// sqlValueToGoal converts a single value returned by the sql driver to a Goal
// value according to the SqlToGoalTypeMapping invariant.
func sqlValueToGoal(v any) goal.V {
	if v == nil {
		return goal.NewF(math.NaN()) // 0n
	}
	switch x := v.(type) {
	case int64:
		return goal.NewI(x)
	case int32:
		return goal.NewI(int64(x))
	case int16:
		return goal.NewI(int64(x))
	case int8:
		return goal.NewI(int64(x))
	case int:
		return goal.NewI(int64(x))
	case uint64:
		return goal.NewI(int64(x)) //nolint:gosec // G115: intentional widening; overflow risk is documented
	case uint32:
		return goal.NewI(int64(x))
	case uint16:
		return goal.NewI(int64(x))
	case uint8:
		return goal.NewI(int64(x))
	case uint:
		return goal.NewI(int64(x)) //nolint:gosec // G115: intentional widening; overflow only for values > MaxInt64
	case float64:
		return goal.NewF(x)
	case float32:
		return goal.NewF(float64(x))
	case string:
		return goal.NewS(x)
	case []byte:
		b := make([]byte, len(x))
		copy(b, x)
		return goal.NewAB(b)
	case bool:
		if x {
			return goal.NewI(1)
		}
		return goal.NewI(0)
	case time.Time:
		// Unix microseconds since 1970-01-01 UTC, following ari's convention.
		return goal.NewI(x.UnixMicro())
	default:
		// Fallback: represent as a string via fmt.
		return goal.NewS(fmt.Sprintf("%v", x))
	}
}
