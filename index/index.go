package index

import (
	"bytes"
	"errors"

	"github.com/asdine/genji/engine"
)

const (
	separator byte = 0x1E
)

var (
	// ErrDuplicate is returned when a value is already associated with a key
	ErrDuplicate = errors.New("duplicate")
)

// An Index associates encoded values with keys.
// It is sorted by value following the lexicographic order.
type Index interface {
	// Set associates a value with a key.
	Set(value []byte, key []byte) error

	// Delete all the references to the key from the index.
	Delete(value []byte, key []byte) error

	// AscendGreaterOrEqual seeks for the pivot and then goes through all the subsequent key value pairs in increasing order and calls the given function for each pair.
	// If the given function returns an error, the iteration stops and returns that error.
	// If the pivot is nil, starts from the beginning.
	AscendGreaterOrEqual(pivot []byte, fn func(value []byte, key []byte) error) error

	// DescendLessOrEqual seeks for the pivot and then goes through all the subsequent key value pairs in descreasing order and calls the given function for each pair.
	// If the given function returns an error, the iteration stops and returns that error.
	// If the pivot is nil, starts from the end.
	DescendLessOrEqual(pivot []byte, fn func(value, key []byte) error) error
}

// Options of the index.
type Options struct {
	// If set to true, values will be associated with at most one key. False by default.
	Unique bool
}

// New creates an index with the given store and options.
func New(store engine.Store, opts Options) Index {
	if opts.Unique {
		return &uniqueIndex{
			store: store,
		}
	}

	return &listIndex{
		store: store,
	}
}

// listIndex is an implementation that associates a value with a list of keys.
type listIndex struct {
	store engine.Store
}

// Set associates a value with a key. It is possible to associate multiple keys for the same value
// but a key can be associated to only one value.
func (i *listIndex) Set(value, key []byte) error {
	if len(value) == 0 {
		return errors.New("value cannot be nil")
	}

	buf := make([]byte, 0, len(value)+len(key)+1)
	buf = append(buf, value...)
	buf = append(buf, separator)
	buf = append(buf, key...)

	return i.store.Put(buf, nil)
}

func (i *listIndex) Delete(value, key []byte) error {
	buf := make([]byte, 0, len(value)+len(key)+1)
	buf = append(buf, value...)
	buf = append(buf, separator)
	buf = append(buf, key...)

	return i.store.Delete(buf)
}

func (i *listIndex) AscendGreaterOrEqual(pivot []byte, fn func(value []byte, key []byte) error) error {
	return i.store.AscendGreaterOrEqual(pivot, func(k, v []byte) error {
		idx := bytes.LastIndexByte(k, separator)
		return fn(k[:idx], k[idx+1:])
	})
}

func (i *listIndex) DescendLessOrEqual(pivot []byte, fn func(k, v []byte) error) error {
	if len(pivot) > 0 {
		// ensure the pivot is bigger than the requested value so it doesn't get skipped.
		pivot = append(pivot, separator, 0xFF)
	}
	return i.store.DescendLessOrEqual(pivot, func(k, v []byte) error {
		idx := bytes.LastIndexByte(k, separator)
		return fn(k[:idx], k[idx+1:])
	})
}

// uniqueIndex is an implementation that associates a value with a exactly one key.
type uniqueIndex struct {
	store engine.Store
}

// Set associates a value with exactly one key.
// If the association already exists, it returns an error.
func (i *uniqueIndex) Set(value []byte, key []byte) error {
	if len(value) == 0 {
		return errors.New("value cannot be nil")
	}

	_, err := i.store.Get(value)
	if err == nil {
		return ErrDuplicate
	}
	if err != engine.ErrKeyNotFound {
		return err
	}

	return i.store.Put(value, key)
}

func (i *uniqueIndex) Delete(value, key []byte) error {
	return i.store.Delete(value)
}

func (i *uniqueIndex) AscendGreaterOrEqual(pivot []byte, fn func(value []byte, key []byte) error) error {
	return i.store.AscendGreaterOrEqual(pivot, fn)
}

func (i *uniqueIndex) DescendLessOrEqual(pivot []byte, fn func(k, v []byte) error) error {
	return i.store.DescendLessOrEqual(pivot, fn)
}
