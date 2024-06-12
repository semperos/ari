package ari

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"reflect"
	"time"

	"codeberg.org/anaseto/goal"
	"github.com/spf13/viper"
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
	// these HttpClient structs.
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

func sqlOpen(dbDriver string, dataSourceName string) (*SQLDatabase, error) {
	db, err := sql.Open(dbDriver, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &SQLDatabase{DB: db, IsOpen: true}, nil
}

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

// Implements sql.open to open a SQL DB.
func VFSqlOpen(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	dataSourceName, ok := x.BV().(goal.S)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("sql.open s", "s", x)
		}
		dsn := string(dataSourceName)
		if len(dsn) == 0 {
			// Empty means use the value from config/CLI args
			dsn = viper.GetString("database")
		}
		sqlDB, err := sqlOpen(dbDriver, dsn)
		if err != nil {
			return goal.NewPanicError(err)
		}
		return goal.NewV(sqlDB)
	default:
		return goal.Panicf("sql.open : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements sql.q for SQL querying.
func VFSqlQ(goalContext *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	sqlQuery, ok := x.BV().(goal.S)
	switch len(args) {
	case monadic:
		// Uses the database configured at the Ari level, initializing if not open.
		if !ok {
			return panicType("sql.q s", "s", x)
		}
		if GlobalContext.SQLDatabase == nil || !GlobalContext.SQLDatabase.IsOpen {
			err := ContextInitSQL(&GlobalContext, GlobalContext.SQLDatabase.DataSource)
			if err != nil {
				return goal.NewPanicError(err)
			}
		}
		goalD, err := SQLQueryContext(GlobalContext.SQLDatabase, string(sqlQuery), nil)
		if err != nil {
			return goal.NewPanicError(err)
		}
		// Last result table as sql.t in Goal, to support switching eval mdoes:
		goalContext.AssignGlobal("sql.t", goalD)
		return goalD
	case dyadic:
		// Explicit database as first argument
		sqlDatabase, ok := x.BV().(*SQLDatabase)
		if !ok {
			return panicType("ari.SqlDatabase sql.q s", "ari.SqlDatabase", x)
		}
		y := args[0]
		sqlQuery, ok := y.BV().(goal.S)
		if !ok {
			return panicType("ari.SqlDatabase sql.q s", "s", y)
		}
		queryResult, err := SQLQueryContext(sqlDatabase, string(sqlQuery), nil)
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return queryResult
	default:
		return goal.Panicf("sql.q : too many arguments (%d), expects 1 or 2 arguments", len(args))
	}
}
