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
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
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
	testFunc := func(ts *InMemoryTrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		for _, tc := range testCases {
			res := ts.Get([]byte(tc))
			require.Equal(t, []byte(tc), res)
		}
	}

	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
	testFunc(ts)
}

func TestTrieState_SetGetChildStorage(t *testing.T) {
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())

	for _, tc := range testCases {
		childTrie := inmemory.NewEmptyInmemoryTrie()
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
	testFunc := func(ts *InMemoryTrieState) {
		for _, tc := range testCases {
			childTrie := inmemory.NewEmptyInmemoryTrie()
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

	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
	testFunc(ts)
}

func TestTrieState_Delete(t *testing.T) {
	testFunc := func(ts *InMemoryTrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		ts.Delete([]byte(testCases[0]))
		has := ts.Has([]byte(testCases[0]))
		require.False(t, has)
	}

	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
	testFunc(ts)
}

func TestTrieState_Root(t *testing.T) {
	testFunc := func(ts *InMemoryTrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		expected := ts.MustRoot()
		require.Equal(t, expected, ts.MustRoot())
	}

	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
	testFunc(ts)
}

func TestTrieState_ClearPrefix(t *testing.T) {
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())

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
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
	child := inmemory.NewEmptyInmemoryTrie()

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
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
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
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	ts.StartTransaction()
	testValue := []byte("noot")
	ts.Put([]byte(testCases[0]), testValue)
	ts.CommitTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, testValue, val)
}

func TestTrieState_RollbackStorageTransaction(t *testing.T) {
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	ts.StartTransaction()
	testValue := []byte("noot")
	ts.Put([]byte(testCases[0]), testValue)
	ts.RollbackTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, []byte(testCases[0]), val)
}

func TestTrieState_NestedTransactions(t *testing.T) {
	cases := map[string]struct {
		createTrieState func() *InMemoryTrieState
		assert          func(*testing.T, *InMemoryTrieState)
	}{
		"committing_and_rollback_on_nested_transactions": {
			createTrieState: func() *InMemoryTrieState {
				ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())

				ts.Put([]byte("key-1"), []byte("value-1"))
				ts.Put([]byte("key-2"), []byte("value-2"))
				ts.Put([]byte("key-3"), []byte("value-3"))

				{
					ts.StartTransaction()
					ts.Put([]byte("key-4"), []byte("value-4"))

					{
						ts.StartTransaction()
						ts.Delete([]byte("key-3"))
						ts.CommitTransaction() // commit the most nested transaction
					}

					// rollback this transaction will discard the modifications
					// made by the most nested transactions so this original trie
					// should not be affected
					ts.RollbackTransaction()
				}
				return ts
			},
			assert: func(t *testing.T, ts *InMemoryTrieState) {
				require.NotNil(t, ts.Get([]byte("key-1")))
				require.NotNil(t, ts.Get([]byte("key-2")))
				require.NotNil(t, ts.Get([]byte("key-3")))

				require.Nil(t, ts.Get([]byte("key-4")))
				require.Equal(t, 1, ts.transactions.Len())
			},
		},
		"committing_all_nested_transactions": {
			createTrieState: func() *InMemoryTrieState {
				ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
				{
					ts.StartTransaction()
					ts.Put([]byte("key-1"), []byte("value-1"))
					{
						ts.StartTransaction()
						ts.Put([]byte("key-2"), []byte("value-2"))
						{
							ts.StartTransaction()
							ts.Put([]byte("key-3"), []byte("value-3"))
							{
								ts.StartTransaction()
								ts.Put([]byte("key-4"), []byte("value-4"))
								{
									ts.StartTransaction()
									ts.Delete([]byte("key-3"))
									ts.CommitTransaction()
								}
								ts.CommitTransaction()
							}
							ts.CommitTransaction()
						}
						ts.CommitTransaction()
					}
					ts.CommitTransaction()
				}
				return ts
			},
			assert: func(t *testing.T, ts *InMemoryTrieState) {
				require.NotNil(t, ts.Get([]byte("key-1")))
				require.NotNil(t, ts.Get([]byte("key-2")))
				require.NotNil(t, ts.Get([]byte("key-4")))
				require.Equal(t, 1, ts.transactions.Len())
			},
		},
		"rollback_without_transaction_should_panic": {
			createTrieState: func() *InMemoryTrieState {
				return NewTrieState(inmemory.NewEmptyInmemoryTrie())
			},
			assert: func(t *testing.T, ts *InMemoryTrieState) {
				require.PanicsWithValue(t, "no transactions to rollback", func() { ts.RollbackTransaction() })
			},
		},
		"commit_without_transaction_should_panic": {
			createTrieState: func() *InMemoryTrieState {
				return NewTrieState(inmemory.NewEmptyInmemoryTrie())
			},
			assert: func(t *testing.T, ts *InMemoryTrieState) {
				require.PanicsWithValue(t, "no transactions to commit", func() { ts.CommitTransaction() })
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ts := tt.createTrieState()
			tt.assert(t, ts)
		})
	}
}

func TestTrieState_DeleteChildLimit(t *testing.T) {
	ts := NewTrieState(inmemory.NewEmptyInmemoryTrie())
	child := inmemory.NewEmptyInmemoryTrie()

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
