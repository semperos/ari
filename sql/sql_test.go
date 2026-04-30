package sql_test

import (
	"math"
	"testing"

	goal "codeberg.org/anaseto/goal"
	gos "codeberg.org/anaseto/goal/os"

	goalsql "github.com/semperos/ari/sql"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newCtx(t *testing.T) *goal.Context {
	t.Helper()
	ctx := goal.NewContext()
	gos.Import(ctx, "")
	goalsql.Import(ctx, "")
	return ctx
}

// eval evaluates a Goal expression, fails the test on panic, and returns the
// result value.
func eval(t *testing.T, ctx *goal.Context, src string) goal.V {
	t.Helper()
	v, err := ctx.Eval(src)
	if err != nil {
		t.Fatalf("eval %q: %v", src, err)
	}
	if v.IsPanic() {
		t.Fatalf("eval %q returned panic: %s", src, v.Sprint(ctx, true))
	}
	return v
}

// evalPanic evaluates a Goal expression and asserts that a panic (error) is
// returned. It returns the panic message string.
func evalPanic(t *testing.T, ctx *goal.Context, src string) string {
	t.Helper()
	v, err := ctx.Eval(src)
	if err != nil {
		return err.Error()
	}
	if !v.IsPanic() {
		t.Fatalf("eval %q: expected panic, got %v", src, v.Sprint(ctx, true))
	}
	return v.Sprint(ctx, true)
}

// openMem opens an in-memory SQLite database and returns the Goal value.
func openMem(t *testing.T, ctx *goal.Context) goal.V {
	t.Helper()
	return eval(t, ctx, `sql.open["sqlite://:memory:"]`)
}

// mustI extracts an int64 from a Goal I value.
func mustI(t *testing.T, v goal.V) int64 {
	t.Helper()
	if !v.IsI() {
		t.Fatalf("expected integer, got %q", v.Type())
	}
	return v.I()
}

// dictLookup retrieves a key from a Goal dict (keys AS, values any array).
// Returns the element at the matching index from the values array.
func dictLookup(t *testing.T, _ *goal.Context, d *goal.D, key string) goal.V {
	t.Helper()
	keys := d.Keys()
	vals := d.Values()

	kas, ok := keys.BV().(*goal.AS)
	if !ok {
		t.Fatalf("dict keys: expected AS, got %q", keys.Type())
	}

	for i, k := range kas.Slice {
		if k != key {
			continue
		}
		// values can be AV (per-column arrays in QueryResult) or AI/AF/AS
		// (scalar columns in ExecResult)
		switch va := vals.BV().(type) {
		case *goal.AV:
			return va.Slice[i]
		case *goal.AI:
			return goal.NewI(va.Slice[i])
		case *goal.AF:
			return goal.NewF(va.Slice[i])
		case *goal.AS:
			return goal.NewS(va.Slice[i])
		default:
			t.Fatalf("dict values: unexpected type %q", vals.Type())
		}
	}
	t.Fatalf("key %q not found in dict", key)
	return goal.V{}
}

// mustDict asserts the value is a Goal dict and returns it.
func mustDict(t *testing.T, ctx *goal.Context, v goal.V) *goal.D {
	t.Helper()
	d, ok := v.BV().(*goal.D)
	if !ok {
		t.Fatalf("expected dict, got %q (%s)", v.Type(), v.Sprint(ctx, true))
	}
	return d
}

// execResult asserts an ExecResult dict and returns (lastInsertId, rowsAffected).
func execResult(t *testing.T, ctx *goal.Context, v goal.V) (int64, int64) {
	t.Helper()
	d := mustDict(t, ctx, v)
	lastID := mustI(t, dictLookup(t, ctx, d, "lastInsertId"))
	rowsAff := mustI(t, dictLookup(t, ctx, d, "rowsAffected"))
	return lastID, rowsAff
}

// ---------------------------------------------------------------------------
// Goal calling-convention notes
//
// In Goal, bracket notation f[a;b;c] pushes args right-to-left:
//   args[0] = c (last bracket arg)
//   args[1] = b
//   args[2] = a (first bracket arg)
//
// x f[a;b] is parsed as x applied to the result of f[a;b] — the left value x
// is NOT automatically included in the variadic args.
//
// Therefore the parameterised query form is:
//   sql.q[db ; "SELECT ... WHERE x=?" ; ,25]    -- args[0]=,25  args[1]="..." args[2]=db
//   sql.exec[db ; "INSERT ... VALUES(?)" ; ,42]
//
// The non-parameterised dyadic form is:
//   db sql.q "SELECT ..."      -- args[0]="..." args[1]=db
//   db sql.exec "INSERT ..."
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// TestOpenClose
// ---------------------------------------------------------------------------

func TestOpenClose(t *testing.T) {
	ctx := newCtx(t)

	db := openMem(t, ctx)
	if db.Type() != "sql.conn" {
		t.Fatalf("sql.open: expected sql.conn, got %q", db.Type())
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

func TestOpenBadScheme(t *testing.T) {
	ctx := newCtx(t)
	evalPanic(t, ctx, `sql.open "duckdb://:memory:"`)
}

func TestOpenBadURI(t *testing.T) {
	ctx := newCtx(t)
	evalPanic(t, ctx, `sql.open "no-scheme-here"`)
}

// ---------------------------------------------------------------------------
// TestExecDDL
// ---------------------------------------------------------------------------

func TestExecDDL(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	v := eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"]`)
	lastID, rowsAff := execResult(t, ctx, v)
	if lastID != 0 || rowsAff != 0 {
		t.Fatalf("CREATE TABLE: expected (0,0), got (%d,%d)", lastID, rowsAff)
	}
}

// ---------------------------------------------------------------------------
// TestInsertAndLastInsertId
// ---------------------------------------------------------------------------

func TestInsertAndLastInsertId(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"]`)

	// Parameterised INSERT — bracket form: sql.exec[db; "query"; args]
	v := eval(t, ctx, `sql.exec[db; "INSERT INTO users (name, age) VALUES (?, ?)" ; ("Alice";30)]`)
	lastID, rowsAff := execResult(t, ctx, v)
	if lastID != 1 {
		t.Fatalf("INSERT: expected lastInsertId=1, got %d", lastID)
	}
	if rowsAff != 1 {
		t.Fatalf("INSERT: expected rowsAffected=1, got %d", rowsAff)
	}

	// Second INSERT.
	v2 := eval(t, ctx, `sql.exec[db; "INSERT INTO users (name, age) VALUES (?, ?)" ; ("Bob";25)]`)
	lastID2, _ := execResult(t, ctx, v2)
	if lastID2 != 2 {
		t.Fatalf("INSERT: expected lastInsertId=2, got %d", lastID2)
	}
}

// ---------------------------------------------------------------------------
// TestQueryColumnarResult
// ---------------------------------------------------------------------------

func TestQueryColumnarResult(t *testing.T) { //nolint:gocognit
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO users (name, age) VALUES (?, ?)" ; ("Alice";30)]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO users (name, age) VALUES (?, ?)" ; ("Bob";25)]`)

	t.Run("all_rows", func(t *testing.T) {
		v := eval(t, ctx, `sql.q[db;"SELECT id, name, age FROM users ORDER BY id"]`)
		d := mustDict(t, ctx, v)

		// id column → AI
		idCol := dictLookup(t, ctx, d, "id")
		ids, ok := idCol.BV().(*goal.AI)
		if !ok {
			t.Fatalf("id column: expected AI, got %q", idCol.Type())
		}
		if len(ids.Slice) != 2 || ids.Slice[0] != 1 || ids.Slice[1] != 2 {
			t.Fatalf("id column: expected [1 2], got %v", ids.Slice)
		}

		// name column → AS
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
		// sql.q[db; "query"; params] — bracket form with conn explicit
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
// TestQueryEmptyResult
// ---------------------------------------------------------------------------

func TestQueryEmptyResult(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER, name TEXT)"]`)

	v := eval(t, ctx, `sql.q[db;"SELECT id, name FROM t"]`)
	d := mustDict(t, ctx, v)

	// Column names must be present even when there are no rows.
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

	// name → empty AS (TEXT type affinity)
	names, ok := nameCol.BV().(*goal.AS)
	if !ok {
		t.Fatalf("empty name column: expected AS, got %q", nameCol.Type())
	}
	if len(names.Slice) != 0 {
		t.Fatalf("empty name column: expected length 0, got %d", len(names.Slice))
	}
}

// ---------------------------------------------------------------------------
// TestNullHandling
// ---------------------------------------------------------------------------

func TestNullHandling(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (id INTEGER, val INTEGER)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?, ?)" ; (1;10)]`)
	eval(t, ctx, `sql.exec[db;"INSERT INTO t VALUES (2, NULL)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?, ?)" ; (3;30)]`)

	v := eval(t, ctx, `sql.q[db;"SELECT val FROM t ORDER BY id"]`)
	d := mustDict(t, ctx, v)
	valCol := dictLookup(t, ctx, d, "val")

	// An INTEGER column containing NULLs: Goal's NewAV normalises
	// [I(10), F(NaN), I(30)] to AF because integers and floats share a
	// common numeric supertype. The NULL is still represented as 0n (NaN).
	//
	// A TEXT column with NULLs would produce AV because S and F have no
	// common scalar type.
	af, ok := valCol.BV().(*goal.AF)
	if !ok {
		t.Fatalf("val column (int+NULLs): expected AF, got %q", valCol.Type())
	}
	if len(af.Slice) != 3 {
		t.Fatalf("val column: expected 3 elements, got %d", len(af.Slice))
	}

	// Row 0: 10.0 (was int64(10), widened to float)
	if af.Slice[0] != 10.0 {
		t.Fatalf("val[0]: expected 10.0, got %v", af.Slice[0])
	}
	// Row 1: NaN (NULL)
	if !math.IsNaN(af.Slice[1]) {
		t.Fatalf("val[1]: expected NaN (NULL), got %v", af.Slice[1])
	}
	// Row 2: 30.0
	if af.Slice[2] != 30.0 {
		t.Fatalf("val[2]: expected 30.0, got %v", af.Slice[2])
	}
}

// ---------------------------------------------------------------------------
// TestFloatColumn
// ---------------------------------------------------------------------------

func TestFloatColumn(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE prices (id INTEGER, price REAL)"]`)
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
// TestTransaction_Commit
// ---------------------------------------------------------------------------

func TestTransactionCommit(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// Transaction that commits (lambda returns a non-error value).
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
// TestTransaction_Rollback
// ---------------------------------------------------------------------------

func TestTransactionRollback(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// Transaction that rolls back: insert into a non-existent table →
	// sql.exec returns a panic value → sql.tx rolls back.
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
// TestTransactionNestedError
// ---------------------------------------------------------------------------

func TestTransactionNestedError(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// sql.tx on a sql.tx (not *Conn) must be an error.
	msg := evalPanic(t, ctx, `sql.tx[db;{[tx2] sql.tx[tx2;{[tx3] 1i}]}]`)
	if msg == "" {
		t.Fatal("expected panic for nested transaction, got none")
	}
}

// ---------------------------------------------------------------------------
// TestMultipleParams
// ---------------------------------------------------------------------------

func TestMultipleParams(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (a INTEGER, b TEXT)"]`)
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
// TestBlobColumn
// ---------------------------------------------------------------------------

func TestBlobColumn(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE blobs (id INTEGER, data BLOB)"]`)
	eval(t, ctx, `sql.exec[db; "INSERT INTO blobs VALUES (?, ?)" ; (1;"hello")]`)

	v := eval(t, ctx, `sql.q[db;"SELECT data FROM blobs"]`)
	d := mustDict(t, ctx, v)
	dataCol := dictLookup(t, ctx, d, "data")
	// SQLite may return BLOB data as []byte or string depending on affinity.
	// Either AB or AS is valid; we just check the column is non-empty.
	switch bv := dataCol.BV().(type) {
	case *goal.AB:
		if len(bv.Slice) == 0 {
			t.Fatal("blob data column: got empty AB")
		}
	case *goal.AS:
		if len(bv.Slice) == 0 {
			t.Fatal("blob data column: got empty AS")
		}
	case *goal.AV:
		if len(bv.Slice) == 0 {
			t.Fatal("blob data column: got empty AV")
		}
	default:
		t.Fatalf("blob data column: unexpected type %q", dataCol.Type())
	}
}

// ---------------------------------------------------------------------------
// TestQueryAfterClose
// ---------------------------------------------------------------------------

func TestQueryAfterClose(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)
	eval(t, ctx, "sql.close[db]")

	evalPanic(t, ctx, `sql.q[db;"SELECT x FROM t"]`)
}

// ---------------------------------------------------------------------------
// TestExecAfterClose
// ---------------------------------------------------------------------------

func TestExecAfterClose(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))
	eval(t, ctx, "sql.close[db]")

	evalPanic(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)
}

// ---------------------------------------------------------------------------
// TestBadSQL
// ---------------------------------------------------------------------------

func TestBadSQL(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	evalPanic(t, ctx, `sql.q[db;"THIS IS NOT SQL"]`)
	evalPanic(t, ctx, `sql.exec[db;"ALSO NOT VALID SQL @@@@"]`)
}

// ---------------------------------------------------------------------------
// TestNullParam
// ---------------------------------------------------------------------------

func TestNullParam(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)

	// Pass 0n (NaN) as a parameter → should become SQL NULL.
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?)" ; ,0n]`)

	v := eval(t, ctx, `sql.q[db;"SELECT x FROM t"]`)
	d := mustDict(t, ctx, v)
	xCol := dictLookup(t, ctx, d, "x")
	// Single NULL in an INTEGER column → AF([NaN]) after Goal normalises
	// [F(NaN)] → AF. The NULL is still 0n.
	af, ok := xCol.BV().(*goal.AF)
	if !ok {
		t.Fatalf("x column: expected AF (has NULL), got %q", xCol.Type())
	}
	if len(af.Slice) != 1 {
		t.Fatalf("x column: expected 1 row, got %d", len(af.Slice))
	}
	if !math.IsNaN(af.Slice[0]) {
		t.Fatalf("x[0]: expected NaN (NULL), got %v", af.Slice[0])
	}
}

// ---------------------------------------------------------------------------
// TestUpdateRowsAffected
// ---------------------------------------------------------------------------

func TestUpdateRowsAffected(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

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
// TestTxQuery
// ---------------------------------------------------------------------------

func TestTxQuery(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

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
// TestSingleEnlistedParam
// ---------------------------------------------------------------------------

func TestSingleEnlistedParam(t *testing.T) {
	ctx := newCtx(t)
	ctx.AssignGlobal("db", openMem(t, ctx))

	eval(t, ctx, `sql.exec[db;"CREATE TABLE t (x INTEGER)"]`)
	// ,99 is enlist 99 → 1-element AI
	eval(t, ctx, `sql.exec[db; "INSERT INTO t VALUES (?)" ; ,99]`)

	v := eval(t, ctx, `sql.q[db; "SELECT x FROM t WHERE x = ?" ; ,99]`)
	d := mustDict(t, ctx, v)
	xCol := dictLookup(t, ctx, d, "x")
	xs, ok := xCol.BV().(*goal.AI)
	if !ok || len(xs.Slice) != 1 || xs.Slice[0] != 99 {
		t.Fatalf("x: expected AI([99]), got %v", xCol.Sprint(ctx, true))
	}
}
