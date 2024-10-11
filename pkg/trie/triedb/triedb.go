// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

var (
	ErrIncompleteDB     = errors.New("incomplete database")
	ErrInvalidStateRoot = errors.New("invalid state root")
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "triedb"))
)

type TrieDBOpts[H hash.Hash, Hasher hash.Hasher[H]] func(*TrieDB[H, Hasher])

func WithCache[H hash.Hash, Hasher hash.Hasher[H]](c TrieCache[H]) TrieDBOpts[H, Hasher] {
	return func(t *TrieDB[H, Hasher]) {
		t.cache = c
	}
}
func WithRecorder[H hash.Hash, Hasher hash.Hasher[H]](r TrieRecorder) TrieDBOpts[H, Hasher] {
	return func(t *TrieDB[H, Hasher]) {
		t.recorder = r
	}
}

// TrieDB is a DB-backed patricia merkle trie implementation
// using lazy loading to fetch nodes
type TrieDB[H hash.Hash, Hasher hash.Hasher[H]] struct {
	rootHash H
	db       db.RWDatabase
	version  trie.TrieLayout
	// rootHandle is an in-memory-trie-like representation of the node
	// references and new inserted nodes in the trie
	rootHandle NodeHandle
	// Storage is an in memory storage for nodes that we need to use during this
	// trieDB session (before nodes are committed to db)
	storage nodeStorage[H]
	// deathRow is a set of nodes that we want to delete from db
	// uses string since it's comparable []byte
	deathRow map[string]interface{}
	// Optional cache to speed up the db lookups
	cache TrieCache[H]
	// Optional recorder for recording trie accesses
	recorder TrieRecorder
}

func NewEmptyTrieDB[H hash.Hash, Hasher hash.Hasher[H]](
	db db.RWDatabase, opts ...TrieDBOpts[H, Hasher]) *TrieDB[H, Hasher] {
	hasher := *new(Hasher)
	root := hasher.Hash([]byte{0})
	return NewTrieDB[H, Hasher](root, db, opts...)
}

// NewTrieDB creates a new TrieDB using the given root and db
func NewTrieDB[H hash.Hash, Hasher hash.Hasher[H]](
	rootHash H, db db.RWDatabase, opts ...TrieDBOpts[H, Hasher]) *TrieDB[H, Hasher] {
	rootHandle := persisted[H]{rootHash}

	trieDB := &TrieDB[H, Hasher]{
		rootHash:   rootHash,
		version:    trie.V0,
		db:         db,
		storage:    newNodeStorage[H](),
		rootHandle: rootHandle,
		deathRow:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(trieDB)
	}

	return trieDB
}

func (t *TrieDB[H, Hasher]) SetVersion(v trie.TrieLayout) {
	if v < t.version {
		panic("cannot regress trie version")
	}

	t.version = v
}

// Hash returns the hashed root of the trie.
func (t *TrieDB[H, Hasher]) Hash() (H, error) {
	err := t.commit()
	if err != nil {
		root := (*new(Hasher)).Hash([]byte{0})
		return root, err
	}
	// This is trivial since it is a read only trie, but will change when we
	// support writes
	return t.rootHash, nil
}

// MustHash returns the hashed root of the trie.
// It panics if it fails to hash the root node.
func (t *TrieDB[H, Hasher]) MustHash() H {
	h, err := t.Hash()
	if err != nil {
		panic(err)
	}

	return h
}

// Get returns the value in the node of the trie
// which matches its key with the key given.
// Note the key argument is given in little Endian format.
func (t *TrieDB[H, Hasher]) Get(key []byte) []byte {
	val, err := t.lookup(key, t.rootHandle)
	if err != nil {
		return nil
	}

	return val
}

func (t *TrieDB[H, Hasher]) lookup(fullKey []byte, handle NodeHandle) ([]byte, error) {
	prefix := fullKey
	partialKey := nibbles.NewNibbles(fullKey)
	for {
		var partialIdx uint
		switch node := handle.(type) {
		case persisted[H]:
			lookup := NewTrieLookup[H, Hasher, []byte](
				t.db,
				node.hash,
				nil, // no cache intentionally
				t.recorder,
				func(data []byte) []byte {
					return data
				},
			)
			qi, err := lookup.Lookup(fullKey)
			if err != nil {
				return nil, err
			}
			if qi == nil {
				return nil, nil
			}
			return *qi, nil
		case inMemory:
			switch n := t.storage.get(storageHandle(node)).(type) {
			case Empty:
				return nil, nil
			case Leaf[H]:
				if nibbles.NewNibblesFromNodeKey(n.partialKey).Equal(partialKey) {
					return inMemoryFetchedValue[H](n.value, prefix, t.db)
				} else {
					return nil, nil
				}
			case Branch[H]:
				slice := nibbles.NewNibblesFromNodeKey(n.partialKey)
				if slice.Equal(partialKey) {
					return inMemoryFetchedValue[H](n.value, prefix, t.db)
				} else if partialKey.StartsWith(slice) {
					idx := partialKey.At(slice.Len())
					child := n.children[idx]
					if child != nil {
						partialIdx = slice.Len() + 1
						handle = child
					} else {
						return nil, nil
					}
				} else {
					return nil, nil
				}
			}
		}
		partialKey = partialKey.Mid(partialIdx)
	}
}

func (t *TrieDB[H, Hasher]) getNodeOrLookup(
	nodeHandle codec.MerkleValue, partialKey nibbles.Prefix, recordAccess bool,
) (codec.EncodedNode, *H, error) {
	var nodeHash *H
	var nodeData []byte
	switch nodeHandle := nodeHandle.(type) {
	case codec.HashedNode[H]:
		prefixedKey := append(partialKey.JoinedBytes(), nodeHandle.Hash.Bytes()...)
		var err error
		nodeData, err = t.db.Get(prefixedKey)
		if err != nil {
			return nil, nil, err
		}
		if len(nodeData) == 0 {
			if partialKey.Key == nil && partialKey.Padded == nil {
				return nil, nil, fmt.Errorf("%w: %v", ErrInvalidStateRoot, nodeHandle.Hash)
			}
			return nil, nil, fmt.Errorf("%w: %v", ErrIncompleteDB, nodeHandle.Hash)
		}
		nodeHash = &nodeHandle.Hash
	case codec.InlineNode:
		nodeHash = nil
		nodeData = nodeHandle
	}

	reader := bytes.NewReader(nodeData)
	decoded, err := codec.Decode[H](reader)
	if err != nil {
		return nil, nil, err
	}

	if recordAccess {
		t.recordAccess(EncodedNodeAccess[H]{Hash: t.rootHash, EncodedNode: nodeData})
	}
	return decoded, nodeHash, nil
}

func (t *TrieDB[H, Hasher]) fetchValue(hash H, prefix nibbles.Prefix) ([]byte, error) {
	prefixedKey := append(prefix.JoinedBytes(), hash.Bytes()...)
	value, err := t.db.Get(prefixedKey)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%w: %v", ErrIncompleteDB, hash)
	}
	t.recordAccess(ValueAccess[H]{Hash: t.rootHash, Value: value, FullKey: prefix.Key})
	return value, nil
}

// Remove removes the given key from the trie
func (t *TrieDB[H, Hasher]) remove(keyNibbles nibbles.Nibbles) error {
	var oldValue nodeValue
	rootHandle := t.rootHandle

	removeResult, err := t.removeAt(rootHandle, &keyNibbles, &oldValue)
	if err != nil {
		return err
	}
	if removeResult != nil {
		t.rootHandle = inMemory(removeResult.handle)
	} else {
		hashedNullNode := (*new(Hasher)).Hash([]byte{0})
		t.rootHandle = persisted[H]{hashedNullNode}
		t.rootHash = hashedNullNode
	}

	return nil
}

// Delete deletes the given key from the trie
func (t *TrieDB[H, Hasher]) Delete(key []byte) error {
	return t.remove(nibbles.NewNibbles(key))
}

// insert inserts the node and update the rootHandle
func (t *TrieDB[H, Hasher]) insert(keyNibbles nibbles.Nibbles, value []byte) error {
	var oldValue nodeValue
	rootHandle := t.rootHandle
	newHandle, _, err := t.insertAt(rootHandle, &keyNibbles, value, &oldValue)
	if err != nil {
		return err
	}
	t.rootHandle = inMemory(newHandle)

	return nil
}

// Put inserts the given key / value pair into the trie
func (t *TrieDB[H, Hasher]) Put(key, value []byte) error {
	return t.insert(nibbles.NewNibbles(key), value)
}

// insertAt inserts the given key / value pair into the node referenced by the
// node handle `handle`
func (t *TrieDB[H, Hasher]) insertAt(
	handle NodeHandle,
	keyNibbles *nibbles.Nibbles,
	value []byte,
	oldValue *nodeValue,
) (strgHandle storageHandle, changed bool, err error) {
	switch h := handle.(type) {
	case inMemory:
		strgHandle = storageHandle(h)
	case persisted[H]:
		strgHandle, err = t.lookupNode(h.hash, keyNibbles.Left())
		if err != nil {
			return -1, false, err
		}
	}

	stored := t.storage.destroy(strgHandle)
	result, err := t.inspect(stored, keyNibbles, func(node Node, keyNibbles *nibbles.Nibbles) (action, error) {
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
	handle  storageHandle
	changed bool
}

func (t *TrieDB[H, Hasher]) removeAt(
	handle NodeHandle,
	keyNibbles *nibbles.Nibbles,
	oldValue *nodeValue,
) (*RemoveAtResult, error) {
	var stored StoredNode
	switch h := handle.(type) {
	case inMemory:
		stored = t.storage.destroy(storageHandle(h))
	case persisted[H]:
		handle, err := t.lookupNode(h.hash, keyNibbles.Left())
		if err != nil {
			return nil, err
		}
		stored = t.storage.destroy(handle)
	}

	result, err := t.inspect(stored, keyNibbles, func(node Node, keyNibbles *nibbles.Nibbles) (action, error) {
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

type inspectResult struct {
	stored  StoredNode
	changed bool
}

// inspect inspects the given node `stored` and calls the `inspector` function
// then returns the new node and a boolean indicating if the node has changed
func (t *TrieDB[H, Hasher]) inspect(
	stored StoredNode,
	key *nibbles.Nibbles,
	inspector func(Node, *nibbles.Nibbles) (action, error),
) (*inspectResult, error) {
	// shallow copy since key will change offset through inspector
	currentKey := *key
	switch n := stored.(type) {
	case NewStoredNode:
		res, err := inspector(n.node, key)
		if err != nil {
			return nil, err
		}
		switch a := res.(type) {
		case restoreNode:
			return &inspectResult{NewStoredNode(a), false}, nil
		case replaceNode:
			return &inspectResult{NewStoredNode(a), true}, nil
		case deleteNode:
			return nil, nil
		default:
			panic("unreachable")
		}
	case CachedStoredNode[H]:
		res, err := inspector(n.node, key)
		if err != nil {
			return nil, err
		}
		switch a := res.(type) {
		case restoreNode:
			return &inspectResult{CachedStoredNode[H]{a.node, n.hash}, false}, nil
		case replaceNode:
			prefixedKey := append(currentKey.Left().JoinedBytes(), n.hash.Bytes()...)
			t.deathRow[string(prefixedKey)] = nil
			return &inspectResult{NewStoredNode(a), true}, nil
		case deleteNode:
			prefixedKey := append(currentKey.Left().JoinedBytes(), n.hash.Bytes()...)
			t.deathRow[string(prefixedKey)] = nil
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
func (t *TrieDB[H, Hasher]) fix(branch Branch[H], key nibbles.Nibbles) (Node, error) {
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
		return Leaf[H]{branch.partialKey, branch.value}, nil
	} else if len(usedIndex) == 1 && branch.value == nil {
		// Only one onward node. use child instead
		idx := usedIndex[0]
		// take child and replace it to nil
		child := branch.children[idx]
		branch.children[idx] = nil

		key2 := key.Clone()
		key2.Advance(uint(len(branch.partialKey.Data))*
			nibbles.NibblesPerByte - branch.partialKey.Offset)

		var (
			start      []byte
			allocStart []byte
			prefixEnd  *byte
		)
		prefix := key2.Left()
		switch prefix.Padded {
		case nil:
			start = prefix.Key
			allocStart = nil
			pushed := nibbles.PushAtLeft(0, idx, 0)
			prefixEnd = &pushed
		default:
			so := prefix.Key
			so = append(so, nibbles.PadLeft(*prefix.Padded)|idx)
			start = prefix.Key
			allocStart = so
			prefixEnd = nil
		}
		var childPrefix nibbles.Prefix
		if allocStart != nil {
			childPrefix = nibbles.Prefix{
				Key:    allocStart,
				Padded: prefixEnd,
			}
		} else {
			childPrefix = nibbles.Prefix{
				Key:    start,
				Padded: prefixEnd,
			}
		}

		var stored StoredNode
		switch n := child.(type) {
		case inMemory:
			stored = t.storage.destroy(storageHandle(n))
		case persisted[H]:
			handle, err := t.lookupNode(n.hash, childPrefix)
			if err != nil {
				return nil, fmt.Errorf("looking up node: %w", err)
			}
			stored = t.storage.destroy(handle)
		}

		var childNode Node
		switch n := stored.(type) {
		case NewStoredNode:
			childNode = n.node
		case CachedStoredNode[H]:
			prefixedKey := append(childPrefix.JoinedBytes(), n.hash.Bytes()...)
			t.deathRow[string(prefixedKey)] = nil
			childNode = n.node
		}

		switch n := childNode.(type) {
		case Leaf[H]:
			combinedKey := combineKey(branch.partialKey, nodeKey{Offset: nibbles.NibblesPerByte - 1, Data: []byte{idx}})
			combinedKey = combineKey(combinedKey, n.partialKey)
			return Leaf[H]{combinedKey, n.value}, nil
		case Branch[H]:
			combinedKey := combineKey(branch.partialKey, nodeKey{Offset: nibbles.NibblesPerByte - 1, Data: []byte{idx}})
			combinedKey = combineKey(combinedKey, n.partialKey)
			return Branch[H]{combinedKey, n.children, n.value}, nil
		default:
			panic("unreachable")
		}
	} else {
		// Restore branch
		return branch, nil
	}
}

func combineKey(start nodeKey, end nodeKey) nodeKey {
	if !(start.Offset < nibbles.NibblesPerByte) {
		panic("invalid start offset")
	}
	if !(end.Offset < nibbles.NibblesPerByte) {
		panic("invalid end offset")
	}
	finalOffset := (start.Offset + end.Offset) % nibbles.NibblesPerByte
	_ = start.ShiftKey(finalOffset)
	var st uint
	if end.Offset > 0 {
		sl := len(start.Data)
		start.Data[sl-1] |= nibbles.PadRight(end.Data[0])
		st = 1
	} else {
		st = 0
	}
	for i := st; i < uint(len(end.Data)); i++ {
		start.Data = append(start.Data, end.Data[i])
	}
	return start
}

// removeInspector removes the key node from the given node `stored`
func (t *TrieDB[H, Hasher]) removeInspector(
	stored Node, keyNibbles *nibbles.Nibbles, oldValue *nodeValue,
) (action, error) {
	partial := keyNibbles.Clone()

	switch n := stored.(type) {
	case Empty:
		return deleteNode{}, nil
	case Leaf[H]:
		existingKey := nibbles.NewNibblesFromNodeKey(n.partialKey)
		if existingKey.Equal(partial) {
			// This is the node we are looking for so we delete it
			keyVal := keyNibbles.Clone()
			keyVal.Advance(existingKey.Len())
			t.replaceOldValue(oldValue, n.value, keyVal.Left())
			return deleteNode{}, nil
		}
		// Wrong partial, so we return the node as is
		return restoreNode{n}, nil
	case Branch[H]:
		if partial.Len() == 0 {
			if n.value == nil {
				// Nothing to delete since the branch doesn't contains a value
				return restoreNode{n}, nil
			}
			// The branch contains the value so we delete it
			t.replaceOldValue(oldValue, n.value, keyNibbles.Left())
			newNode, err := t.fix(Branch[H]{n.partialKey, n.children, nil}, *keyNibbles)
			if err != nil {
				return nil, err
			}
			return replaceNode{newNode}, nil
		}

		existingKey := nibbles.NewNibblesFromNodeKey(n.partialKey)

		common := existingKey.CommonPrefix(partial)
		existingLength := existingKey.Len()

		if common == existingLength && common == partial.Len() {
			// Replace value
			if n.value != nil {
				keyVal := keyNibbles.Clone()
				keyVal.Advance(existingLength)
				t.replaceOldValue(oldValue, n.value, keyVal.Left())
				newNode, err := t.fix(Branch[H]{n.partialKey, n.children, nil}, *keyNibbles)
				return replaceNode{newNode}, err
			}
			return restoreNode{Branch[H]{n.partialKey, n.children, nil}}, nil
		} else if common < existingLength {
			return restoreNode{n}, nil
		}
		// Check children
		idx := partial.At(common)
		// take child and replace it to nil
		child := n.children[idx]
		n.children[idx] = nil

		if child == nil {
			return restoreNode{n}, nil
		}
		prefix := *keyNibbles
		keyNibbles.Advance(common + 1)

		removeAtResult, err := t.removeAt(child, keyNibbles, oldValue)
		if err != nil {
			return nil, err
		}

		if removeAtResult != nil {
			n.children[idx] = inMemory(removeAtResult.handle)
			if removeAtResult.changed {
				return replaceNode{n}, nil
			}
			return restoreNode{n}, nil
		}

		newNode, err := t.fix(n, prefix)
		if err != nil {
			return nil, err
		}
		return replaceNode{newNode}, nil
	default:
		panic("unreachable")
	}
}

// insertInspector inserts the new key / value pair into the given node `stored`
func (t *TrieDB[H, Hasher]) insertInspector(
	stored Node, keyNibbles *nibbles.Nibbles, value []byte, oldValue *nodeValue,
) (action, error) {
	partial := keyNibbles.Clone()

	switch n := stored.(type) {
	case Empty:
		// If the node is empty we have to replace it with a leaf node with the
		// new value
		value := NewValue[H](value, t.version.MaxInlineValue())
		pnk := partial.NodeKey()
		return replaceNode{node: Leaf[H]{partialKey: pnk, value: value}}, nil
	case Leaf[H]:
		existingKey := nibbles.NewNibblesFromNodeKey(n.partialKey)
		common := existingKey.CommonPrefix(partial)

		if common == existingKey.Len() && common == partial.Len() {
			// We are trying to insert a value in the same leaf so we just need
			// to replace the value
			value := NewValue[H](value, t.version.MaxInlineValue())
			unchanged := n.value.equal(value)
			keyVal := keyNibbles.Clone()
			keyVal.Advance(existingKey.Len())
			t.replaceOldValue(oldValue, n.value, keyVal.Left())
			leaf := Leaf[H]{partialKey: n.partialKey, value: value}
			if unchanged {
				// If the value didn't change we can restore this leaf previously
				// taken from storage
				return restoreNode{leaf}, nil
			}
			return replaceNode{leaf}, nil
		} else if common < existingKey.Len() {
			// If the common prefix is less than this leaf's key then we need to
			// create a branch node. Then add this leaf and the new value to the
			// branch
			var children [codec.ChildrenCapacity]NodeHandle

			idx := existingKey.At(common)

			// Modify the existing leaf partial key and add it as a child
			newLeaf := Leaf[H]{existingKey.Mid(common + 1).NodeKey(), n.value}
			children[idx] = inMemory(t.storage.alloc(NewStoredNode{node: newLeaf}))
			branch := Branch[H]{
				partialKey: partial.NodeKeyRange(common),
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
			var branch Node = Branch[H]{
				partialKey: existingKey.NodeKey(),
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
	case Branch[H]:
		existingKey := nibbles.NewNibblesFromNodeKey(n.partialKey)
		common := partial.CommonPrefix(existingKey)

		if common == existingKey.Len() && common == partial.Len() {
			// We are trying to insert a value in the same branch so we just need
			// to replace the value
			value := NewValue[H](value, t.version.MaxInlineValue())
			var unchanged bool
			if n.value != nil {
				unchanged = n.value.equal(value)
			}
			branch := Branch[H]{existingKey.NodeKey(), n.children, value}

			keyVal := keyNibbles.Clone()
			keyVal.Advance(existingKey.Len())
			t.replaceOldValue(oldValue, n.value, keyVal.Left())
			if unchanged {
				// If the value didn't change we can restore this leaf previously
				// taken from storage
				return restoreNode{branch}, nil
			}
			return replaceNode{branch}, nil
		} else if common < existingKey.Len() {
			// If the common prefix is less than this branch's key then we need to
			// create a branch node in between.
			// Then add this branch and the new value to the new branch

			// So we take this branch and we add it as a child of the new one
			branchPartial := existingKey.Mid(common + 1).NodeKey()
			lowerBranch := Branch[H]{branchPartial, n.children, n.value}
			allocStorage := t.storage.alloc(NewStoredNode{node: lowerBranch})

			children := [codec.ChildrenCapacity]NodeHandle{}
			ix := existingKey.At(common)
			children[ix] = inMemory(allocStorage)

			value := NewValue[H](value, t.version.MaxInlineValue())

			if partial.Len()-common == 0 {
				// The value should be part of the branch
				return replaceNode{
					Branch[H]{
						existingKey.NodeKeyRange(common),
						children,
						value,
					},
				}, nil
			} else {
				// Value is in a leaf under the branch so we have to create it
				storedLeaf := Leaf[H]{partial.Mid(common + 1).NodeKey(), value}
				leaf := t.storage.alloc(NewStoredNode{node: storedLeaf})

				ix = partial.At(common)
				children[ix] = inMemory(leaf)
				return replaceNode{
					Branch[H]{
						existingKey.NodeKeyRange(common),
						children,
						nil,
					},
				}, nil
			}
		} else {
			// append after common == existing_key and partial > common
			idx := partial.At(common)
			keyNibbles.Advance(common + 1)
			child := n.children[idx]
			if child != nil {
				n.children[idx] = nil
				// We have to add the new value to the child
				newChild, changed, err := t.insertAt(child, keyNibbles, value, oldValue)
				if err != nil {
					return nil, err
				}
				n.children[idx] = inMemory(newChild)
				if !changed {
					// Our branch is untouched so we can restore it
					branch := Branch[H]{
						existingKey.NodeKey(),
						n.children,
						n.value,
					}

					return restoreNode{branch}, nil
				}
			} else {
				// Original has nothing here so we have to create a new leaf
				value := NewValue[H](value, t.version.MaxInlineValue())
				leaf := t.storage.alloc(NewStoredNode{node: Leaf[H]{keyNibbles.NodeKey(), value}})
				n.children[idx] = inMemory(leaf)
			}
			return replaceNode{Branch[H]{
				existingKey.NodeKey(),
				n.children,
				n.value,
			}}, nil
		}
	default:
		panic("unreachable")
	}
}

func (t *TrieDB[H, Hasher]) replaceOldValue(
	oldValue *nodeValue,
	storedValue nodeValue,
	prefix nibbles.Prefix,
) {
	switch oldv := storedValue.(type) {
	case valueRef[H]:
		hash := oldv.getHash()
		if hash != (*new(H)) {
			prefixedKey := append(prefix.JoinedBytes(), hash.Bytes()...)
			t.deathRow[string(prefixedKey)] = nil
		}
	case newValueRef[H]:
		hash := oldv.getHash()
		if hash != (*new(H)) {
			prefixedKey := append(prefix.JoinedBytes(), hash.Bytes()...)
			t.deathRow[string(prefixedKey)] = nil
		}
	}
	*oldValue = storedValue
}

// lookup node in DB and add it in storage, return storage handle
func (t *TrieDB[H, Hasher]) lookupNode(hash H, key nibbles.Prefix) (storageHandle, error) {
	var newNode = func() (Node, error) {
		prefixedKey := append(key.JoinedBytes(), hash.Bytes()...)
		encodedNode, err := t.db.Get(prefixedKey)
		if err != nil {
			return nil, ErrIncompleteDB
		}

		t.recordAccess(EncodedNodeAccess[H]{Hash: t.rootHash, EncodedNode: encodedNode})

		return newNodeFromEncoded[H](hash, encodedNode, &t.storage)
	}
	// We only check the `cache` for a node with `get_node` and don't insert
	// the node if it wasn't there, because in substrate we only access the node while computing
	// a new trie (aka some branch). We assume that this node isn't that important
	// to have it being cached.
	var node Node
	if t.cache != nil {
		nodeOwned := t.cache.GetNode(hash)
		if nodeOwned == nil {
			var err error
			node, err = newNode()
			if err != nil {
				return -1, err
			}
		} else {
			t.recordAccess(CachedNodeAccess[H]{Hash: hash, Node: nodeOwned})
			node = newNodeFromCachedNode(nodeOwned, &t.storage)
		}
	} else {
		var err error
		node, err = newNode()
		if err != nil {
			return -1, err
		}
	}

	return t.storage.alloc(CachedStoredNode[H]{
		node: node,
		hash: hash,
	}), nil
}

// commit writes all trie changes to the underlying db
func (t *TrieDB[H, Hasher]) commit() error {
	logger.Debug("Committing trie changes to db")
	logger.Debugf("%d nodes to remove from db", len(t.deathRow))

	dbBatch := t.db.NewBatch()
	defer func() {
		if err := dbBatch.Close(); err != nil {
			logger.Criticalf("cannot close triedb commit batcher: %w", err)
		}
	}()

	for hash := range t.deathRow {
		err := dbBatch.Del([]byte(hash))
		if err != nil {
			return err
		}
	}

	// Reset deathRow
	t.deathRow = make(map[string]interface{})

	var handle storageHandle
	switch h := t.rootHandle.(type) {
	case persisted[H]:
		return nil // nothing to commit since the root is already in db
	case inMemory:
		handle = storageHandle(h)
	}

	switch stored := t.storage.destroy(handle).(type) {
	case NewStoredNode:
		// Reconstructs the full key for root node
		var fullKey *nibbles.NibbleSlice
		if pk := stored.getNode().getPartialKey(); pk != nil {
			fk := nibbles.NewNibblesFromNodeKey(*pk)
			ns := nibbles.NewNibbleSliceFromNibbles(fk)
			fullKey = &ns
		}

		var k nibbles.NibbleSlice

		encodedNode, err := newEncodedNode[H](
			stored.node,
			func(node nodeToEncode, partialKey *nibbles.Nibbles, childIndex *byte) (ChildReference, error) {
				mov := k.AppendOptionalSliceAndNibble(partialKey, childIndex)
				switch n := node.(type) {
				case newNodeToEncode:
					hash := (*new(Hasher)).Hash(n.value)
					prefixedKey := append(k.Prefix().JoinedBytes(), hash.Bytes()...)
					err := dbBatch.Put(prefixedKey, n.value)
					if err != nil {
						return nil, err
					}
					t.cacheValue(k.Inner(), n.value, hash)
					k.DropLasts(mov)
					return HashChildReference[H]{hash}, nil
				case trieNodeToEncode:
					result, err := t.commitChild(dbBatch, n.child, &k)
					if err != nil {
						return nil, err
					}

					k.DropLasts(mov)
					return result, nil
				default:
					panic("unreachable")
				}
			},
		)

		if err != nil {
			return err
		}

		hash := (*new(Hasher)).Hash(encodedNode)
		err = dbBatch.Put(hash.Bytes(), encodedNode)
		if err != nil {
			return err
		}

		t.rootHash = hash
		t.cacheNode(hash, encodedNode, fullKey)
		t.rootHandle = persisted[H]{t.rootHash}

		// Flush all db changes
		return dbBatch.Flush()
	case CachedStoredNode[H]:
		t.rootHash = stored.hash
		t.rootHandle = inMemory(
			t.storage.alloc(CachedStoredNode[H]{stored.node, stored.hash}),
		)
		return nil
	default:
		panic("unreachable")
	}
}

// Commit a node by hashing it and writing it to the db.
func (t *TrieDB[H, Hasher]) commitChild(
	dbBatch database.Batch,
	child NodeHandle,
	prefixKey *nibbles.NibbleSlice,
) (ChildReference, error) {
	switch nh := child.(type) {
	case persisted[H]:
		// Already persisted we have to do nothing
		return HashChildReference[H]{nh.hash}, nil
	case inMemory:
		stored := t.storage.destroy(storageHandle(nh))
		switch storedNode := stored.(type) {
		case CachedStoredNode[H]:
			return HashChildReference[H]{storedNode.hash}, nil
		case NewStoredNode:
			// Reconstructs the full key
			var fullKey *nibbles.NibbleSlice
			prefix := prefixKey.Clone()
			if partial := stored.getNode().getPartialKey(); partial != nil {
				fk := nibbles.NewNibblesFromNodeKey(*partial)
				prefix.AppendPartial(fk.RightPartial())
			}
			fullKey = &prefix

			// We have to store the node in the DB
			commitChildFunc := func(node nodeToEncode, partialKey *nibbles.Nibbles, childIndex *byte) (ChildReference, error) {
				mov := prefixKey.AppendOptionalSliceAndNibble(partialKey, childIndex)
				switch n := node.(type) {
				case newNodeToEncode:
					hash := (*new(Hasher)).Hash(n.value)
					prefixedKey := append(prefixKey.Prefix().JoinedBytes(), hash.Bytes()...)
					err := dbBatch.Put(prefixedKey, n.value)
					if err != nil {
						panic("inserting in db")
					}

					t.cacheValue(prefixKey.Inner(), n.value, hash)
					prefixKey.DropLasts(mov)
					return HashChildReference[H]{hash}, nil
				case trieNodeToEncode:
					result, err := t.commitChild(dbBatch, n.child, prefixKey)
					if err != nil {
						return nil, err
					}

					prefixKey.DropLasts(mov)
					return result, nil
				default:
					panic("unreachable")
				}
			}

			encoded, err := newEncodedNode[H](storedNode.node, commitChildFunc)
			if err != nil {
				panic("encoding node")
			}

			// Not inlined node
			if len(encoded) >= (*new(H)).Length() {
				hash := (*new(Hasher)).Hash(encoded)
				prefixedKey := append(prefixKey.Prefix().JoinedBytes(), hash.Bytes()...)
				err := dbBatch.Put(prefixedKey, encoded)
				if err != nil {
					return nil, err
				}

				t.cacheNode(hash, encoded, fullKey)

				return HashChildReference[H]{hash}, nil
			} else {
				return InlineChildReference(encoded), nil
			}
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}

type valueToCache[H any] struct {
	KeyBytes []byte
	CachedValue[H]
}

func cacheChildValues[H hash.Hash](
	node CachedNode[H],
	valuesToCache *[]valueToCache[H],
	fullKey nibbles.NibbleSlice,
) {
	for _, child := range node.children() {
		switch nho := child.NodeHandleOwned.(type) {
		case NodeHandleOwnedInline[H]:
			n := child.nibble
			c := nho.CachedNode
			key := fullKey.Clone()
			if n != nil {
				key.Push(*n)
			}
			if pk := c.partialKey(); pk != nil {
				key.Append(*pk)
			}

			if d := c.data(); d != nil {
				if h := c.dataHash(); h != nil {
					*valuesToCache = append(*valuesToCache, valueToCache[H]{
						KeyBytes: key.Inner(),
						CachedValue: ExistingCachedValue[H]{
							Hash: *h,
							Data: d,
						},
					})
				}
			}

			cacheChildValues(c, valuesToCache, key)
		}
	}
}

// Cache the given encoded node.
func (t *TrieDB[H, Hasher]) cacheNode(hash H, encoded []byte, fullKey *nibbles.NibbleSlice) {
	if t.cache == nil {
		return
	}
	node, err := t.cache.GetOrInsertNode(hash, func() (CachedNode[H], error) {
		buf := bytes.NewBuffer(encoded)
		decoded, err := codec.Decode[H](buf)
		if err != nil {
			return nil, err
		}
		return newCachedNodeFromNode[H, Hasher](decoded)
	})
	if err != nil {
		panic("Just encoded the node, so it should decode without any errors; qed")
	}

	valuesToCache := []valueToCache[H]{}
	// If the given node has data attached, the fullKey is the full key to this node.
	if fullKey != nil {
		if v := node.data(); v != nil {
			if h := node.dataHash(); h != nil {
				valuesToCache = append(valuesToCache, valueToCache[H]{
					KeyBytes: fullKey.Inner(),
					CachedValue: NewCachedValue[H](
						ExistingCachedValue[H]{
							Hash: *h,
							Data: v,
						},
					),
				})
			}
		}

		// Also cache values of inline nodes.
		cacheChildValues(node, &valuesToCache, *fullKey)
	}

	for _, valueToCache := range valuesToCache {
		k := valueToCache.KeyBytes
		v := valueToCache.CachedValue
		t.cache.SetValue(k, v)
	}
}

// Cache the given value.
//
// The supplied hash should be the hash of value.
func (t *TrieDB[H, Hasher]) cacheValue(fullKey []byte, value []byte, hash H) {
	if t.cache == nil {
		return
	}
	var val []byte
	node, err := t.cache.GetOrInsertNode(hash, func() (CachedNode[H], error) {
		return ValueCachedNode[H]{
			Value: value,
			Hash:  hash,
		}, nil
	})
	if err != nil {
		panic("this should never happen")
	}
	if node != nil {
		val = node.data()
	}

	if val != nil {
		t.cache.SetValue(fullKey, ExistingCachedValue[H]{
			Hash: hash,
			Data: val,
		})
	}
}

func (t *TrieDB[H, Hasher]) recordAccess(access TrieAccess) {
	if t.recorder != nil {
		t.recorder.Record(access)
	}
}

// Returns the hash of the value for key.
func (t *TrieDB[H, Hasher]) GetHash(key []byte) (*H, error) {
	// TODO: look into moving query into Lookup method
	lookup := NewTrieLookup[H, Hasher](
		t.db, t.rootHash, t.cache, t.recorder,
		func([]byte) any { return nil },
	)
	return lookup.LookupHash(key)
}

// Search for the key with the given query parameter.
func GetWith[H hash.Hash, Hasher hash.Hasher[H], QueryItem any](
	t *TrieDB[H, Hasher], key []byte, query Query[QueryItem],
) (*QueryItem, error) {
	lookup := NewTrieLookup[H, Hasher](
		t.db, t.rootHash, t.cache, t.recorder, query,
	)
	return lookup.Lookup(key)
}
