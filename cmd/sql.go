package cmd

import "sort"

// From https://en.wikipedia.org/wiki/List_of_SQL_reserved_words
var acSqlKeywords = func() []string {
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
}()

// EXAMPLE 1
// cols, err := rows.Columns() // Remember to check err afterwards
// vals := make([]interface{}, len(cols))
// for i, _ := range cols {
// 	vals[i] = new(sql.RawBytes)
// }
// for rows.Next() {
// 	err = rows.Scan(vals...)
// 	// Now you can check each element of vals for nil-ness,
// 	// and you can use type introspection and type assertions
// 	// to fetch the column into a typed variable.
// }

// EXAMPLE 2
// var myMap = make(map[string]interface{})
// rows, err := db.Query("SELECT * FROM myTable")
// defer rows.Close()
// if err != nil {
//     log.Fatal(err)
// }
// colNames, err := rows.Columns()
// if err != nil {
//     log.Fatal(err)
// }
// cols := make([]interface{}, len(colNames))
// colPtrs := make([]interface{}, len(colNames))
// for i := 0; i < len(colNames); i++ {
//     colPtrs[i] = &cols[i]
// }
// for rows.Next() {
//     err = rows.Scan(colPtrs...)
//     if err != nil {
//         log.Fatal(err)
//     }
//     for i, col := range cols {
//         myMap[colNames[i]] = col
//     }
//     // Do something with the map
//     for key, val := range myMap {
//         fmt.Println("Key:", key, "Value Type:", reflect.TypeOf(val))
//     }
// }

// EXAMPLE 3
// from https://go.dev/wiki/SQLInterface#dealing-with-null
// func getTable[T any](rows *sql.Rows) (out []T) {
//     var table []T
//     for rows.Next() {
//         var data T
//         s := reflect.ValueOf(&data).Elem()
//         numCols := s.NumField()
//         columns := make([]interface{}, numCols)

//         for i := 0; i < numCols; i++ {
//             field := s.Field(i)
//             columns[i] = field.Addr().Interface()
//         }

//         if err := rows.Scan(columns...); err != nil {
//             fmt.Println("Case Read Error ", err)
//         }

//         table = append(table, data)
//     }
//     return table
// }

// type User struct {
// 	UUID  sql.NullString
// 	Name  sql.NullString
//   }

//   rows, err := db.Query("SELECT * FROM Users")
//   cases := getTable[User](rows)

// for  := goal.NewAV(goal.NewV())
// d := goal.NewD()
// APPROACH: sql.RawBytes
// cols, err := rows.Columns()
// if err != nil {
// 	panic(err)
// }
// vals := make([]interface{}, len(cols))
// for i := range cols {
// 	vals[i] = new(sql.RawBytes)
// }
// for rows.Next() {
// 	err = rows.Scan(vals...)
// 	if err != nil {
// 		panic(err)
// 	}
// 	colTypes, err := rows.ColumnTypes()
// 	if err != nil {
// 		panic(err)
// 	}
// 	for i, val := range vals {
// 		colType := colTypes[i]
// 		if val == nil {
// 			fmt.Println("Found null!")
// 		}
// 		// colType
// 		switch val.(type) {
// 		case int:
// 			fmt.Println("Found int", val)
// 			strconv.ParseInt()
// 		case float32:
// 			fmt.Println("Found float32", val)
// 		case float64:
// 			fmt.Println("Found float64", val)
// 		default:
// 			fmt.Println("Found something...", val)
// 		}
// 	}
// }
// for rows.Next() {
// 	cols, err := rows.Columns()
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("COLS: %q\n", cols)
// 	colTypes, err := rows.ColumnTypes()
// 	if err != nil {
// 		panic(err)
// 	}
// 	var values [][]interface{}
// 	for _, colType := range colTypes {
// 		fmt.Printf("COLTYPES %q\n", colType.DatabaseTypeName())
// 	}
// 	rows.Scan(args...)
// }
