// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"slices"
	"testing"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertions(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trieEntries []trie.Entry
		key         []uint8
		value       []uint8
		stored      nodeStorage[hash.H256]
		dontCheck   bool
	}{
		"nil_parent": {
			trieEntries: []trie.Entry{},
			key:         []byte{0x01},
			value:       []byte("leaf"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x01}, Offset: 0},
							value:      inline([]byte("leaf")),
						},
					},
				},
			},
		},
		"branch_parent": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{0x01},
					Value: []byte("branch"),
				},
			},
			key:   []byte{0x01, 0x01},
			value: []byte("leaf"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x01}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x01}},
							value:      inline([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"branch_in_between_rearrange": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0, 1},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1, 0},
			value: []byte("in between branch"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline([]byte("in between branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(1), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x01}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}, Offset: 0},
							value:      inline([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"branch_in_between": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1, 0},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0, 1},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("top branch"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), nil, nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("top branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(1), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"override_branch_value": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("new branch"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("new branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
			dontCheck: true,
		},
		"override_branch_value_same_value": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("branch"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"override_leaf_of_branch_value_same_value": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(1), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
				},
			},
		},
		"override_leaf_parent": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("new leaf"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("new leaf")),
						},
					},
				},
			},
			dontCheck: true,
		},
		"write_same_leaf_value_to_leaf_parent": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("same"),
				},
			},
			key:   []byte{1},
			value: []byte("same"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("same")),
						},
					},
				},
			},
		},
		"write_leaf_as_divergent_child_next_to_parent_leaf": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{0x01, 0x02},
					Value: []byte("original leaf"),
				},
			},
			key:   []byte{0x02, 0x03},
			value: []byte("leaf"),
			stored: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x02}},
							value:      inline([]byte("original leaf")),
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x03}},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x00}, Offset: 1},
							value:      nil,
							children: [codec.ChildrenCapacity]NodeHandle{
								nil,
								inMemory(0),
								inMemory(1),
								nil,
								nil, nil, nil, nil,
								nil, nil, nil, nil,
								nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Setup trie
			inmemoryDB := NewMemoryDB()
			trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

			for _, entry := range testCase.trieEntries {
				require.NoError(t, trie.Set(entry.Key, entry.Value))
			}
			// Add new key-value pair
			err := trie.Set(testCase.key, testCase.value)
			require.NoError(t, err)

			if !testCase.dontCheck {
				// Check values for keys
				for _, entry := range testCase.trieEntries {
					val, err := trie.Get(entry.Key)
					require.NoError(t, err)
					require.Equal(t, entry.Value, val)
				}
			}
			val, err := trie.Get(testCase.key)
			require.NoError(t, err)
			require.Equal(t, testCase.value, val)

			// Check we have what we expect
			assert.Equal(t, testCase.stored.nodes, trie.storage.nodes)
		})
	}
}

func TestDeletes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trieEntries []trie.Entry
		key         []byte
		expected    nodeStorage[hash.H256]
	}{
		"nil_key": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("leaf"),
				},
			},
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("leaf")),
						},
					},
				},
			},
		},
		"empty_trie": {
			key: []byte{1},
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{NewStoredNode{Empty{}}},
			},
		},
		"delete_leaf": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("leaf"),
				},
			},
			key: []byte{1},
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{NewStoredNode{Empty{}}},
			},
		},
		"delete_branch": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0},
					Value: []byte("leaf"),
				},
			},
			key: []byte{1},
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{Empty{}},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1, 0}},
							value:      inline([]byte("leaf")),
						},
					},
				},
			},
		},
		"delete_branch_without_value_should_do_nothing": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1, 0},
					Value: []byte("leaf1"),
				},
				{
					Key:   []byte{1, 1},
					Value: []byte("leaf2"),
				},
			},
			key: []byte{1},
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: make([]byte, 0)},
							value:      inline([]byte("leaf1")),
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: make([]byte, 0)},
							value:      inline([]byte("leaf2")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x00, 0x10}, Offset: 1},
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), inMemory(1),
							},
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup trie
			inmemoryDB := NewMemoryDB()
			trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

			for _, entry := range testCase.trieEntries {
				assert.NoError(t, trie.Set(entry.Key, entry.Value))
			}

			// Remove key
			err := trie.Delete(testCase.key)
			assert.NoError(t, err)

			// Check we have what we expect
			assert.Equal(t, testCase.expected.nodes, trie.storage.nodes)
		})
	}
}

func TestInsertAfterDelete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trieEntries []trie.Entry
		key         []byte
		value       []byte
		expected    nodeStorage[hash.H256]
	}{
		"insert_leaf_after_delete": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("new leaf"),
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("new leaf")),
						},
					},
				},
			},
		},
		"insert_branch_after_delete": {
			trieEntries: []trie.Entry{
				{
					Key:   []byte{1},
					Value: []byte("branch"),
				},
				{
					Key:   []byte{1, 0},
					Value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("new branch"),
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline([]byte("new branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0),
							},
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup trie
			inmemoryDB := NewMemoryDB()
			trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

			for _, entry := range testCase.trieEntries {
				assert.NoError(t, trie.insert(nibbles.NewNibbles(entry.Key), entry.Value))
			}

			// Remove key
			err := trie.remove(nibbles.NewNibbles(testCase.key))
			assert.NoError(t, err)

			// Add again
			err = trie.insert(nibbles.NewNibbles(testCase.key), testCase.value)
			assert.NoError(t, err)

			// Check we have what we expect
			assert.Equal(t, testCase.expected.nodes, trie.storage.nodes)
		})
	}
}

func TestDBCommits(t *testing.T) {
	t.Parallel()

	t.Run("commit_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := trie.Set([]byte("leaf"), []byte("leafvalue"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		// 1 leaf
		assert.Len(t, inmemoryDB.Keys(), 1)

		// Get values using lazy loading
		value, _ := trie.Get([]byte("leaf"))
		assert.Equal(t, []byte("leafvalue"), value)
	})

	t.Run("commit_branch_and_inlined_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := trie.Set([]byte("branchleaf"), []byte("leafvalue"))
		assert.NoError(t, err)
		err = trie.Set([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		// 1 branch with its inlined leaf
		assert.Len(t, inmemoryDB.Keys(), 1)

		// Get values using lazy loading
		value, _ := trie.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), value)
		value, _ = trie.Get([]byte("branchleaf"))
		assert.Equal(t, []byte("leafvalue"), value)
	})

	t.Run("commit_branch_and_hashed_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := tr.Set([]byte("branchleaf"), make([]byte, 40))
		assert.NoError(t, err)
		err = tr.Set([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 branch with 1 hashed leaf child
		// 1 hashed leaf
		assert.Len(t, inmemoryDB.Keys(), 2)

		// Get values using lazy loading
		value, _ := tr.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), value)
		value, _ = tr.Get([]byte("branchleaf"))
		assert.Equal(t, make([]byte, 40), value)
	})

	t.Run("commit_leaf_with_hashed_value", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Set([]byte("leaf"), make([]byte, 40))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.Keys(), 2)

		// Get values using lazy loading
		value, _ := tr.Get([]byte("leaf"))
		assert.Equal(t, make([]byte, 40), value)
	})

	t.Run("commit_leaf_with_hashed_value_then_remove_it", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Set([]byte("leaf"), make([]byte, 40))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.Keys(), 2)

		// Get values using lazy loading
		err = tr.Delete([]byte("leaf"))
		assert.NoError(t, err)
		tr.commit()
		assert.Len(t, inmemoryDB.Keys(), 0)
	})

	t.Run("commit_branch_and_hashed_leaf_with_hashed_value", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Set([]byte("branchleaf"), make([]byte, 40))
		assert.NoError(t, err)
		err = tr.Set([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 branch with 1 hashed leaf child
		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.Keys(), 3)

		// Get values using lazy loading
		value, _ := tr.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), value)
		value, _ = tr.Get([]byte("branchleaf"))
		assert.Equal(t, make([]byte, 40), value)
	})

	t.Run("commit_branch_and_hashed_leaf_with_hashed_value_then_delete_it", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Set([]byte("branchleaf"), make([]byte, 40))
		assert.NoError(t, err)
		err = tr.Set([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 branch with 1 hashed leaf child
		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.Keys(), 3)

		err = tr.Delete([]byte("branchleaf"))
		assert.NoError(t, err)
		tr.commit()

		// 1 branch transformed in a leaf
		// previous leaf was deleted
		// previous hashed (V1) value was deleted too
		assert.Len(t, inmemoryDB.Keys(), 1)
	})

	t.Run("commit_branch_with_leaf_then_delete_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB()
		trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := trie.Set([]byte("branchleaf"), []byte("leafvalue"))
		assert.NoError(t, err)
		err = trie.Set([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		err = trie.Delete([]byte("branchleaf"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		// 1 branch transformed in a leaf
		// previous leaf was deleted
		assert.Len(t, inmemoryDB.Keys(), 1)

		v, _ := trie.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), v)
		v, _ = trie.Get([]byte("branchleaf"))
		assert.Nil(t, v)
	})
}

func Test_TrieDB(t *testing.T) {
	t.Run("recorder", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 64)},
					{[]byte("AB"), bytes.Repeat([]byte{3}, 64)},
					{[]byte("B"), bytes.Repeat([]byte{4}, 64)},
				}

				// Add some initial data to the trie
				db := NewMemoryDB()
				trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
				trie.SetVersion(version)

				for _, entry := range keyValues[:1] {
					require.NoError(t, trie.Set(entry.key, entry.value))
				}
				err := trie.commit()
				require.NoError(t, err)
				require.NotEmpty(t, trie.rootHash)
				root := trie.rootHash

				// Add more data, but this time only to the overlay.
				// While doing that we record all trie accesses to replay this operation.
				recorder := NewRecorder[hash.H256]()
				overlay := db.Clone()
				newRoot := root
				{
					trie := NewTrieDB(newRoot, &overlay,
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					for _, entry := range keyValues[1:] {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					newRoot = trie.rootHash
				}

				partialDB := NewMemoryDB()
				for _, record := range recorder.Drain() {
					// key := runtime.BlakeTwo256{}.Hash(record.Data).Bytes()
					// require.NoError(t, partialDB.Set(key, record.Data))
					partialDB.Insert(hashdb.EmptyPrefix, record.Data)
				}

				// Replay the it, but this time we use the proof.
				var validatedRoot hash.H256
				{
					trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, partialDB)
					trie.SetVersion(version)
					for _, entry := range keyValues[1:] {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					validatedRoot = trie.rootHash
				}
				assert.Equal(t, validatedRoot, newRoot)
			})
		}
	})

	t.Run("recorder_with_cache", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 64)},
					{[]byte("AB"), bytes.Repeat([]byte{3}, 64)},
					{[]byte("B"), bytes.Repeat([]byte{4}, 64)},
				}

				// Add some initial data to the trie
				db := NewMemoryDB()
				trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
				trie.SetVersion(version)

				for _, entry := range keyValues[:1] {
					require.NoError(t, trie.Set(entry.key, entry.value))
				}
				err := trie.commit()
				require.NoError(t, err)
				require.NotEmpty(t, trie.rootHash)
				root := trie.rootHash

				cache := NewTestTrieCache[hash.H256]()

				{
					trie := NewTrieDB(trie.rootHash, db, WithCache[hash.H256, runtime.BlakeTwo256](cache))
					trie.SetVersion(version)
					// Only read one entry, using GetWith which should cache the root node
					_, err := GetWith(trie, keyValues[0].key, func(v []byte) []byte { return v })
					assert.NoError(t, err)
				}

				// Root should now be cached.
				require.NotNil(t, cache.GetNode(trie.rootHash))

				// Add more data, but this time only to the overlay.
				// While doing that we record all trie accesses to replay this operation.
				recorder := NewRecorder[hash.H256]()
				overlay := db.Clone()
				var newRoot hash.H256
				{
					trie := NewTrieDB(trie.rootHash, &overlay,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					for _, entry := range keyValues[1:] {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					newRoot = trie.rootHash
				}

				for i, entry := range keyValues[1:] {
					cachedValue := cache.GetValue(entry.key)
					require.Equal(t, ExistingCachedValue[hash.H256]{
						Hash: runtime.BlakeTwo256{}.Hash(keyValues[i+1].value),
						Data: keyValues[i+1].value,
					}, cachedValue)
				}

				partialDB := NewMemoryDB()
				for _, record := range recorder.Drain() {
					// key := runtime.BlakeTwo256{}.Hash(record.Data).Bytes()
					// require.NoError(t, partialDB.Set(key, record.Data))
					partialDB.Insert(hashdb.EmptyPrefix, record.Data)
				}

				// Replay the it, but this time we use the proof.
				var validatedRoot hash.H256
				{
					trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, partialDB)
					trie.SetVersion(version)
					for _, entry := range keyValues[1:] {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					validatedRoot = trie.rootHash
				}
				assert.Equal(t, validatedRoot, newRoot)
			})
		}
	})

	t.Run("insert_remove_with_cache", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 64)},
					// Should be inlined in v1
					{[]byte("AC"), bytes.Repeat([]byte{7}, 4)},
					{[]byte("AB"), bytes.Repeat([]byte{3}, 64)},
					{[]byte("B"), bytes.Repeat([]byte{4}, 64)},
				}

				cache := NewTestTrieCache[hash.H256]()
				recorder := NewRecorder[hash.H256]()
				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)

					// Add all values
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}

					// Remove only the last 2 elements
					for _, entry := range keyValues[3:] {
						require.NoError(t, trie.Delete(entry.key))
					}

					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				// Then only the first 3 elements should be in the cache and the last
				// two ones should not be there.
				for _, entry := range keyValues[:3] {
					cachedValue := cache.GetValue(entry.key)
					require.NotNil(t, cachedValue)

					require.Equal(t, entry.value, cachedValue.data())
					require.Equal(t, runtime.BlakeTwo256{}.Hash(entry.value), *cachedValue.hash())
				}

				for _, entry := range keyValues[3:] {
					require.Nil(t, cache.GetValue(entry.key))
				}

				// get values again using cache
				for _, entry := range keyValues[:3] {
					trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, val)
					require.Equal(t, entry.value, *val)
				}
			})
		}
	})

	t.Run("insert_with_cache_more_nodes", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 1)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 2)},
					{[]byte("AAA"), bytes.Repeat([]byte{3}, 3)},
					{[]byte("AAAA"), bytes.Repeat([]byte{4}, 4)},
					{[]byte("AB"), bytes.Repeat([]byte{5}, 5)},
					{[]byte("ABA"), bytes.Repeat([]byte{6}, 6)},
					{[]byte("ABB"), bytes.Repeat([]byte{7}, 7)},
					{[]byte("AC"), bytes.Repeat([]byte{8}, 8)},
				}

				cache := NewTestTrieCache[hash.H256]()
				recorder := NewRecorder[hash.H256]()
				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)

					// Add all values
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(slices.Clone(entry.key), entry.value))
					}

					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				// Then only the first 3 elements should be in the cache and the last
				// two ones should not be there.
				for _, entry := range keyValues {
					cachedValue := cache.GetValue(entry.key)
					require.NotNil(t, cachedValue)

					require.Equal(t, entry.value, cachedValue.data())
					require.Equal(t, runtime.BlakeTwo256{}.Hash(entry.value), *cachedValue.hash())
				}

				// get values again using cache
				for _, entry := range keyValues {
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, val)
					require.Equal(t, entry.value, *val)
				}
			})
		}
	})

	t.Run("insert_with_cache_insert_after_commit", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 1)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 2)},
					{[]byte("AAA"), bytes.Repeat([]byte{3}, 3)},
					{[]byte("AAAA"), bytes.Repeat([]byte{4}, 4)},
					{[]byte("AB"), bytes.Repeat([]byte{5}, 5)},
					{[]byte("ABA"), bytes.Repeat([]byte{6}, 6)},
					{[]byte("ABB"), bytes.Repeat([]byte{7}, 7)},
					{[]byte("AC"), bytes.Repeat([]byte{8}, 8)},
				}

				cache := NewTestTrieCache[hash.H256]()
				recorder := NewRecorder[hash.H256]()
				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)

					// Add all values
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(slices.Clone(entry.key), entry.value))
					}

					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				// Then only the first 3 elements should be in the cache and the last
				// two ones should not be there.
				for _, entry := range keyValues {
					cachedValue := cache.GetValue(entry.key)
					require.NotNil(t, cachedValue)

					require.Equal(t, entry.value, cachedValue.data())
					require.Equal(t, runtime.BlakeTwo256{}.Hash(entry.value), *cachedValue.hash())
				}

				// get values again using cache
				for _, entry := range keyValues {
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, val)
					require.Equal(t, entry.value, *val)
				}

				// ensure we can insert a new leaf node on a new triedb
				// use lookup functions to validate we're using cached version
				{
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)

					require.NoError(t, trie.Set([]byte("AAB"), []byte{1, 1, 1, 1}))

					val, _ := trie.Get([]byte("AAB"))
					require.NotNil(t, val)
					require.Equal(t, []byte{1, 1, 1, 1}, val)

					err := trie.commit()
					require.NoError(t, err)

					val, _ = trie.Get([]byte("AAB"))
					require.NotNil(t, val)
					require.Equal(t, []byte{1, 1, 1, 1}, val)

					oVal, err := GetWith(trie, []byte("AAB"), func(d []byte) []byte { return d })
					require.NoError(t, err)
					require.NotNil(t, oVal)
					require.Equal(t, []byte{1, 1, 1, 1}, *oVal)
				}
			})
		}
	})

	t.Run("insert_into_cache_and_lookup_using_cache", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 1)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 2)},
					{[]byte("AAA"), bytes.Repeat([]byte{3}, 3)},
					{[]byte("AAAA"), bytes.Repeat([]byte{4}, 4)},
					{[]byte("AB"), bytes.Repeat([]byte{5}, 5)},
					{[]byte("ABA"), bytes.Repeat([]byte{6}, 6)},
					{[]byte("ABB"), bytes.Repeat([]byte{7}, 7)},
					{[]byte("AC"), bytes.Repeat([]byte{8}, 8)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)

					// Add all values
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(slices.Clone(entry.key), entry.value))
					}

					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				cache := NewTestTrieCache[hash.H256]()

				// get all keys to populate cache
				{
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
					)
					trie.SetVersion(version)
					// get values again using cache
					for _, entry := range keyValues {
						val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
						require.NoError(t, err)
						require.NotNil(t, val)
						require.Equal(t, entry.value, *val)
					}
				}

				// ensure all values are cached
				for _, entry := range keyValues {
					cachedValue := cache.GetValue(entry.key)
					require.NotNil(t, cachedValue)

					require.Equal(t, entry.value, cachedValue.data())
					require.Equal(t, runtime.BlakeTwo256{}.Hash(entry.value), *cachedValue.hash())
				}

				// get all keys again from cache, by passing in brand new db
				{
					db := NewMemoryDB()
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
					)
					trie.SetVersion(version)
					// get values again using cache
					for _, entry := range keyValues {
						val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
						require.NoError(t, err)
						require.NotNil(t, val)
						require.Equal(t, entry.value, *val)
					}
				}
			})
		}
	})

	t.Run("insert_into_cache_and_lookup_hash_using_cache", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 1)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 2)},
					{[]byte("AAA"), bytes.Repeat([]byte{3}, 3)},
					{[]byte("AAAA"), bytes.Repeat([]byte{4}, 4)},
					{[]byte("AB"), bytes.Repeat([]byte{5}, 5)},
					{[]byte("ABA"), bytes.Repeat([]byte{6}, 6)},
					{[]byte("ABB"), bytes.Repeat([]byte{7}, 7)},
					{[]byte("AC"), bytes.Repeat([]byte{8}, 8)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)

					// Add all values
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(slices.Clone(entry.key), entry.value))
					}

					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				cache := NewTestTrieCache[hash.H256]()

				// get all keys to populate cache
				{
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
					)

					trie.SetVersion(version)
					// get hashes for all entries populating cache
					for _, entry := range keyValues {
						h, err := trie.GetHash(entry.key)
						require.NoError(t, err)
						require.NotNil(t, h)
					}
				}

				// get all keys again from cache, by passing in brand new db
				{
					db := NewMemoryDB()
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
					)
					trie.SetVersion(version)
					// get hashes for all entries from cache
					for _, entry := range keyValues {
						h, err := trie.GetHash(entry.key)
						require.NoError(t, err)
						require.NotNil(t, h)
					}
				}

				// get all values, by using cache and previous db
				{
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
					)
					trie.SetVersion(version)
					// get hashes for all entries from cache
					for _, entry := range keyValues {
						val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
						require.NoError(t, err)
						require.NotNil(t, val)
						require.Equal(t, entry.value, *val)
					}
				}
			})
		}
	})

	t.Run("trie_nodes_recorded", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 64)},
					{[]byte("AB"), bytes.Repeat([]byte{3}, 64)},
					{[]byte("B"), bytes.Repeat([]byte{4}, 64)},
					{[]byte("BC"), bytes.Repeat([]byte{4}, 64)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				for _, cache := range []TrieCache[hash.H256]{NewTestTrieCache[hash.H256](), nil} {
					for _, getHash := range []bool{
						true,
						false,
					} {
						recorder := NewRecorder[hash.H256]()
						{
							trie := NewTrieDB(
								root, db,
								WithCache[hash.H256, runtime.BlakeTwo256](cache),
								WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
							)
							trie.SetVersion(version)

							for _, entry := range keyValues {
								if getHash {
									h, err := trie.GetHash(entry.key)
									assert.NoError(t, err)
									assert.NotNil(t, h)
								} else {
									val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
									require.NoError(t, err)
									require.NotNil(t, val)
									require.Equal(t, entry.value, *val)
								}
							}

							if getHash {
								h, err := trie.GetHash([]byte("nonexistent"))
								require.Nil(t, h)
								require.NoError(t, err)
							} else {
								val, err := GetWith(trie, []byte("nonexistent"), func(d []byte) []byte { return d })
								require.NoError(t, err)
								require.Nil(t, val)
							}
						}

						for _, entry := range keyValues {
							recorded := recorder.TrieNodesRecordedForKey(entry.key)

							var isInline bool
							switch version {
							case trie.V0:
								isInline = true
							case trie.V1:
								if len(entry.value) > 32 {
									isInline = false
								} else {
									isInline = true
								}
							}

							var expected RecordedForKey
							if getHash && !isInline {
								expected = RecordedHash
							} else {
								expected = RecordedValue
							}
							require.Equal(t, expected, recorded)
						}

					}
				}
			})
		}
	})

	t.Run("trie_nodes_recorded_get_hashes_and_values", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{trie.V0, trie.V1} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 64)},
					{[]byte("AB"), bytes.Repeat([]byte{3}, 64)},
					{[]byte("B"), bytes.Repeat([]byte{4}, 64)},
					{[]byte("BC"), bytes.Repeat([]byte{4}, 64)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				cache := NewTestTrieCache[hash.H256]()
				recorder := NewRecorder[hash.H256]()
				{
					trie := NewTrieDB(
						root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						h, err := trie.GetHash(entry.key)
						assert.NoError(t, err)
						assert.NotNil(t, h)
					}
				}

				// get all values, by using cache and previous db
				{
					trie := NewTrieDB(root, db,
						WithCache[hash.H256, runtime.BlakeTwo256](cache),
					)
					trie.SetVersion(version)
					// get values for all entries from cache
					for _, entry := range keyValues {
						val, err := GetWith(trie, entry.key, func(d []byte) []byte { return d })
						require.NoError(t, err)
						require.NotNil(t, val)
						require.Equal(t, entry.value, *val)
					}
				}

				// get all hashes, without using cache and previous db
				// pass in the recorder as well
				{
					trie := NewTrieDB(root, db,
						WithRecorder[hash.H256, runtime.BlakeTwo256](recorder),
					)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						h, err := trie.GetHash(entry.key)
						assert.NoError(t, err)
						assert.NotNil(t, h)
					}
				}

			})
		}
	})

	t.Run("test_merkle_value_internal", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{
			trie.V0,
			trie.V1,
		} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("A"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AA"), bytes.Repeat([]byte{2}, 64)},
					{[]byte("AAAA"), bytes.Repeat([]byte{3}, 64)},
					{[]byte("AAB"), bytes.Repeat([]byte{4}, 64)},
					{[]byte("AABBBB"), bytes.Repeat([]byte{4}, 1)},
					{[]byte("AB"), bytes.Repeat([]byte{5}, 1)},
					{[]byte("B"), bytes.Repeat([]byte{6}, 1)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
				trie.SetVersion(version)
				for _, entry := range keyValues {
					h, err := trie.LookupFirstDescendant(entry.key)
					require.NoError(t, err)
					require.NotNil(t, h)
				}

				// Key is not present and has no descedant, but shares a prefix.
				for _, key := range []string{
					"AAAAX", "AABX", "ABX", "AABBBX", "BX", "AC", "AAAAX",
					"C", // Key shares the first nibble with b"A".
				} {
					mv, err := trie.LookupFirstDescendant([]byte(key))
					require.NoError(t, err)
					require.Nil(t, mv)
				}

				// Key not present, but has a descendent.
				hash, err := trie.LookupFirstDescendant([]byte("AAA"))
				require.NoError(t, err)
				require.NotNil(t, hash)
				expected, err := trie.LookupFirstDescendant([]byte("AAAA"))
				require.NoError(t, err)
				require.NotNil(t, expected)
				require.Equal(t, expected, hash)

				hash, err = trie.LookupFirstDescendant([]byte("AABB"))
				require.NoError(t, err)
				require.NotNil(t, hash)
				expected, err = trie.LookupFirstDescendant([]byte("AABBBB"))
				require.NoError(t, err)
				require.NotNil(t, expected)
				require.Equal(t, expected, hash)

				hash, err = trie.LookupFirstDescendant([]byte("AABBB"))
				require.NoError(t, err)
				require.NotNil(t, hash)
				expected, err = trie.LookupFirstDescendant([]byte("AABBBB"))
				require.NoError(t, err)
				require.NotNil(t, expected)
				require.Equal(t, expected, hash)

				// Prefix AABB in between AAB and AABBBB, but has different ending char.
				hash, err = trie.LookupFirstDescendant([]byte("AABBX"))
				require.NoError(t, err)
				require.Nil(t, hash)
			})
		}
	})

	t.Run("test_merkle_value_branches_internal", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{
			trie.V0,
			trie.V1,
		} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("AAAA"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AABA"), bytes.Repeat([]byte{2}, 64)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
				trie.SetVersion(version)

				// The hash is returned from the branch node.
				hash, err := trie.LookupFirstDescendant([]byte("A"))
				require.NoError(t, err)
				require.NotNil(t, hash)
				aaaa_hash, err := trie.LookupFirstDescendant([]byte("AAAA"))
				require.NoError(t, err)
				require.NotNil(t, aaaa_hash)
				aaba_hash, err := trie.LookupFirstDescendant([]byte("AABA"))
				require.NoError(t, err)
				require.NotNil(t, aaba_hash)

				assert.NotEqual(t, hash, aaaa_hash)
				assert.NotEqual(t, hash, aaba_hash)
			})
		}
	})

	t.Run("test_merkle_value_empty_trie_internal", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{
			trie.V0,
			trie.V1,
		} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					// test for both empty cases
					{[]byte{}, []byte{}},
					{nil, nil},
					{[]byte{}, nil},
					{nil, []byte{}},
				}

				for _, entry := range keyValues {
					db := NewMemoryDB()
					var root hash.H256
					{
						trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
						trie.SetVersion(version)

						require.NoError(t, trie.Set(entry.key, entry.value))

						err := trie.commit()
						require.NoError(t, err)
						require.NotEmpty(t, trie.rootHash)
						// Valid state root.
						root = trie.rootHash
					}

					// Data set is empty.
					trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
					trie.SetVersion(version)
					hash, err := trie.LookupFirstDescendant([]byte("A"))
					require.NoError(t, err)
					require.Nil(t, hash)

					hash, err = trie.LookupFirstDescendant([]byte("A"))
					require.NoError(t, err)
					require.Nil(t, hash)

					hash, err = trie.LookupFirstDescendant([]byte("AA"))
					require.NoError(t, err)
					require.Nil(t, hash)

					hash, err = trie.LookupFirstDescendant([]byte("AAA"))
					require.NoError(t, err)
					require.Nil(t, hash)

					hash, err = trie.LookupFirstDescendant([]byte("AAAA"))
					require.NoError(t, err)
					require.Nil(t, hash)
				}
			})
		}
	})

	t.Run("test_merkle_value_modification_internal", func(t *testing.T) {
		for _, version := range []trie.TrieLayout{
			trie.V0,
			trie.V1,
		} {
			t.Run(version.String(), func(t *testing.T) {
				keyValues := []struct {
					key   []byte
					value []byte
				}{
					{[]byte("AAAA"), bytes.Repeat([]byte{1}, 64)},
					{[]byte("AABA"), bytes.Repeat([]byte{2}, 64)},
				}

				db := NewMemoryDB()
				var root hash.H256
				{
					trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
					trie.SetVersion(version)
					for _, entry := range keyValues {
						require.NoError(t, trie.Set(entry.key, entry.value))
					}
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					root = trie.rootHash
				}

				var (
					aHashLHS    MerkleValue[hash.H256]
					aaaaHashLHS MerkleValue[hash.H256]
					aabaHashLHS MerkleValue[hash.H256]
				)
				{
					trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
					trie.SetVersion(version)

					// The hash is returned from the branch node.
					hash, err := trie.LookupFirstDescendant([]byte("A"))
					require.NoError(t, err)
					require.NotNil(t, hash)
					aaaa_hash, err := trie.LookupFirstDescendant([]byte("AAAA"))
					require.NoError(t, err)
					require.NotNil(t, aaaa_hash)
					aaba_hash, err := trie.LookupFirstDescendant([]byte("AABA"))
					require.NoError(t, err)
					require.NotNil(t, aaba_hash)

					// Ensure the hash is not from any leaf.
					assert.NotEqual(t, hash, aaaa_hash)
					assert.NotEqual(t, hash, aaba_hash)

					aHashLHS = hash
					aaaaHashLHS = aaaa_hash
					aabaHashLHS = aaba_hash
				}

				var (
					aHashRHS    MerkleValue[hash.H256]
					aaaaHashRHS MerkleValue[hash.H256]
					aabaHashRHS MerkleValue[hash.H256]
				)
				// Modify AABA and expect AAAA to return the same merkle value
				{
					trie := NewTrieDB[hash.H256, runtime.BlakeTwo256](root, db)
					trie.SetVersion(version)
					require.NoError(t, trie.Set([]byte("AABA"), bytes.Repeat([]byte{3}, 64)))
					err := trie.commit()
					require.NoError(t, err)
					require.NotEmpty(t, trie.rootHash)
					require.NotEqual(t, root, trie.rootHash)

					// The hash is returned from the branch node.
					hash, err := trie.LookupFirstDescendant([]byte("A"))
					require.NoError(t, err)
					require.NotNil(t, hash)
					aaaa_hash, err := trie.LookupFirstDescendant([]byte("AAAA"))
					require.NoError(t, err)
					require.NotNil(t, aaaa_hash)
					aaba_hash, err := trie.LookupFirstDescendant([]byte("AABA"))
					require.NoError(t, err)
					require.NotNil(t, aaba_hash)

					// Ensure the hash is not from any leaf.
					require.NotEqual(t, hash, aaaa_hash)
					require.NotEqual(t, hash, aaba_hash)

					aHashRHS = hash
					aaaaHashRHS = aaaa_hash
					aabaHashRHS = aaba_hash
				}

				// AAAA was not modified.
				require.Equal(t, aaaaHashLHS, aaaaHashRHS)
				// Changes to AABA must propagate to the root.
				require.NotEqual(t, aabaHashLHS, aabaHashRHS)
				require.NotEqual(t, aHashLHS, aHashRHS)

			})
		}
	})
}
