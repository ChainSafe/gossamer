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

// Delete deletes the given key from the trie
func (t *TrieDB) Delete(key []byte) error {
	var oldValue Value

	rootHandle := t.rootHandle
	keyNibbles := nibbles.KeyLEToNibbles(key)
	removeResult, err := t.removeAt(rootHandle, keyNibbles, &oldValue)
	if err != nil {
		return err
	}
	if removeResult != nil {
		t.rootHandle = InMemory{idx: removeResult.handle}
	} else {
		t.rootHandle = Hash{HashedNullNode}
		t.rootHash = HashedNullNode
	}

	return nil
}

// Put inserts the given key / value pair into the trie
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

// insertAt inserts the given key / value pair into the node referenced by the
// node handle `handle`
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
	result, err := t.inspect(stored, keyNibbles, func(node Node, keyNibbles []byte) (Action, error) {
		return t.insertInspector(node, keyNibbles, value, oldValue)
	})
	if err != nil {
		return StorageHandle{}, false, err
	}

	if result == nil {
		panic("Insertion never deletes")

	}
	return t.storage.alloc(result.stored), result.changed, nil
}

type RemoveAtResult struct {
	handle  StorageHandle
	changed bool
}

func (t *TrieDB) removeAt(
	handle NodeHandle,
	keyNibbles []byte,
	oldValue *Value,
) (*RemoveAtResult, error) {
	var stored StoredNode
	switch h := handle.(type) {
	case InMemory:
		stored = t.storage.destroy(h.idx)
	case Hash:
		handle, err := t.lookupNode(h.hash)
		if err != nil {
			return nil, err
		}
		stored = t.storage.destroy(handle)
	}

	result, err := t.inspect(stored, keyNibbles, func(node Node, keyNibbles []byte) (Action, error) {
		return t.removeInspector(node, keyNibbles, oldValue)
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return &RemoveAtResult{t.storage.alloc(result.stored), result.changed}, err
}

type InspectResult struct {
	stored  StoredNode
	changed bool
}

// inspect inspects the given node `stored` and calls the `inspector` function
// then returns the new node and a boolean indicating if the node has changed
func (t *TrieDB) inspect(
	stored StoredNode,
	key []byte,
	inspector func(Node, []byte) (Action, error),
) (*InspectResult, error) {
	switch n := stored.(type) {
	case New:
		action, err := inspector(n.node, key)
		if err != nil {
			return nil, err
		}
		switch a := action.(type) {
		case Restore:
			return &InspectResult{NewStoredNodeNew(a.node), false}, nil
		case Replace:
			return &InspectResult{NewStoredNodeNew(a.node), true}, nil
		case Delete:
			return nil, nil
		default:
			panic("unreachable")
		}
	case Cached:
		action, err := inspector(n.node, key)
		if err != nil {
			return nil, err
		}
		switch a := action.(type) {
		case Restore:
			return &InspectResult{Cached{a.node, n.hash}, false}, nil
		case Replace:
			t.deathRow[n.hash] = nil
			return &InspectResult{NewStoredNodeNew(a.node), true}, nil
		case Delete:
			t.deathRow[n.hash] = nil
			return nil, nil
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

func (t *TrieDB) fix(node Node) (Node, error) {
	usedIndex := make([]byte, 0)

	switch branch := node.(type) {
	case Branch:
		for i := 0; i < codec.ChildrenCapacity; i++ {
			if branch.children[i] == nil || len(usedIndex) > 1 {
				continue
			}
			if branch.children[i] != nil {
				if len(usedIndex) == 0 {
					usedIndex = append(usedIndex, byte(i))
				} else if len(usedIndex) == 1 {
					usedIndex = append(usedIndex, byte(i))
					break
				}
			}
		}

		if len(usedIndex) == 0 {
			if branch.value == nil {
				panic("branch with no subvalues. Something went wrong.")
			}

			// Make it a leaf
			return Leaf{branch.partialKey, branch.value}, nil
		} else if len(usedIndex) == 1 && branch.value == nil {
			// Only one onward node. use child instead
			idx := usedIndex[0] //nolint:gosec
			// take child and replace it to nil
			child := branch.children[idx]
			branch.children[idx] = nil

			var stored StoredNode
			switch n := child.(type) {
			case InMemory:
				stored = t.storage.destroy(n.idx)
			case Hash:
				handle, err := t.lookupNode(n.hash)
				if err != nil {
					return nil, err
				}
				stored = t.storage.destroy(handle)
			}

			var childNode Node
			switch n := stored.(type) {
			case New:
				childNode = n.node
			case Cached:
				t.deathRow[n.hash] = nil
				childNode = n.node
			}

			combinedKey := branch.partialKey
			combinedKey = append(combinedKey, idx)
			combinedKey = append(combinedKey, childNode.getPartialKey()...)

			switch n := childNode.(type) {
			case Leaf:
				return Leaf{combinedKey, n.value}, nil
			case Branch:
				return Branch{combinedKey, n.children, n.value}, nil
			default:
				panic("unreachable")
			}
		} else {
			// Restore branch
			return Branch{branch.partialKey, branch.children, branch.value}, nil
		}
	default:
		panic("fix should be only called with branch nodes")
	}
}

// removeInspector removes the key node from the given node `stored`
func (t *TrieDB) removeInspector(stored Node, keyNibbles []byte, oldValue *Value) (Action, error) {
	partial := keyNibbles

	switch n := stored.(type) {
	case Empty:
		return Delete{}, nil
	case Leaf:
		existingKey := n.partialKey

		if bytes.Equal(existingKey, partial) {
			// This is the node we are looking for so we delete it
			t.replaceOldValue(oldValue, n.value)
			return Delete{}, nil
		} else {
			// Wrong partial, so we return the node as is
			return Restore{n}, nil
		}
	case Branch:
		if len(partial) == 0 {
			if n.value == nil {
				// Nothing to delete since the branch doesn't contains a value
				return Restore{n}, nil
			} else {
				// The branch contains the value so we delete it
				t.replaceOldValue(oldValue, n.value)
				newNode, err := t.fix(Branch{n.partialKey, n.children, nil})
				if err != nil {
					return nil, err
				}
				return Replace{newNode}, nil
			}
		} else {
			common := nibbles.CommonPrefix(n.partialKey, partial)
			existingLength := len(n.partialKey)

			if common == existingLength && common == len(partial) {
				// Replace value
				if n.value != nil {
					t.replaceOldValue(oldValue, n.value)
					newNode, err := t.fix(Branch{n.partialKey, n.children, nil})
					return Replace{newNode}, err
				} else {
					return Restore{Branch{n.partialKey, n.children, nil}}, nil
				}
			} else if common < existingLength {
				return Restore{n}, nil
			} else {
				// Check children
				idx := partial[common]
				// take child and replace it to nil
				child := n.children[idx]
				n.children[idx] = nil

				if child != nil {
					removeAtResult, err := t.removeAt(child, keyNibbles[len(n.partialKey)+1:], oldValue)
					if err != nil {
						return nil, err
					}

					if removeAtResult != nil {
						n.children[idx] = removeAtResult.handle.toNodeHandle()
						if removeAtResult.changed {
							return Replace{n}, nil
						} else {
							return Restore{n}, nil
						}
					} else {
						newNode, err := t.fix(n)
						if err != nil {
							return nil, err
						}
						return Replace{newNode}, nil
					}
				}
				return Restore{n}, nil
			}
		}
	default:
		panic("unreachable")
	}
}

// insertInspector inserts the new key / value pair into the given node `stored`
func (t *TrieDB) insertInspector(stored Node, keyNibbles []byte, value []byte, oldValue *Value) (Action, error) {
	partial := keyNibbles

	switch n := stored.(type) {
	case Empty:
		// If the node is empty we have to replace it with a leaf node with the
		// new value
		value := NewValue(value, t.layout.MaxInlineValue())
		return Replace{node: Leaf{partialKey: partial, value: value}}, nil
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
				return Restore{leaf}, nil
			}
			return Replace{leaf}, nil
		} else if common < len(existingKey) {
			// If the common prefix is less than this leaf's key then we need to
			// create a branch node. Then add this leaf and the new value to the
			// branch
			var children [codec.ChildrenCapacity]NodeHandle

			idx := existingKey[common]

			// Modify the existing leaf partial key and add it as a child
			newLeaf := Leaf{existingKey[common+1:], n.value}
			children[idx] = t.storage.alloc(New{node: newLeaf}).toNodeHandle()
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
			return Replace{branchAction.getNode()}, nil
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
			return Replace{branch}, nil
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
				return Restore{branch}, nil
			}
			return Replace{branch}, nil
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
			children[ix] = allocStorage.toNodeHandle()

			value := NewValue(value, t.layout.MaxInlineValue())

			if len(partial)-common == 0 {
				// The value should be part of the branch
				return Replace{
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
				children[ix] = leaf.toNodeHandle()
				return Replace{
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
				n.children[idx] = newChild.toNodeHandle()
				if !changed {
					// Our branch is untouched so we can restore it
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
