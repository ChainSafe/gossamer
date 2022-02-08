// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/lib/common"
)

// EmptyHash is the empty trie hash.
var EmptyHash, _ = NewEmptyTrie().Hash()

// Trie is a base 16 modified Merkle Patricia trie.
type Trie struct {
	generation  uint64
	root        Node
	childTries  map[common.Hash]*Trie
	deletedKeys map[common.Hash]struct{}
}

// NewEmptyTrie creates a trie with a nil root
func NewEmptyTrie() *Trie {
	return NewTrie(nil)
}

// NewTrie creates a trie with an existing root node
func NewTrie(root Node) *Trie {
	return &Trie{
		root:        root,
		childTries:  make(map[common.Hash]*Trie),
		generation:  0, // Initially zero but increases after every snapshot.
		deletedKeys: make(map[common.Hash]struct{}),
	}
}

// Snapshot creates a copy of the trie.
// Note it does not deep copy the trie, but will
// copy on write as modifications are done on this new trie.
// It does a snapshot of all child tries as well, and resets
// the set of deleted hashes.
func (t *Trie) Snapshot() (newTrie *Trie) {
	childTries := make(map[common.Hash]*Trie, len(t.childTries))
	for rootHash, childTrie := range t.childTries {
		childTries[rootHash] = &Trie{
			generation:  childTrie.generation + 1,
			root:        childTrie.root.Copy(false),
			deletedKeys: make(map[common.Hash]struct{}),
		}
	}

	return &Trie{
		generation:  t.generation + 1,
		root:        t.root,
		childTries:  childTries,
		deletedKeys: make(map[common.Hash]struct{}),
	}
}

func (t *Trie) maybeUpdateGeneration(currentNode Node) (newNode Node) {
	if currentNode.GetGeneration() == t.generation {
		// No need to update the current node, just return it
		// since its generation matches the one of the trie.
		return currentNode
	}

	// The node is from an older trie generation (snapshot)
	// so we need to deep copy the node and update the generation
	// on the newer copy.
	return updateGeneration(currentNode, t.generation, t.deletedKeys)
}

func updateGeneration(currentNode Node, trieGeneration uint64,
	deletedHashes map[common.Hash]struct{}) (newNode Node) {
	const copyChildren = false
	newNode = currentNode.Copy(copyChildren)
	newNode.SetGeneration(trieGeneration)

	// The hash of the node from a previous snapshotted trie
	// is usually already computed.
	deletedHashBytes := currentNode.GetHash()
	if len(deletedHashBytes) > 0 {
		deletedHash := common.BytesToHash(deletedHashBytes)
		deletedHashes[deletedHash] = struct{}{}
	}

	return newNode
}

// DeepCopy deep copies the trie and returns
// the copy. Note this method is meant to be used
// in tests and should not be used in production
// since it's rather inefficient compared to the copy
// on write mechanism achieved through snapshots.
func (t *Trie) DeepCopy() (trieCopy *Trie) {
	if t == nil {
		return nil
	}

	trieCopy = &Trie{
		generation: t.generation,
	}

	if t.deletedKeys != nil {
		trieCopy.deletedKeys = make(map[common.Hash]struct{}, len(t.deletedKeys))
		for k := range t.deletedKeys {
			trieCopy.deletedKeys[k] = struct{}{}
		}
	}

	if t.childTries != nil {
		trieCopy.childTries = make(map[common.Hash]*Trie, len(t.childTries))
		for hash, trie := range t.childTries {
			trieCopy.childTries[hash] = trie.DeepCopy()
		}
	}

	if t.root != nil {
		const copyChildren = true
		trieCopy.root = t.root.Copy(copyChildren)
	}

	return trieCopy
}

// RootNode returns a copy of the root node of the trie.
func (t *Trie) RootNode() Node {
	const copyChildren = false
	return t.root.Copy(copyChildren)
}

// encodeRoot writes the encoding of the root node to the buffer.
func encodeRoot(root node.Node, buffer node.Buffer) (err error) {
	if root == nil {
		_, err = buffer.Write([]byte{0})
		if err != nil {
			return fmt.Errorf("cannot write nil root node to buffer: %w", err)
		}
		return nil
	}
	return root.Encode(buffer)
}

// MustHash returns the hashed root of the trie.
// It panics if it fails to hash the root node.
func (t *Trie) MustHash() common.Hash {
	h, err := t.Hash()
	if err != nil {
		panic(err)
	}

	return h
}

// Hash returns the hashed root of the trie.
func (t *Trie) Hash() (rootHash common.Hash, err error) {
	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

	err = encodeRoot(t.root, buffer)
	if err != nil {
		return [32]byte{}, err
	}

	return common.Blake2bHash(buffer.Bytes()) // TODO optimisation: use hashers sync pools
}

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (t *Trie) Entries() map[string][]byte {
	return entries(t.root, nil, make(map[string][]byte))
}

func entries(parent Node, prefix []byte, kv map[string][]byte) map[string][]byte {
	if parent == nil {
		return kv
	}

	if parent.Type() == node.LeafType {
		parentKey := parent.GetKey()
		fullKeyNibbles := make([]byte, 0, len(prefix)+len(parentKey))
		fullKeyNibbles = append(fullKeyNibbles, prefix...)
		fullKeyNibbles = append(fullKeyNibbles, parentKey...)
		keyLE := string(codec.NibblesToKeyLE(fullKeyNibbles))
		kv[keyLE] = parent.GetValue()
		return kv
	}

	// Branch with/without value
	branch := parent.(*node.Branch)

	if branch.Value != nil {
		fullKeyNibbles := make([]byte, 0, len(prefix)+len(branch.Key))
		fullKeyNibbles = append(fullKeyNibbles, prefix...)
		fullKeyNibbles = append(fullKeyNibbles, branch.Key...)
		keyLE := string(codec.NibblesToKeyLE(fullKeyNibbles))
		kv[keyLE] = branch.Value
	}

	for i, child := range branch.Children {
		childPrefix := make([]byte, 0, len(prefix)+len(branch.Key)+1)
		childPrefix = append(childPrefix, prefix...)
		childPrefix = append(childPrefix, branch.Key...)
		childPrefix = append(childPrefix, byte(i))
		entries(child, childPrefix, kv)
	}

	return kv
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (t *Trie) NextKey(keyLE []byte) (nextKeyLE []byte) {
	prefix := []byte(nil)
	key := codec.KeyLEToNibbles(keyLE)

	nextKey := findNextKey(t.root, prefix, key)
	if nextKey == nil {
		return nil
	}

	nextKeyLE = codec.NibblesToKeyLE(nextKey)
	return nextKeyLE
}

func findNextKey(parent Node, prefix, searchKey []byte) (nextKey []byte) {
	if parent == nil {
		return nil
	}

	if parent.Type() == node.LeafType {
		parentLeaf := parent.(*node.Leaf)
		return findNextKeyLeaf(parentLeaf, prefix, searchKey)
	}

	// Branch
	parentBranch := parent.(*node.Branch)
	return findNextKeyBranch(parentBranch, prefix, searchKey)
}

func findNextKeyLeaf(leaf *node.Leaf, prefix, searchKey []byte) (nextKey []byte) {
	parentLeafKey := leaf.Key
	fullKey := make([]byte, 0, len(prefix)+len(parentLeafKey))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, parentLeafKey...)

	searchKeyBigger :=
		(len(searchKey) < len(fullKey) &&
			bytes.Compare(searchKey, fullKey[:len(searchKey)]) == 1) ||
			(len(searchKey) >= len(fullKey) &&
				bytes.Compare(searchKey[:len(fullKey)], fullKey) != -1)
	if searchKeyBigger {
		return nil
	}

	return fullKey
}

func findNextKeyBranch(parentBranch *node.Branch, prefix, searchKey []byte) (nextKey []byte) {
	fullKey := make([]byte, 0, len(prefix)+len(parentBranch.Key))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, parentBranch.Key...)

	if bytes.Equal(searchKey, fullKey) {
		const startChildIndex = 0
		return findNextKeyChild(parentBranch.Children, startChildIndex, fullKey, searchKey)
	}

	searchKeyShorter := len(searchKey) < len(fullKey)
	searchKeyLonger := len(searchKey) > len(fullKey)

	searchKeyBigger :=
		(searchKeyShorter &&
			bytes.Compare(searchKey, fullKey[:len(searchKey)]) == 1) ||
			(!searchKeyShorter &&
				bytes.Compare(searchKey[:len(fullKey)], fullKey) != -1)

	if searchKeyBigger {
		if searchKeyShorter {
			return nil
		} else if searchKeyLonger {
			startChildIndex := searchKey[len(fullKey)]
			return findNextKeyChild(parentBranch.Children,
				startChildIndex, fullKey, searchKey)
		}
	}

	// search key is smaller than full key
	if parentBranch.Value != nil {
		return fullKey
	}
	const startChildIndex = 0
	return findNextKeyChild(parentBranch.Children, startChildIndex,
		fullKey, searchKey)
}

// findNextKeyChild searches for a next key in the children
// given and returns a next key or nil if no next key is found.
func findNextKeyChild(children [16]node.Node, startIndex byte,
	fullKey, key []byte) (nextKey []byte) {
	for i := startIndex; i < node.ChildrenCapacity; i++ {
		child := children[i]
		if child == nil {
			continue
		}

		childFullKey := make([]byte, 0, len(fullKey)+1)
		childFullKey = append(childFullKey, fullKey...)
		childFullKey = append(childFullKey, i)
		next := findNextKey(child, childFullKey, key)
		if len(next) > 0 {
			return next
		}
	}

	return nil
}

// Put inserts a value into the trie at the
// key specified in little Endian format.
func (t *Trie) Put(keyLE, value []byte) {
	nibblesKey := codec.KeyLEToNibbles(keyLE)
	t.put(nibblesKey, value)
}

func (t *Trie) put(key, value []byte) {
	nodeToInsert := &node.Leaf{
		Value:      value,
		Generation: t.generation,
		Dirty:      true,
	}
	t.root = t.insert(t.root, key, nodeToInsert)
}

// insert attempts to insert a key with value into the trie
func (t *Trie) insert(parent Node, key []byte, value Node) (newParent Node) {
	// TODO change value node to be value []byte?
	value.SetGeneration(t.generation) // just in case it's not set by the caller.

	if parent == nil {
		value.SetKey(key)
		return value
	}

	// TODO ensure all values have dirty set to true
	newParent = t.maybeUpdateGeneration(parent)

	switch newParent.Type() {
	case node.BranchType, node.BranchWithValueType:
		parentBranch := newParent.(*node.Branch)
		return t.insertInBranch(parentBranch, key, value)
	default:
		parentLeaf := newParent.(*node.Leaf)
		return t.insertInLeaf(parentLeaf, key, value)
	}
}

func (t *Trie) insertInBranch(parentBranch *node.Branch, key []byte,
	value Node) (newParent Node) {
	newParent = t.updateBranch(parentBranch, key, value)

	if newParent.IsDirty() {
		// the older parent branch might had been pushed down the trie
		// under the new parent branch, so mark it dirty.
		parentBranch.SetDirty(true)
	}

	return newParent
}

func (t *Trie) insertInLeaf(parentLeaf *node.Leaf, key []byte,
	value Node) (newParent Node) {
	newValue := value.(*node.Leaf).Value

	if bytes.Equal(parentLeaf.Key, key) {
		if !bytes.Equal(newValue, parentLeaf.Value) {
			parentLeaf.Value = newValue
			parentLeaf.SetDirty(true)
		}
		return parentLeaf
	}

	commonPrefixLength := lenCommonPrefix(key, parentLeaf.Key)

	// Convert the current leaf parent into a branch parent
	newBranchParent := &node.Branch{
		Key:        key[:commonPrefixLength],
		Generation: t.generation,
		Dirty:      true,
	}
	parentLeafKey := parentLeaf.Key

	if len(key) == commonPrefixLength {
		// key is included in parent leaf key
		newBranchParent.Value = newValue

		if len(key) < len(parentLeafKey) {
			// Move the current leaf parent as a child to the new branch.
			childIndex := parentLeafKey[commonPrefixLength]
			parentLeaf.Key = parentLeaf.Key[commonPrefixLength+1:]
			parentLeaf.Dirty = true
			newBranchParent.Children[childIndex] = parentLeaf
		}

		return newBranchParent
	}

	value.SetKey(key[commonPrefixLength+1:])

	if len(parentLeaf.Key) == commonPrefixLength {
		// the key of the parent leaf is at this new branch
		newBranchParent.Value = parentLeaf.Value
	} else {
		// make the leaf a child of the new branch
		childIndex := parentLeafKey[commonPrefixLength]
		parentLeaf.Key = parentLeaf.Key[commonPrefixLength+1:]
		parentLeaf.SetDirty(true)
		newBranchParent.Children[childIndex] = parentLeaf
	}
	childIndex := key[commonPrefixLength]
	newBranchParent.Children[childIndex] = value

	return newBranchParent
}

func (t *Trie) updateBranch(parentBranch *node.Branch, key []byte, value Node) (newParent Node) {
	if bytes.Equal(key, parentBranch.Key) {
		parentBranch.SetDirty(true)
		parentBranch.Value = value.GetValue()
		return parentBranch
	}

	if bytes.HasPrefix(key, parentBranch.Key) {
		// key is included in parent branch key
		commonPrefixLength := lenCommonPrefix(key, parentBranch.Key)
		childIndex := key[commonPrefixLength]
		remainingKey := key[commonPrefixLength+1:]
		child := parentBranch.Children[childIndex]

		if child == nil {
			child = &node.Leaf{
				Key:        remainingKey,
				Value:      value.GetValue(),
				Generation: t.generation,
				Dirty:      true,
			}
		} else {
			child = t.insert(child, remainingKey, value)
			child.SetDirty(true)
		}

		parentBranch.Children[childIndex] = child
		parentBranch.SetDirty(true)
		return parentBranch
	}

	// we need to branch out at the point where the keys diverge
	// update partial keys, new branch has key up to matching length
	commonPrefixLength := lenCommonPrefix(key, parentBranch.Key)
	newParentBranch := &node.Branch{
		Key:        key[:commonPrefixLength],
		Generation: t.generation,
		Dirty:      true,
	}

	oldParentIndex := parentBranch.Key[commonPrefixLength]
	remainingOldParentKey := parentBranch.Key[commonPrefixLength+1:]
	newParentBranch.Children[oldParentIndex] = t.insert(nil, remainingOldParentKey, parentBranch)

	if len(key) <= commonPrefixLength {
		newParentBranch.Value = value.(*node.Leaf).Value
	} else {
		childIndex := key[commonPrefixLength]
		remainingKey := key[commonPrefixLength+1:]
		newParentBranch.Children[childIndex] = t.insert(nil, remainingKey, value)
	}

	newParentBranch.SetDirty(true)
	return newParentBranch
}

// LoadFromMap loads the given data mapping of key to value into the trie.
// The keys are in hexadecimal little Endian encoding and the values
// are hexadecimal encoded.
func (t *Trie) LoadFromMap(data map[string]string) (err error) {
	for key, value := range data {
		keyLEBytes, err := common.HexToBytes(key)
		if err != nil {
			return fmt.Errorf("cannot convert key hex to bytes: %w", err)
		}

		valueBytes, err := common.HexToBytes(value)
		if err != nil {
			return fmt.Errorf("cannot convert value hex to bytes: %w", err)
		}

		t.Put(keyLEBytes, valueBytes)
	}

	return nil
}

// GetKeysWithPrefix returns all keys in little Endian
// format from nodes in the trie that have the given little
// Endian formatted prefix in their key.
func (t *Trie) GetKeysWithPrefix(prefixLE []byte) (keysLE [][]byte) {
	var prefixNibbles []byte
	if len(prefixLE) > 0 {
		prefixNibbles = codec.KeyLEToNibbles(prefixLE)
		prefixNibbles = bytes.TrimSuffix(prefixNibbles, []byte{0})
	}

	prefix := []byte{}
	key := prefixNibbles
	return getKeysWithPrefix(t.root, prefix, key, keysLE)
}

// getKeysWithPrefix returns all keys in little Endian format that have the
// prefix given. The prefix and key byte slices are in nibbles format.
// TODO pass in map of keysLE if order is not needed.
// TODO do all processing on nibbles keys and then convert to LE.
func getKeysWithPrefix(parent Node, prefix, key []byte,
	keysLE [][]byte) (newKeysLE [][]byte) {
	if parent == nil {
		return keysLE
	}

	if parent.Type() == node.LeafType {
		parentLeaf := parent.(*node.Leaf)
		return getKeysWithPrefixFromLeaf(parentLeaf, prefix, key, keysLE)
	}

	parentBranch := parent.(*node.Branch)
	return getKeysWithPrefixFromBranch(parentBranch, prefix, key, keysLE)
}

func getKeysWithPrefixFromLeaf(parent *node.Leaf, prefix, key []byte,
	keysLE [][]byte) (newKeysLE [][]byte) {
	if len(key) == 0 || bytes.HasPrefix(parent.Key, key) {
		fullKeyLE := makeFullKeyLE(prefix, parent.Key)
		keysLE = append(keysLE, fullKeyLE)
	}
	return keysLE
}

func getKeysWithPrefixFromBranch(parent *node.Branch, prefix, key []byte,
	keysLE [][]byte) (newKeysLE [][]byte) {
	if len(key) == 0 || bytes.HasPrefix(parent.Key, key) {
		return addAllKeys(parent, prefix, keysLE)
	}

	noPossiblePrefixedKeys :=
		len(parent.Key) > len(key) &&
			!bytes.HasPrefix(parent.Key, key)
	if noPossiblePrefixedKeys {
		return keysLE
	}

	key = key[len(parent.Key):]
	childIndex := key[0]
	child := parent.Children[childIndex]
	childPrefix := makeChildPrefix(prefix, parent.Key, int(childIndex))
	childKey := key[1:]
	return getKeysWithPrefix(child, childPrefix, childKey, keysLE)
}

// addAllKeys appends all keys of descendant nodes of the parent node
// to the slice of keys given and returns this slice.
// It uses the prefix in nibbles format to determine the full key.
// The slice of keys has its keys formatted in little Endian.
func addAllKeys(parent Node, prefix []byte, keysLE [][]byte) (newKeysLE [][]byte) {
	if parent == nil {
		return keysLE
	}

	if parent.Type() == node.LeafType {
		keyLE := makeFullKeyLE(prefix, parent.GetKey())
		keysLE = append(keysLE, keyLE)
		return keysLE
	}

	// Branches
	branchParent := parent.(*node.Branch)
	if branchParent.Value != nil {
		keyLE := makeFullKeyLE(prefix, branchParent.Key)
		keysLE = append(keysLE, keyLE)
	}

	for i, child := range branchParent.Children {
		childPrefix := makeChildPrefix(prefix, branchParent.Key, i)
		keysLE = addAllKeys(child, childPrefix, keysLE)
	}

	return keysLE
}

func makeFullKeyLE(prefix, nodeKey []byte) (fullKeyLE []byte) {
	fullKey := make([]byte, 0, len(prefix)+len(nodeKey))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, nodeKey...)
	fullKeyLE = codec.NibblesToKeyLE(fullKey)
	return fullKeyLE
}

func makeChildPrefix(branchPrefix, branchKey []byte,
	childIndex int) (childPrefix []byte) {
	childPrefix = make([]byte, 0, len(branchPrefix)+len(branchKey)+1)
	childPrefix = append(childPrefix, branchPrefix...)
	childPrefix = append(childPrefix, branchKey...)
	childPrefix = append(childPrefix, byte(childIndex))
	return childPrefix
}

// Get returns the value in the node of the trie
// which matches its key with the key given.
// Note the key argument is given in little Endian format.
func (t *Trie) Get(keyLE []byte) (value []byte) {
	keyNibbles := codec.KeyLEToNibbles(keyLE)
	return retrieve(t.root, keyNibbles)
}

func retrieve(parent Node, key []byte) (value []byte) {
	if parent == nil {
		return nil
	}

	if parent.Type() == node.LeafType {
		leaf := parent.(*node.Leaf)
		return retrieveFromLeaf(leaf, key)
	}

	// Branches
	branch := parent.(*node.Branch)
	return retrieveFromBranch(branch, key)
}

func retrieveFromLeaf(leaf *node.Leaf, key []byte) (value []byte) {
	if bytes.Equal(leaf.Key, key) {
		return leaf.Value
	}
	return nil
}

func retrieveFromBranch(branch *node.Branch, key []byte) (value []byte) {
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		return branch.Value
	}

	if len(branch.Key) > len(key) && bytes.HasPrefix(branch.Key, key) {
		return nil
	}

	commonPrefixLength := lenCommonPrefix(branch.Key, key)
	childIndex := key[commonPrefixLength]
	childKey := key[commonPrefixLength+1:]
	child := branch.Children[childIndex]
	return retrieve(child, childKey)
}

// ClearPrefixLimit deletes the keys having the prefix given in little
// Endian format for up to `limit` keys. It returns the number of deleted
// keys and a boolean indicating if all keys with the prefix were deleted
// within the limit.
func (t *Trie) ClearPrefixLimit(prefixLE []byte, limit uint32) (deleted uint32, allDeleted bool) {
	if limit == 0 {
		return 0, false
	}

	prefix := codec.KeyLEToNibbles(prefixLE)
	prefix = bytes.TrimSuffix(prefix, []byte{0})

	initialLimit := limit
	t.root, _, allDeleted = t.clearPrefixLimit(t.root, prefix, &limit)
	deleted = initialLimit - limit
	return deleted, allDeleted
}

// clearPrefixLimit deletes the keys having the prefix till limit reached and returns updated trie root node,
// true if any node in the trie got updated, and next bool returns true if there is no keys left with prefix.
// TODO return deleted count and deduce updated from deleted count, do not pass limit as pointer.
func (t *Trie) clearPrefixLimit(parent Node, prefix []byte, limit *uint32) (
	newParent Node, updated bool, allDeleted bool) {
	if parent == nil {
		return nil, false, true
	}

	newParent = t.maybeUpdateGeneration(parent)

	if newParent.Type() == node.LeafType {
		leaf := newParent.(*node.Leaf)
		// if prefix is not found, it's also all deleted.
		// TODO check this is the same behaviour as in substrate
		const allDeleted = true
		if bytes.HasPrefix(leaf.Key, prefix) {
			*limit--
			return nil, true, allDeleted
		}
		// not modified so return the leaf of the original
		// trie generation. The copied leaf newParent will be
		// garbage collected.
		return parent, false, allDeleted
	}

	branch := newParent.(*node.Branch)
	newParent, updated, allDeleted = t.clearPrefixLimitBranch(branch, prefix, limit)
	if !updated {
		// not modified so return the node of the original
		// trie generation. The copied newParent will be
		// garbage collected.
		newParent = parent
	}

	return newParent, updated, allDeleted
}

func (t *Trie) clearPrefixLimitBranch(branch *node.Branch, prefix []byte, limit *uint32) (
	newParent Node, updated, allDeleted bool) {
	newParent = branch

	if bytes.HasPrefix(branch.Key, prefix) {
		updated = true // at least one node will be deleted
		nilPrefix := ([]byte)(nil)
		// TODO return deleted count to replace updated boolean and update limit
		newParent = t.deleteNodesLimit(branch, nilPrefix, limit)
		allDeleted = newParent == nil
		return newParent, updated, allDeleted
	}

	if len(prefix) == len(branch.Key)+1 &&
		bytes.HasPrefix(branch.Key, prefix[:len(prefix)-1]) {
		// Prefix is one the children of the branch
		return t.clearPrefixLimitChild(branch, prefix, limit)
	}

	noPrefixForNode := len(prefix) <= len(branch.Key) ||
		lenCommonPrefix(branch.Key, prefix) < len(branch.Key)
	if noPrefixForNode {
		updated, allDeleted = false, true
		return newParent, updated, allDeleted
	}

	childIndex := prefix[len(branch.Key)]
	childPrefix := prefix[len(branch.Key)+1:]
	child := branch.Children[childIndex]

	newParent = branch // mostly just a reminder for the reader
	branch.Children[childIndex], updated, allDeleted = t.clearPrefixLimit(child, childPrefix, limit)
	if updated {
		branch.SetDirty(true)
		newParent = handleDeletion(branch, prefix)
	}

	return newParent, newParent.IsDirty(), allDeleted
}

func (t *Trie) clearPrefixLimitChild(branch *node.Branch, prefix []byte, limit *uint32) (
	newParent Node, updated, allDeleted bool) {
	newParent = branch

	childIndex := prefix[len(branch.Key)]
	child := branch.Children[childIndex]

	if child == nil {
		// TODO ensure this is the same behaviour as in substrate
		updated, allDeleted = false, true
		return newParent, updated, allDeleted
	}

	nilPrefix := ([]byte)(nil)
	branch.Children[childIndex] = t.deleteNodesLimit(child, nilPrefix, limit)
	branch.SetDirty(true)

	newParent = handleDeletion(branch, prefix)

	updated = true
	allDeleted = branch.Children[childIndex] == nil
	return newParent, updated, allDeleted
}

func (t *Trie) deleteNodesLimit(parent Node, prefix []byte, limit *uint32) (newParent Node) {
	if *limit == 0 {
		return parent
	}

	if parent == nil {
		return nil
	}

	newParent = t.maybeUpdateGeneration(parent)

	if newParent.Type() == node.LeafType {
		*limit--
		return nil
	}

	branch := newParent.(*node.Branch)

	fullKey := make([]byte, 0, len(prefix)+len(branch.Key))
	fullKey = append(fullKey, prefix...)
	fullKey = append(fullKey, branch.Key...)

	nilChildren := node.ChildrenCapacity - branch.NumChildren()

	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		branch.Children[i] = t.deleteNodesLimit(child, fullKey, limit)
		if branch.Children[i] == nil {
			nilChildren++
		}

		branch.SetDirty(true)
		newParent = handleDeletion(branch, fullKey)
		if nilChildren == node.ChildrenCapacity &&
			branch.Value == nil {
			return nil
		}

		if *limit == 0 {
			return newParent
		}
	}

	if branch.Value != nil {
		*limit--
	}

	return nil
}

// ClearPrefix deletes all nodes in the trie for which the key contains the
// prefix given in little Endian format.
func (t *Trie) ClearPrefix(prefixLE []byte) {
	if len(prefixLE) == 0 {
		t.root = nil
		return
	}

	prefix := codec.KeyLEToNibbles(prefixLE)
	prefix = bytes.TrimSuffix(prefix, []byte{0})

	t.root, _ = t.clearPrefix(t.root, prefix)
}

func (t *Trie) clearPrefix(parent Node, prefix []byte) (
	newParent Node, updated bool) {
	if parent == nil {
		return nil, false
	}

	newParent = t.maybeUpdateGeneration(parent)

	if bytes.HasPrefix(newParent.GetKey(), prefix) {
		return nil, true
	}

	if newParent.Type() == node.LeafType {
		// not modified so return the leaf of the original
		// trie generation. The copied newParent will be
		// garbage collected.
		return parent, false
	}

	branch := newParent.(*node.Branch)

	if len(prefix) == len(branch.Key)+1 &&
		bytes.HasPrefix(branch.Key, prefix[:len(prefix)-1]) {
		// Prefix is one of the children of the branch
		childIndex := prefix[len(branch.Key)]
		child := branch.Children[childIndex]

		if child == nil {
			// child is already nil at the child index
			// node is not modified so return the branch of the original
			// trie generation. The copied newParent will be
			// garbage collected.
			return parent, false
		}

		branch.Children[childIndex] = nil
		branch.SetDirty(true)
		newParent = handleDeletion(branch, prefix)
		return newParent, true
	}

	noPrefixForNode := len(prefix) <= len(branch.Key) ||
		lenCommonPrefix(branch.Key, prefix) < len(branch.Key)
	if noPrefixForNode {
		// not modified so return the branch of the original
		// trie generation. The copied newParent will be
		// garbage collected.
		return parent, false
	}

	childIndex := prefix[len(branch.Key)]
	childPrefix := prefix[len(branch.Key)+1:]
	child := branch.Children[childIndex]

	branch.Children[childIndex], updated = t.clearPrefix(child, childPrefix)
	if !updated {
		// branch not modified so return the branch of the original
		// trie generation. The copied newParent will be
		// garbage collected.
		return parent, false
	}

	branch.SetDirty(true)
	newParent = handleDeletion(branch, prefix)
	return newParent, true
}

// Delete removes the node of the trie with the key
// matching the key given in little Endian format.
// If no node is found at this key, nothing is deleted.
func (t *Trie) Delete(keyLE []byte) {
	key := codec.KeyLEToNibbles(keyLE)
	t.root, _ = t.delete(t.root, key)
}

func (t *Trie) delete(parent Node, key []byte) (newParent Node, deleted bool) {
	if parent == nil {
		return nil, false
	}

	newParent = t.maybeUpdateGeneration(parent)

	if newParent.Type() == node.LeafType {
		newParent = deleteLeaf(newParent, key)
		if newParent == nil {
			return nil, true
		}
		// The leaf was not deleted so return the original
		// parent without its generation updated.
		// The copied newParent will be garbage collected.
		return parent, false
	}

	branch := newParent.(*node.Branch)
	newParent, deleted = t.deleteBranch(branch, key)
	if !deleted {
		// Nothing was deleted so return the original
		// parent without its generation updated.
		// The copied newParent will be garbage collected.
		return parent, false
	}

	return newParent, true
}

func deleteLeaf(parent Node, key []byte) (newParent Node) {
	if len(key) == 0 || bytes.Equal(key, parent.GetKey()) {
		return nil
	}
	return parent
}

func (t *Trie) deleteBranch(branch *node.Branch, key []byte) (newParent Node, deleted bool) {
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		branch.Value = nil
		branch.SetDirty(true)
		return handleDeletion(branch, key), true
	}

	commonPrefixLength := lenCommonPrefix(branch.Key, key)
	childIndex := key[commonPrefixLength]
	childKey := key[commonPrefixLength+1:]
	child := branch.Children[childIndex]

	newChild, deleted := t.delete(child, childKey)
	if !deleted {
		return branch, false
	}

	branch.Children[childIndex] = newChild
	branch.SetDirty(true)
	newParent = handleDeletion(branch, key)
	return newParent, true
}

// handleDeletion is called when a value is deleted from a branch to handle
// the eventual mutation of the branch depending on its children.
// If the branch has no value and a single child, it will be combined with this child.
// If the branch has a value and no child, it will be changed into a leaf.
func handleDeletion(branch *node.Branch, key []byte) (newNode Node) {
	// TODO try to remove key argument just use branch.Key instead?
	childrenCount := 0
	firstChildIndex := -1
	for i, child := range branch.Children {
		if child == nil {
			continue
		}
		if firstChildIndex == -1 {
			firstChildIndex = i
		}
		childrenCount++
	}

	switch {
	default:
		return branch
	case childrenCount == 0 && branch.Value != nil:
		commonPrefixLength := lenCommonPrefix(branch.Key, key)
		return &node.Leaf{
			Key:        key[:commonPrefixLength],
			Value:      branch.Value,
			Dirty:      true,
			Generation: branch.Generation,
		}
	case childrenCount == 1 && branch.Value == nil:
		childIndex := firstChildIndex
		child := branch.Children[firstChildIndex]

		if child.Type() == node.LeafType {
			childLeafKey := child.GetKey()
			newLeafKey := make([]byte, 0, len(branch.Key)+1+len(childLeafKey))
			newLeafKey = append(newLeafKey, branch.Key...)
			newLeafKey = append(newLeafKey, byte(childIndex))
			newLeafKey = append(newLeafKey, childLeafKey...)
			return &node.Leaf{
				Key:        newLeafKey,
				Value:      child.GetValue(),
				Dirty:      true,
				Generation: branch.Generation,
			}
		}

		childBranch := child.(*node.Branch)
		newBranchKey := make([]byte, 0, len(branch.Key)+1+len(childBranch.Key))
		newBranchKey = append(newBranchKey, branch.Key...)
		newBranchKey = append(newBranchKey, byte(childIndex))
		newBranchKey = append(newBranchKey, childBranch.Key...)
		newBranch := &node.Branch{
			Key:        newBranchKey,
			Value:      childBranch.Value,
			Generation: branch.Generation,
			Dirty:      true,
		}

		// Adopt the grand-children
		for i, grandChild := range childBranch.Children {
			if grandChild != nil {
				newBranch.Children[i] = grandChild
				// No need to copy and update the generation
				// of the grand children since they are not modified.
			}
		}

		return newBranch
	}
}

// lenCommonPrefix returns the length of the
// common prefix between two byte slices.
func lenCommonPrefix(a, b []byte) (length int) {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}

	for length = 0; length < min; length++ {
		if a[length] != b[length] {
			break
		}
	}

	return length
}
