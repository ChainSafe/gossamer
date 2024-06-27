// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"
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
		[]byte("key1"),
		[]byte("key2"),
		[]byte("key3"),
	}

	sortedValues := [][]byte{
		[]byte("val1"),
		[]byte("val2"),
		[]byte("val3"),
		[]byte("val4"),
		[]byte("val5"),
		[]byte("val6"),
	}

	sortedKeyToChild := [][]byte{
		[]byte("ktc1"),
		[]byte("ktc2"),
		[]byte("ktc3"),
		[]byte("ktc4"),
		[]byte("ktc5"),
		[]byte("ktc6"),
	}

	keyToChild := []byte("keytochild")

	cases := map[string]struct {
		changes func(t *testing.T, ts *TrieState)
		checks  func(t *testing.T, ts *TrieState, isTransactionRunning bool)
	}{
		"set_get": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range testCases {
					err := ts.Put([]byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				for i, tc := range testCases {
					res := ts.Get([]byte(tc))
					require.Equal(t, sortedValues[i], res)
				}
			},
		},
		"set_child_storage": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range testCases {
					err := ts.SetChildStorage(sortedKeyToChild[i], []byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				for i, tc := range testCases {
					res, err := ts.GetChildStorage(sortedKeyToChild[i], []byte(tc))
					require.NoError(t, err)
					require.Equal(t, sortedValues[i], res)
				}
			},
		},
		"set_and_clear_from_child": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range testCases {
					err := ts.SetChildStorage(sortedKeyToChild[i], []byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				for i, tc := range testCases {
					err := ts.ClearChildStorage(sortedKeyToChild[i], []byte(tc))
					require.NoError(t, err)

					val, err := ts.GetChildStorage(sortedKeyToChild[i], []byte(tc))

					require.Nil(t, val)

					if isTransactionRunning {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, "child trie does not exist at key")
					}
				}
			},
		},
		"delete": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range testCases {
					ts.Put([]byte(tc), sortedValues[i])
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				ts.Delete([]byte(testCases[0]))
				has := ts.Has([]byte(testCases[0]))
				require.False(t, has)
			},
		},
		"delete_child": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range prefixedKeys {
					ts.SetChildStorage(keyToChild, tc, sortedValues[i])
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				err := ts.DeleteChild(keyToChild)
				require.NoError(t, err)

				root, err := ts.GetChildStorage(keyToChild, prefixedKeys[0])
				require.NotNil(t, err)
				require.Nil(t, root)
			},
		},
		"clear_prefix": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range prefixedKeys {
					err := ts.Put(key, []byte{byte(i)})
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				err := ts.ClearPrefix([]byte("noo"))
				require.NoError(t, err)

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
		"clear_prefix_with_limit_1": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range prefixedKeys {
					err := ts.Put(key, []byte{byte(i)})
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				deleted, allDeleted, err := ts.ClearPrefixLimit([]byte("noo"), uint32(1))
				require.NoError(t, err)

				if isTransactionRunning {
					// New keys are not considered towards the limit
					require.Equal(t, uint32(2), deleted)
					require.False(t, allDeleted)
				} else {
					require.Equal(t, uint32(1), deleted)
					require.False(t, allDeleted)
				}
			},
		},
		"clear_prefix_in_child": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range prefixedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				err := ts.ClearPrefixInChild(keyToChild, []byte("noo"))
				require.NoError(t, err)

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
		"clear_prefix_in_child_with_limit_1": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range prefixedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.NoError(t, err)
				}

			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				deleted, allDeleted, err := ts.ClearPrefixInChildWithLimit(keyToChild, []byte("noo"), uint32(1))

				require.NoError(t, err)
				require.False(t, allDeleted)

				if isTransactionRunning {
					require.Equal(t, uint32(2), deleted)
				} else {
					require.Equal(t, uint32(1), deleted)
				}
			},
		},
		"delete_child_limit_child_not_exists": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, key := range sortedKeys {
					err := ts.SetChildStorage(keyToChild, key, []byte{byte(i)})
					require.NoError(t, err)
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
					require.NoError(t, err)
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
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, isTransactionRunning bool) {
				deleted, all, err := ts.DeleteChildLimit(keyToChild, nil)

				require.NoError(t, err)
				require.Equal(t, uint32(3), deleted)
				require.Equal(t, true, all)
			},
		},
		"next_key": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range sortedKeys {
					err := ts.Put(tc, sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				for i, tc := range sortedKeys {
					next := ts.NextKey(tc)
					if i == len(sortedKeys)-1 {
						require.Nil(t, next)
					} else {
						require.Equal(t, sortedKeys[i+1], next, common.BytesToHex(tc))
					}
				}
			},
		},
		"child_next_key": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range sortedKeys {
					err := ts.SetChildStorage(keyToChild, tc, sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				for i, tc := range sortedKeys {
					next, err := ts.GetChildNextKey(keyToChild, tc)
					require.NoError(t, err)

					if i == len(sortedKeys)-1 {
						require.Nil(t, next)
					} else {
						require.Equal(t, sortedKeys[i+1], next, common.BytesToHex(tc))
					}
				}
			},
		},
		"entries": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range testCases {
					err := ts.Put([]byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				entries := ts.TrieEntries()
				require.Len(t, entries, len(testCases))

				for _, tc := range testCases {
					require.Contains(t, entries, tc)
				}
			},
		},
		"get_keys_with_prefix_from_child": {
			changes: func(t *testing.T, ts *TrieState) {
				for i, tc := range prefixedKeys {
					err := ts.SetChildStorage(keyToChild, tc, sortedValues[i])
					require.NoError(t, err)
				}
			},
			checks: func(t *testing.T, ts *TrieState, _ bool) {
				values, err := ts.GetKeysWithPrefixFromChild(keyToChild, []byte("noo"))

				require.NoError(t, err)
				require.Len(t, values, 2)
				require.Contains(t, values, []byte("noot"))
				require.Contains(t, values, []byte("noodle"))
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			t.Run("without_transactions", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(inmemory_trie.NewEmptyTrie())
				tt.changes(t, ts)
				tt.checks(t, ts, false)
			})

			t.Run("during_transaction", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(inmemory_trie.NewEmptyTrie())
				ts.StartTransaction()
				tt.changes(t, ts)
				tt.checks(t, ts, true)
				ts.CommitTransaction()
			})

			t.Run("after_transaction_committed", func(t *testing.T) {
				t.Parallel()

				ts := NewTrieState(inmemory_trie.NewEmptyTrie())
				ts.StartTransaction()
				tt.changes(t, ts)
				ts.CommitTransaction()
				tt.checks(t, ts, false)
			})
		})
	}
}

func TestNextKeys(t *testing.T) {
	cases := map[string]struct {
		keysOnState        [][]byte
		underTransactionFn func(t *testing.T, ts *TrieState)
		expectedNextKey    []byte
		searchKey          []byte
	}{
		"key_already_on_state": {
			searchKey: []byte("acc:abc123:fff"),
			keysOnState: [][]byte{
				[]byte("acc:abc123:ddd"),
				[]byte("acc:abc123:eee"),
				[]byte("acc:abc123:fff"),
				[]byte("completely_diff_key"),
			},
			underTransactionFn: func(t *testing.T, ts *TrieState) {
			},
			expectedNextKey: []byte("completely_diff_key"),
		},
		"key_removed_inside_tx": {
			searchKey: []byte("acc:abc123"),
			keysOnState: [][]byte{
				[]byte("acc:abc123:ddd"),
				[]byte("acc:abc123:eee"),
				[]byte("acc:abc123:fff"),
				[]byte("completely_diff_key"),
			},
			underTransactionFn: func(t *testing.T, ts *TrieState) {
				require.NoError(t, ts.Delete([]byte("acc:abc123:ddd")))
			},
			expectedNextKey: []byte("acc:abc123:eee"),
		},
		"remove_all_acc:abc123_keys_should_return_completely_diff_key": {
			searchKey: []byte("acc:abc123"),
			keysOnState: [][]byte{
				[]byte("acc:abc123:ddd"),
				[]byte("acc:abc123:eee"),
				[]byte("acc:abc123:fff"),
				[]byte("completely_diff_key"),
			},
			underTransactionFn: func(t *testing.T, ts *TrieState) {
				require.NoError(t, ts.Delete([]byte("acc:abc123:ddd")))
				require.NoError(t, ts.Delete([]byte("acc:abc123:eee")))
				require.NoError(t, ts.Delete([]byte("acc:abc123:fff")))
			},
			expectedNextKey: []byte("completely_diff_key"),
		},
		"find_key_on_state_and_on_tx_should_return_key_on_tx": {
			searchKey: []byte("acc:abc123"),
			keysOnState: [][]byte{
				[]byte("acc:abc123:ddd"),
				[]byte("acc:abc123:eee"),
				[]byte("acc:abc123:fff"),
				[]byte("completely_diff_key"),
			},
			underTransactionFn: func(t *testing.T, ts *TrieState) {
				require.NoError(t, ts.Delete([]byte("acc:abc123:ddd")))
				require.NoError(t, ts.Delete([]byte("acc:abc123:eee")))
				require.NoError(t, ts.Delete([]byte("acc:abc123:fff")))
				require.NoError(t, ts.Put([]byte("acc:abc123:ggg"), []byte("0x010")))
			},
			expectedNextKey: []byte("acc:abc123:ggg"),
		},
		"no_next_key": {
			searchKey: []byte("zzzz"),
			keysOnState: [][]byte{
				[]byte("acc:abc123:ddd"),
				[]byte("acc:abc123:eee"),
				[]byte("acc:abc123:fff"),
				[]byte("completely_diff_key"),
			},
			underTransactionFn: func(t *testing.T, ts *TrieState) {},
			expectedNextKey:    nil,
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ts := NewTrieState(inmemory_trie.NewEmptyTrie())

			// inserting first keys on state
			for _, key := range tt.keysOnState {
				require.NoError(t, ts.Put(key, []byte("0x10")))
			}

			ts.StartTransaction()
			tt.underTransactionFn(t, ts)

			nxt := ts.NextKey(tt.searchKey)
			require.Equal(t, tt.expectedNextKey, nxt)

			ts.CommitTransaction()
		})
	}
}

func TestClearPrefixSortedKeys(t *testing.T) {
	t.Run("with_limit", func(t *testing.T) {
		ts := NewTrieState(inmemory_trie.NewEmptyTrie())
		ts.SetVersion(trie.V1)

		setOfKeysWithSamePrefix := [][]byte{
			[]byte("same_prefix_key::A"),
			[]byte("same_prefix_key::B"),
			[]byte("same_prefix_key::C"),
			[]byte("same_prefix_key::D"),
			[]byte("same_prefix_key::E"),
		}

		// insert the keys under a transaction
		ts.StartTransaction()
		for _, k := range setOfKeysWithSamePrefix {
			ts.Put(k, []byte("some_value"))
		}
		ts.CommitTransaction()

		// clear just 1 key using the prefix
		commonPrefix := []byte("same_prefix_key::")
		ts.ClearPrefixLimit(commonPrefix, 1)

		ts.StartTransaction()
		lastKey := commonPrefix

		// we should be able to retrieve
		for i := 0; i < len(setOfKeysWithSamePrefix)-1; i++ {
			nextKey := ts.NextKey(lastKey)
			require.True(t, bytes.HasPrefix(nextKey, commonPrefix), "%v does not have prefix %s", nextKey, commonPrefix)
			lastKey = nextKey
		}

		// the 5th next key call should return nil
		require.Nil(t, ts.NextKey(lastKey))
		ts.CommitTransaction()

	})

	t.Run("without_limit", func(t *testing.T) {
		ts := NewTrieState(inmemory_trie.NewEmptyTrie())
		ts.SetVersion(trie.V1)

		setOfKeysWithSamePrefix := [][]byte{
			[]byte("same_prefix_key::A"),
			[]byte("same_prefix_key::B"),
			[]byte("same_prefix_key::C"),
			[]byte("same_prefix_key::D"),
			[]byte("same_prefix_key::E"),
		}

		{
			// insert the keys under a transaction
			ts.StartTransaction()
			for _, k := range setOfKeysWithSamePrefix {
				ts.Put(k, []byte("some_value"))
			}
			ts.CommitTransaction()
		}

		// clear just 1 key using the prefix
		commonPrefix := []byte("same_prefix_key::")
		ts.ClearPrefix(commonPrefix)

		ts.StartTransaction()
		// should not exist any key
		require.Nil(t, ts.NextKey(commonPrefix))
		ts.CommitTransaction()

	})
}

func TestTrieState_Root(t *testing.T) {
	ts := NewTrieState(inmemory_trie.NewEmptyTrie())

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	expected := ts.MustRoot()
	require.Equal(t, expected, ts.MustRoot())
}

func TestTrieState_ChildRoot(t *testing.T) {
	ts := NewTrieState(inmemory_trie.NewEmptyTrie())

	keyToChild := []byte("child")

	for _, tc := range testCases {
		ts.SetChildStorage(keyToChild, []byte(tc), []byte(tc))
	}

	root, err := ts.GetChildRoot(keyToChild)
	require.NoError(t, err)
	require.NotNil(t, root)
}

func TestTrieState_NestedTransactions(t *testing.T) {
	cases := map[string]struct {
		createTrieState func() *TrieState
		assert          func(*testing.T, *TrieState)
	}{
		"committing_and_rollback_on_nested_transactions": {
			createTrieState: func() *TrieState {
				ts := NewTrieState(inmemory_trie.NewEmptyTrie())

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
				ts := NewTrieState(inmemory_trie.NewEmptyTrie())
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
				return NewTrieState(inmemory_trie.NewEmptyTrie())
			},
			assert: func(t *testing.T, ts *TrieState) {
				require.PanicsWithValue(t, "no transactions to rollback", func() { ts.RollbackTransaction() })
			},
		},
		"commit_without_transaction_should_panic": {
			createTrieState: func() *TrieState {
				return NewTrieState(inmemory_trie.NewEmptyTrie())
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

func BenchmarkNextKey(b *testing.B) {
	ts := NewTrieState(inmemory_trie.NewEmptyTrie())

	// Keys / values already present in state
	maxKeys := 2000
	sortedKeys := make([][]byte, maxKeys)

	for i := 0; i < maxKeys/2; i++ {
		key := []byte(fmt.Sprintf("key%04d", i))
		sortedKeys[i] = key
		err := ts.Put(key, key)
		require.Nil(b, err)
	}

	// Keys / values added after a transaction starts
	ts.StartTransaction()

	for i := maxKeys / 2; i < maxKeys; i++ {
		key := []byte(fmt.Sprintf("key%04d", i))
		sortedKeys[i] = key
		err := ts.Put(key, key)
		require.Nil(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, tc := range sortedKeys {
			next := ts.NextKey(tc)
			if i == len(sortedKeys)-1 {
				require.Nil(b, next)
			} else {
				require.Equal(b, sortedKeys[i+1], next, common.BytesToHex(tc))
			}
		}
	}
}
