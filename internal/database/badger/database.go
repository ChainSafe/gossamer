// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package badger provides a database implementation using badger v3.
package badger

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/database"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/ristretto/z"
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

	badgerOptions := badger.DefaultOptions(*settings.Path)
	badgerOptions = badgerOptions.WithLogger(nil)
	badgerOptions = badgerOptions.WithInMemory(*settings.InMemory)
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

	return value, transformError(err)
}

// Set sets a value at the given key in the database.
func (db *Database) Set(key, value []byte) (err error) {
	err = db.badgerDatabase.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	return transformError(err)
}

// Delete deletes the given key from the database.
// If the key is not found, no error is returned.
func (db *Database) Delete(key []byte) (err error) {
	err = db.badgerDatabase.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	return transformError(err)
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

// Stream streams data from the database to the `handle`
// function given. The `prefix` is used to filter the keys
// as well as the `chooseKey` function. Note the `prefix`
// argument is more performant than checking the prefix within
// the `chooseKey` function.
func (db *Database) Stream(ctx context.Context,
	prefix []byte,
	chooseKey func(key []byte) bool,
	handle func(key, value []byte) error,
) error {
	stream := db.badgerDatabase.NewStream()

	if prefix != nil {
		stream.Prefix = make([]byte, len(prefix))
		copy(stream.Prefix, prefix)
	}

	stream.ChooseKey = func(item *badger.Item) bool {
		key := item.Key()
		return chooseKey(key)
	}

	stream.Send = func(buf *z.Buffer) (err error) {
		kvList, err := badger.BufferToKVList(buf)
		if err != nil {
			return fmt.Errorf("decoding badger proto key value: %w", err)
		}

		for _, keyValue := range kvList.Kv {
			err = handle(keyValue.Key, keyValue.Value)
			if err != nil {
				return fmt.Errorf("handling key value: %w", err)
			}
		}
		return nil
	}

	return stream.Orchestrate(ctx)
}

// Close closes the database.
func (db *Database) Close() (err error) {
	err = db.badgerDatabase.Close()
	return transformError(err)
}

// DropAll drops all data from the database.
func (db *Database) DropAll() (err error) {
	err = db.badgerDatabase.DropAll()
	return transformError(err)
}
