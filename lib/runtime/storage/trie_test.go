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
	"github.com/stretchr/testify/assert"
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
	const stateVersion = trie.V0
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Set([]byte(tc), []byte(tc), stateVersion)
		}

		for _, tc := range testCases {
			res := ts.Get([]byte(tc))
			require.Equal(t, []byte(tc), res)
		}
	}

	ts := &TrieState{t: trie.NewEmptyTrie()}
	testFunc(ts)
}

func TestTrieState_Delete(t *testing.T) {
	const stateVersion = trie.V0
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Set([]byte(tc), []byte(tc), stateVersion)
		}

		ts.Delete([]byte(testCases[0]))
		has := ts.Has([]byte(testCases[0]))
		require.False(t, has)
	}

	ts := &TrieState{t: trie.NewEmptyTrie()}
	testFunc(ts)
}

func TestTrieState_Root(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues    map[string][]byte
		expectedRoot common.Hash
		version      trie.Version
	}{
		"empty v0 trie": {
			expectedRoot: trie.EmptyHash,
			version:      trie.V0,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ts := &TrieState{t: trie.NewEmptyTrie()}
			for k, v := range testCase.keyValues {
				ts.Set([]byte(k), v, testCase.version)
			}

			root, err := ts.Root(testCase.version)
			require.NoError(t, err)

			assert.Equal(t, testCase.expectedRoot, root)
		})
	}
}

func TestTrieState_ClearPrefix(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	const stateVersion = trie.V0
	for i, key := range keys {
		ts.Set([]byte(key), []byte{byte(i)}, stateVersion)
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

	const stateVersion = trie.V0
	for i, key := range keys {
		child.Put([]byte(key), []byte{byte(i)}, stateVersion)
	}

	keyToChild := []byte("keytochild")

	err := ts.SetChild(keyToChild, child, trie.V0)
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

	const stateVersion = trie.V0
	for _, tc := range testCases {
		ts.Set([]byte(tc), []byte(tc), stateVersion)
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

	const stateVersion = trie.V0
	for _, tc := range testCases {
		ts.Set([]byte(tc), []byte(tc), stateVersion)
	}

	ts.BeginStorageTransaction()
	testValue := []byte("noot")
	ts.Set([]byte(testCases[0]), testValue, stateVersion)
	ts.CommitStorageTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, testValue, val)
}

func TestTrieState_RollbackStorageTransaction(t *testing.T) {
	ts := &TrieState{t: trie.NewEmptyTrie()}

	const stateVersion = trie.V0
	for _, tc := range testCases {
		ts.Set([]byte(tc), []byte(tc), stateVersion)
	}

	ts.BeginStorageTransaction()
	testValue := []byte("noot")
	ts.Set([]byte(testCases[0]), testValue, stateVersion)
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

	const stateVersion = trie.V0
	for i, key := range keys {
		child.Put([]byte(key), []byte{byte(i)}, stateVersion)
	}

	keyToChild := []byte("keytochild")

	err := ts.SetChild(keyToChild, child, trie.V0)
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
