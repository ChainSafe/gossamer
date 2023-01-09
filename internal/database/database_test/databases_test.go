// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package test

import (
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
	DropAll() error
	Close() error
}

func Test_Databases(t *testing.T) {
	t.Parallel()

	databaseBuilders := []func() Database{
		func() Database { return memory.New() },
		func() Database {
			settings := badger.Settings{
				Path: t.TempDir(),
			}
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

		err = db.Close()
		require.NoError(t, err)

		err = db.Set([]byte{1}, []byte{2})
		assert.ErrorIs(t, err, database.ErrClosed)
	}
}
