package genji

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/asdine/genji/internal/scanner"
	"github.com/asdine/genji/record"
)

// parseUpdateStatement parses a update string and returns a Statement AST object.
// This function assumes the UPDATE token has already been consumed.
func (p *parser) parseUpdateStatement() (updateStmt, error) {
	var stmt updateStmt
	var err error

	// Parse table name
	stmt.tableName, err = p.ParseIdent()
	if err != nil {
		return stmt, err
	}

	// Parse assignment: "SET field = EXPR".
	stmt.pairs, err = p.parseSetClause()
	if err != nil {
		return stmt, err
	}

	// Parse condition: "WHERE EXPR".
	stmt.whereExpr, err = p.parseCondition()
	if err != nil {
		return stmt, err
	}

	return stmt, nil
}

// parseSetClause parses the "SET" clause of the query.
func (p *parser) parseSetClause() (map[string]expr, error) {
	// Check if the SET token exists.
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.SET {
		return nil, newParseError(scanner.Tokstr(tok, lit), []string{"SET"}, pos)
	}

	pairs := make(map[string]expr)

	firstPair := true
	for {
		if !firstPair {
			// Scan for a comma.
			tok, _, _ := p.ScanIgnoreWhitespace()
			if tok != scanner.COMMA {
				p.Unscan()
				break
			}
		}

		// Scan the identifier for the field name.
		tok, pos, lit := p.ScanIgnoreWhitespace()
		if tok != scanner.IDENT {
			return nil, newParseError(scanner.Tokstr(tok, lit), []string{"identifier"}, pos)
		}

		// Scan the eq sign
		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != scanner.EQ {
			return nil, newParseError(scanner.Tokstr(tok, lit), []string{"="}, pos)
		}

		// Scan the expr for the value.
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		pairs[lit] = expr

		firstPair = false
	}

	return pairs, nil
}

// updateStmt is a DSL that allows creating a full Update query.
type updateStmt struct {
	tableName string
	pairs     map[string]expr
	whereExpr expr
}

// IsReadOnly always returns false. It implements the Statement interface.
func (stmt updateStmt) IsReadOnly() bool {
	return false
}

// Run runs the Update table statement in the given transaction.
// It implements the Statement interface.
func (stmt updateStmt) Run(tx *Tx, args []driver.NamedValue) (Result, error) {
	var res Result

	if stmt.tableName == "" {
		return res, errors.New("missing table name")
	}

	if len(stmt.pairs) == 0 {
		return res, errors.New("Set method not called")
	}

	stack := evalStack{
		Tx:     tx,
		Params: args,
	}

	t, err := tx.GetTable(stmt.tableName)
	if err != nil {
		return res, err
	}

	st := record.NewStream(t)
	st = st.Filter(whereClause(stmt.whereExpr, stack))

	indexes, err := t.Indexes()
	if err != nil {
		return res, err
	}

	err = st.Iterate(func(r record.Record) error {
		rk, ok := r.(record.Keyer)
		if !ok {
			return errors.New("attempt to update record without key")
		}

		var fb record.FieldBuffer
		err := fb.ScanRecord(r)
		if err != nil {
			return err
		}

		for fname, e := range stmt.pairs {
			f, err := fb.GetField(fname)
			if err != nil {
				continue
			}

			v, err := e.Eval(evalStack{
				Tx:     tx,
				Record: r,
				Params: args,
			})
			if err != nil {
				return err
			}

			if v.IsList {
				return fmt.Errorf("expected value got list")
			}

			f.Type = v.Value.Type
			f.Data = v.Value.Data
			err = fb.Replace(f.Name, f)
			if err != nil {
				return err
			}
		}

		err = t.replace(indexes, rk.Key(), &fb)
		if err != nil {
			return err
		}

		return nil
	})
	return res, err
}
