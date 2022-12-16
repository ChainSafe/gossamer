// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrTo[T any](value T) *T { return &value }

func assertDBValue(t *testing.T, db *Database, key, expectedValue []byte) {
	t.Helper()

	value, err := db.Get(key)
	require.NoError(t, err)

	assert.Equal(t, expectedValue, value)
}

func assertDBKeyNotFound(t *testing.T, db *Database, key []byte) {
	t.Helper()

	_, err := db.Get(key)
	assert.ErrorIs(t, err, database.ErrKeyNotFound)
}

func logAllKeyValues(t *testing.T, badgerDB *badger.DB) { //nolint:unused,deadcode
	t.Helper()

	keyValues := make(map[string][]byte)
	err := badgerDB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		iterator := txn.NewIterator(opts)
		defer iterator.Close()
		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			item := iterator.Item()
			key := item.Key()
			err := item.Value(func(v []byte) error {
				keyValues[string(key)] = v
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	message := "Database contains the following key value pairs:\n"
	for key, value := range keyValues {
		keyBytes := []byte(key)
		message += fmt.Sprintf("  0x%x <-> 0x%x\n", keyBytes, value)
	}
	t.Log(message)
}
