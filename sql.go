package ari

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"time"

	"codeberg.org/anaseto/goal"
)

type SQLDatabase struct {
	DataSource string
	DB         *sql.DB
	IsOpen     bool
}

const dbDriver = "duckdb"

// Append implements goal.BV.
func (sqlDatabase *SQLDatabase) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("<%v %#v>", sqlDatabase.Type(), sqlDatabase.DB)...)
}

// LessT implements goal.BV.
func (sqlDatabase *SQLDatabase) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these structs.
	return sqlDatabase.Type() < y.Type()
}

// Matches implements goal.BV.
func (sqlDatabase *SQLDatabase) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *SQLDatabase:
		return sqlDatabase.DB == yv.DB
	default:
		return false
	}
}

// Type implements goal.BV.
func (sqlDatabase *SQLDatabase) Type() string {
	return "ari.SqlDatabase"
}

func (sqlDatabase *SQLDatabase) Open() error {
	db, err := sql.Open(dbDriver, sqlDatabase.DataSource)
	if err != nil {
		return err
	}
	sqlDatabase.DB = db
	sqlDatabase.IsOpen = true
	return nil
}

// Close will close the underlying sql.DB, if one exists.
func (sqlDatabase *SQLDatabase) Close() error {
	if sqlDatabase.DB != nil {
		err := sqlDatabase.DB.Close()
		if err != nil {
			return err
		}
		sqlDatabase.IsOpen = false
	}
	return nil
}

//nolint:gocognit
func SQLQueryContext(sqlDatabase *SQLDatabase, sqlQuery string, args []any) (goal.V, error) {
	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = sqlDatabase.DB.QueryContext(context.Background(), sqlQuery, args...)
	} else {
		rows, err = sqlDatabase.DB.QueryContext(context.Background(), sqlQuery)
	}
	if err != nil {
		return goal.V{}, err
	}
	// defer rows.Close()
	colNames, err := rows.Columns()
	if err != nil {
		return goal.V{}, err
	}
	// NB: For future type introspection, see below.
	// colTypes, err := rows.ColumnTypes()
	// if err != nil {
	// 	panic(err)
	// }
	cols := make([]interface{}, len(colNames))
	colPtrs := make([]interface{}, len(colNames))
	rowValues := make([][]goal.V, len(cols))
	for i := 0; i < len(colNames); i++ {
		colPtrs[i] = &cols[i]
	}
	for rows.Next() {
		err = rows.Scan(colPtrs...)
		if err != nil {
			return goal.V{}, err
		}
		for i, col := range cols {
			// fmt.Printf("SQL %v // Go %v\n", colTypes[i].DatabaseTypeName(), reflect.TypeOf(col))
			if col == nil {
				rowValues[i] = append(rowValues[i], goal.NewF(math.NaN()))
			} else {
				switch col := col.(type) {
				case bool:
					if col {
						rowValues[i] = append(rowValues[i], goal.NewF(math.Inf(1)))
					} else {
						rowValues[i] = append(rowValues[i], goal.NewF(math.Inf(-1)))
					}
				case float32:
					rowValues[i] = append(rowValues[i], goal.NewF(float64(col)))
				case float64:
					rowValues[i] = append(rowValues[i], goal.NewF(col))
				case int32:
					rowValues[i] = append(rowValues[i], goal.NewI(int64(col)))
				case int64:
					rowValues[i] = append(rowValues[i], goal.NewI(col))
				case string:
					rowValues[i] = append(rowValues[i], goal.NewS(col))
				case time.Time:
					// From DuckDB docs: https://duckdb.org/docs/sql/data_types/timestamp
					// "DuckDB represents instants as the number of microseconds (µs) since 1970-01-01 00:00:00+00"
					rowValues[i] = append(rowValues[i], goal.NewI(col.UnixMicro()))
				default:
					fmt.Fprintf(os.Stderr, "Go Type %v\n", reflect.TypeOf(col))
					rowValues[i] = append(rowValues[i], goal.NewS(fmt.Sprintf("%v", col)))
				}
			}
		}
	}
	if err = rows.Close(); err != nil {
		return goal.V{}, err
	}
	if err = rows.Err(); err != nil {
		return goal.V{}, err
	}
	// NB: For future type introspection, see above.
	// fmt.Printf("COLS %v\n", colNames)
	// fmt.Printf("ROWS %v\n", rowValues)
	dictVs := make([]goal.V, len(rowValues))
	for i, slc := range rowValues {
		// NB: Goal's underlying NewArray function specializes for us.
		dictVs[i] = goal.NewAV(slc)
	}
	goalD := goal.NewD(goal.NewAS(colNames), goal.NewAV(dictVs))
	return goalD, nil
}

func SQLExec(sqlDatabase *SQLDatabase, sqlQuery string, args []any) (goal.V, error) {
	result, err := sqlDatabase.DB.Exec(sqlQuery, args...)
	if err != nil {
		return goal.V{}, err
	}
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return goal.V{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return goal.V{}, err
	}
	// Consider: Formatting this as a table for consistency with )sql, but it's really a simpler dict
	ks := goal.NewAS([]string{"lastInsertId", "rowsAffected"})
	// vs := goal.NewAI([]int64{lastInsertId, rowsAffected})
	vs := goal.NewAV([]goal.V{goal.NewAI([]int64{lastInsertID}), goal.NewAI([]int64{rowsAffected})})
	return goal.NewD(ks, vs), nil
}

// Copied from Goal's implementation, a panic value for type mismatches.
func panicType(op, sym string, x goal.V) goal.V {
	return goal.Panicf("%s : bad type %q in %s", op, x.Type(), sym)
}

// Implements sql.open to open a SQL database.
func VFSqlOpen(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	dataSourceName, ok := x.BV().(goal.S)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("sql.open s", "s", x)
		}
		dsn := string(dataSourceName)
		sqlDatabase, err := NewSQLDatabase(dsn)
		if err != nil {
			return goal.NewPanicError(err)
		}
		err = sqlDatabase.Open()
		if err != nil {
			return goal.NewPanicError(err)
		}
		return goal.NewV(sqlDatabase)
	default:
		return goal.Panicf("sql.open : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements sql.close to close the SQL database.
func VFSqlClose(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	sqlDatabase, ok := x.BV().(*SQLDatabase)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("sql.close ari.SqlDatabase", "ari.SqlDatabase", x)
		}
		err := sqlDatabase.Close()
		if err != nil {
			return goal.NewPanicError(err)
		}
		return goal.NewI(1)
	default:
		return goal.Panicf("sql.close : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements sql.q for SQL querying.
func VFSqlQFn(sqlDatabase *SQLDatabase) func(goalContext *goal.Context, args []goal.V) goal.V {
	return func(goalContext *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		switch len(args) {
		case monadic:
			return sqlQMonadic(x, sqlDatabase, goalContext)
		case dyadic:
			return sqlQDyadic(x, args)
		default:
			return goal.Panicf("sql.q : too many arguments (%d), expects 1 or 2 arguments", len(args))
		}
	}
}

func sqlQMonadic(x goal.V, sqlDatabase *SQLDatabase, goalContext *goal.Context) goal.V {
	sqlQuery, ok := x.BV().(goal.S)
	if !ok {
		return panicType("sql.q s", "s", x)
	}
	var err error
	if sqlDatabase.DB == nil || !sqlDatabase.IsOpen {
		fmt.Fprintf(os.Stdout, "Opening database %q\n", sqlDatabase.DataSource)
		err = sqlDatabase.Open()
		if err != nil {
			return goal.NewPanicError(err)
		}
	}
	goalD, err := SQLQueryContext(sqlDatabase, string(sqlQuery), nil)
	if err != nil {
		return goal.NewPanicError(err)
	}
	goalContext.AssignGlobal("sql.p", goalD)
	return goalD
}

func sqlQDyadic(x goal.V, args []goal.V) goal.V {
	sqlDatabase, ok := x.BV().(*SQLDatabase)
	if !ok {
		return panicType("ari.SqlDatabase sql.q s", "ari.SqlDatabase", x)
	}
	y := args[0]
	sqlQuery, ok := y.BV().(goal.S)
	if !ok {
		return panicType("ari.SqlDatabase sql.q s", "s", y)
	}
	goalD, err := SQLQueryContext(sqlDatabase, string(sqlQuery), nil)
	if err != nil {
		return goal.NewPanicError(err)
	}
	return goalD
}

// Implements sql.exec for executing SQL statements.
func VFSqlExecFn(sqlDatabase *SQLDatabase) func(goalContext *goal.Context, args []goal.V) goal.V {
	return func(goalContext *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		switch len(args) {
		case monadic:
			return sqlExecMonadic(x, sqlDatabase, goalContext)
		case dyadic:
			return sqlExecDyadic(goalContext, x, args)
		default:
			return goal.Panicf("sql.exec too many arguments (%d), expects 1 or 2 arguments", len(args))
		}
	}
}

func sqlExecMonadic(x goal.V, sqlDatabase *SQLDatabase, goalContext *goal.Context) goal.V {
	sqlStmt, ok := x.BV().(goal.S)
	if !ok {
		return panicType("sql.exec s", "s", x)
	}
	var err error
	if sqlDatabase.DB == nil || !sqlDatabase.IsOpen {
		fmt.Fprintf(os.Stdout, "Opening database %q\n", sqlDatabase.DataSource)
		err = sqlDatabase.Open()
		if err != nil {
			return goal.NewPanicError(err)
		}
	}
	goalD, err := SQLExec(sqlDatabase, string(sqlStmt), nil)
	if err != nil {
		return goal.NewPanicError(err)
	}
	goalContext.AssignGlobal("sql.p", goalD)
	return goalD
}

func sqlExecDyadic(goalContext *goal.Context, x goal.V, args []goal.V) goal.V {
	sqlDatabase, ok := x.BV().(*SQLDatabase)
	if !ok {
		return panicType("ari.SqlDatabase sql.exec s", "ari.SqlDatabase", x)
	}
	y := args[0]
	sqlStmt, ok := y.BV().(goal.S)
	if !ok {
		return panicType("ari.SqlDatabase sql.exec s", "s", y)
	}
	goalD, err := SQLExec(sqlDatabase, string(sqlStmt), nil)
	if err != nil {
		return goal.NewPanicError(err)
	}
	goalContext.AssignGlobal("sql.p", goalD)
	return goalD
}

// From https://en.wikipedia.org/wiki/List_of_SQL_reserved_words
//
//nolint:funlen
func SQLKeywords() []string {
	s := []string{
		"ABORT",
		"ABORTSESSION",
		"ABS",
		"ABSENT",
		"ABSOLUTE",
		"ACCESS",
		"ACCESSIBLE",
		"ACCESS_LOCK",
		"ACCOUNT",
		"ACOS",
		"ACOSH",
		"ACTION",
		"ADD",
		"ADD_MONTHS",
		"ADMIN",
		"AFTER",
		"AGGREGATE",
		"ALIAS",
		"ALL",
		"ALLOCATE",
		"ALLOW",
		"ALTER",
		"ALTERAND",
		"AMP",
		"ANALYSE",
		"ANALYZE",
		"AND",
		"ANSIDATE",
		"ANY",
		"ANY_VALUE",
		"ARE",
		"ARRAY",
		"ARRAY_AGG",
		"ARRAY_EXISTS",
		"ARRAY_MAX_CARDINALITY",
		"AS",
		"ASC",
		"ASENSITIVE",
		"ASIN",
		"ASINH",
		"ASSERTION",
		"ASSOCIATE",
		"ASUTIME",
		"ASYMMETRIC",
		"AT",
		"ATAN",
		"ATAN2",
		"ATANH",
		"ATOMIC",
		"AUDIT",
		"AUTHORIZATION",
		"AUX",
		"AUXILIARY",
		"AVE",
		"AVERAGE",
		"AVG",
		"BACKUP",
		"BEFORE",
		"BEGIN",
		"BEGIN_FRAME",
		"BEGIN_PARTITION",
		"BETWEEN",
		"BIGINT",
		"BINARY",
		"BIT",
		"BLOB",
		"BOOLEAN",
		"BOTH",
		"BREADTH",
		"BREAK",
		"BROWSE",
		"BT",
		"BTRIM",
		"BUFFERPOOL",
		"BULK",
		"BUT",
		"BY",
		"BYTE",
		"BYTEINT",
		"BYTES",
		"CALL",
		"CALLED",
		"CAPTURE",
		"CARDINALITY",
		"CASCADE",
		"CASCADED",
		"CASE",
		"CASESPECIFIC",
		"CASE_N",
		"CAST",
		"CATALOG",
		"CCSID",
		"CD",
		"CEIL",
		"CEILING",
		"CHANGE",
		"CHAR",
		"CHAR2HEXINT",
		"CHARACTER",
		"CHARACTERS",
		"CHARACTER_LENGTH",
		"CHARS",
		"CHAR_LENGTH",
		"CHECK",
		"CHECKPOINT",
		"CLASS",
		"CLASSIFIER",
		"CLOB",
		"CLONE",
		"CLOSE",
		"CLUSTER",
		"CLUSTERED",
		"CM",
		"COALESCE",
		"COLLATE",
		"COLLATION",
		"COLLECT",
		"COLLECTION",
		"COLLID",
		"COLUMN",
		"COLUMN_VALUE",
		"COMMENT",
		"COMMIT",
		"COMPLETION",
		"COMPRESS",
		"COMPUTE",
		"CONCAT",
		"CONCURRENTLY",
		"CONDITION",
		"CONNECT",
		"CONNECTION",
		"CONSTRAINT",
		"CONSTRAINTS",
		"CONSTRUCTOR",
		"CONTAINS",
		"CONTAINSTABLE",
		"CONTENT",
		"CONTINUE",
		"CONVERT",
		"CONVERT_TABLE_HEADER",
		"COPY",
		"CORR",
		"CORRESPONDING",
		"COS",
		"COSH",
		"COUNT",
		"COVAR_POP",
		"COVAR_SAMP",
		"CREATE",
		"CROSS",
		"CS",
		"CSUM",
		"CT",
		"CUBE",
		"CUME_DIST",
		"CURRENT",
		"CURRENT_CATALOG",
		"CURRENT_DATE",
		"CURRENT_DEFAULT_TRANSFORM_GROUP",
		"CURRENT_LC_CTYPE",
		"CURRENT_PATH",
		"CURRENT_ROLE",
		"CURRENT_ROW",
		"CURRENT_SCHEMA",
		"CURRENT_SERVER",
		"CURRENT_TIME",
		"CURRENT_TIMESTAMP",
		"CURRENT_TIMEZONE",
		"CURRENT_TRANSFORM_GROUP_FOR_TYPE",
		"CURRENT_USER",
		"CURRVAL",
		"CURSOR",
		"CV",
		"CYCLE",
		"DATA",
		"DATABASE",
		"DATABASES",
		"DATABLOCKSIZE",
		"DATE",
		"DATEFORM",
		"DAY",
		"DAYS",
		"DAY_HOUR",
		"DAY_MICROSECOND",
		"DAY_MINUTE",
		"DAY_SECOND",
		"DBCC",
		"DBINFO",
		"DEALLOCATE",
		"DEC",
		"DECFLOAT",
		"DECIMAL",
		"DECLARE",
		"DEFAULT",
		"DEFERRABLE",
		"DEFERRED",
		"DEFINE",
		"DEGREES",
		"DEL",
		"DELAYED",
		"DELETE",
		"DENSE_RANK",
		"DENY",
		"DEPTH",
		"DEREF",
		"DESC",
		"DESCRIBE",
		"DESCRIPTOR",
		"DESTROY",
		"DESTRUCTOR",
		"DETERMINISTIC",
		"DIAGNOSTIC",
		"DIAGNOSTICS",
		"DICTIONARY",
		"DISABLE",
		"DISABLED",
		"DISALLOW",
		"DISCONNECT",
		"DISK",
		"DISTINCT",
		"DISTINCTROW",
		"DISTRIBUTED",
		"DIV",
		"DO",
		"DOCUMENT",
		"DOMAIN",
		"DOUBLE",
		"DROP",
		"DSSIZE",
		"DUAL",
		"DUMP",
		"DYNAMIC",
		"EACH",
		"ECHO",
		"EDITPROC",
		"ELEMENT",
		"ELSE",
		"ELSEIF",
		"EMPTY",
		"ENABLED",
		"ENCLOSED",
		"ENCODING",
		"ENCRYPTION",
		"END",
		"END-EXEC",
		"ENDING",
		"END_FRAME",
		"END_PARTITION",
		"EQ",
		"EQUALS",
		"ERASE",
		"ERRLVL",
		"ERROR",
		"ERRORFILES",
		"ERRORTABLES",
		"ESCAPE",
		"ESCAPED",
		"ET",
		"EVERY",
		"EXCEPT",
		"EXCEPTION",
		"EXCLUSIVE",
		"EXEC",
		"EXECUTE",
		"EXISTS",
		"EXIT",
		"EXP",
		"EXPLAIN",
		"EXTERNAL",
		"EXTRACT",
		"FALLBACK",
		"FALSE",
		"FASTEXPORT",
		"FENCED",
		"FETCH",
		"FIELDPROC",
		"FILE",
		"FILLFACTOR",
		"FILTER",
		"FINAL",
		"FIRST",
		"FIRST_VALUE",
		"FLOAT",
		"FLOAT4",
		"FLOAT8",
		"FLOOR",
		"FOR",
		"FORCE",
		"FOREIGN",
		"FORMAT",
		"FOUND",
		"FRAME_ROW",
		"FREE",
		"FREESPACE",
		"FREETEXT",
		"FREETEXTTABLE",
		"FREEZE",
		"FROM",
		"FULL",
		"FULLTEXT",
		"FUNCTION",
		"FUSION",
		"GE",
		"GENERAL",
		"GENERATED",
		"GET",
		"GIVE",
		"GLOBAL",
		"GO",
		"GOTO",
		"GRANT",
		"GRAPHIC",
		"GREATEST",
		"GROUP",
		"GROUPING",
		"GROUPS",
		"GT",
		"HANDLER",
		"HASH",
		"HASHAMP",
		"HASHBAKAMP",
		"HASHBUCKET",
		"HASHROW",
		"HAVING",
		"HELP",
		"HIGH_PRIORITY",
		"HOLD",
		"HOLDLOCK",
		"HOST",
		"HOUR",
		"HOURS",
		"HOUR_MICROSECOND",
		"HOUR_MINUTE",
		"HOUR_SECOND",
		"IDENTIFIED",
		"IDENTITY",
		"IDENTITYCOL",
		"IDENTITY_INSERT",
		"IF",
		"IGNORE",
		"ILIKE",
		"IMMEDIATE",
		"IN",
		"INCLUSIVE",
		"INCONSISTENT",
		"INCREMENT",
		"INDEX",
		"INDICATOR",
		"INFILE",
		"INHERIT",
		"INITIAL",
		"INITIALIZE",
		"INITIALLY",
		"INITIATE",
		"INNER",
		"INOUT",
		"INPUT",
		"INS",
		"INSENSITIVE",
		"INSERT",
		"INSTEAD",
		"INT",
		"INT1",
		"INT2",
		"INT3",
		"INT4",
		"INT8",
		"INTEGER",
		"INTEGERDATE",
		"INTERSECT",
		"INTERSECTION",
		"INTERVAL",
		"INTO",
		"IO_AFTER_GTIDS",
		"IO_BEFORE_GTIDS",
		"IS",
		"ISNULL",
		"ISOBID",
		"ISOLATION",
		"ITERATE",
		"JAR",
		"JOIN",
		"JOURNAL",
		"JSON",
		"JSON_ARRAY",
		"JSON_ARRAYAGG",
		"JSON_EXISTS",
		"JSON_OBJECT",
		"JSON_OBJECTAGG",
		"JSON_QUERY",
		"JSON_SCALAR",
		"JSON_SERIALIZE",
		"JSON_TABLE",
		"JSON_TABLE_PRIMITIVE",
		"JSON_VALUE",
		"KEEP",
		"KEY",
		"KEYS",
		"KILL",
		"KURTOSIS",
		"LABEL",
		"LAG",
		"LANGUAGE",
		"LARGE",
		"LAST",
		"LAST_VALUE",
		"LATERAL",
		"LC_CTYPE",
		"LE",
		"LEAD",
		"LEADING",
		"LEAST",
		"LEAVE",
		"LEFT",
		"LESS",
		"LEVEL",
		"LIKE",
		"LIKE_REGEX",
		"LIMIT",
		"LINEAR",
		"LINENO",
		"LINES",
		"LISTAGG",
		"LN",
		"LOAD",
		"LOADING",
		"LOCAL",
		"LOCALE",
		"LOCALTIME",
		"LOCALTIMESTAMP",
		"LOCATOR",
		"LOCATORS",
		"LOCK",
		"LOCKING",
		"LOCKMAX",
		"LOCKSIZE",
		"LOG",
		"LOG10",
		"LOGGING",
		"LOGON",
		"LONG",
		"LONGBLOB",
		"LONGTEXT",
		"LOOP",
		"LOWER",
		"LOW_PRIORITY",
		"LPAD",
		"LT",
		"LTRIM",
		"MACRO",
		"MAINTAINED",
		"MAP",
		"MASTER_BIND",
		"MASTER_SSL_VERIFY_SERVER_CERT",
		"MATCH",
		"MATCHES",
		"MATCH_NUMBER",
		"MATCH_RECOGNIZE",
		"MATERIALIZED",
		"MAVG",
		"MAX",
		"MAXEXTENTS",
		"MAXIMUM",
		"MAXVALUE",
		"MCHARACTERS",
		"MDIFF",
		"MEDIUMBLOB",
		"MEDIUMINT",
		"MEDIUMTEXT",
		"MEMBER",
		"MERGE",
		"METHOD",
		"MICROSECOND",
		"MICROSECONDS",
		"MIDDLEINT",
		"MIN",
		"MINDEX",
		"MINIMUM",
		"MINUS",
		"MINUTE",
		"MINUTES",
		"MINUTE_MICROSECOND",
		"MINUTE_SECOND",
		"MLINREG",
		"MLOAD",
		"MLSLABEL",
		"MOD",
		"MODE",
		"MODIFIES",
		"MODIFY",
		"MODULE",
		"MONITOR",
		"MONRESOURCE",
		"MONSESSION",
		"MONTH",
		"MONTHS",
		"MSUBSTR",
		"MSUM",
		"MULTISET",
		"NAMED",
		"NAMES",
		"NATIONAL",
		"NATURAL",
		"NCHAR",
		"NCLOB",
		"NE",
		"NESTED_TABLE_ID",
		"NEW",
		"NEW_TABLE",
		"NEXT",
		"NEXTVAL",
		"NO",
		"NOAUDIT",
		"NOCHECK",
		"NOCOMPRESS",
		"NONCLUSTERED",
		"NONE",
		"NORMALIZE",
		"NOT",
		"NOTNULL",
		"NOWAIT",
		"NO_WRITE_TO_BINLOG",
		"NTH_VALUE",
		"NTILE",
		"NULL",
		"NULLIF",
		"NULLIFZERO",
		"NULLS",
		"NUMBER",
		"NUMERIC",
		"NUMPARTS",
		"OBID",
		"OBJECT",
		"OBJECTS",
		"OCCURRENCES_REGEX",
		"OCTET_LENGTH",
		"OF",
		"OFF",
		"OFFLINE",
		"OFFSET",
		"OFFSETS",
		"OLD",
		"OLD_TABLE",
		"OMIT",
		"ON",
		"ONE",
		"ONLINE",
		"ONLY",
		"OPEN",
		"OPENDATASOURCE",
		"OPENQUERY",
		"OPENROWSET",
		"OPENXML",
		"OPERATION",
		"OPTIMIZATION",
		"OPTIMIZE",
		"OPTIMIZER_COSTS",
		"OPTION",
		"OPTIONALLY",
		"OR",
		"ORDER",
		"ORDINALITY",
		"ORGANIZATION",
		"OUT",
		"OUTER",
		"OUTFILE",
		"OUTPUT",
		"OVER",
		"OVERLAPS",
		"OVERLAY",
		"OVERRIDE",
		"PACKAGE",
		"PAD",
		"PADDED",
		"PARAMETER",
		"PARAMETERS",
		"PART",
		"PARTIAL",
		"PARTITION",
		"PARTITIONED",
		"PARTITIONING",
		"PASSWORD",
		"PATH",
		"PATTERN",
		"PCTFREE",
		"PER",
		"PERCENT",
		"PERCENTILE_CONT",
		"PERCENTILE_DISC",
		"PERCENT_RANK",
		"PERIOD",
		"PERM",
		"PERMANENT",
		"PIECESIZE",
		"PIVOT",
		"PLACING",
		"PLAN",
		"PORTION",
		"POSITION",
		"POSITION_REGEX",
		"POSTFIX",
		"POWER",
		"PRECEDES",
		"PRECISION",
		"PREFIX",
		"PREORDER",
		"PREPARE",
		"PRESERVE",
		"PREVVAL",
		"PRIMARY",
		"PRINT",
		"PRIOR",
		"PRIQTY",
		"PRIVATE",
		"PRIVILEGES",
		"PROC",
		"PROCEDURE",
		"PROFILE",
		"PROGRAM",
		"PROPORTIONAL",
		"PROTECTION",
		"PSID",
		"PTF",
		"PUBLIC",
		"PURGE",
		"QUALIFIED",
		"QUALIFY",
		"QUANTILE",
		"QUERY",
		"QUERYNO",
		"RADIANS",
		"RAISERROR",
		"RANDOM",
		"RANGE",
		"RANGE_N",
		"RANK",
		"RAW",
		"READ",
		"READS",
		"READTEXT",
		"READ_WRITE",
		"REAL",
		"RECONFIGURE",
		"RECURSIVE",
		"REF",
		"REFERENCES",
		"REFERENCING",
		"REFRESH",
		"REGEXP",
		"REGR_AVGX",
		"REGR_AVGY",
		"REGR_COUNT",
		"REGR_INTERCEPT",
		"REGR_R2",
		"REGR_SLOPE",
		"REGR_SXX",
		"REGR_SXY",
		"REGR_SYY",
		"RELATIVE",
		"RELEASE",
		"RENAME",
		"REPEAT",
		"REPLACE",
		"REPLICATION",
		"REPOVERRIDE",
		"REQUEST",
		"REQUIRE",
		"RESIGNAL",
		"RESOURCE",
		"RESTART",
		"RESTORE",
		"RESTRICT",
		"RESULT",
		"RESULT_SET_LOCATOR",
		"RESUME",
		"RET",
		"RETRIEVE",
		"RETURN",
		"RETURNING",
		"RETURNS",
		"REVALIDATE",
		"REVERT",
		"REVOKE",
		"RIGHT",
		"RIGHTS",
		"RLIKE",
		"ROLE",
		"ROLLBACK",
		"ROLLFORWARD",
		"ROLLUP",
		"ROUND_CEILING",
		"ROUND_DOWN",
		"ROUND_FLOOR",
		"ROUND_HALF_DOWN",
		"ROUND_HALF_EVEN",
		"ROUND_HALF_UP",
		"ROUND_UP",
		"ROUTINE",
		"ROW",
		"ROWCOUNT",
		"ROWGUIDCOL",
		"ROWID",
		"ROWNUM",
		"ROWS",
		"ROWSET",
		"ROW_NUMBER",
		"RPAD",
		"RULE",
		"RUN",
		"RUNNING",
		"SAMPLE",
		"SAMPLEID",
		"SAVE",
		"SAVEPOINT",
		"SCHEMA",
		"SCHEMAS",
		"SCOPE",
		"SCRATCHPAD",
		"SCROLL",
		"SEARCH",
		"SECOND",
		"SECONDS",
		"SECOND_MICROSECOND",
		"SECQTY",
		"SECTION",
		"SECURITY",
		"SECURITYAUDIT",
		"SEEK",
		"SEL",
		"SELECT",
		"SEMANTICKEYPHRASETABLE",
		"SEMANTICSIMILARITYDETAILSTABLE",
		"SEMANTICSIMILARITYTABLE",
		"SENSITIVE",
		"SEPARATOR",
		"SEQUENCE",
		"SESSION",
		"SESSION_USER",
		"SET",
		"SETRESRATE",
		"SETS",
		"SETSESSRATE",
		"SETUSER",
		"SHARE",
		"SHOW",
		"SHUTDOWN",
		"SIGNAL",
		"SIMILAR",
		"SIMPLE",
		"SIN",
		"SINH",
		"SIZE",
		"SKEW",
		"SKIP",
		"SMALLINT",
		"SOME",
		"SOUNDEX",
		"SOURCE",
		"SPACE",
		"SPATIAL",
		"SPECIFIC",
		"SPECIFICTYPE",
		"SPOOL",
		"SQL",
		"SQLEXCEPTION",
		"SQLSTATE",
		"SQLTEXT",
		"SQLWARNING",
		"SQL_BIG_RESULT",
		"SQL_CALC_FOUND_ROWS",
		"SQL_SMALL_RESULT",
		"SQRT",
		"SS",
		"SSL",
		"STANDARD",
		"START",
		"STARTING",
		"STARTUP",
		"STATE",
		"STATEMENT",
		"STATIC",
		"STATISTICS",
		"STAY",
		"STDDEV_POP",
		"STDDEV_SAMP",
		"STEPINFO",
		"STOGROUP",
		"STORED",
		"STORES",
		"STRAIGHT_JOIN",
		"STRING_CS",
		"STRUCTURE",
		"STYLE",
		"SUBMULTISET",
		"SUBSCRIBER",
		"SUBSET",
		"SUBSTR",
		"SUBSTRING",
		"SUBSTRING_REGEX",
		"SUCCEEDS",
		"SUCCESSFUL",
		"SUM",
		"SUMMARY",
		"SUSPEND",
		"SYMMETRIC",
		"SYNONYM",
		"SYSDATE",
		"SYSTEM",
		"SYSTEM_TIME",
		"SYSTEM_USER",
		"SYSTIMESTAMP",
		"TABLE",
		"TABLESAMPLE",
		"TABLESPACE",
		"TAN",
		"TANH",
		"TBL_CS",
		"TEMPORARY",
		"TERMINATE",
		"TERMINATED",
		"TEXTSIZE",
		"THAN",
		"THEN",
		"THRESHOLD",
		"TIME",
		"TIMESTAMP",
		"TIMEZONE_HOUR",
		"TIMEZONE_MINUTE",
		"TINYBLOB",
		"TINYINT",
		"TINYTEXT",
		"TITLE",
		"TO",
		"TOP",
		"TRACE",
		"TRAILING",
		"TRAN",
		"TRANSACTION",
		"TRANSLATE",
		"TRANSLATE_CHK",
		"TRANSLATE_REGEX",
		"TRANSLATION",
		"TREAT",
		"TRIGGER",
		"TRIM",
		"TRIM_ARRAY",
		"TRUE",
		"TRUNCATE",
		"TRY_CONVERT",
		"TSEQUAL",
		"TYPE",
		"UC",
		"UESCAPE",
		"UID",
		"UNDEFINED",
		"UNDER",
		"UNDO",
		"UNION",
		"UNIQUE",
		"UNKNOWN",
		"UNLOCK",
		"UNNEST",
		"UNPIVOT",
		"UNSIGNED",
		"UNTIL",
		"UPD",
		"UPDATE",
		"UPDATETEXT",
		"UPPER",
		"UPPERCASE",
		"USAGE",
		"USE",
		"USER",
		"USING",
		"UTC_DATE",
		"UTC_TIME",
		"UTC_TIMESTAMP",
		"VALIDATE",
		"VALIDPROC",
		"VALUE",
		"VALUES",
		"VALUE_OF",
		"VARBINARY",
		"VARBYTE",
		"VARCHAR",
		"VARCHAR2",
		"VARCHARACTER",
		"VARGRAPHIC",
		"VARIABLE",
		"VARIADIC",
		"VARIANT",
		"VARYING",
		"VAR_POP",
		"VAR_SAMP",
		"VCAT",
		"VERBOSE",
		"VERSIONING",
		"VIEW",
		"VIRTUAL",
		"VOLATILE",
		"VOLUMES",
		"WAIT",
		"WAITFOR",
		"WHEN",
		"WHENEVER",
		"WHERE",
		"WHILE",
		"WIDTH_BUCKET",
		"WINDOW",
		"WITH",
		"WITHIN",
		"WITHIN_GROUP",
		"WITHOUT",
		"WLM",
		"WORK",
		"WRITE",
		"WRITETEXT",
		"XMLCAST",
		"XMLEXISTS",
		"XMLNAMESPACES",
		"XOR",
		"YEAR",
		"YEARS",
		"YEAR_MONTH",
		"ZEROFILL",
		"ZEROIFNULL",
		"ZONE",
	}
	sort.Strings(s)
	return s
}
