package sql_test

// DuckDB-specific tests.
//
// Key behavioural differences from SQLite:
//
//   - INTEGER is 32-bit (int32) in DuckDB; BIGINT is 64-bit.  Both map to
//     Goal's AI via sqlValueToGoal's int32 → int64 widening.
//   - DuckDB does not support ROWID-based AUTOINCREMENT.  Tests supply
//     explicit primary key values.
//   - LastInsertId() always returns 0 for DuckDB (no ROWID concept);
//     RowsAffected() works normally.
//   - VARCHAR is DuckDB's canonical string type; TEXT is accepted as an alias.
//   - In-memory DSN is the empty string (duckdb://) or ":memory:"
//     (duckdb://:memory:).

import (
	"math"
	"testing"

	goal "codeberg.org/anaseto/goal"
)

// openDuckDB opens an in-memory DuckDB database and returns the Goal value.
func openDuckDB(t *testing.T, ctx *goal.Context) goal.V {
	t.Helper()
	return eval(t, ctx, `sql.open["duckdb://"]`)
}

// ---------------------------------------------------------------------------
// TestDuckDBOpenClose
// ---------------------------------------------------------------------------

func TestDuckDBOpenClose(t *testing.T) {
	ctx := newCtx(t)

	db := openDuckDB(t, ctx)
	if db.Type() != "sql.conn" {
		t.Fatalf("sql.open duckdb: expected sql.conn, got %q", db.Type())
	}
	ctx.AssignGlobal("db", db)

	// Close succeeds and returns 1i.
	v := eval(t, ctx, "sql.close[db]")
	if mustI(t, v) != 1 {
		t.Fatalf("sql.close: expected 1, got %v", v.Sprint(ctx, true))
	}

	// Closing again is an error.
	evalPanic(t, ctx, "sql.close[db]")
}

func TestDuckDBOpenMemoryAlias(t *testing.T) {
	// Both duckdb:// and duckdb://:memory: should open in-memory databases.
	ctx := newCtx(t)
	db := eval(t, ctx, `sql.open["duckdb://:memory:"]`)
	if db.Type() != "sql.conn" {
		t.Fatalf("sql.open duckdb://:memory:: expected sql.conn, got %q", db.Type())
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBExecDDL
// ---------------------------------------------------------------------------

func TestDuckDBExecDDL(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	v := eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER, name VARCHAR, age INTEGER)"]`)
	_, rowsAff := execResult(t, ctx, v)
	// DuckDB returns 0 rowsAffected for DDL.
	if rowsAff != 0 {
		t.Fatalf("CREATE TABLE: expected rowsAffected=0, got %d", rowsAff)
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBInsert
// ---------------------------------------------------------------------------

func TestDuckDBInsert(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE users (id INTEGER, name VARCHAR, age INTEGER)"]`)

	// Parameterised INSERT.
	v := eval(t, ctx, `sql.exec[db; "INSERT INTO users VALUES (?, ?, ?)" ; (1;"Alice";30)]`)
	lastID, rowsAff := execResult(t, ctx, v)
	// DuckDB does not support LastInsertId; it always returns 0.
	if lastID != 0 {
		t.Fatalf("INSERT: expected lastInsertId=0 (DuckDB), got %d", lastID)
	}
	if rowsAff != 1 {
		t.Fatalf("INSERT: expected rowsAffected=1, got %d", rowsAff)
	}

	// Second INSERT.
	v2 := eval(t, ctx, `sql.exec[db; "INSERT INTO users VALUES (?, ?, ?)" ; (2;"Bob";25)]`)
	_, rowsAff2 := execResult(t, ctx, v2)
	if rowsAff2 != 1 {
		t.Fatalf("INSERT: expected rowsAffected=1, got %d", rowsAff2)
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBQueryColumnarResult
// ---------------------------------------------------------------------------

func TestDuckDBQueryColumnarResult(t *testing.T) { //nolint:gocognit
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE users (id INTEGER, name VARCHAR, age INTEGER)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO users VALUES (?, ?, ?)" ; (1;"Alice";30)]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO users VALUES (?, ?, ?)" ; (2;"Bob";25)]`)

	t.Run("all_rows", func(t *testing.T) {
		v := eval(t, ctx, `sql.q[db;"SELECT id, name, age FROM users ORDER BY id"]`)
		d := mustDict(t, ctx, v)

		// id column: DuckDB INTEGER = int32, widened to int64 → AI
		idCol := dictLookup(t, ctx, d, "id")
		ids, ok := idCol.BV().(*goal.AI)
		if !ok {
			t.Fatalf("id column: expected AI, got %q", idCol.Type())
		}
		if len(ids.Slice) != 2 || ids.Slice[0] != 1 || ids.Slice[1] != 2 {
			t.Fatalf("id column: expected [1 2], got %v", ids.Slice)
		}

		// name column: DuckDB VARCHAR → string → AS
		nameCol := dictLookup(t, ctx, d, "name")
		names, ok := nameCol.BV().(*goal.AS)
		if !ok {
			t.Fatalf("name column: expected AS, got %q", nameCol.Type())
		}
		if len(names.Slice) != 2 || names.Slice[0] != "Alice" || names.Slice[1] != "Bob" {
			t.Fatalf("name column: expected [Alice Bob], got %v", names.Slice)
		}

		// age column → AI
		ageCol := dictLookup(t, ctx, d, "age")
		ages, ok := ageCol.BV().(*goal.AI)
		if !ok {
			t.Fatalf("age column: expected AI, got %q", ageCol.Type())
		}
		if len(ages.Slice) != 2 || ages.Slice[0] != 30 || ages.Slice[1] != 25 {
			t.Fatalf("age column: expected [30 25], got %v", ages.Slice)
		}
	})

	t.Run("parameterised_where", func(t *testing.T) {
		v := eval(t, ctx, `sql.q[db; "SELECT name FROM users WHERE age > ?" ; ,28]`)
		d := mustDict(t, ctx, v)
		nameCol := dictLookup(t, ctx, d, "name")
		names, ok := nameCol.BV().(*goal.AS)
		if !ok {
			t.Fatalf("name column: expected AS, got %q", nameCol.Type())
		}
		if len(names.Slice) != 1 || names.Slice[0] != "Alice" {
			t.Fatalf("name column: expected [Alice], got %v", names.Slice)
		}
	})

	t.Run("two_params", func(t *testing.T) {
		v := eval(t, ctx, `sql.q[db; "SELECT id FROM users WHERE age > ? AND age < ?" ; (20;28)]`)
		d := mustDict(t, ctx, v)
		idCol := dictLookup(t, ctx, d, "id")
		ids, ok := idCol.BV().(*goal.AI)
		if !ok {
			t.Fatalf("id column: expected AI, got %q", idCol.Type())
		}
		if len(ids.Slice) != 1 || ids.Slice[0] != 2 {
			t.Fatalf("id column: expected [2] (Bob, age=25), got %v", ids.Slice)
		}
	})
}

// ---------------------------------------------------------------------------
// TestDuckDBQueryEmptyResult
// ---------------------------------------------------------------------------

func TestDuckDBQueryEmptyResult(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER, name VARCHAR)"]`)

	v := eval(t, ctx, `sql.q[db;"SELECT id, name FROM t"]`)
	d := mustDict(t, ctx, v)

	// Column names must be present even with no rows.
	idCol := dictLookup(t, ctx, d, "id")
	nameCol := dictLookup(t, ctx, d, "name")

	// id → empty AI (INTEGER type affinity)
	ids, ok := idCol.BV().(*goal.AI)
	if !ok {
		t.Fatalf("empty id column: expected AI, got %q", idCol.Type())
	}
	if len(ids.Slice) != 0 {
		t.Fatalf("empty id column: expected length 0, got %d", len(ids.Slice))
	}

	// name → empty AS (VARCHAR → contains "CHAR" → AS)
	names, ok := nameCol.BV().(*goal.AS)
	if !ok {
		t.Fatalf("empty name column: expected AS, got %q", nameCol.Type())
	}
	if len(names.Slice) != 0 {
		t.Fatalf("empty name column: expected length 0, got %d", len(names.Slice))
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBNullHandling
// ---------------------------------------------------------------------------

func TestDuckDBNullHandling(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER, val INTEGER)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?, ?)" ; (1;10)]`)
	eval(t, ctx, `sql.exec[db;"INSERT INTO t VALUES (2, NULL)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?, ?)" ; (3;30)]`)

	v := eval(t, ctx, `sql.q[db;"SELECT val FROM t ORDER BY id"]`)
	d := mustDict(t, ctx, v)
	valCol := dictLookup(t, ctx, d, "val")

	// An INTEGER column containing NULLs: Goal's NewAV normalises
	// [I(10), F(NaN), I(30)] → AF.  The NULL slot stays as 0n.
	af, ok := valCol.BV().(*goal.AF)
	if !ok {
		t.Fatalf("val column (int+NULLs): expected AF, got %q", valCol.Type())
	}
	if len(af.Slice) != 3 {
		t.Fatalf("val column: expected 3 elements, got %d", len(af.Slice))
	}
	if af.Slice[0] != 10.0 {
		t.Fatalf("val[0]: expected 10.0, got %v", af.Slice[0])
	}
	if !math.IsNaN(af.Slice[1]) {
		t.Fatalf("val[1]: expected NaN (NULL), got %v", af.Slice[1])
	}
	if af.Slice[2] != 30.0 {
		t.Fatalf("val[2]: expected 30.0, got %v", af.Slice[2])
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBFloatColumn
// ---------------------------------------------------------------------------

func TestDuckDBFloatColumn(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE prices (id INTEGER, price DOUBLE)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO prices VALUES (?, ?)" ; (1;9.99)]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO prices VALUES (?, ?)" ; (2;4.5)]`)

	v := eval(t, ctx, `sql.q[db;"SELECT price FROM prices ORDER BY id"]`)
	d := mustDict(t, ctx, v)
	priceCol := dictLookup(t, ctx, d, "price")

	af, ok := priceCol.BV().(*goal.AF)
	if !ok {
		t.Fatalf("price column: expected AF, got %q", priceCol.Type())
	}
	if len(af.Slice) != 2 {
		t.Fatalf("price column: expected 2 rows, got %d", len(af.Slice))
	}
	if af.Slice[0] != 9.99 {
		t.Fatalf("price[0]: expected 9.99, got %v", af.Slice[0])
	}
	if af.Slice[1] != 4.5 {
		t.Fatalf("price[1]: expected 4.5, got %v", af.Slice[1])
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBBigint
// ---------------------------------------------------------------------------

func TestDuckDBBigint(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id BIGINT, val BIGINT)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?, ?)" ; (1;9000000000)]`)

	v := eval(t, ctx, `sql.q[db;"SELECT val FROM t"]`)
	d := mustDict(t, ctx, v)
	valCol := dictLookup(t, ctx, d, "val")

	ai, ok := valCol.BV().(*goal.AI)
	if !ok {
		t.Fatalf("val column: expected AI, got %q", valCol.Type())
	}
	if len(ai.Slice) != 1 || ai.Slice[0] != 9000000000 {
		t.Fatalf("val[0]: expected 9000000000, got %v", ai.Slice)
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBTransactionCommit
// ---------------------------------------------------------------------------

func TestDuckDBTransactionCommit(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// Transaction that commits.
	v := eval(t, ctx, `sql.tx[db;{[tx] sql.exec[tx; "INSERT INTO t VALUES (?)" ; ,42]}]`)
	_, rowsAff := execResult(t, ctx, v)
	if rowsAff != 1 {
		t.Fatalf("tx commit: expected rowsAffected=1, got %d", rowsAff)
	}

	// Verify the row is visible outside the transaction.
	qv := eval(t, ctx, `sql.q[db;"SELECT x FROM t"]`)
	d := mustDict(t, ctx, qv)
	xCol := dictLookup(t, ctx, d, "x")
	xs, ok := xCol.BV().(*goal.AI)
	if !ok || len(xs.Slice) != 1 || xs.Slice[0] != 42 {
		t.Fatalf("after commit: expected x=[42], got %v", qv.Sprint(ctx, true))
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBTransactionRollback
// ---------------------------------------------------------------------------

func TestDuckDBTransactionRollback(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// Transaction that rolls back: insert into a non-existent table.
	_, _ = ctx.Eval(`sql.tx[db;{[tx] sql.exec[tx;"INSERT INTO nonexistent VALUES (1)"]}]`)

	// The main table must still be empty.
	qv := eval(t, ctx, `sql.q[db;"SELECT x FROM t"]`)
	d := mustDict(t, ctx, qv)
	xCol := dictLookup(t, ctx, d, "x")
	xs, ok := xCol.BV().(*goal.AI)
	if !ok {
		t.Fatalf("after rollback: expected AI, got %q", xCol.Type())
	}
	if len(xs.Slice) != 0 {
		t.Fatalf("after rollback: expected empty table, got %v", xs.Slice)
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBMultipleParams
// ---------------------------------------------------------------------------

func TestDuckDBMultipleParams(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (a INTEGER, b VARCHAR)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?, ?)" ; (7;"hello")]`)

	v := eval(t, ctx, `sql.q[db; "SELECT a, b FROM t WHERE a = ? AND b = ?" ; (7;"hello")]`)
	d := mustDict(t, ctx, v)

	aCol := dictLookup(t, ctx, d, "a")
	bCol := dictLookup(t, ctx, d, "b")

	ai, ok := aCol.BV().(*goal.AI)
	if !ok || len(ai.Slice) != 1 || ai.Slice[0] != 7 {
		t.Fatalf("a column: expected AI([7]), got %v", aCol.Sprint(ctx, true))
	}
	as, ok := bCol.BV().(*goal.AS)
	if !ok || len(as.Slice) != 1 || as.Slice[0] != "hello" {
		t.Fatalf("b column: expected AS([hello]), got %v", bCol.Sprint(ctx, true))
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBUpdateRowsAffected
// ---------------------------------------------------------------------------

func TestDuckDBUpdateRowsAffected(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER, val INTEGER)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?,?)" ; (1;10)]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?,?)" ; (2;20)]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?,?)" ; (3;30)]`)

	v := eval(t, ctx, `sql.exec[db;"UPDATE t SET val = 99 WHERE val < 25"]`)
	_, rowsAff := execResult(t, ctx, v)
	if rowsAff != 2 {
		t.Fatalf("UPDATE: expected rowsAffected=2, got %d", rowsAff)
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBNullParam
// ---------------------------------------------------------------------------

func TestDuckDBNullParam(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// Pass 0n (NaN) as a parameter → should become SQL NULL.
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?)" ; ,0n]`)

	v := eval(t, ctx, `sql.q[db;"SELECT x FROM t"]`)
	d := mustDict(t, ctx, v)
	xCol := dictLookup(t, ctx, d, "x")

	// Single NULL in an INTEGER column → AF([NaN]) after Goal normalises.
	af, ok := xCol.BV().(*goal.AF)
	if !ok {
		t.Fatalf("x column: expected AF (has NULL), got %q", xCol.Type())
	}
	if len(af.Slice) != 1 || !math.IsNaN(af.Slice[0]) {
		t.Fatalf("x[0]: expected NaN (NULL), got %v", af.Slice)
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBTxQuery
// ---------------------------------------------------------------------------

func TestDuckDBTxQuery(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?)" ; ,7]`)

	// Query inside a transaction.
	v := eval(t, ctx, `sql.tx[db;{[tx] sql.q[tx;"SELECT x FROM t"]}]`)
	d := mustDict(t, ctx, v)
	xCol := dictLookup(t, ctx, d, "x")
	xs, ok := xCol.BV().(*goal.AI)
	if !ok || len(xs.Slice) != 1 || xs.Slice[0] != 7 {
		t.Fatalf("tx query: expected x=[7], got %v", xCol.Sprint(ctx, true))
	}
}

// ---------------------------------------------------------------------------
// TestDuckDBQueryAfterClose
// ---------------------------------------------------------------------------

func TestDuckDBQueryAfterClose(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)
	eval(t, ctx, "sql.close[db]")

	evalPanic(t, ctx, `sql.q[db;"SELECT x FROM t"]`)
}

// ---------------------------------------------------------------------------
// TestDuckDBBadSQL
// ---------------------------------------------------------------------------

func TestDuckDBBadSQL(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openDuckDB(t, ctx))

	evalPanic(t, ctx, `sql.q[db;"THIS IS NOT SQL"]`)
	evalPanic(t, ctx, `sql.exec[db;"ALSO NOT VALID SQL @@@@"]`)
}
