// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Database(t *testing.T) {
	t.Parallel()

	settings := Settings{
		Path: t.TempDir(),
	}
	db, err := New(settings)
	require.NoError(t, err)

	err = db.Set([]byte{1}, []byte{2})
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
	assert.ErrorIs(t, err, badger.ErrDBClosed)
}

func Test_New(t *testing.T) {
	t.Parallel()

	settings := Settings{
		Path: t.TempDir(),
	}
	database, err := New(settings)
	require.NoError(t, err)

	err = database.Close()
	require.NoError(t, err)
}

func Test_Database_Get(t *testing.T) {
	t.Parallel()

	t.Run("get error", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		_, err = db.Get([]byte{})
		assert.ErrorIs(t, err, badger.ErrEmptyKey)
		assert.EqualError(t, err, "getting item from transaction: Key cannot be empty")
	})

	t.Run("key not found", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		_, err = db.Get([]byte{1})
		assert.ErrorIs(t, err, database.ErrKeyNotFound)
		assert.EqualError(t, err, "key not found: 0x01")
	})

	t.Run("key found", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		key := []byte{1}
		value := []byte{2}

		err = db.badgerDatabase.Update(func(txn *badger.Txn) error {
			return txn.Set(key, value)
		})
		require.NoError(t, err)

		valueRetrieved, err := db.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, value, valueRetrieved)

		// Check for mutation safety
		value[0]++
		assert.NotEqual(t, value, valueRetrieved)

		valueRetrieved[0]++
		valueRetrievedAgain, err := db.Get([]byte{1})
		require.NoError(t, err)
		assert.NotEqual(t, valueRetrieved, valueRetrievedAgain)
	})
}

func Test_Database_Set(t *testing.T) {
	t.Parallel()

	t.Run("set error", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		err = db.Set([]byte{}, []byte{2})
		assert.ErrorIs(t, err, badger.ErrEmptyKey)
		assert.EqualError(t, err, "Key cannot be empty")
	})

	t.Run("set new key", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		err = db.Set([]byte{1}, []byte{2})
		require.NoError(t, err)

		value, err := db.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, []byte{2}, value)
	})

	t.Run("override at existing key", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		err = db.badgerDatabase.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte{1}, []byte{2})
		})
		require.NoError(t, err)

		value := []byte{3}
		err = db.Set([]byte{1}, value)
		require.NoError(t, err)

		valueRetrieved, err := db.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, []byte{3}, valueRetrieved)

		// Check for mutation safety
		value[0]++
		assert.NotEqual(t, value, valueRetrieved)
		valueRetrieved, err = db.Get([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, []byte{3}, valueRetrieved)
	})
}

func Test_Database_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		err = db.Delete([]byte{})
		assert.ErrorIs(t, err, badger.ErrEmptyKey)
		assert.EqualError(t, err, "Key cannot be empty")
	})

	t.Run("key not found", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		err = db.Delete([]byte{1})
		require.NoError(t, err)
	})

	t.Run("delete existing key", func(t *testing.T) {
		t.Parallel()

		db, err := New(Settings{Path: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := db.Close()
			require.NoError(t, err)
		})

		err = db.badgerDatabase.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte{1}, []byte{2})
		})
		require.NoError(t, err)

		err = db.Delete([]byte{1})
		require.NoError(t, err)

		_, err = db.Get([]byte{1})
		require.ErrorIs(t, err, database.ErrKeyNotFound)
	})
}

func Test_Database_NewWriteBatch(t *testing.T) {
	t.Parallel()

	db, err := New(Settings{Path: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	writeBatch := db.NewWriteBatch()

	err = writeBatch.Set([]byte{1}, []byte{1})
	require.NoError(t, err)
	writeBatch.Cancel()

	writeBatch = db.NewWriteBatch()
	err = writeBatch.Set([]byte{2}, []byte{2})
	require.NoError(t, err)
	err = writeBatch.Set([]byte{3}, []byte{3})
	require.NoError(t, err)
	err = writeBatch.Delete([]byte{2})
	require.NoError(t, err)
	err = writeBatch.Flush()
	require.NoError(t, err)

	_, err = db.Get([]byte{1})
	require.ErrorIs(t, err, database.ErrKeyNotFound)
	_, err = db.Get([]byte{2})
	require.ErrorIs(t, err, database.ErrKeyNotFound)
	value, err := db.Get([]byte{3})
	require.NoError(t, err)
	assert.Equal(t, []byte{3}, value)
}

func Test_Database_NewTable(t *testing.T) {
	t.Parallel()

	db, err := New(Settings{Path: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	const prefix = "prefix"
	prefixBytes := []byte(prefix)
	table := db.NewTable(prefix)

	// err = table.Set([]byte{1}, []byte{1})
	// require.NoError(t, err)
	// assertDBValue(t, db, append(prefixBytes, []byte{1}...), []byte{1})
	// err = table.Delete([]byte{1})
	// require.NoError(t, err)

	writeBatch := table.NewWriteBatch()
	err = writeBatch.Set([]byte{1}, []byte{1})
	require.NoError(t, err)
	err = writeBatch.Set([]byte{2}, []byte{2})
	require.NoError(t, err)
	err = writeBatch.Set([]byte{3}, []byte{3})
	require.NoError(t, err)
	err = writeBatch.Delete([]byte{2})
	require.NoError(t, err)
	err = writeBatch.Flush()
	require.NoError(t, err)

	// assertDBValue(t, db, append(prefixBytes, []byte{1}...), []byte{1})
	// assertDBKeyNotFound(t, db, append(prefixBytes, []byte{2}...))
	assertDBValue(t, db, append(prefixBytes, []byte{3}...), []byte{3})
}
