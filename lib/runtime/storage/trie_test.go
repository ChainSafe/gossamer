// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

var testCases = []string{
	"asdf",
	"ghjk",
	"qwerty",
	"uiopl",
	"zxcv",
	"bnm",
}

func TestTrieState_SetGet(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		for _, tc := range testCases {
			res := ts.Get([]byte(tc))
			require.Equal(t, []byte(tc), res)
		}
	}

	ts := &TrieState{t: trie.NewEmptyTrie()}
	testFunc(ts)
}

func TestTrieState_SetGetChildStorage(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	for _, tc := range testCases {
		childTrie := trie.NewEmptyTrie()
		err := ts.SetChild([]byte(tc), childTrie)
		require.NoError(t, err)

		err = ts.SetChildStorage([]byte(tc), []byte(tc), []byte(tc))
		require.NoError(t, err)
	}

	for _, tc := range testCases {
		res, err := ts.GetChildStorage([]byte(tc), []byte(tc))
		require.NoError(t, err)
		require.Equal(t, []byte(tc), res)
	}
}

func TestTrieState_SetAndClearFromChild(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			childTrie := trie.NewEmptyTrie()
			err := ts.SetChild([]byte(tc), childTrie)
			require.NoError(t, err)

			err = ts.SetChildStorage([]byte(tc), []byte(tc), []byte(tc))
			require.NoError(t, err)
		}

		for _, tc := range testCases {
			err := ts.ClearChildStorage([]byte(tc), []byte(tc))
			require.NoError(t, err)

			_, err = ts.GetChildStorage([]byte(tc), []byte(tc))
			require.ErrorContains(t, err, "child trie does not exist at key")
		}
	}

	ts := &TrieState{t: trie.NewEmptyTrie()}
	testFunc(ts)
}

func TestTrieState_Delete(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		ts.Delete([]byte(testCases[0]))
		has := ts.Has([]byte(testCases[0]))
		require.False(t, has)
	}

	ts := &TrieState{t: trie.NewEmptyTrie()}
	testFunc(ts)
}

func TestTrieState_Root(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		expected := ts.MustRoot(trie.NoMaxInlineValueSize)
		require.Equal(t, expected, ts.MustRoot(trie.NoMaxInlineValueSize))
	}

	ts := &TrieState{t: trie.NewEmptyTrie()}
	testFunc(ts)
}

func TestTrieState_ClearPrefix(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	for i, key := range keys {
		ts.Put([]byte(key), []byte{byte(i)})
	}

	ts.ClearPrefix([]byte("noo"))

	for i, key := range keys {
		val := ts.Get([]byte(key))
		if i < 2 {
			require.Nil(t, val)
		} else {
			require.NotNil(t, val)
		}
	}
}

func TestTrieState_ClearPrefixInChild(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}
	child := trie.NewEmptyTrie()

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	for i, key := range keys {
		child.Put([]byte(key), []byte{byte(i)})
	}

	keyToChild := []byte("keytochild")

	err := ts.SetChild(keyToChild, child)
	require.NoError(t, err)

	err = ts.ClearPrefixInChild(keyToChild, []byte("noo"))
	require.NoError(t, err)

	for i, key := range keys {
		val, err := ts.GetChildStorage(keyToChild, []byte(key))
		require.NoError(t, err)
		if i < 2 {
			require.Nil(t, val)
		} else {
			require.NotNil(t, val)
		}
	}
}

func TestTrieState_NextKey(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	sort.Slice(testCases, func(i, j int) bool {
		return bytes.Compare([]byte(testCases[i]), []byte(testCases[j])) == -1
	})

	for i, tc := range testCases {
		next := ts.NextKey([]byte(tc))
		if i == len(testCases)-1 {
			require.Nil(t, next)
		} else {
			require.Equal(t, []byte(testCases[i+1]), next, common.BytesToHex([]byte(tc)))
		}
	}
}

func TestTrieState_CommitStorageTransaction(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	ts.BeginStorageTransaction()
	testValue := []byte("noot")
	ts.Put([]byte(testCases[0]), testValue)
	ts.CommitStorageTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, testValue, val)
}

func TestTrieState_RollbackStorageTransaction(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	ts.BeginStorageTransaction()
	testValue := []byte("noot")
	ts.Put([]byte(testCases[0]), testValue)
	ts.RollbackStorageTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, []byte(testCases[0]), val)
}

func TestTrieState_DeleteChildLimit(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}
	child := trie.NewEmptyTrie()

	keys := []string{
		"key3",
		"key1",
		"key2",
	}

	for i, key := range keys {
		child.Put([]byte(key), []byte{byte(i)})
	}

	keyToChild := []byte("keytochild")

	err := ts.SetChild(keyToChild, child)
	require.NoError(t, err)

	testLimitBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(testLimitBytes, uint32(2))
	optLimit2 := &testLimitBytes

	testCases := []struct {
		key             []byte
		limit           *[]byte
		expectedDeleted uint32
		expectedDelAll  bool
		errMsg          string
	}{
		{
			key:             []byte("fakekey"),
			limit:           optLimit2,
			expectedDeleted: 0,
			expectedDelAll:  false,
			errMsg:          fmt.Sprintf("child trie does not exist at key 0x%x", ":child_storage:default:fakekey"),
		},
		{key: []byte("keytochild"), limit: optLimit2, expectedDeleted: 2, expectedDelAll: false},
		{key: []byte("keytochild"), limit: nil, expectedDeleted: 1, expectedDelAll: true},
	}
	for _, test := range testCases {
		deleted, all, err := ts.DeleteChildLimit(test.key, test.limit)
		if test.errMsg != "" {
			require.Error(t, err)
			require.EqualError(t, err, test.errMsg)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, test.expectedDeleted, deleted)
		require.Equal(t, test.expectedDelAll, all)
	}
}
