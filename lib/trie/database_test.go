// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) chaindb.Database {
	testDatadirPath := t.TempDir()
	db, err := utils.SetupDatabase(testDatadirPath, true)
	require.NoError(t, err)
	return chaindb.NewTable(db, "trie")
}

type keyValue struct {
	key   []byte
	value []byte
}

func getDBKeyValuesA() []keyValue {
	return []keyValue{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}
}

func getDBKeyValuesB() []keyValue {
	return []keyValue{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x30}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
	}
}

func getDBKeyValuesC() []keyValue {
	return []keyValue{
		{key: []byte("asdf"), value: []byte("asdf")},
		{key: []byte("ghjk"), value: []byte("ghjk")},
		{key: []byte("qwerty"), value: []byte("qwerty")},
		{key: []byte("uiopl"), value: []byte("uiopl")},
		{key: []byte("zxcv"), value: []byte("zxcv")},
		{key: []byte("bnm"), value: []byte("bnm")},
	}
}

func TestTrie_DatabaseStoreAndLoad(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
		},
		"second": {
			keyValues: getDBKeyValuesB(),
		},
		"third": {
			keyValues: getDBKeyValuesC(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := NewEmptyTrie()

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
			}

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			res := NewEmptyTrie()
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.MustHash(), res.MustHash())

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
				require.NoError(t, err)
				require.Equal(t, keyValue.value, val)
			}
		})
	}
}

func TestTrie_WriteDirty_Put(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
		},
		"second": {
			keyValues: getDBKeyValuesB(),
		},
		"third": {
			keyValues: getDBKeyValuesC(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := NewEmptyTrie()
			db := newTestDB(t)

			for i, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
				err := trie.WriteDirty(db)
				require.NoError(t, err)

				for j, kv := range testCase.keyValues {
					if j > i {
						break
					}

					val, err := GetFromDB(db, trie.MustHash(), kv.key)
					require.NoError(t, err)
					require.Equal(t, kv.value, val, fmt.Sprintf("key=%x", kv.key))
				}
			}

			err := trie.Store(db)
			require.NoError(t, err)

			trie.Put([]byte("asdf"), []byte("notapenguin"))
			err = trie.WriteDirty(db)
			require.NoError(t, err)

			res := NewEmptyTrie()
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.MustHash(), res.MustHash())

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
				require.NoError(t, err)
				if bytes.Equal(keyValue.key, []byte("asdf")) {
					continue
				}
				require.Equal(t, keyValue.value, val)
			}

			val, err := GetFromDB(db, trie.MustHash(), []byte("asdf"))
			require.NoError(t, err)
			require.Equal(t, []byte("notapenguin"), val)
		})
	}
}

func TestTrie_WriteDirty_PutReplace(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
		},
		"second": {
			keyValues: getDBKeyValuesB(),
		},
		"third": {
			keyValues: getDBKeyValuesC(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := NewEmptyTrie()
			db := newTestDB(t)

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)

				err := trie.WriteDirty(db)
				require.NoError(t, err)
			}

			for _, keyValue := range testCase.keyValues {
				// overwrite existing values
				trie.Put(keyValue.key, keyValue.key)

				err := trie.WriteDirty(db)
				require.NoError(t, err)
			}

			res := NewEmptyTrie()
			err := res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.MustHash(), res.MustHash())

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
				require.NoError(t, err)
				require.Equal(t, keyValue.key, val)
			}
		})
	}
}

func TestTrie_WriteDirty_Delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
		},
		"second": {
			keyValues: getDBKeyValuesB(),
		},
		"third": {
			keyValues: getDBKeyValuesC(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, curr := range testCase.keyValues {
				trie := NewEmptyTrie()

				for _, keyValue := range testCase.keyValues {
					trie.Put(keyValue.key, keyValue.value)
				}

				db := newTestDB(t)
				err := trie.Store(db)
				require.NoError(t, err)

				err = trie.DeleteFromDB(db, curr.key)
				require.NoError(t, err)

				res := NewEmptyTrie()
				err = res.Load(db, trie.MustHash())
				require.NoError(t, err)
				require.Equal(t, trie.MustHash(), res.MustHash())

				for _, keyValue := range testCase.keyValues {
					val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
					require.NoError(t, err)

					if bytes.Equal(keyValue.key, curr.key) {
						require.Nil(t, val, fmt.Sprintf("key=%x", keyValue.key))
						continue
					}

					require.Equal(t, keyValue.value, val)
				}
			}
		})
	}
}

func TestTrie_WriteDirty_ClearPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
		},
		"second": {
			keyValues: getDBKeyValuesB(),
		},
		"third": {
			keyValues: getDBKeyValuesC(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := NewEmptyTrie()

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
			}

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			err = trie.ClearPrefixFromDB(db, []byte{0x01, 0x35})
			require.NoError(t, err)

			res := NewEmptyTrie()
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)

			require.Equal(t, trie.MustHash(), res.MustHash())
		})
	}
}

func TestTrie_GetFromDB(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
		},
		"second": {
			keyValues: getDBKeyValuesB(),
		},
		"third": {
			keyValues: getDBKeyValuesC(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := NewEmptyTrie()

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
			}

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			root := trie.MustHash()

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, root, keyValue.key)
				require.NoError(t, err)
				require.Equal(t, keyValue.value, val)
			}
		})
	}
}
