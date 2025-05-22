// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func newInstance(backend statemachine.Backend[hash.H256, runtime.BlakeTwo256]) TrieState {
	if backend == nil {
		backend = statemachine.NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
	}
	overlay := overlayedchanges.NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	ext := overlayedchanges.NewExt(overlay, backend)
	return NewExtBackedTrieState(ext)
}

func TestTrieState_WithAndWithoutTransactions(t *testing.T) {
	t.Parallel()

	prefixedKeys := [][]byte{
		[]byte("noot"),
		[]byte("noodle"),
		[]byte("other"),
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
		changes func(t *testing.T, ts TrieState) TrieState
		checks  func(t *testing.T, ts TrieState, isTransactionRunning bool)
	}{
		"set_get": {
			changes: func(t *testing.T, ts TrieState) TrieState {
				for i, tc := range testCases {
					err := ts.Put([]byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
				return ts
			},
			checks: func(t *testing.T, ts TrieState, _ bool) {
				for i, tc := range testCases {
					res := ts.Get([]byte(tc))
					require.Equal(t, sortedValues[i], res)
				}
			},
		},
		"set_child_storage": {
			changes: func(t *testing.T, ts TrieState) TrieState {
				for i, tc := range testCases {
					err := ts.SetChildStorage(sortedKeyToChild[i], []byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
				return ts
			},
			checks: func(t *testing.T, ts TrieState, _ bool) {
				for i, tc := range testCases {
					res, err := ts.GetChildStorage(sortedKeyToChild[i], []byte(tc))
					require.NoError(t, err)
					require.Equal(t, sortedValues[i], res)
				}
			},
		},
		"set_and_clear_from_child": {
			changes: func(t *testing.T, ts TrieState) TrieState {
				for i, tc := range testCases {
					err := ts.SetChildStorage(sortedKeyToChild[i], []byte(tc), sortedValues[i])
					require.NoError(t, err)
				}
				return ts
			},
			checks: func(t *testing.T, ts TrieState, isTransactionRunning bool) {
				for i, tc := range testCases {
					err := ts.ClearChildStorage(sortedKeyToChild[i], []byte(tc))
					require.NoError(t, err)

					val, err := ts.GetChildStorage(sortedKeyToChild[i], []byte(tc))

					require.NoError(t, err)
					require.Nil(t, val)
				}
			},
		},
		"delete": {
			changes: func(t *testing.T, ts TrieState) TrieState {
				for i, tc := range testCases {
					ts.Put([]byte(tc), sortedValues[i])
				}
				return ts
			},
			checks: func(t *testing.T, ts TrieState, _ bool) {
				ts.Delete([]byte(testCases[0]))
				has := ts.Has([]byte(testCases[0]))
				require.False(t, has)
			},
		},
		"delete_child": {
			changes: func(t *testing.T, ts TrieState) TrieState {
				for i, tc := range prefixedKeys {
					ts.SetChildStorage(keyToChild, tc, sortedValues[i])
				}
				return ts
			},
			checks: func(t *testing.T, ts TrieState, _ bool) {
				err := ts.DeleteChild(keyToChild)
				require.NoError(t, err)

				root, err := ts.GetChildStorage(keyToChild, prefixedKeys[0])
				require.NoError(t, err)
				require.Nil(t, root)
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			t.Run("without_transactions", func(t *testing.T) {
				t.Parallel()

				ts := newInstance(nil)
				tt.changes(t, ts)
				tt.checks(t, ts, false)
			})

			t.Run("during_transaction", func(t *testing.T) {
				t.Parallel()

				ts := newInstance(nil)
				ts.StartTransaction()
				tt.changes(t, ts)
				tt.checks(t, ts, true)
				ts.CommitTransaction()
			})

			t.Run("after_transaction_committed", func(t *testing.T) {
				t.Parallel()

				ts := newInstance(nil)
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
		underTransactionFn func(t *testing.T, ts TrieState)
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
			underTransactionFn: func(t *testing.T, ts TrieState) {
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
			underTransactionFn: func(t *testing.T, ts TrieState) {
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
			underTransactionFn: func(t *testing.T, ts TrieState) {
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
			underTransactionFn: func(t *testing.T, ts TrieState) {
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
			underTransactionFn: func(t *testing.T, ts TrieState) {},
			expectedNextKey:    nil,
		},
		"nothing_on_state_only_on_tx": {
			searchKey:   []byte("acc:abc123"),
			keysOnState: [][]byte{},
			underTransactionFn: func(t *testing.T, ts TrieState) {
				require.NoError(t, ts.Put([]byte("acc:abc123:ddd"), []byte("0x10")))
			},
			expectedNextKey: []byte("acc:abc123:ddd"),
		},
		"search_key_longer_but_next_key_exists": {
			searchKey: []byte("abz"),
			keysOnState: [][]byte{
				[]byte("a"),
				[]byte("b"),
				[]byte("c"),
			},
			underTransactionFn: func(t *testing.T, ts TrieState) {},
			expectedNextKey:    []byte("b"),
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			keyOnState := make(map[string][]byte)

			// inserting first keys on state
			for _, key := range tt.keysOnState {
				keyOnState[string(key)] = []byte("0x10")
			}

			backend := statemachine.NewMemoryDBTrieBackendFromMap[hash.H256, runtime.BlakeTwo256](
				keyOnState,
				storage.StateVersionV1,
			)
			ts := newInstance(backend)

			ts.StartTransaction()
			tt.underTransactionFn(t, ts)

			nxt := ts.NextKey(tt.searchKey)
			require.Equal(t, tt.expectedNextKey, nxt)

			ts.CommitTransaction()
		})
	}
}

func TestClearPrefixSortedKeys(t *testing.T) {
	setOfKeysWithSamePrefix := map[string][]byte{
		"same_prefix_key::A": []byte("some_value"),
		"same_prefix_key::B": []byte("some_value"),
		"same_prefix_key::C": []byte("some_value"),
		"same_prefix_key::D": []byte("some_value"),
		"same_prefix_key::E": []byte("some_value"),
	}

	backend := statemachine.NewMemoryDBTrieBackendFromMap[hash.H256, runtime.BlakeTwo256](
		setOfKeysWithSamePrefix,
		storage.StateVersionV1,
	)

	t.Run("with_limit", func(t *testing.T) {
		ts := newInstance(backend)
		// clear just 1 key using the prefix
		commonPrefix := []byte("same_prefix_key::")
		ts.ClearPrefixLimit(commonPrefix, 1)

		lastKey := commonPrefix

		// we should be able to retrieve
		for range len(setOfKeysWithSamePrefix) - 1 {
			nextKey := ts.NextKey(lastKey)
			t.Logf("nextKey: %s", nextKey)

			require.True(t, bytes.HasPrefix(nextKey, commonPrefix), "%v does not have prefix %s", nextKey, commonPrefix)
			lastKey = nextKey
		}

		// the 5th next key call should return nil
		require.Nil(t, ts.NextKey(lastKey))
	})

	t.Run("without_limit", func(t *testing.T) {
		ts := newInstance(backend)

		// clear all keys using the prefix
		commonPrefix := []byte("same_prefix_key::")
		ts.ClearPrefix(commonPrefix)

		// should not exist any key
		require.Nil(t, ts.NextKey(commonPrefix))
	})
}

func TestTrieState_Root(t *testing.T) {
	ts := newInstance(nil)

	for _, tc := range testCases {
		ts.Put([]byte(tc), []byte(tc))
	}

	expectedHash := common.Hash{
		0x1a, 0x9, 0x9c, 0x3d, 0x3a, 0x8, 0x17, 0xaf, 0x96, 0x27, 0x2c, 0xb0, 0x46, 0x97, 0x44, 0x4,
		0x7a, 0xae, 0xb7, 0x5b, 0x87, 0x6a, 0xe5, 0x72, 0xcf, 0x91, 0xfc, 0xe6, 0x90, 0xc7, 0x85, 0x91,
	}
	rootHash, err := ts.Root()
	require.NoError(t, err)
	require.Equal(t, expectedHash, rootHash)
}

func TestTrieState_ChildRoot(t *testing.T) {
	ts := newInstance(nil)

	keyToChild := []byte("child")

	for _, tc := range testCases {
		ts.SetChildStorage(keyToChild, []byte(tc), []byte(tc))
	}

	expectedHash := common.Hash{
		0x1a, 0x9, 0x9c, 0x3d, 0x3a, 0x8, 0x17, 0xaf, 0x96, 0x27, 0x2c, 0xb0, 0x46, 0x97, 0x44, 0x4,
		0x7a, 0xae, 0xb7, 0x5b, 0x87, 0x6a, 0xe5, 0x72, 0xcf, 0x91, 0xfc, 0xe6, 0x90, 0xc7, 0x85, 0x91,
	}
	root, err := ts.GetChildRoot(keyToChild)
	require.NoError(t, err)
	require.Equal(t, expectedHash, root)
}

func TestTrieState_NestedTransactions(t *testing.T) {
	cases := map[string]struct {
		createTrieState func(ts TrieState)
		assert          func(*testing.T, TrieState)
	}{
		"committing_and_rollback_on_nested_transactions": {
			createTrieState: func(ts TrieState) {
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
			},
			assert: func(t *testing.T, ts TrieState) {
				require.NotNil(t, ts.Get([]byte("key-1")))
				require.NotNil(t, ts.Get([]byte("key-2")))
				require.NotNil(t, ts.Get([]byte("key-3")))

				require.Nil(t, ts.Get([]byte("key-4")))
			},
		},
		"committing_all_nested_transactions": {
			createTrieState: func(ts TrieState) {
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
			},
			assert: func(t *testing.T, ts TrieState) {
				require.NotNil(t, ts.Get([]byte("key-1")))
				require.NotNil(t, ts.Get([]byte("key-2")))
				require.NotNil(t, ts.Get([]byte("key-4")))
			},
		},
		"rollback_without_transaction_should_panic": {
			createTrieState: func(ts TrieState) {
				// do nothing
			},
			assert: func(t *testing.T, ts TrieState) {
				require.ErrorIs(t, ts.RollbackTransaction(), ErrNoTransactionsToRollback)
			},
		},
		"commit_without_transaction_should_panic": {
			createTrieState: func(ts TrieState) {
				// do nothing
			},
			assert: func(t *testing.T, ts TrieState) {
				require.ErrorIs(t, ts.CommitTransaction(), ErrNoTransactionsToCommit)
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ts := newInstance(nil)
			tt.createTrieState(ts)
			tt.assert(t, ts)
		})
	}
}

func BenchmarkExtBackedTrieState_NextKey(b *testing.B) {
	ts := newInstance(nil)

	// Keys / values already present in state
	maxKeys := 2000
	sortedKeys := make([][]byte, maxKeys)

	for i := range maxKeys / 2 {
		key := fmt.Appendf(nil, "key%04d", i)
		sortedKeys[i] = key
		err := ts.Put(key, key)
		require.Nil(b, err)
	}

	// Keys / values added after a transaction starts
	ts.StartTransaction()

	for i := maxKeys / 2; i < maxKeys; i++ {
		key := fmt.Appendf(nil, "key%04d", i)
		sortedKeys[i] = key
		err := ts.Put(key, key)
		require.Nil(b, err)
	}

	for b.Loop() {
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
