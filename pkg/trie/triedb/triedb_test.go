// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/db"
	"github.com/stretchr/testify/assert"
)

func TestInsertions(t *testing.T) {
	t.Parallel()

	type entry struct {
		key   []byte
		value []byte
	}

	testCases := map[string]struct {
		trieEntries []entry
		key         []byte
		value       []byte
		stored      NodeStorage
	}{
		"nil_parent": {
			trieEntries: []entry{},
			key:         []byte{1},
			value:       []byte("leaf"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{1},
							value:      inline{Data: []byte("leaf")},
						},
					},
				},
			},
		},
		"branch_parent": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("branch"),
				},
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{},
							value:      inline{Data: []byte("leaf")},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{1},
							value:      inline{Data: []byte("branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								InMemory{StorageHandle(0)}, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"branch_in_between_rearrange": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("branch"),
				},
				{
					key:   []byte{1, 0, 1},
					value: []byte("leaf"),
				},
			},
			key:   []byte{1, 0},
			value: []byte("in between branch"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Branch{
							partialKey: []byte{},
							value:      inline{Data: []byte("in between branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								nil, InMemory{StorageHandle(1)}, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Leaf{
							partialKey: []byte{},
							value:      inline{Data: []byte("leaf")},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{1},
							value:      inline{Data: []byte("branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								InMemory{StorageHandle(0)}, nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"branch_in_between": {
			trieEntries: []entry{
				{
					key:   []byte{1, 0},
					value: []byte("branch"),
				},
				{
					key:   []byte{1, 0, 1},
					value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("top branch"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{},
							value:      inline{Data: []byte("leaf")},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{},
							value:      inline{Data: []byte("branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								nil, InMemory{StorageHandle(0)}, nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{1},
							value:      inline{Data: []byte("top branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								InMemory{StorageHandle(1)}, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"override_branch_value": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("branch"),
				},
				{
					key:   []byte{1, 0},
					value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("new branch"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{},
							value:      inline{Data: []byte("leaf")},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{1},
							value:      inline{Data: []byte("new branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								InMemory{StorageHandle(0)}, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"override_branch_value_same_value": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("branch"),
				},
				{
					key:   []byte{1, 0},
					value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("branch"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{},
							value:      inline{Data: []byte("leaf")},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{1},
							value:      inline{Data: []byte("branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								InMemory{StorageHandle(0)}, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
				},
			},
		},
		"override_leaf_of_branch_value_same_value": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("branch"),
				},
				{
					key:   []byte{1, 0},
					value: []byte("leaf"),
				},
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Branch{
							partialKey: []byte{1},
							value:      inline{Data: []byte("branch")},
							children: [codec.ChildrenCapacity]NodeHandle{
								InMemory{StorageHandle(1)}, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
							},
						},
					},
					NewStoredNode{
						Leaf{
							partialKey: []byte{},
							value:      inline{Data: []byte("leaf")},
						},
					},
				},
			},
		},
		"override_leaf_parent": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("leaf"),
				},
			},
			key:   []byte{1},
			value: []byte("new leaf"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{1},
							value:      inline{Data: []byte("new leaf")},
						},
					},
				},
			},
		},
		"write_same_leaf_value_to_leaf_parent": {
			trieEntries: []entry{
				{
					key:   []byte{1},
					value: []byte("same"),
				},
			},
			key:   []byte{1},
			value: []byte("same"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{1},
							value:      inline{Data: []byte("same")},
						},
					},
				},
			},
		},
		"write_leaf_as_divergent_child_next_to_parent_leaf": {
			trieEntries: []entry{
				{
					key:   []byte{1, 2},
					value: []byte("original leaf"),
				},
			},
			key:   []byte{2, 3},
			value: []byte("leaf"),
			stored: NodeStorage{
				nodes: []StoredNode{
					NewStoredNode{
						Leaf{
							partialKey: []byte{2},
							value:      inline{Data: []byte("original leaf")},
						},
					},
					NewStoredNode{
						Leaf{
							partialKey: []byte{3},
							value:      inline{Data: []byte("leaf")},
						},
					},
					NewStoredNode{
						Branch{
							partialKey: []byte{},
							value:      nil,
							children: [codec.ChildrenCapacity]NodeHandle{
								nil,
								InMemory{StorageHandle(0)}, InMemory{StorageHandle(1)},
								nil, nil, nil, nil, nil, nil, nil, nil,
								nil, nil, nil, nil, nil,
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
			inmemoryDB := db.NewMemoryDB(make([]byte, 1))
			trie := NewEmptyTrieDB(inmemoryDB, nil)

			for _, entry := range testCase.trieEntries {
				assert.NoError(t, trie.insert(entry.key, entry.value))
			}

			// Add new key-value pair
			err := trie.insert(testCase.key, testCase.value)
			assert.NoError(t, err)

			// Check we have what we expect
			assert.Equal(t, testCase.stored.nodes, trie.storage.nodes)
		})
	}
}
