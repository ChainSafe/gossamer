// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
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
	t.Parallel()

	prefixedKeys := [][]byte{
		[]byte("noot"),
		[]byte("noodle"),
		[]byte("other"),
	}
	sortedKeys := [][]byte{
		[]byte("key3"),
		[]byte("key1"),
		[]byte("key2"),
	}

	keyToChild := []byte("keytochild")

	cases := map[string]struct {
		changes func(t *testing.T, ts *TrieState)
		checks  func(t *testing.T, ts *TrieState, isTransactionRunning bool)
	}{
		"set_get": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					err := ts.Put([]byte(tc), []byte(tc))
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
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
			checks: func(t *testing.T, ts *TrieState, _ bool) {
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
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				for _, tc := range testCases {
					err := ts.ClearChildStorage([]byte(tc), []byte(tc))
					require.NoError(t, err)

					val, err := ts.GetChildStorage([]byte(tc), []byte(tc))

					require.Nil(t, val)

					if isTransactionRunning {
						require.Nil(t, err)
					} else {
						require.ErrorContains(t, err, "child trie does not exist at key")
					}
				}
			},
		},
		"delete": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					ts.Put([]byte(tc), []byte(tc))
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				ts.Delete([]byte(testCases[0]))
				has := ts.Has([]byte(testCases[0]))
				require.False(t, has)
			},
		},
		"clear_prefix": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range prefixedKeys {
					err := ts.Put(key, []byte{byte(i)})
					require.Nil(t, err)
				}

				err := ts.ClearPrefix([]byte("noo"))
				require.Nil(t, err)
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				for i, key := range prefixedKeys {
					val := ts.Get(key)
					if i < 2 {
						require.Nil(t, val)
					} else {
						require.NotNil(t, val)
					}
				}
			},
		},
		"clear_prefix_in_child": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range prefixedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.NoError(t, err)
				}

				err := ts.ClearPrefixInChild(keyToChild, []byte("noo"))
				require.NoError(t, err)
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				for i, key := range prefixedKeys {
					val, err := ts.GetChildStorage(keyToChild, key)
					require.NoError(t, err)
					if i < 2 {
						require.Nil(t, val)
					} else {
						require.NotNil(t, val)
					}
				}
			},
		},
		"delete_child_limit_child_not_exists": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range sortedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.Nil(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				testLimitBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(testLimitBytes, uint32(2))
				optLimit2 := &testLimitBytes

				errMsg := fmt.Sprintf("child trie does not exist at key 0x%x", ":child_storage:default:fakekey")

				_, _, err := ts.DeleteChildLimit([]byte("fakekey"), optLimit2)
				require.Error(t, err)
				require.EqualError(t, err, errMsg)

			},
		},
		"delete_child_limit_with_limit": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range sortedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.Nil(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				testLimitBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(testLimitBytes, uint32(2))
				optLimit2 := &testLimitBytes

				deleted, all, err := ts.DeleteChildLimit(keyToChild, optLimit2)
				require.NoError(t, err)

				if isTransactionRunning {
					require.Equal(t, uint32(3), deleted)
					require.Equal(t, true, all)
				} else {
					require.Equal(t, uint32(2), deleted)
					require.Equal(t, false, all)
				}
			},
		},
		"delete_child_limit_nil": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range sortedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.Nil(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				deleted, all, err := ts.DeleteChildLimit(keyToChild, nil)

				require.Nil(t, err)
				if isTransactionRunning {
					require.Equal(t, uint32(3), deleted)
					require.Equal(t, true, all)
				} else {
					require.Equal(t, uint32(3), deleted)
					require.Equal(t, true, all)
				}
			},
		},
		"next_key": {
			changes: func(t *testing.T, ts *TrieState) {
				for _, tc := range testCases {
					err := ts.Put([]byte(tc), []byte(tc))
					require.Nil(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				testCasesCopy := slices.Clone(testCases)

				sort.Slice(testCasesCopy, func(i, j int) bool {
					return bytes.Compare([]byte(testCasesCopy[i]), []byte(testCasesCopy[j])) == -1
				})

				for i, tc := range testCasesCopy {
					next := ts.NextKey([]byte(tc))
					if i == len(testCasesCopy)-1 {
						require.Nil(t, next)
					} else {
						require.Equal(t, []byte(testCasesCopy[i+1]), next, common.BytesToHex([]byte(tc)))
					}
				}
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
				tt.checks(t, ts, false)
			})

			t.Run("during_transaction", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(trie.NewEmptyTrie())
				ts.StartTransaction()
				tt.changes(t, ts)
				tt.checks(t, ts, true)
				ts.CommitTransaction()
			})

			t.Run("after_transaction_committed", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(trie.NewEmptyTrie())
				ts.StartTransaction()
				tt.changes(t, ts)
				ts.CommitTransaction()
				tt.checks(t, ts, false)
			})
		})
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
