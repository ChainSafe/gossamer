// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorykvdb

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/internal/kvdb"
	"github.com/stretchr/testify/require"
)

var _ kvdb.KeyValueDB = &MemoryKVDB{}

// / The number of columns required to run `test_delete_prefix`.
const DeletePrefixNumColumns uint32 = 7

func Test_MemoryKVDB(t *testing.T) {
	t.Run("get_fails_with_non_existing_column", func(t *testing.T) {
		db := New(1)
		_, err := db.Get(1, []byte{})
		require.Error(t, err)
	})

	t.Run("put_and_get", func(t *testing.T) {
		db := New(1)
		key1 := []byte("key1")

		var transaction kvdb.DBTransaction
		transaction.Put(0, key1, []byte("horse"))
		require.NoError(t, db.Write(transaction))

		val, err := db.Get(0, key1)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("horse"), val)
	})

	t.Run("delete_and_get", func(t *testing.T) {
		db := New(DeletePrefixNumColumns)
		key1 := []byte("key1")

		var transaction kvdb.DBTransaction = kvdb.NewDBTransaction()
		transaction.Put(0, key1, []byte("horse"))
		require.NoError(t, db.Write(transaction))

		val, err := db.Get(0, key1)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("horse"), val)

		transaction = kvdb.NewDBTransaction()
		transaction.Delete(0, key1)
		err = db.Write(transaction)
		require.NoError(t, err)
		val, err = db.Get(0, key1)
		require.NoError(t, err)
		require.Nil(t, val)
	})

	t.Run("delete_prefix", func(t *testing.T) {
		db := New(DeletePrefixNumColumns)
		keys := [][]byte{
			{},
			{0},
			{0, 1},
			{1},
			{1, 0},
			{1, 255},
			{1, 255, 255},
			{2},
			{2, 0},
			{2, 255},
			bytes.Repeat([]byte{255}, 16),
		}

		var initDB = func(ix uint32) {
			var transaction kvdb.DBTransaction
			for i, key := range keys {
				transaction.Put(ix, key, []byte{uint8(i)})
			}
			err := db.Write(transaction)
			require.NoError(t, err)
		}

		var checkDB = func(ix uint32, content [11]bool) {
			var state [11]bool
			for i, key := range keys {
				val, err := db.Get(ix, key)
				require.NoError(t, err)
				if val != nil {
					state[i] = true
				}
			}
			require.Equal(t, content, state)
		}

		tests := []struct {
			Prefix  []byte
			Content [11]bool
		}{
			// standard
			{
				Prefix:  []byte{1},
				Content: [11]bool{true, true, true, false, false, false, false, true, true, true, true},
			},
			// edge
			{
				Prefix:  []byte{1, 255, 255},
				Content: [11]bool{true, true, true, true, true, true, false, true, true, true, true},
			},
			// none 1
			{
				Prefix:  []byte{1, 2},
				Content: [11]bool{true, true, true, true, true, true, true, true, true, true, true},
			},
			// none 2
			{
				Prefix:  []byte{8},
				Content: [11]bool{true, true, true, true, true, true, true, true, true, true, true},
			},
			// last value
			{
				Prefix:  []byte{255, 255},
				Content: [11]bool{true, true, true, true, true, true, true, true, true, true, false},
			},
			// last value, limit prefix
			{
				Prefix:  []byte{255},
				Content: [11]bool{true, true, true, true, true, true, true, true, true, true, false},
			},
			// all
			{
				Prefix:  []byte{},
				Content: [11]bool{false, false, false, false, false, false, false, false, false, false, false},
			},
		}

		for i, test := range tests {
			ix := uint32(i)
			initDB(ix)
			batch := kvdb.NewDBTransaction()
			batch.DeletePrefix(ix, test.Prefix)
			err := db.Write(batch)
			require.NoError(t, err)
			checkDB(ix, test.Content)
		}
	})

	t.Run("iter", func(t *testing.T) {
		db := New(1)
		key1 := []byte("key1")
		key2 := []byte("key2")

		transaction := kvdb.NewDBTransaction()
		transaction.Put(0, key1, key1)
		transaction.Put(0, key2, key2)
		require.NoError(t, db.Write(transaction))

		var contents []kvdb.DBKeyValue
		for kv, err := range db.Iter(0) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}

		require.Len(t, contents, 2)
		require.Equal(t, kvdb.DBKey(key1), contents[0].Key)
		require.Equal(t, kvdb.DBValue(key1), contents[0].Value)
		require.Equal(t, kvdb.DBKey(key2), contents[1].Key)
		require.Equal(t, kvdb.DBValue(key2), contents[1].Value)
	})

	t.Run("iter_with_prefix", func(t *testing.T) {
		db := New(1)
		key1 := []byte("0")
		key2 := []byte("ab")
		key3 := []byte("abc")
		key4 := []byte("abcd")

		transaction := kvdb.NewDBTransaction()
		transaction.Put(0, key1, key1)
		transaction.Put(0, key2, key2)
		transaction.Put(0, key3, key3)
		transaction.Put(0, key4, key4)
		require.NoError(t, db.Write(transaction))

		// empty prefix
		var contents []kvdb.DBKeyValue
		for kv, err := range db.PrefixIter(0, []byte("")) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Len(t, contents, 4)
		require.Equal(t, kvdb.DBKey(key1), contents[0].Key)
		require.Equal(t, kvdb.DBKey(key2), contents[1].Key)
		require.Equal(t, kvdb.DBKey(key3), contents[2].Key)
		require.Equal(t, kvdb.DBKey(key4), contents[3].Key)

		// prefix a
		contents = nil
		for kv, err := range db.PrefixIter(0, []byte("a")) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Equal(t, kvdb.DBKey(key2), contents[0].Key)
		require.Equal(t, kvdb.DBKey(key3), contents[1].Key)
		require.Equal(t, kvdb.DBKey(key4), contents[2].Key)

		// prefix abc
		contents = nil
		for kv, err := range db.PrefixIter(0, []byte("abc")) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Equal(t, kvdb.DBKey(key3), contents[0].Key)
		require.Equal(t, kvdb.DBKey(key4), contents[1].Key)

		// prefix abcde
		contents = nil
		for kv, err := range db.PrefixIter(0, []byte("abcde")) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Len(t, contents, 0)

		// prefix 0
		contents = nil
		for kv, err := range db.PrefixIter(0, []byte("0")) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Len(t, contents, 1)
		require.Equal(t, kvdb.DBKey(key1), contents[0].Key)
	})

	t.Run("complex", func(t *testing.T) {
		db := New(1)
		key1 := []byte("02c69be41d0b7e40352fc85be1cd65eb03d40ef8427a0ca4596b1ead9a00e9fc")
		key2 := []byte("03c69be41d0b7e40352fc85be1cd65eb03d40ef8427a0ca4596b1ead9a00e9fc")
		key3 := []byte("04c00000000b7e40352fc85be1cd65eb03d40ef8427a0ca4596b1ead9a00e9fc")
		key4 := []byte("04c01111110b7e40352fc85be1cd65eb03d40ef8427a0ca4596b1ead9a00e9fc")
		key5 := []byte("04c02222220b7e40352fc85be1cd65eb03d40ef8427a0ca4596b1ead9a00e9fc")

		transaction := kvdb.NewDBTransaction()
		transaction.Put(0, key1, []byte("cat"))
		transaction.Put(0, key2, []byte("dog"))
		transaction.Put(0, key3, []byte("caterpillar"))
		transaction.Put(0, key4, []byte("beef"))
		transaction.Put(0, key5, []byte("fish"))
		require.NoError(t, db.Write(transaction))

		val, err := db.Get(0, key1)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("cat"), val)

		var contents []kvdb.DBKeyValue
		for kv, err := range db.Iter(0) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Len(t, contents, 5)
		require.Equal(t, kvdb.DBKey(key1), contents[0].Key)
		require.Equal(t, kvdb.DBValue("cat"), contents[0].Value)
		require.Equal(t, kvdb.DBKey(key2), contents[1].Key)
		require.Equal(t, kvdb.DBValue("dog"), contents[1].Value)

		contents = nil
		for kv, err := range db.PrefixIter(0, []byte("04c0")) {
			require.NoError(t, err)
			contents = append(contents, kv)
		}
		require.Len(t, contents, 3)
		require.Equal(t, kvdb.DBValue("caterpillar"), contents[0].Value)
		require.Equal(t, kvdb.DBValue("beef"), contents[1].Value)
		require.Equal(t, kvdb.DBValue("fish"), contents[2].Value)

		transaction = kvdb.NewDBTransaction()
		transaction.Delete(0, key1)
		require.NoError(t, db.Write(transaction))

		val, err = db.Get(0, key1)
		require.NoError(t, err)
		require.Nil(t, val)

		transaction = kvdb.NewDBTransaction()
		transaction.Put(0, key1, []byte("cat"))
		require.NoError(t, db.Write(transaction))

		transaction = kvdb.NewDBTransaction()
		transaction.Put(0, key3, []byte("elephant"))
		transaction.Delete(0, key1)
		require.NoError(t, db.Write(transaction))
		val, err = db.Get(0, key1)
		require.NoError(t, err)
		require.Nil(t, val)
		val, err = db.Get(0, key3)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("elephant"), val)

		val, err = db.PrefixGet(0, key3)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("elephant"), val)
		val, err = db.PrefixGet(0, key2)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("dog"), val)

		transaction = kvdb.NewDBTransaction()
		transaction.Put(0, key1, []byte("horse"))
		transaction.Delete(0, key3)
		require.NoError(t, db.Write(transaction))
		val, err = db.Get(0, key3)
		require.NoError(t, err)
		require.Nil(t, val)
		val, err = db.Get(0, key1)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("horse"), val)

		val, err = db.Get(0, key3)
		require.NoError(t, err)
		require.Nil(t, val)
		val, err = db.Get(0, key1)
		require.NoError(t, err)
		require.Equal(t, kvdb.DBValue("horse"), val)
	})
}
