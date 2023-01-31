// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package memory provides an in-memory database implementation.
package memory

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ChainSafe/gossamer/internal/database"
)

// Database is an in-memory database implementation.
type Database struct {
	closed    *atomic.Bool
	keyValues map[string][]byte
	mutex     sync.RWMutex
}

// New returns a new in-memory database.
func New() *Database {
	return &Database{
		closed:    new(atomic.Bool),
		keyValues: make(map[string][]byte),
	}
}

// Get retrieves a value from the database using the given key.
// It returns `ErrKeyNotFound` if the key is not found.
func (db *Database) Get(key []byte) (value []byte, err error) {
	if db.closed.Load() {
		return nil, fmt.Errorf("%w", database.ErrClosed)
	}

	db.mutex.RLock()
	defer db.mutex.RUnlock()
	value, ok := db.keyValues[string(key)]
	if !ok {
		return nil, fmt.Errorf("%w: 0x%x", database.ErrKeyNotFound, key)
	}

	return value, nil
}

// Set sets a value at the given key in the database.
// The value byte slice is deep copied to avoid any mutation surprises.
// The error returned is always nil.
func (db *Database) Set(key, value []byte) (err error) {
	if db.closed.Load() {
		return fmt.Errorf("%w", database.ErrClosed)
	}

	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.keyValues[string(key)] = copyBytes(value)

	return nil
}

// Delete deletes a the given key in the database.
// If the key is not found, no error is returned.
func (db *Database) Delete(key []byte) (err error) {
	if db.closed.Load() {
		return fmt.Errorf("%w", database.ErrClosed)
	}

	db.mutex.Lock()
	defer db.mutex.Unlock()

	delete(db.keyValues, string(key))

	return nil
}

// NewWriteBatch returns a new write batch for the database.
// It is not thread-safe to write to the batch, but flushing it is
// thread-safe for the database.
func (db *Database) NewWriteBatch() (writeBatch database.WriteBatch) {
	const prefix = ""
	return newWriteBatch(prefix, db)
}

// NewTable returns a new table using the database.
// All keys on the table will be prefixed with the given prefix.
func (db *Database) NewTable(prefix string) (writeBatch database.Table) {
	return &table{
		prefix:   prefix,
		database: db,
	}
}

// Stream streams data from the database to the `handle`
// function given. The `prefix` is used to filter the keys
// as well as the `chooseKey` function. Note the whole stream
// operation locks the database for reading.
func (db *Database) Stream(_ context.Context, prefix []byte,
	chooseKey func(key []byte) bool,
	handle func(key, value []byte) error) (err error) {
	if db.closed.Load() {
		return fmt.Errorf("%w", database.ErrClosed)
	}

	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for keyString, value := range db.keyValues {
		key := []byte(keyString)
		if !bytes.HasPrefix(key, prefix) || !chooseKey(key) {
			continue
		}

		if err := handle(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the database.
func (db *Database) Close() (err error) {
	closed := db.closed.Swap(true)
	if closed {
		return fmt.Errorf("%w", database.ErrClosed)
	}

	db.keyValues = nil
	return nil
}

// DropAll drops all data from the database.
func (db *Database) DropAll() (err error) {
	if db.closed.Load() {
		return fmt.Errorf("%w", database.ErrClosed)
	}

	db.keyValues = make(map[string][]byte)
	return nil
}
