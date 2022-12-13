// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package memory provides an in-memory database implementation.
package memory

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/database"
)

// Database is an in-memory database implementation.
type Database struct {
	closed    bool
	keyValues map[string][]byte
	mutex     sync.RWMutex
}

// New returns a new in-memory database.
func New() *Database {
	return &Database{
		keyValues: make(map[string][]byte),
	}
}

// Get retrieves a value from the database using the given key.
// It returns `ErrKeyNotFound` if the key is not found.
func (db *Database) Get(key []byte) (value []byte, err error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	db.panicOnClosed()

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
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.panicOnClosed()

	db.keyValues[string(key)] = copyBytes(value)

	return nil
}

// Delete deletes a the given key in the database.
// If the key is not found, no error is returned.
func (db *Database) Delete(key []byte) (err error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.panicOnClosed()

	delete(db.keyValues, string(key))

	return nil
}

// NewWriteBatch returns a new write batch for the database.
// It is not thread-safe to write to the batch, but flushing it is
// thread-safe for the database.
func (db *Database) NewWriteBatch() (writeBatch database.WriteBatch) {
	db.panicOnClosed()
	const prefix = ""
	return newWriteBatch(prefix, db)
}

// NewTable returns a new table using the database.
// All keys on the table will be prefixed with the given prefix.
func (db *Database) NewTable(prefix string) (writeBatch database.Table) {
	db.panicOnClosed()
	return newTable(prefix, db)
}

// Close closes the database.
func (db *Database) Close() (err error) {
	db.closed = true
	db.keyValues = nil
	return nil
}

// DropAll drops all data from the database.
func (db *Database) DropAll() (err error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.panicOnClosed()

	db.keyValues = make(map[string][]byte)
	return nil
}
