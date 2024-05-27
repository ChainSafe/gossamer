// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"

	"github.com/ChainSafe/gossamer/pkg/trie/cache"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

var ErrIncompleteDB = errors.New("incomplete database")

type entry struct {
	key   []byte
	value []byte
}

// TrieDB is a DB-backed patricia merkle trie implementation
// using lazy loading to fetch nodes
type TrieDB struct {
	rootHash common.Hash
	db       db.DBGetter
	cache    cache.TrieCache
	layout   trie.TrieLayout
	// rootHandle is an in-memory-trie-like representation of the node
	// references and new inserted nodes in the trie
	rootHandle NodeHandle
	// Storage is an in memory storage for nodes that we need to use during this
	// trieDB session (before nodes are committed to db)
	storage NodeStorage
	// deathRow is a set of nodes that we want to delete from db
	deathRow map[common.Hash]interface{}
}

func NewEmptyTrieDB(db db.Database, cache cache.TrieCache) *TrieDB {
	root := HashedNullNode
	return NewTrieDB(root, db, cache)
}

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB(rootHash common.Hash, db db.DBGetter, cache cache.TrieCache) *TrieDB {
	rootHandle := Hash{hash: rootHash}

	return &TrieDB{
		rootHash:   rootHash,
		cache:      cache,
		db:         db,
		storage:    NewNodeStorage(),
		rootHandle: rootHandle,
		deathRow:   make(map[common.Hash]interface{}),
	}
}

// Hash returns the hashed root of the trie.
func (t *TrieDB) Hash() (common.Hash, error) {
	// This is trivial since it is a read only trie, but will change when we
	// support writes
	return t.rootHash, nil
}

// MustHash returns the hashed root of the trie.
// It panics if it fails to hash the root node.
func (t *TrieDB) MustHash() common.Hash {
	h, err := t.Hash()
	if err != nil {
		panic(err)
	}

	return h
}

// Get returns the value in the node of the trie
// which matches its key with the key given.
// Note the key argument is given in little Endian format.
func (t *TrieDB) Get(key []byte) []byte {
	keyNibbles := nibbles.KeyLEToNibbles(key)

	val, err := t.lookup(keyNibbles, keyNibbles, t.rootHandle)
	if err != nil {
		return nil
	}

	return val
}

func (t *TrieDB) lookup(fullKey []byte, partialKey []byte, handle NodeHandle) ([]byte, error) {
	prefix := fullKey

	for {
		var partialIdx int
		switch node := handle.(type) {
		case Hash:
			lookup := NewTrieLookup(t.db, node.hash, t.cache)
			val, err := lookup.lookupValue(fullKey)
			if err != nil {
				return nil, err
			}
			return val, nil
		case InMemory:
			switch n := t.storage.get(node.idx).(type) {
			case Empty:
				return nil, nil
			case Leaf:
				if bytes.Equal(n.partialKey, partialKey) {
					return inMemoryFetchedValue(n.value, prefix, t.db, fullKey)
				} else {
					return nil, nil
				}
			case Branch:
				if bytes.Equal(n.partialKey, partialKey) {
					return inMemoryFetchedValue(n.value, prefix, t.db, fullKey)
				} else if bytes.HasPrefix(partialKey, n.partialKey) {
					idx := partialKey[len(n.partialKey)]
					child := n.children[idx]
					if child != nil {
						partialIdx = 1 + len(n.partialKey)
						handle = child
					}
				} else {
					return nil, nil
				}
			}
		}
		partialKey = partialKey[partialIdx:]
	}
}

// Internal methods
func (t *TrieDB) getRootNode() (codec.EncodedNode, error) {
	encodedNode, err := t.db.Get(t.rootHash[:])
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(encodedNode)
	return codec.Decode(reader)
}

// Internal methods

func (t *TrieDB) getNodeAt(key []byte) (codec.EncodedNode, error) {
	lookup := NewTrieLookup(t.db, t.rootHash, t.cache)
	node, err := lookup.lookupNode(nibbles.KeyLEToNibbles(key))
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (t *TrieDB) getNode(
	merkleValue codec.MerkleValue,
) (node codec.EncodedNode, err error) {
	switch n := merkleValue.(type) {
	case codec.InlineNode:
		reader := bytes.NewReader(n.Data)
		return codec.Decode(reader)
	case codec.HashedNode:
		encodedNode, err := t.db.Get(n.Data)
		if err != nil {
			return nil, err
		}
		reader := bytes.NewReader(encodedNode)
		return codec.Decode(reader)
	default: // should never happen
		panic("unreachable")
	}
}

// Put inserts the given key / value pair into the trie
func (t *TrieDB) Put(key, value []byte) error {
	// Insert the node and update the rootHandle
	var oldValue nodeValue

	rootHandle := t.rootHandle
	keyNibbles := nibbles.KeyLEToNibbles(key)
	newHandle, _, err := t.insertAt(rootHandle, keyNibbles, value, &oldValue)
	if err != nil {
		return err
	}
	t.rootHandle = InMemory{idx: newHandle}
	return nil
}

// insertAt inserts the given key / value pair into the node referenced by the
// node handle `handle`
func (t *TrieDB) insertAt(
	handle NodeHandle,
	keyNibbles,
	value []byte,
	oldValue *nodeValue,
) (storageHandle StorageHandle, changed bool, err error) {
	switch h := handle.(type) {
	case InMemory:
		storageHandle = h.idx
	case Hash:
		storageHandle, err = t.lookupNode(h.hash)
		if err != nil {
			return StorageHandle{}, false, err
		}
	}

	stored := t.storage.destroy(storageHandle)
	newStored, changed, err := t.inspect(stored, keyNibbles, func(stored Node, keyNibbles []byte) (action, error) {
		return t.insertInspector(stored, keyNibbles, value, oldValue)
	})
	if err != nil {
		return StorageHandle{}, false, err
	}
	return t.storage.alloc(newStored), changed, nil
}

// inspect inspects the given node `stored` and calls the `inspector` function
// then returns the new node and a boolean indicating if the node has changed
func (t *TrieDB) inspect(
	stored StoredNode,
	key []byte,
	inspector func(Node, []byte) (action, error),
) (StoredNode, bool, error) {
	switch n := stored.(type) {
	case New:
		action, err := inspector(n.node, key)
		if err != nil {
			return nil, false, err
		}
		switch a := action.(type) {
		case restore:
			return NewStoredNodeNew(a.node), false, nil
		case replace:
			return NewStoredNodeNew(a.node), true, nil
		case delete:
			return nil, false, nil
		default:
			panic("unreachable")
		}
	case Cached:
		action, err := inspector(n.node, key)
		if err != nil {
			return nil, false, err
		}
		switch a := action.(type) {
		case restore:
			return Cached{a.node, n.hash}, false, nil
		case replace:
			t.deathRow[n.hash] = nil
			return NewStoredNodeNew(a.node), true, nil
		case delete:
			t.deathRow[n.hash] = nil
			return nil, false, nil
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

// insertInspector inserts the new key / value pair into the given node `stored`
func (t *TrieDB) insertInspector(stored Node, keyNibbles []byte, value []byte, oldValue *nodeValue) (action, error) {
	partial := keyNibbles

	switch n := stored.(type) {
	case Empty:
		// If the node is empty we have to replace it with a leaf node with the
		// new value
		value := NewValue(value, t.layout.MaxInlineValue())
		return replace{node: Leaf{partialKey: partial, value: value}}, nil
	case Leaf:
		existingKey := n.partialKey
		common := nibbles.CommonPrefix(partial, existingKey)

		if common == len(existingKey) && common == len(partial) {
			// We are trying to insert a value in the same leaf so we just need
			// to replace the value
			value := NewValue(value, t.layout.MaxInlineValue())
			unchanged := n.value == value
			t.replaceOldValue(oldValue, n.value)
			leaf := Leaf{partialKey: n.partialKey, value: n.value}
			if unchanged {
				// If the value didn't change we can restore this leaf previously
				// taken from storage
				return restore{leaf}, nil
			}
			return replace{leaf}, nil
		} else if common < len(existingKey) {
			// If the common prefix is less than this leaf's key then we need to
			// create a branch node. Then add this leaf and the new value to the
			// branch
			var children [codec.ChildrenCapacity]NodeHandle

			idx := existingKey[common]

			// Modify the existing leaf partial key and add it as a child
			newLeaf := Leaf{existingKey[common+1:], n.value}
			children[idx] = newInMemoryNodeHandle(t.storage.alloc(New{node: newLeaf}))
			branch := Branch{
				partialKey: partial[:common],
				children:   children,
				value:      nil,
			}

			// Use the inspector to add the new leaf as part of this branch
			branchAction, err := t.insertInspector(branch, keyNibbles, value, oldValue)
			if err != nil {
				return nil, err
			}
			return replace{branchAction.getNode()}, nil
		} else {
			// we have a common prefix but the new key is longer than the existing
			// then we turn this leaf into a branch and add the new leaf as a child
			var branch Node = Branch{
				partialKey: n.partialKey,
				children:   [codec.ChildrenCapacity]NodeHandle{},
				value:      n.value,
			}
			// Use the inspector to add the new leaf as part of this branch
			// And replace the node with the new branch
			action, err := t.insertInspector(branch, keyNibbles, value, oldValue)
			if err != nil {
				return nil, err
			}
			branch = action.getNode()
			return replace{branch}, nil
		}
	case Branch:
		existingKey := n.partialKey
		common := nibbles.CommonPrefix(partial, existingKey)

		if common == len(existingKey) && common == len(partial) {
			// We are trying to insert a value in the same branch so we just need
			// to replace the value
			value := NewValue(value, t.layout.MaxInlineValue())
			unchanged := n.value == value
			branch := Branch{existingKey, n.children, value}

			t.replaceOldValue(oldValue, n.value)
			if unchanged {
				// If the value didn't change we can restore this leaf previously
				// taken from storage
				return restore{branch}, nil
			}
			return replace{branch}, nil
		} else if common < len(existingKey) {
			// If the common prefix is less than this branch's key then we need to
			// create a branch node in between.
			// Then add this branch and the new value to the new branch

			// So we take this branch and we add it as a child of the new one
			branchPartial := existingKey[common+1:]
			lowerBranch := Branch{branchPartial, n.children, n.value}
			allocStorage := t.storage.alloc(New{node: lowerBranch})

			children := [codec.ChildrenCapacity]NodeHandle{}
			ix := existingKey[common]
			children[ix] = newInMemoryNodeHandle(allocStorage)

			value := NewValue(value, t.layout.MaxInlineValue())

			if len(partial)-common == 0 {
				// The value should be part of the branch
				return replace{
					Branch{
						existingKey[:common],
						children,
						value,
					},
				}, nil
			} else {
				// Value is in a leaf under the branch so we have to create it
				storedLeaf := Leaf{partial[common+1:], value}
				leaf := t.storage.alloc(New{node: storedLeaf})

				ix = partial[common]
				children[ix] = newInMemoryNodeHandle(leaf)
				return replace{
					Branch{
						existingKey[:common],
						children,
						nil,
					},
				}, nil
			}
		} else {
			// append after common == existing_key and partial > common
			idx := partial[common]
			keyNibbles = keyNibbles[common+1:]
			child := n.children[idx]
			if child != nil {
				// We have to add the new value to the child
				newChild, changed, err := t.insertAt(child, keyNibbles, value, oldValue)
				if err != nil {
					return nil, err
				}
				n.children[idx] = newInMemoryNodeHandle(newChild)
				if !changed {
					// Our branch is untouched so we can restore it
					branch := Branch{
						existingKey,
						n.children,
						n.value,
					}

					return restore{branch}, nil
				}
			} else {
				// Original has nothing here so we have to create a new leaf
				value := NewValue(value, t.layout.MaxInlineValue())
				leaf := t.storage.alloc(New{node: Leaf{keyNibbles, value}})
				n.children[idx] = newInMemoryNodeHandle(leaf)
			}
			return replace{Branch{
				existingKey,
				n.children,
				n.value,
			}}, nil
		}
	default:
		panic("unreachable")
	}
}

func (t *TrieDB) replaceOldValue(
	oldValue *nodeValue,
	storedValue nodeValue,
) {
	switch oldv := storedValue.(type) {
	case valueRef, newValueRef:
		hash := oldv.getHash()
		if hash != common.EmptyHash {
			t.deathRow[oldv.getHash()] = nil
		}
	}
	*oldValue = storedValue
}

// lookup node in DB and add it in storage, return storage handle
// TODO: implement cache to improve performance
func (t *TrieDB) lookupNode(hash common.Hash) (StorageHandle, error) {
	encodedNode, err := t.db.Get(hash[:])
	if err != nil {
		return StorageHandle{}, ErrIncompleteDB
	}

	node, err := newNodeFromEncoded(hash, encodedNode, t.storage)
	if err != nil {
		return StorageHandle{-1}, err
	}

	return t.storage.alloc(Cached{
		node: node,
		hash: hash,
	}), nil
}

var _ trie.TrieRead = (*TrieDB)(nil)
