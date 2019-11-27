package genji

import (
	"database/sql"
	"database/sql/driver"

	"github.com/asdine/genji/record"
)

// A query can execute statements against the database. It can read or write data
// from any table, or even alter the structure of the database.
// Results are returned as streams.
type query struct {
	Statements []statement
}

// Run executes all the statements in their own transaction and returns the last result.
func (q query) Run(db *DB, args []driver.NamedValue) (*Result, error) {
	var res Result
	var tx *Tx
	var err error

	for _, stmt := range q.Statements {
		// it there is an opened transaction but there are still statements
		// to be executed, close the current transaction.
		if tx != nil {
			if tx.Writable() {
				err := tx.Commit()
				if err != nil {
					return nil, err
				}
			} else {
				err := tx.Rollback()
				if err != nil {
					return nil, err
				}
			}
		}

		// start a new transaction for every statement
		tx, err = db.Begin(!stmt.IsReadOnly())
		if err != nil {
			return nil, err
		}

		res, err = stmt.Run(tx, args)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// the returned result will now own the transaction.
	// its Close method is expected to be called.
	res.tx = tx

	return &res, nil
}

// Exec the query within the given transaction. If the one of the statements requires a read-write
// transaction and tx is not, tx will get promoted.
func (q query) Exec(tx *Tx, args []driver.NamedValue, forceReadOnly bool) (*Result, error) {
	var res Result
	var err error

	for _, stmt := range q.Statements {
		// if the statement requires a writable transaction,
		// promote the current transaction.
		if !forceReadOnly && !tx.Writable() && !stmt.IsReadOnly() {
			err := tx.Promote()
			if err != nil {
				return nil, err
			}
		}

		res, err = stmt.Run(tx, args)
		if err != nil {
			return nil, err
		}
	}

	return &res, nil
}

// newQuery creates a new query with the given statements.
func newQuery(statements ...statement) query {
	return query{Statements: statements}
}

// A statement represents a unique action that can be executed against the database.
type statement interface {
	Run(*Tx, []driver.NamedValue) (Result, error)
	IsReadOnly() bool
}

// Result of a query.
type Result struct {
	record.Stream
	rowsAffected  driver.RowsAffected
	lastInsertKey []byte
	tx            *Tx
	closed        bool
}

// LastInsertId is not supported and returns an error.
// Use LastInsertKey instead.
func (r Result) LastInsertId() (int64, error) {
	return r.rowsAffected.LastInsertId()
}

// LastInsertKey returns the database's auto-generated key
// after, for example, an INSERT into a table with primary
// key.
func (r Result) LastInsertKey() ([]byte, error) {
	return r.lastInsertKey, nil
}

// RowsAffected returns the number of rows affected by the
// query.
func (r Result) RowsAffected() (int64, error) {
	return r.rowsAffected.RowsAffected()
}

// Close the result stream.
// After closing the result, Stream is not supposed to be used.
// If the result stream was already closed, it returns
// ErrResultClosed.
func (r *Result) Close() error {
	if r.closed {
		return ErrResultClosed
	}

	r.closed = true

	var err error
	if r.tx != nil {
		if r.tx.Writable() {
			err = r.tx.Commit()
		} else {
			err = r.tx.Rollback()
		}
	}

	return err
}

func whereClause(e expr, stack evalStack) func(r record.Record) (bool, error) {
	if e == nil {
		return func(r record.Record) (bool, error) {
			return true, nil
		}
	}

	return func(r record.Record) (bool, error) {
		stack.Record = r
		v, err := e.Eval(stack)
		if err != nil {
			return false, err
		}

		return v.Truthy(), nil
	}
}

func argsToNamedValues(args []interface{}) []driver.NamedValue {
	nv := make([]driver.NamedValue, len(args))
	for i := range args {
		switch t := args[i].(type) {
		case sql.NamedArg:
			nv[i].Name = t.Name
			nv[i].Value = t.Value
		case *sql.NamedArg:
			nv[i].Name = t.Name
			nv[i].Value = t.Value
		case driver.NamedValue:
			nv[i] = t
		case *driver.NamedValue:
			nv[i] = *t
		default:
			nv[i].Ordinal = i + 1
			nv[i].Value = args[i]
		}
	}

	return nv
}
