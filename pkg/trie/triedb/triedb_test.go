// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

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
							value:      inline[hash.H256]([]byte("leaf")),
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
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x01}},
							value:      inline[hash.H256]([]byte("branch")),
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
							value:      inline[hash.H256]([]byte("in between branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(1), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x01}, Offset: 1},
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}, Offset: 0},
							value:      inline[hash.H256]([]byte("branch")),
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
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline[hash.H256]([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(0), nil, nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline[hash.H256]([]byte("top branch")),
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
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline[hash.H256]([]byte("new branch")),
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
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline[hash.H256]([]byte("branch")),
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
							value:      inline[hash.H256]([]byte("branch")),
							children: [codec.ChildrenCapacity]NodeHandle{
								inMemory(1), nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0}, Offset: 1},
							value:      inline[hash.H256]([]byte("leaf")),
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
							value:      inline[hash.H256]([]byte("new leaf")),
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
							value:      inline[hash.H256]([]byte("same")),
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
							value:      inline[hash.H256]([]byte("original leaf")),
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{0x03}},
							value:      inline[hash.H256]([]byte("leaf")),
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
			inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
			trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

			for _, entry := range testCase.trieEntries {
				require.NoError(t, trie.Put(entry.Key, entry.Value))
			}
			// Add new key-value pair
			err := trie.Put(testCase.key, testCase.value)
			require.NoError(t, err)

			if !testCase.dontCheck {
				// Check values for keys
				for _, entry := range testCase.trieEntries {
					require.Equal(t, entry.Value, trie.Get(entry.Key))
				}
			}
			require.Equal(t, testCase.value, trie.Get(testCase.key))

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
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
				},
			},
		},
		"empty_trie": {
			key: []byte{1},
			expected: nodeStorage[hash.H256]{
				nodes: []StoredNode{nil},
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
				nodes: []StoredNode{nil},
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
					nil,
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: []byte{1, 0}},
							value:      inline[hash.H256]([]byte("leaf")),
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
							value:      inline[hash.H256]([]byte("leaf1")),
						},
					},
					NewStoredNode{
						Leaf[hash.H256]{
							partialKey: nodeKey{Data: make([]byte, 0)},
							value:      inline[hash.H256]([]byte("leaf2")),
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
			inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
			trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

			for _, entry := range testCase.trieEntries {
				assert.NoError(t, trie.Put(entry.Key, entry.Value))
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
							value:      inline[hash.H256]([]byte("new leaf")),
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
							value:      inline[hash.H256]([]byte("leaf")),
						},
					},
					NewStoredNode{
						Branch[hash.H256]{
							partialKey: nodeKey{Data: []byte{1}},
							value:      inline[hash.H256]([]byte("new branch")),
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
			inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
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

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := trie.Put([]byte("leaf"), []byte("leafvalue"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		// 1 leaf
		assert.Len(t, inmemoryDB.data, 1)

		// Get values using lazy loading
		value := trie.Get([]byte("leaf"))
		assert.Equal(t, []byte("leafvalue"), value)
	})

	t.Run("commit_branch_and_inlined_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := trie.Put([]byte("branchleaf"), []byte("leafvalue"))
		assert.NoError(t, err)
		err = trie.Put([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		// 1 branch with its inlined leaf
		assert.Len(t, inmemoryDB.data, 1)

		// Get values using lazy loading
		value := trie.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), value)
		value = trie.Get([]byte("branchleaf"))
		assert.Equal(t, []byte("leafvalue"), value)
	})

	t.Run("commit_branch_and_hashed_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := tr.Put([]byte("branchleaf"), make([]byte, 40))
		assert.NoError(t, err)
		err = tr.Put([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 branch with 1 hashed leaf child
		// 1 hashed leaf
		assert.Len(t, inmemoryDB.data, 2)

		// Get values using lazy loading
		value := tr.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), value)
		value = tr.Get([]byte("branchleaf"))
		assert.Equal(t, make([]byte, 40), value)
	})

	t.Run("commit_leaf_with_hashed_value", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Put([]byte("leaf"), make([]byte, 40))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.data, 2)

		// Get values using lazy loading
		value := tr.Get([]byte("leaf"))
		assert.Equal(t, make([]byte, 40), value)
	})

	t.Run("commit_leaf_with_hashed_value_then_remove_it", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Put([]byte("leaf"), make([]byte, 40))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.data, 2)

		// Get values using lazy loading
		err = tr.Delete([]byte("leaf"))
		assert.NoError(t, err)
		tr.commit()
		assert.Len(t, inmemoryDB.data, 0)
	})

	t.Run("commit_branch_and_hashed_leaf_with_hashed_value", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Put([]byte("branchleaf"), make([]byte, 40))
		assert.NoError(t, err)
		err = tr.Put([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 branch with 1 hashed leaf child
		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.data, 3)

		// Get values using lazy loading
		value := tr.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), value)
		value = tr.Get([]byte("branchleaf"))
		assert.Equal(t, make([]byte, 40), value)
	})

	t.Run("commit_branch_and_hashed_leaf_with_hashed_value_then_delete_it", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		tr := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)
		tr.SetVersion(trie.V1)

		err := tr.Put([]byte("branchleaf"), make([]byte, 40))
		assert.NoError(t, err)
		err = tr.Put([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = tr.commit()
		assert.NoError(t, err)

		// 1 branch with 1 hashed leaf child
		// 1 hashed leaf with hashed value
		// 1 hashed value
		assert.Len(t, inmemoryDB.data, 3)

		err = tr.Delete([]byte("branchleaf"))
		assert.NoError(t, err)
		tr.commit()

		// 1 branch transformed in a leaf
		// previous leaf was deleted
		// previous hashed (V1) value was deleted too
		assert.Len(t, inmemoryDB.data, 1)
	})

	t.Run("commit_branch_with_leaf_then_delete_leaf", func(t *testing.T) {
		t.Parallel()

		inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
		trie := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](inmemoryDB)

		err := trie.Put([]byte("branchleaf"), []byte("leafvalue"))
		assert.NoError(t, err)
		err = trie.Put([]byte("branch"), []byte("branchvalue"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		err = trie.Delete([]byte("branchleaf"))
		assert.NoError(t, err)

		err = trie.commit()
		assert.NoError(t, err)

		// 1 branch transformed in a leaf
		// previous leaf was deleted
		assert.Len(t, inmemoryDB.data, 1)

		v := trie.Get([]byte("branch"))
		assert.Equal(t, []byte("branchvalue"), v)
		v = trie.Get([]byte("branchleaf"))
		assert.Nil(t, v)
	})
}
