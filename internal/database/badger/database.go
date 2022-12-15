// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package badger provides a database implementation using badger v3.
package badger

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/dgraph-io/badger/v3"
)

// Database is database implementation using a badger/v3 database.
type Database struct {
	badgerDatabase *badger.DB
}

// New returns a new database based on a badger v3 database.
func New(settings Settings) (database *Database, err error) {
	settings.SetDefaults()
	err = settings.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}

	badgerOptions := badger.DefaultOptions(settings.Path)
	badgerOptions = badgerOptions.WithLogger(nil)
	// TODO enable once we share the same instance
	// See https://github.com/ChainSafe/gossamer/issues/2981
	// badgerOptions = badgerOptions.WithBypassLockGuard(true)
	badgerDatabase, err := badger.Open(badgerOptions)
	if err != nil {
		return nil, fmt.Errorf("opening badger database: %w", err)
	}

	return &Database{
		badgerDatabase: badgerDatabase,
	}, nil
}

// Get retrieves a value from the database using the given key.
// It returns the wrapped error `database.ErrKeyNotFound` if the
// key is not found.
func (db *Database) Get(key []byte) (value []byte, err error) {
	err = db.badgerDatabase.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("getting item from transaction: %w", err)
		}

		value, err = item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("copying value: %w", err)
		}

		return nil
	})

	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("%w: 0x%x", database.ErrKeyNotFound, key)
	}

	return value, err
}

// Set sets a value at the given key in the database.
func (db *Database) Set(key, value []byte) (err error) {
	return db.badgerDatabase.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// Delete deletes the given key from the database.
// If the key is not found, no error is returned.
func (db *Database) Delete(key []byte) (err error) {
	return db.badgerDatabase.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// NewWriteBatch returns a new write batch for the database.
func (db *Database) NewWriteBatch() (writeBatch database.WriteBatch) {
	prefix := []byte(nil)
	badgerWriteBatch := db.badgerDatabase.NewWriteBatch()
	return newWriteBatch(prefix, badgerWriteBatch)
}

// NewTable returns a new table using the database.
// All keys on the table will be prefixed with the given prefix.
func (db *Database) NewTable(prefix string) (dbTable database.Table) {
	return &table{
		prefix:   []byte(prefix),
		database: db,
	}
}

// Close closes the database.
func (db *Database) Close() (err error) {
	return db.badgerDatabase.Close()
}

// DropAll drops all data from the database.
func (db *Database) DropAll() (err error) {
	return db.badgerDatabase.DropAll()
}
