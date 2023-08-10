// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type testAssertion struct {
	input    string
	expected string
}

func testSetup() []testAssertion {
	tests := []testAssertion{
		{"camel", "camel"},
		{"walrus", "walrus"},
		{"296204", "296204"},
		{"\x00123\x00", "\x00123\x00"},
	}
	return tests
}

func testNewPebble(t *testing.T) Database {
	t.Helper()

	db, err := NewPebble(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	return db
}

func TestPebbleDatabaseImplementations(t *testing.T) {
	db := testNewPebble(t)

	testPutGetter(t, db)
	testHasGetter(t, db)
	testUpdateGetter(t, db)
	testDelGetter(t, db)
	testGetPath(t, db)
}

func TestPebbleDBBatch(t *testing.T) {
	db := testNewPebble(t)
	testBatchPutAndDelete(t, db)
}

func TestPebbleDBIterator(t *testing.T) {
	db := testNewPebble(t)
	testNextKeyIterator(t, db)
	testSeekKeyValueIterator(t, db)
}

func testPutGetter(t *testing.T, db Database) {
	tests := testSetup()
	for _, v := range tests {
		err := db.Put([]byte(v.input), []byte(v.input))
		require.NoError(t, err)

		data, err := db.Get([]byte(v.input))
		require.NoError(t, err)

		require.Equal(t, data, []byte(v.expected))
	}
}

func testHasGetter(t *testing.T, db Database) {
	tests := testSetup()

	for _, v := range tests {
		exists, err := db.Has([]byte(v.input))
		require.NoError(t, err)
		require.True(t, exists)
	}
}

func testUpdateGetter(t *testing.T, db Database) {
	tests := testSetup()

	for _, v := range tests {
		err := db.Put([]byte(v.input), []byte("?"))
		require.NoError(t, err)

		data, err := db.Get([]byte(v.input))
		require.NoError(t, err)

		require.Equal(t, data, []byte("?"))
	}
}

func testDelGetter(t *testing.T, db Database) {
	tests := testSetup()

	for _, v := range tests {
		err := db.Del([]byte(v.input))
		require.NoError(t, err)

		d, _ := db.Get([]byte(v.input))
		require.Greater(t, len(d), 1)
	}
}

func testGetPath(t *testing.T, db Database) {
	dir := db.Path()
	fi, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, fi.IsDir())
}

func testBatchPutAndDelete(t *testing.T, db Database) {
	key := []byte("camel")
	value := []byte("camel-value")

	batch := db.NewBatch()
	err := batch.Put(key, value)
	require.NoError(t, err)

	testFlushAndClose(t, batch, 1)

	deleteBatch := db.NewBatch()
	err = deleteBatch.Del(key)
	require.NoError(t, err)

	retrievedValue, err := db.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, retrievedValue)

	testFlushAndClose(t, deleteBatch, 1)

	_, err = db.Get(key)
	require.ErrorIs(t, err, ErrNotFound)
}

func testFlushAndClose(t *testing.T, batch Batch, expectedSize int) {
	t.Helper()

	err := batch.Flush()
	require.NoError(t, err)

	size := batch.ValueSize()
	require.Equal(t, expectedSize, size)

	batch.Close()
	size = batch.ValueSize()
	require.Equal(t, 0, size)
}

func testIteratorSetup(t *testing.T, db Database) {
	t.Helper()
	batch := db.NewBatch()

	for i := 0; i < 5; i++ {
		key := []byte(fmt.Sprintf("camel-%d", i))
		value := []byte(fmt.Sprintf("camel-value-%d", i))
		err := batch.Put(key, value)
		require.NoError(t, err)
	}

	err := batch.Flush()
	require.NoError(t, err)
}

func testNextKeyIterator(t *testing.T, db Database) {
	testIteratorSetup(t, db)

	it := db.NewIterator()
	defer it.Release()

	counter := 0
	for succ := it.First(); succ; succ = it.Next() {
		require.NotNil(t, it.Key())
		require.NotNil(t, it.Value())
		counter++
	}

	// testIteratorSetup creates 5 entries
	const expected = 5
	require.Equal(t, expected, counter)
}

func testSeekKeyValueIterator(t *testing.T, db Database) {
	testIteratorSetup(t, db)
	kv := map[string]string{
		"camel-0": "camel-value-0",
		"camel-1": "camel-value-1",
		"camel-2": "camel-value-2",
		"camel-3": "camel-value-3",
		"camel-4": "camel-value-4",
	}

	it := db.NewIterator()
	defer it.Release()

	for succ := it.SeekGE([]byte("camel-")); succ; succ = it.Next() {
		expectedValue, ok := kv[string(it.Key())]
		require.True(t, ok)

		require.True(t, it.Valid())
		require.Equal(t, it.Value(), []byte(expectedValue))
	}
}
