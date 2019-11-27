package genji

import (
	"bytes"
	"database/sql"
	"testing"

	"github.com/asdine/genji/engine/memoryengine"
	"github.com/asdine/genji/record"
	"github.com/stretchr/testify/require"
)

func TestParserInsert(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected statement
		fails    bool
	}{
		{"Values / No columns", `INSERT INTO test VALUES ('a', -1, true)`,
			insertStmt{tableName: "test", values: litteralExprList{litteralExprList{stringValue("a"), int8Value(-1), boolValue(true)}}}, false},
		{"Values / With columns", "INSERT INTO test (a, b) VALUES ('c', 'd', 'e')",
			insertStmt{
				tableName:  "test",
				fieldNames: []string{"a", "b"},
				values: litteralExprList{
					litteralExprList{stringValue("c"), stringValue("d"), stringValue("e")},
				},
			}, false},
		{"Values / Multiple", "INSERT INTO test (a, b) VALUES ('c', 'd'), ('e', 'f')",
			insertStmt{
				tableName:  "test",
				fieldNames: []string{"a", "b"},
				values: litteralExprList{
					litteralExprList{stringValue("c"), stringValue("d")},
					litteralExprList{stringValue("e"), stringValue("f")},
				},
			}, false},

		{"Records", `INSERT INTO test RECORDS (a: 'a', b: 2.3, "c ": 1 = 1)`,
			insertStmt{
				tableName: "test",
				records: []interface{}{
					[]kvPair{
						kvPair{K: "a", V: stringValue("a")},
						kvPair{K: "b", V: float64Value(2.3)},
						kvPair{K: "c ", V: eq(int8Value(1), int8Value(1))},
					},
				},
			}, false},
		{"Records / Multiple", `INSERT INTO test RECORDS ("a": 'a', b: -2.3), (a: 1, d: true)`,
			insertStmt{
				tableName: "test",
				records: []interface{}{
					[]kvPair{
						kvPair{K: "a", V: stringValue("a")},
						kvPair{K: "b", V: float64Value(-2.3)},
					},
					[]kvPair{kvPair{K: "a", V: int8Value(1)}, kvPair{K: "d", V: boolValue(true)}},
				},
			}, false},
		{"Records / Positional Param", "INSERT INTO test RECORDS ?, ?",
			insertStmt{
				tableName: "test",
				records:   []interface{}{positionalParam(1), positionalParam(2)},
			},
			false},
		{"Records / Named Param", "INSERT INTO test RECORDS $foo, $bar",
			insertStmt{
				tableName: "test",
				records:   []interface{}{namedParam("foo"), namedParam("bar")},
			},
			false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := parseQuery(test.s)
			if test.fails {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, q.Statements, 1)
			require.EqualValues(t, test.expected, q.Statements[0])
		})
	}
}

func TestInsertStmt(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		fails    bool
		expected string
		params   []interface{}
	}{
		{"Values / No columns", `INSERT INTO test VALUES ("a", 'b', 'c')`, true, ``, nil},
		{"Values / With columns", `INSERT INTO test (a, b, c) VALUES ('a', 'b', 'c')`, false, "1,a,b,c\n", nil},
		{"Values / Ident", `INSERT INTO test (a) VALUES (a)`, true, ``, nil},
		{"Values / Ident string", `INSERT INTO test (a) VALUES ("a")`, true, ``, nil},
		{"Values / With fields ident string", `INSERT INTO test (a, "foo bar") VALUES ('c', 'd')`, false, "1,c,d\n", nil},
		{"Values / Positional Params", "INSERT INTO test (a, b, c) VALUES (?, 'e', ?)", false, "1,d,e,f\n", []interface{}{"d", "f"}},
		{"Values / Named Params", "INSERT INTO test (a, b, c) VALUES ($d, 'e', $f)", false, "1,d,e,f\n", []interface{}{sql.Named("f", "f"), sql.Named("d", "d")}},
		{"Values / Invalid params", "INSERT INTO test (a, b, c) VALUES ('d', ?)", true, "", []interface{}{'e'}},
		{"Values / List", `INSERT INTO test (a, b, c) VALUES ("a", 'b', (1, 2, 3))`, true, "", nil},
		{"Records", "INSERT INTO test RECORDS (a: 'a', b: 2.3, c: 1 = 1)", false, "1,a,2.3,true\n", nil},
		{"Records / Positional Params", "INSERT INTO test RECORDS (a: ?, b: 2.3, c: ?)", false, "1,a,2.3,true\n", []interface{}{"a", true}},
		{"Records / Named Params", "INSERT INTO test RECORDS (a: $a, b: 2.3, c: $c)", false, "1,1,2.3,true\n", []interface{}{sql.Named("c", true), sql.Named("a", 1)}},
		{"Records / List ", "INSERT INTO test RECORDS (a: (1, 2, 3))", true, "", nil},
		{"Records / strings", `INSERT INTO test RECORDS ('a': 'a', b: 2.3)`, true, "", nil},
		{"Records / ident value", `INSERT INTO test RECORDS ("a": "a")`, true, "", nil},
	}

	for _, test := range tests {
		testFn := func(withIndexes bool) func(t *testing.T) {
			return func(t *testing.T) {
				db, err := New(memoryengine.NewEngine())
				require.NoError(t, err)
				defer db.Close()

				err = db.Exec("CREATE TABLE test")
				require.NoError(t, err)
				if withIndexes {
					err = db.Exec(`
						CREATE INDEX idx_a ON test (a);
						CREATE INDEX idx_b ON test (b);
						CREATE INDEX idx_c ON test (c);
					`)
					require.NoError(t, err)
				}
				err = db.Exec(test.query, test.params...)
				if test.fails {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				st, err := db.Query("SELECT key(), * FROM test")
				require.NoError(t, err)
				defer st.Close()

				var buf bytes.Buffer
				err = record.IteratorToCSV(&buf, st)
				require.NoError(t, err)
				require.Equal(t, test.expected, buf.String())
			}
		}

		t.Run("No Index/"+test.name, testFn(false))
		t.Run("With Index/"+test.name, testFn(true))
	}

	t.Run("with primary key", func(t *testing.T) {
		db, err := New(memoryengine.NewEngine())
		require.NoError(t, err)
		defer db.Close()

		err = db.Exec("CREATE TABLE test (foo INTEGER PRIMARY KEY)")
		require.NoError(t, err)

		err = db.Exec(`INSERT INTO test (bar) VALUES (1)`)
		require.Error(t, err)
		err = db.Exec(`INSERT INTO test (bar, foo) VALUES (1, 2)`)
		require.NoError(t, err)

		err = db.Exec(`INSERT INTO test (bar, foo) VALUES (1, 2)`)
		require.Equal(t, err, ErrDuplicateRecord)
	})

	t.Run("with shadowing", func(t *testing.T) {
		db, err := New(memoryengine.NewEngine())
		require.NoError(t, err)
		defer db.Close()

		err = db.Exec("CREATE TABLE test")
		require.NoError(t, err)

		err = db.Exec(`INSERT INTO test ("key()", "key") VALUES (1, 2)`)
		require.NoError(t, err)
	})
}
