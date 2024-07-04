// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/pkg/trie/cache"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

var ErrIncompleteDB = errors.New("incomplete database")

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "triedb"))
)

type entry struct {
	key   []byte
	value []byte
}

// TrieDB is a DB-backed patricia merkle trie implementation
// using lazy loading to fetch nodes
type TrieDB struct {
	rootHash common.Hash
	db       db.RWDatabase
	version  trie.TrieLayout
	// rootHandle is an in-memory-trie-like representation of the node
	// references and new inserted nodes in the trie
	rootHandle NodeHandle
	// Storage is an in memory storage for nodes that we need to use during this
	// trieDB session (before nodes are committed to db)
	storage NodeStorage
	// deathRow is a set of nodes that we want to delete from db
	deathRow map[common.Hash]interface{}
	// Optional cache to speed up the db lookups
	cache cache.TrieCache
	// Optional recorder for recording trie accesses
	recorder *Recorder
}

func NewEmptyTrieDB(db db.RWDatabase, cache cache.TrieCache, recorder *Recorder) *TrieDB {
	root := hashedNullNode
	return NewTrieDB(root, db, cache, recorder)
}

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB(rootHash common.Hash, db db.RWDatabase, cache cache.TrieCache, recorder *Recorder) *TrieDB {
	rootHandle := Persisted{hash: rootHash}

	return &TrieDB{
		rootHash:   rootHash,
		cache:      cache,
		recorder:   recorder,
		version:    trie.V0,
		db:         db,
		storage:    NewNodeStorage(),
		rootHandle: rootHandle,
		deathRow:   make(map[common.Hash]interface{}),
	}
}

func (t *TrieDB) SetVersion(v trie.TrieLayout) {
	if v < t.version {
		panic("cannot regress trie version")
	}

	t.version = v
}

// Hash returns the hashed root of the trie.
func (t *TrieDB) Hash() (common.Hash, error) {
	err := t.commit()
	if err != nil {
		return common.EmptyHash, err
	}
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
		case Persisted:
			lookup := NewTrieLookup(t.db, node.hash, t.cache, t.recorder)
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
					return inMemoryFetchedValue(n.value, prefix, t.db)
				} else {
					return nil, nil
				}
			case Branch:
				if bytes.Equal(n.partialKey, partialKey) {
					return inMemoryFetchedValue(n.value, prefix, t.db)
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

	t.recordAccess(EncodedNodeAccess{hash: t.rootHash, encodedNode: encodedNode})

	reader := bytes.NewReader(encodedNode)
	return codec.Decode(reader)
}

// Internal methods

func (t *TrieDB) getNodeAt(key []byte) (codec.EncodedNode, error) {
	lookup := NewTrieLookup(t.db, t.rootHash, t.cache, t.recorder)
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
		t.recordAccess(EncodedNodeAccess{hash: t.rootHash, encodedNode: encodedNode})

		reader := bytes.NewReader(encodedNode)
		return codec.Decode(reader)
	default: // should never happen
		panic("unreachable")
	}
}

// Remove removes the given key from the trie
func (t *TrieDB) remove(keyNibbles []byte) error {
	var oldValue nodeValue
	rootHandle := t.rootHandle

	removeResult, err := t.removeAt(rootHandle, keyNibbles, &oldValue)
	if err != nil {
		return err
	}
	if removeResult != nil {
		t.rootHandle = InMemory{idx: removeResult.handle}
	} else {
		t.rootHandle = Persisted{hashedNullNode}
		t.rootHash = hashedNullNode
	}

	return nil
}

// Delete deletes the given key from the trie
func (t *TrieDB) Delete(key []byte) error {

	keyNibbles := nibbles.KeyLEToNibbles(key)
	return t.remove(keyNibbles)
}

// insert inserts the node and update the rootHandle
func (t *TrieDB) insert(keyNibbles, value []byte) error {
	var oldValue nodeValue
	rootHandle := t.rootHandle
	newHandle, _, err := t.insertAt(rootHandle, keyNibbles, value, &oldValue)
	if err != nil {
		return err
	}
	t.rootHandle = InMemory{idx: newHandle}

	return nil
}

// Put inserts the given key / value pair into the trie
func (t *TrieDB) Put(key, value []byte) error {
	keyNibbles := nibbles.KeyLEToNibbles(key)
	return t.insert(keyNibbles, value)
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
	case Persisted:
		storageHandle, err = t.lookupNode(h.hash)
		if err != nil {
			return -1, false, err
		}
	}

	stored := t.storage.destroy(storageHandle)
	result, err := t.inspect(stored, keyNibbles, func(node Node, keyNibbles []byte) (action, error) {
		return t.insertInspector(node, keyNibbles, value, oldValue)
	})
	if err != nil {
		return -1, false, err
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
	oldValue *nodeValue,
) (*RemoveAtResult, error) {
	var stored StoredNode
	switch h := handle.(type) {
	case InMemory:
		stored = t.storage.destroy(h.idx)
	case Persisted:
		handle, err := t.lookupNode(h.hash)
		if err != nil {
			return nil, err
		}
		stored = t.storage.destroy(handle)
	}

	result, err := t.inspect(stored, keyNibbles, func(node Node, keyNibbles []byte) (action, error) {
		return t.removeInspector(node, keyNibbles, oldValue)
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return &RemoveAtResult{
		handle:  t.storage.alloc(result.stored),
		changed: result.changed,
	}, err
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
	inspector func(Node, []byte) (action, error),
) (*InspectResult, error) {
	switch n := stored.(type) {
	case NewStoredNode:
		res, err := inspector(n.node, key)
		if err != nil {
			return nil, err
		}
		switch a := res.(type) {
		case restoreNode:
			return &InspectResult{BuildNewStoredNode(a.node), false}, nil
		case replaceNode:
			return &InspectResult{BuildNewStoredNode(a.node), true}, nil
		case deleteNode:
			return nil, nil
		default:
			panic("unreachable")
		}
	case CachedStoredNode:
		res, err := inspector(n.node, key)
		if err != nil {
			return nil, err
		}
		switch a := res.(type) {
		case restoreNode:
			return &InspectResult{CachedStoredNode{a.node, n.hash}, false}, nil
		case replaceNode:
			t.deathRow[n.hash] = nil
			return &InspectResult{BuildNewStoredNode(a.node), true}, nil
		case deleteNode:
			t.deathRow[n.hash] = nil
			return nil, nil
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

// fix is a helper function to reorganise the nodes after deleting a branch.
// For example, if the node we are deleting is the only child for a branch node, we can transform that branch in a leaf
func (t *TrieDB) fix(branch Branch) (Node, error) {
	usedIndex := make([]byte, 0)

	for i := 0; i < codec.ChildrenCapacity; i++ {
		if branch.children[i] != nil {
			if len(usedIndex) == 2 {
				break
			}
			usedIndex = append(usedIndex, byte(i))
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
		case Persisted:
			handle, err := t.lookupNode(n.hash)
			if err != nil {
				return nil, fmt.Errorf("looking up node: %w", err)
			}
			stored = t.storage.destroy(handle)
		}

		var childNode Node
		switch n := stored.(type) {
		case NewStoredNode:
			childNode = n.node
		case CachedStoredNode:
			t.deathRow[n.hash] = nil
			childNode = n.node
		}

		combinedKey := bytes.Join([][]byte{branch.partialKey, {idx}, childNode.getPartialKey()}, nil)

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
		return branch, nil
	}
}

// removeInspector removes the key node from the given node `stored`
func (t *TrieDB) removeInspector(stored Node, keyNibbles []byte, oldValue *nodeValue) (action, error) {
	partial := keyNibbles

	switch n := stored.(type) {
	case Empty:
		return deleteNode{}, nil
	case Leaf:
		if bytes.Equal(n.partialKey, partial) {
			// This is the node we are looking for so we delete it
			t.replaceOldValue(oldValue, n.value)
			return deleteNode{}, nil
		}
		// Wrong partial, so we return the node as is
		return restoreNode{n}, nil
	case Branch:
		if len(partial) == 0 {
			if n.value == nil {
				// Nothing to delete since the branch doesn't contains a value
				return restoreNode{n}, nil
			}
			// The branch contains the value so we delete it
			t.replaceOldValue(oldValue, n.value)
			newNode, err := t.fix(Branch{n.partialKey, n.children, nil})
			if err != nil {
				return nil, err
			}
			return replaceNode{newNode}, nil
		}

		common := nibbles.CommonPrefix(n.partialKey, partial)
		existingLength := len(n.partialKey)

		if common == existingLength && common == len(partial) {
			// Replace value
			if n.value != nil {
				t.replaceOldValue(oldValue, n.value)
				newNode, err := t.fix(Branch{n.partialKey, n.children, nil})
				return replaceNode{newNode}, err
			}
			return restoreNode{Branch{n.partialKey, n.children, nil}}, nil
		} else if common < existingLength {
			return restoreNode{n}, nil
		}
		// Check children
		idx := partial[common]
		// take child and replace it to nil
		child := n.children[idx]
		n.children[idx] = nil

		if child == nil {
			return restoreNode{n}, nil
		}

		removeAtResult, err := t.removeAt(child, partial[len(n.partialKey)+1:], oldValue)
		if err != nil {
			return nil, err
		}

		if removeAtResult != nil {
			n.children[idx] = newInMemoryNodeHandle(removeAtResult.handle)
			if removeAtResult.changed {
				return replaceNode{n}, nil
			}
			return restoreNode{n}, nil
		}

		newNode, err := t.fix(n)
		if err != nil {
			return nil, err
		}
		return replaceNode{newNode}, nil
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
		value := NewValue(value, t.version.MaxInlineValue())
		return replaceNode{node: Leaf{partialKey: partial, value: value}}, nil
	case Leaf:
		existingKey := n.partialKey
		common := nibbles.CommonPrefix(partial, existingKey)

		if common == len(existingKey) && common == len(partial) {
			// We are trying to insert a value in the same leaf so we just need
			// to replace the value
			value := NewValue(value, t.version.MaxInlineValue())
			unchanged := n.value.equal(value)
			t.replaceOldValue(oldValue, n.value)
			leaf := Leaf{partialKey: n.partialKey, value: value}
			if unchanged {
				// If the value didn't change we can restore this leaf previously
				// taken from storage
				return restoreNode{leaf}, nil
			}
			return replaceNode{leaf}, nil
		} else if common < len(existingKey) {
			// If the common prefix is less than this leaf's key then we need to
			// create a branch node. Then add this leaf and the new value to the
			// branch
			var children [codec.ChildrenCapacity]NodeHandle

			idx := existingKey[common]

			// Modify the existing leaf partial key and add it as a child
			newLeaf := Leaf{existingKey[common+1:], n.value}
			children[idx] = newInMemoryNodeHandle(t.storage.alloc(NewStoredNode{node: newLeaf}))
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
			return replaceNode{branchAction.getNode()}, nil
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
			return replaceNode{branch}, nil
		}
	case Branch:
		existingKey := n.partialKey
		common := nibbles.CommonPrefix(partial, existingKey)

		if common == len(existingKey) && common == len(partial) {
			// We are trying to insert a value in the same branch so we just need
			// to replace the value
			value := NewValue(value, t.version.MaxInlineValue())
			var unchanged bool
			if n.value != nil {
				unchanged = n.value.equal(value)
			}
			branch := Branch{existingKey, n.children, value}

			t.replaceOldValue(oldValue, n.value)
			if unchanged {
				// If the value didn't change we can restore this leaf previously
				// taken from storage
				return restoreNode{branch}, nil
			}
			return replaceNode{branch}, nil
		} else if common < len(existingKey) {
			// If the common prefix is less than this branch's key then we need to
			// create a branch node in between.
			// Then add this branch and the new value to the new branch

			// So we take this branch and we add it as a child of the new one
			branchPartial := existingKey[common+1:]
			lowerBranch := Branch{branchPartial, n.children, n.value}
			allocStorage := t.storage.alloc(NewStoredNode{node: lowerBranch})

			children := [codec.ChildrenCapacity]NodeHandle{}
			ix := existingKey[common]
			children[ix] = newInMemoryNodeHandle(allocStorage)

			value := NewValue(value, t.version.MaxInlineValue())

			if len(partial)-common == 0 {
				// The value should be part of the branch
				return replaceNode{
					Branch{
						existingKey[:common],
						children,
						value,
					},
				}, nil
			} else {
				// Value is in a leaf under the branch so we have to create it
				storedLeaf := Leaf{partial[common+1:], value}
				leaf := t.storage.alloc(NewStoredNode{node: storedLeaf})

				ix = partial[common]
				children[ix] = newInMemoryNodeHandle(leaf)
				return replaceNode{
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

					return restoreNode{branch}, nil
				}
			} else {
				// Original has nothing here so we have to create a new leaf
				value := NewValue(value, t.version.MaxInlineValue())
				leaf := t.storage.alloc(NewStoredNode{node: Leaf{keyNibbles, value}})
				n.children[idx] = newInMemoryNodeHandle(leaf)
			}
			return replaceNode{Branch{
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
		return -1, ErrIncompleteDB
	}

	t.recordAccess(EncodedNodeAccess{hash: t.rootHash, encodedNode: encodedNode})

	node, err := newNodeFromEncoded(hash, encodedNode, t.storage)
	if err != nil {
		return -1, err
	}

	return t.storage.alloc(CachedStoredNode{
		node: node,
		hash: hash,
	}), nil
}

func (t *TrieDB) commit() error {
	logger.Debug("Committing trie changes to db")
	logger.Debugf("%d nodes to remove from db", len(t.deathRow))

	dbBatch := t.db.NewBatch()
	defer func() {
		if err := dbBatch.Close(); err != nil {
			logger.Criticalf("cannot close triedb commit batcher: %w", err)
		}
	}()

	for hash := range t.deathRow {
		err := dbBatch.Del(hash[:])
		if err != nil {
			return err
		}
	}

	// Reset deathRow
	t.deathRow = make(map[common.Hash]interface{})

	var handle StorageHandle
	switch h := t.rootHandle.(type) {
	case Persisted:
		return nil // nothing to commit since the root is already in db
	case InMemory:
		handle = h.idx
	}

	switch stored := t.storage.destroy(handle).(type) {
	case NewStoredNode:
		// Reconstructs the full key for root node
		var k []byte

		encodedNode, err := NewEncodedNode(
			stored.node,
			func(node NodeToEncode, partialKey []byte, childIndex *byte) (ChildReference, error) {
				k = append(k, partialKey...)
				mov := len(partialKey)
				if childIndex != nil {
					k = append(k, *childIndex)
					mov += 1
				}

				switch n := node.(type) {
				case NewNodeToEncode:
					hash := common.MustBlake2bHash(n.value)
					err := dbBatch.Put(hash[:], n.value)
					if err != nil {
						return nil, err
					}

					k = k[:mov]
					return HashChildReference{Hash: hash}, nil
				case TrieNodeToEncode:
					result, err := t.commitChild(dbBatch, n.child, k)
					if err != nil {
						return nil, err
					}

					k = k[:mov]
					return result, nil
				default:
					panic("unreachable")
				}
			},
		)

		if err != nil {
			return err
		}

		hash := common.MustBlake2bHash(encodedNode)
		err = dbBatch.Put(hash[:], encodedNode)
		if err != nil {
			return err
		}

		t.rootHash = hash
		t.rootHandle = Persisted{t.rootHash}

		// Flush all db changes
		return dbBatch.Flush()
	case CachedStoredNode:
		t.rootHash = stored.hash
		t.rootHandle = InMemory{
			t.storage.alloc(CachedStoredNode{stored.node, stored.hash}),
		}
		return nil
	default:
		panic("unreachable")
	}
}

// Commit a node by hashing it and writing it to the db.
func (t *TrieDB) commitChild(
	dbBatch database.Batch,
	child NodeHandle,
	prefixKey []byte,
) (ChildReference, error) {
	switch nh := child.(type) {
	case Persisted:
		// Already persisted we have to do nothing
		return HashChildReference{Hash: nh.hash}, nil
	case InMemory:
		stored := t.storage.destroy(nh.idx)
		switch storedNode := stored.(type) {
		case CachedStoredNode:
			return HashChildReference{Hash: storedNode.hash}, nil
		case NewStoredNode:
			// We have to store the node in the DB
			commitChildFunc := func(node NodeToEncode, partialKey []byte, childIndex *byte) (ChildReference, error) {
				prefixKey = append(prefixKey, partialKey...)
				mov := len(partialKey)
				if childIndex != nil {
					prefixKey = append(prefixKey, *childIndex)
					mov += 1
				}

				switch n := node.(type) {
				case NewNodeToEncode:
					valueHash := common.MustBlake2bHash(n.value)
					prefixedKey := bytes.Join([][]byte{n.partialKey, valueHash.ToBytes()}, nil)
					err := dbBatch.Put(prefixedKey, n.value)
					if err != nil {
						panic("inserting in db")
					}

					if t.cache != nil {
						t.cache.SetValue(n.partialKey, n.value)
					}

					prefixKey = prefixKey[:mov]
					return HashChildReference{Hash: valueHash}, nil
				case TrieNodeToEncode:
					result, err := t.commitChild(dbBatch, n.child, prefixKey)
					if err != nil {
						return nil, err
					}

					prefixKey = prefixKey[:mov]
					return result, nil
				default:
					panic("unreachable")
				}
			}

			encoded, err := NewEncodedNode(storedNode.node, commitChildFunc)
			if err != nil {
				panic("encoding node")
			}

			// Not inlined node
			if len(encoded) >= common.HashLength {
				hash := common.MustBlake2bHash(encoded)
				err := dbBatch.Put(hash[:], encoded)
				if err != nil {
					return nil, err
				}

				return HashChildReference{Hash: hash}, nil
			} else {
				return InlineChildReference{encoded}, nil
			}
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

func (t *TrieDB) recordAccess(access TrieAccess) {
	if t.recorder != nil {
		t.recorder.record(access)
	}
}

var _ trie.TrieRead = (*TrieDB)(nil)
