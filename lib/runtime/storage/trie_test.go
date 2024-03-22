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
	"github.com/ChainSafe/gossamer/pkg/trie"
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

func TestTrieState_WithAndWithoutTransactions(t *testing.T) {
	cases := map[string]struct {
		changes func(t *testing.T, ts *TrieState)
		checks  func(t *testing.T, ts *TrieState)
	}{
		"set_get": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					err := ts.Put([]byte(tc), []byte(tc))
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					res := ts.Get([]byte(tc))
					require.Equal(t, []byte(tc), res)
				}
			},
		},
		"set_child_storage": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					err := ts.SetChildStorage([]byte(tc), []byte(tc), []byte(tc))
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					res, err := ts.GetChildStorage([]byte(tc), []byte(tc))
					require.NoError(t, err)
					require.Equal(t, []byte(tc), res)
				}
			},
		},
		"set_and_clear_from_child": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					err := ts.SetChildStorage([]byte(tc), []byte(tc), []byte(tc))
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					err := ts.ClearChildStorage([]byte(tc), []byte(tc))
					require.NoError(t, err)

					val, _ := ts.GetChildStorage([]byte(tc), []byte(tc))
					require.Nil(t, val)
				}
			},
		},
		"delete": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					ts.Put([]byte(tc), []byte(tc))
				}
			},
			checks: func(t *testing.T, ts *TrieState) {
				ts.Delete([]byte(testCases[0]))
				has := ts.Has([]byte(testCases[0]))
				require.False(t, has)
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			t.Run("without_transactions", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(trie.NewEmptyTrie())
				tt.changes(t, ts)
				tt.checks(t, ts)
			})

			t.Run("during_transaction", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(trie.NewEmptyTrie())
				ts.StartTransaction()
				tt.changes(t, ts)
				tt.checks(t, ts)
				ts.CommitTransaction()
			})

			t.Run("after_transaction_commited", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(trie.NewEmptyTrie())
				ts.StartTransaction()
				tt.changes(t, ts)
				ts.CommitTransaction()
				tt.checks(t, ts)
			})
		})
	}
}

func TestTrieState_ClearPrefix(t *testing.T) {
	ts := NewTrieState(trie.NewEmptyTrie())

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
	ts := NewTrieState(trie.NewEmptyTrie())
	child := trie.NewEmptyTrie()

	keys := [][]byte{
		[]byte("noot"),
		[]byte("noodle"),
		[]byte("other"),
	}

	keyToChild := []byte("keytochild")

	for i, key := range keys {
		ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
		child.Put(key, []byte{byte(i)})
	}

	err := ts.ClearPrefixInChild(keyToChild, []byte("noo"))
	require.NoError(t, err)

	for i, key := range keys {
		val, err := ts.GetChildStorage(keyToChild, key)
		require.NoError(t, err)
		if i < 2 {
			require.Nil(t, val)
		} else {
			require.NotNil(t, val)
		}
	}
}

func TestTrieState_NextKey(t *testing.T) {
	ts := NewTrieState(trie.NewEmptyTrie())
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

func TestTrieState_DeleteChildLimit(t *testing.T) {
	ts := NewTrieState(trie.NewEmptyTrie())

	keys := [][]byte{
		[]byte("key3"),
		[]byte("key1"),
		[]byte("key2"),
	}

	keyToChild := []byte("keytochild")

	for i, key := range keys {
		ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
	}

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

func TestTrieState_Root(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Put([]byte(tc), []byte(tc))
		}

		expected := ts.MustRoot()
		require.Equal(t, expected, ts.MustRoot())
	}

	ts := NewTrieState(trie.NewEmptyTrie())
	testFunc(ts)
}

func TestTrieState_NestedTransactions(t *testing.T) {
	cases := map[string]struct {
		createTrieState func() *TrieState
		assert          func(*testing.T, *TrieState)
	}{
		"committing_and_rollback_on_nested_transactions": {
			createTrieState: func() *TrieState {
				ts := NewTrieState(trie.NewEmptyTrie())

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
			assert: func(t *testing.T, ts *TrieState) {
				require.NotNil(t, ts.Get([]byte("key-1")))
				require.NotNil(t, ts.Get([]byte("key-2")))
				require.NotNil(t, ts.Get([]byte("key-3")))

				require.Nil(t, ts.Get([]byte("key-4")))
				require.Equal(t, 0, ts.transactions.Len())
			},
		},
		"committing_all_nested_transactions": {
			createTrieState: func() *TrieState {
				ts := NewTrieState(trie.NewEmptyTrie())
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
			assert: func(t *testing.T, ts *TrieState) {
				require.NotNil(t, ts.Get([]byte("key-1")))
				require.NotNil(t, ts.Get([]byte("key-2")))
				require.NotNil(t, ts.Get([]byte("key-4")))
				require.Equal(t, 0, ts.transactions.Len())
			},
		},
		"rollback_without_transaction_should_panic": {
			createTrieState: func() *TrieState {
				return NewTrieState(trie.NewEmptyTrie())
			},
			assert: func(t *testing.T, ts *TrieState) {
				require.PanicsWithValue(t, "no transactions to rollback", func() { ts.RollbackTransaction() })
			},
		},
		"commit_without_transaction_should_panic": {
			createTrieState: func() *TrieState {
				return NewTrieState(trie.NewEmptyTrie())
			},
			assert: func(t *testing.T, ts *TrieState) {
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
