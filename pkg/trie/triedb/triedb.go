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

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB(rootHash common.Hash, db db.DBGetter, cache cache.TrieCache) *TrieDB {
	rootHandle := Hash{hash: rootHash}

	return &TrieDB{
		rootHash:   rootHash,
		cache:      cache,
		db:         db,
		storage:    NewNodeStorage(),
		rootHandle: rootHandle,
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
					return InMemoryFetchedValue(n.value, prefix, t.db, fullKey)
				} else {
					return nil, nil
				}
			case Branch:
				if bytes.Equal(n.partialKey, partialKey) {
					return InMemoryFetchedValue(n.value, prefix, t.db, fullKey)
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

func (t *TrieDB) Put(key, value []byte) error {
	// Insert the node and update the rootHandle
	var oldValue Value

	rootHandle := t.rootHandle
	keyNibbles := nibbles.KeyLEToNibbles(key)
	newHandle, _, err := t.insertAt(rootHandle, keyNibbles, value, &oldValue)
	if err != nil {
		return err
	}
	t.rootHandle = InMemory{idx: newHandle}
	return nil
}

func (t *TrieDB) insertAt(
	handle NodeHandle,
	keyNibbles,
	value []byte,
	oldValue *Value,
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
	newStored, changed, err := t.inspect(stored, keyNibbles, func(stored Node, keyNibbles []byte) (Action, error) {
		return t.insertInspector(stored, keyNibbles, value, oldValue)
	})
	if err != nil {
		return StorageHandle{}, false, err
	}
	return t.storage.alloc(newStored), changed, nil
}

func (t *TrieDB) inspect(
	stored StoredNode,
	key []byte,
	inspector func(Node, []byte) (Action, error),
) (StoredNode, bool, error) {
	//currentKey := key
	switch n := stored.(type) {
	case New:
		action, err := inspector(n.node, key)
		if err != nil {
			return nil, false, err
		}
		switch action.(type) {
		case Restore:
			return NewNewNode(n.node), false, nil
		case Replace:
			return NewNewNode(n.node), true, nil
		case Delete:
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
		case Restore:
			return Cached{a.node, n.hash}, false, nil
		case Replace:
			t.deathRow[n.hash] = nil
			return NewNewNode(a.node), true, nil
		case Delete:
			t.deathRow[n.hash] = nil
			return nil, false, nil
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

func (t *TrieDB) insertInspector(stored Node, keyNibbles []byte, value []byte, oldValue *Value) (Action, error) {
	partial := keyNibbles

	switch n := stored.(type) {
	case Empty:
		value := NewValue(value, t.layout.MaxInlineValue())
		return Replace{node: Leaf{partialKey: partial, value: value}}, nil
	case Leaf:
		existingKey := n.partialKey
		common := nibbles.CommonPrefix(partial, existingKey)
		if common == len(existingKey) && common == len(partial) {
			// Equivalent leaf
			value := NewValue(value, t.layout.MaxInlineValue())
			unchanged := n.value == value
			t.replaceOldValue(oldValue, n.value)
			if unchanged {
				// Unchanged then restore
				return Restore{Leaf{partialKey: n.partialKey, value: n.value}}, nil
			}
			return Replace{Leaf{partialKey: n.partialKey, value: n.value}}, nil
		} else {
			// fully shared prefix then create a branch
			var branch Node = Branch{
				partialKey: existingKey,
				children:   [codec.ChildrenCapacity]NodeHandle{},
				value:      n.value,
			}
			// Insert into new branch
			action, err := t.insertInspector(branch, keyNibbles, value, oldValue)
			if err != nil {
				return nil, err
			}
			branch = action.getNode()
			return Replace{branch}, nil
		}
	case Branch:
		existingKey := n.partialKey
		common := nibbles.CommonPrefix(partial, existingKey)
		if common == len(existingKey) && common == len(partial) {
			value := NewValue(value, t.layout.MaxInlineValue())
			unchanged := n.value == value
			branch := Branch{existingKey, n.children, value}

			t.replaceOldValue(oldValue, n.value)
			if unchanged {
				// Unchanged then restore
				return Restore{branch}, nil
			}
			return Replace{branch}, nil
		} else if common < len(existingKey) {
			// insert a branch value in between
			branchPartial := existingKey[common+1:]
			low := Branch{branchPartial, n.children, n.value}
			ix := existingKey[common]
			children := [codec.ChildrenCapacity]NodeHandle{}
			allocStorage := t.storage.alloc(New{node: low})

			children[ix] = allocStorage.toNodeHandle()

			value := NewValue(value, t.layout.MaxInlineValue())
			if len(partial)-common == 0 {
				// Value should be part of the branch
				return Replace{
					Branch{
						existingKey[common:],
						children,
						value,
					},
				}, nil
			} else {
				// Value is in a leaf under the branch
				ix = partial[common]
				storedLeaf := Leaf{partial[common+1:], value}

				leaf := t.storage.alloc(New{node: storedLeaf})

				children[ix] = leaf.toNodeHandle()
				return Replace{
					Branch{
						existingKey[common:],
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
				newChild, changed, err := t.insertAt(child, keyNibbles, value, oldValue)
				if err != nil {
					return nil, err
				}
				n.children[idx] = newChild.toNodeHandle()
				if !changed {
					// Our branch is untouched
					branch := Branch{
						existingKey,
						n.children,
						n.value,
					}

					return Restore{branch}, nil
				}
			} else {
				// Original has nothing here so we have to create a new leaf
				value := NewValue(value, t.layout.MaxInlineValue())
				leaf := t.storage.alloc(New{node: Leaf{keyNibbles, value}})
				n.children[idx] = leaf.toNodeHandle()
			}
			return Replace{Branch{
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
	oldValue *Value,
	storedValue Value,
) {
	switch oldv := storedValue.(type) {
	case ValueRef, NewValueRef:
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
