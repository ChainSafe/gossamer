// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package test

import (
	"bytes"
	"context"
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/database/badger"
	"github.com/ChainSafe/gossamer/internal/database/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Database interface {
	Get(key []byte) (value []byte, err error)
	Set(key, value []byte) error
	Delete(key []byte) error
	NewWriteBatch() database.WriteBatch
	NewTable(prefix string) database.Table
	Stream(ctx context.Context, prefix []byte, chooseKey func(key []byte) bool,
		handle func(key []byte, value []byte) error) error
	DropAll() error
	Close() error
}

func Test_Databases(t *testing.T) {
	t.Parallel()

	databaseBuilders := []func() Database{
		func() Database { return memory.New() },
		func() Database {
			settings := badger.Settings{}.WithInMemory(true)
			database, err := badger.New(settings)
			require.NoError(t, err)
			return database
		},
	}

	for _, databaseBuilder := range databaseBuilders {
		db := databaseBuilder()

		err := db.Set([]byte{1}, []byte{2})
		require.NoError(t, err)

		value, err := db.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, []byte{2}, value)

		err = db.Delete([]byte{2})
		require.NoError(t, err)

		err = db.Delete([]byte{1})
		require.NoError(t, err)

		_, err = db.Get([]byte{1})
		require.ErrorIs(t, err, database.ErrKeyNotFound)

		err = db.Set([]byte{1}, []byte{2})
		require.NoError(t, err)

		value, err = db.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, []byte{2}, value)

		batch := db.NewWriteBatch()
		err = batch.Set([]byte{3}, []byte{4})
		require.NoError(t, err)
		err = batch.Set([]byte{4}, []byte{5})
		require.NoError(t, err)
		err = batch.Flush()
		require.NoError(t, err)
		value, err = db.Get([]byte{3})
		require.NoError(t, err)
		assert.Equal(t, []byte{4}, value)
		value, err = db.Get([]byte{4})
		require.NoError(t, err)
		assert.Equal(t, []byte{5}, value)

		table := db.NewTable("x")
		err = table.Set([]byte{1}, []byte{3})
		require.NoError(t, err)
		value, err = table.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, []byte{3}, value)
		value, err = db.Get([]byte("x\x01"))
		require.NoError(t, err)
		assert.Equal(t, []byte{3}, value)

		err = db.DropAll()
		require.NoError(t, err)

		_, err = db.Get([]byte{1})
		require.ErrorIs(t, err, database.ErrKeyNotFound)

		streamTest(t, db)

		err = db.Close()
		require.NoError(t, err)

		err = db.Set([]byte{1}, []byte{2})
		assert.ErrorIs(t, err, database.ErrClosed)
	}
}

func streamTest(t *testing.T, db Database) {
	err := db.DropAll()
	require.NoError(t, err)

	keyValues := map[string][]byte{
		"prefix_1":  {1},
		"prefix_12": {1, 2},
		"prefix_3":  {3},
		"4":         {4},
	}
	for keyString, value := range keyValues {
		err := db.Set([]byte(keyString), value)
		require.NoError(t, err)
	}

	ctx := context.Background()
	prefix := []byte("prefix")
	chooseKey := func(key []byte) bool {
		keyWithoutPrefix := bytes.TrimPrefix(key, prefix)
		return keyWithoutPrefix[1] == '1'
	}
	expected := map[string][]byte{
		"prefix_1":  {1},
		"prefix_12": {1, 2},
	}
	handle := func(key []byte, value []byte) error {
		keyString := string(key)
		expectedValue, ok := expected[keyString]
		require.True(t, ok)
		assert.Equal(t, expectedValue, value)
		delete(expected, keyString)
		return nil
	}

	err = db.Stream(ctx, prefix, chooseKey, handle)
	require.NoError(t, err)

	assert.Empty(t, expected)
}
