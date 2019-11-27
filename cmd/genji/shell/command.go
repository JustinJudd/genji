package shell

import (
	"fmt"

	"github.com/asdine/genji"
)

func runTablesCmd(db *genji.DB) error {
	var tables []string
	err := db.View(func(tx *genji.Tx) error {
		var err error

		tables, err = tx.ListTables()
		return err
	})
	if err != nil {
		return err
	}

	for _, t := range tables {
		fmt.Println(t)
	}

	return nil
}
