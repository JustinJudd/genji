// Package badgerengine implements a Badger engine.
package badgerengine

import (
	"bytes"

	"github.com/asdine/genji/engine"
	"github.com/dgraph-io/badger/v2"
)

const (
	separator   byte = 0x1F
	storeKey         = "__genji.store"
	storePrefix      = 's'
)

// Engine represents a Badger engine.
type Engine struct {
	DB *badger.DB
}

// NewEngine creates a Badger engine. It takes the same argument as Badger's Open function.
func NewEngine(opt badger.Options) (*Engine, error) {
	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	return &Engine{
		DB: db,
	}, nil
}

// Begin creates a transaction using Badger's transaction API.
func (e *Engine) Begin(writable bool) (engine.Transaction, error) {
	tx := e.DB.NewTransaction(writable)

	return &Transaction{
		tx:       tx,
		writable: writable,
	}, nil
}

// Close the engine and underlying Badger database.
func (e *Engine) Close() error {
	return e.DB.Close()
}

// A Transaction uses Badger's transactions.
type Transaction struct {
	tx        *badger.Txn
	writable  bool
	discarded bool
}

// Rollback the transaction. Can be used safely after commit.
func (t *Transaction) Rollback() error {
	t.tx.Discard()

	t.discarded = true
	return nil
}

// Commit the transaction.
func (t *Transaction) Commit() error {
	if t.discarded {
		return badger.ErrDiscardedTxn
	}

	if !t.writable {
		return engine.ErrTransactionReadOnly
	}

	t.discarded = true
	return t.tx.Commit()
}

func buildStoreKey(name string) []byte {
	var buf bytes.Buffer
	buf.Grow(len(storeKey) + 1 + len(name))
	buf.WriteString(storeKey)
	buf.WriteByte(separator)
	buf.WriteString(name)

	return buf.Bytes()
}

func buildStorePrefixKey(name string) []byte {
	prefix := make([]byte, 0, len(name)+3)
	prefix = append(prefix, storePrefix)
	prefix = append(prefix, separator)
	prefix = append(prefix, name...)

	return prefix
}

// Store returns a store by name.
func (t *Transaction) Store(name string) (engine.Store, error) {
	key := buildStoreKey(name)

	_, err := t.tx.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, engine.ErrStoreNotFound
		}

		return nil, err
	}

	pkey := buildStorePrefixKey(name)

	return &Store{
		tx:       t.tx,
		prefix:   pkey,
		writable: t.writable,
		name:     name,
	}, nil
}

// CreateStore creates a store.
// If the store already exists, returns engine.ErrStoreAlreadyExists.
func (t *Transaction) CreateStore(name string) error {
	if !t.writable {
		return engine.ErrTransactionReadOnly
	}

	key := buildStoreKey(name)
	_, err := t.tx.Get(key)
	if err == nil {
		return engine.ErrStoreAlreadyExists
	}
	if err != badger.ErrKeyNotFound {
		return err
	}

	return t.tx.Set(key, nil)
}

// DropStore deletes the store and all its keys.
func (t *Transaction) DropStore(name string) error {
	if !t.writable {
		return engine.ErrTransactionReadOnly
	}

	s, err := t.Store(name)
	if err != nil {
		return err
	}

	err = s.Truncate()
	if err != nil {
		return err
	}

	err = t.tx.Delete(buildStoreKey(name))
	if err == badger.ErrKeyNotFound {
		return engine.ErrStoreNotFound
	}

	return err
}

// ListStores returns a list of all the store names.
func (t *Transaction) ListStores(prefix string) ([]string, error) {
	var names []string

	p := buildStoreKey(prefix)
	it := t.tx.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	for it.Seek(p); it.ValidForPrefix(p); it.Next() {
		names = append(names, string(bytes.TrimPrefix(it.Item().Key(), p[:len(p)-len(prefix)])))
	}

	return names, nil
}
