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
		fullKeyNibbles := append(prefix, parent.GetKey()...)
		keyLE := string(codec.NibblesToKeyLE(fullKeyNibbles))
		kv[keyLE] = parent.GetValue()
		return kv
	}

	// Branch with/without value
	branch := parent.(*node.Branch)

	if branch.Value != nil {
		fullKeyNibbles := append(prefix, branch.Key...)
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
	fullKey := append(prefix, parentLeafKey...)

	searchKeyBigger :=
		(len(searchKey) < len(fullKey) &&
			bytes.Compare(searchKey, fullKey[:len(searchKey)]) == 1) ||
			(len(searchKey) >= len(fullKey) &&
				bytes.Compare(searchKey[:len(fullKey)], fullKey) != -1)
	if searchKeyBigger {
		return nil
	}

	nextKey = append(prefix, parentLeafKey...)
	return nextKey
}

func findNextKeyBranch(parentBranch *node.Branch, prefix, searchKey []byte) (nextKey []byte) {
	fullKey := append(prefix, parentBranch.Key...)

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
	return findNextKeyChild(parentBranch.Children, startChildIndex, fullKey, searchKey)
}

// findNextKeyChild searches for a next key in the children
// given and returns a next key or nil if no next key is found.
func findNextKeyChild(children [16]node.Node, startIndex byte,
	fullKey, key []byte) (nextKey []byte) {
	for i := startIndex; i < byte(len(children)); i++ {
		child := children[i]
		if child == nil {
			continue
		}

		childFullKey := append(fullKey, i)
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
		keysLE = append(keysLE, codec.NibblesToKeyLE(append(prefix, parent.GetKey()...)))
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
	fullKey := append(prefix, nodeKey...)
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

// Get returns the value for key stored in the trie at the corresponding key
func (t *Trie) Get(key []byte) []byte {
	keyNibbles := codec.KeyLEToNibbles(key)
	return retrieve(t.root, keyNibbles)
}

func retrieve(parent Node, key []byte) (value []byte) {
	switch p := parent.(type) {
	case *node.Branch:
		length := lenCommonPrefix(p.Key, key)

		// found the value at this node
		if bytes.Equal(p.Key, key) || len(key) == 0 {
			return p.Value
		}

		// did not find value
		if bytes.Equal(p.Key[:length], key) && len(key) < len(p.Key) {
			return nil
		}

		value = retrieve(p.Children[key[length]], key[length+1:])
	case *node.Leaf:
		if bytes.Equal(p.Key, key) {
			value = p.Value
		}
	case nil:
		return nil
	}
	return value // TODO remove
}

// ClearPrefixLimit deletes the keys having the prefix till limit reached
func (t *Trie) ClearPrefixLimit(prefix []byte, limit uint32) (uint32, bool) {
	if limit == 0 {
		return 0, false
	}

	p := codec.KeyLEToNibbles(prefix)
	p = bytes.TrimSuffix(p, []byte{0})

	l := limit
	var allDeleted bool
	t.root, _, allDeleted = t.clearPrefixLimit(t.root, p, &limit)
	return l - limit, allDeleted
}

// clearPrefixLimit deletes the keys having the prefix till limit reached and returns updated trie root node,
// true if any node in the trie got updated, and next bool returns true if there is no keys left with prefix.
func (t *Trie) clearPrefixLimit(cn Node, prefix []byte, limit *uint32) (Node, bool, bool) {
	if cn == nil {
		return nil, false, true
	}

	curr := t.maybeUpdateGeneration(cn)

	switch c := curr.(type) {
	case *node.Branch:
		length := lenCommonPrefix(c.Key, prefix)
		if length == len(prefix) {
			n := t.deleteNodes(c, []byte{}, limit)
			if n == nil {
				return nil, true, true
			}
			return n, true, false
		}

		if len(prefix) == len(c.Key)+1 && length == len(prefix)-1 {
			i := prefix[len(c.Key)]

			if c.Children[i] == nil {
				// child is already nil at the child index
				return c, false, true
			}

			c.Children[i] = t.deleteNodes(c.Children[i], []byte{}, limit)

			c.SetDirty(true)
			curr = handleDeletion(c, prefix)

			if c.Children[i] == nil {
				return curr, true, true
			}
			return c, true, false
		}

		if len(prefix) <= len(c.Key) || length < len(c.Key) {
			// this node doesn't have the prefix, return
			return c, false, true
		}

		i := prefix[len(c.Key)]

		var wasUpdated, allDeleted bool
		c.Children[i], wasUpdated, allDeleted = t.clearPrefixLimit(c.Children[i], prefix[len(c.Key)+1:], limit)
		if wasUpdated {
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.IsDirty(), allDeleted
	case *node.Leaf:
		length := lenCommonPrefix(c.Key, prefix)
		if length == len(prefix) {
			*limit--
			return nil, true, true
		}
		// Prefix not found might be all deleted
		return curr, false, true
	}

	return nil, false, true // TODO remove
}

func (t *Trie) deleteNodes(cn Node, prefix []byte, limit *uint32) (newNode Node) {
	if *limit == 0 || cn == nil {
		return cn
	}

	curr := t.maybeUpdateGeneration(cn)

	switch c := curr.(type) {
	case *node.Leaf:
		*limit--
		return nil
	case *node.Branch:
		if len(c.Key) != 0 {
			prefix = append(prefix, c.Key...)
		}

		for i, child := range c.Children {
			if child == nil {
				continue
			}

			c.Children[i] = t.deleteNodes(child, prefix, limit)

			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
			isAllNil := c.NumChildren() == 0
			if isAllNil && c.Value == nil {
				curr = nil
			}

			if *limit == 0 {
				return curr
			}
		}

		// Delete the current node as well
		if c.Value != nil {
			*limit--
		}
		return nil
	}

	return curr
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (t *Trie) ClearPrefix(prefix []byte) {
	if len(prefix) == 0 {
		t.root = nil
		return
	}

	p := codec.KeyLEToNibbles(prefix)
	p = bytes.TrimSuffix(p, []byte{0})

	t.root, _ = t.clearPrefix(t.root, p)
}

func (t *Trie) clearPrefix(cn Node, prefix []byte) (Node, bool) {
	if cn == nil {
		return nil, false
	}

	curr := t.maybeUpdateGeneration(cn)
	switch c := curr.(type) {
	case *node.Branch:
		length := lenCommonPrefix(c.Key, prefix)

		if length == len(prefix) {
			// found prefix at this branch, delete it
			return nil, true
		}

		// Store the current node and return it, if the trie is not updated.

		if len(prefix) == len(c.Key)+1 && length == len(prefix)-1 {
			// found prefix at child index, delete child
			i := prefix[len(c.Key)]

			if c.Children[i] == nil {
				// child is already nil at the child index
				return c, false
			}

			c.Children[i] = nil
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
			return curr, true
		}

		if len(prefix) <= len(c.Key) || length < len(c.Key) {
			// this node doesn't have the prefix, return
			return c, false
		}

		var wasUpdated bool
		i := prefix[len(c.Key)]

		c.Children[i], wasUpdated = t.clearPrefix(c.Children[i], prefix[len(c.Key)+1:])
		if wasUpdated {
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.IsDirty()
	case *node.Leaf:
		length := lenCommonPrefix(c.Key, prefix)
		if length == len(prefix) {
			return nil, true
		}
		return c, false
	}
	// This should never happen.
	return nil, false // TODO remove
}

// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) {
	k := codec.KeyLEToNibbles(key)
	t.root, _ = t.delete(t.root, k)
}

func (t *Trie) delete(parent Node, key []byte) (Node, bool) {
	if parent == nil {
		return nil, false
	}

	// Store the current node and return it, if the trie is not updated.
	switch p := t.maybeUpdateGeneration(parent).(type) {
	case *node.Branch:

		length := lenCommonPrefix(p.Key, key)
		if bytes.Equal(p.Key, key) || len(key) == 0 {
			// found the value at this node
			p.Value = nil
			p.SetDirty(true)
			return handleDeletion(p, key), true
		}

		n, del := t.delete(p.Children[key[length]], key[length+1:])
		if !del {
			// If nothing was deleted then don't copy the path.
			// Return the parent without its generation updated.
			return parent, false
		}

		p.Children[key[length]] = n
		p.SetDirty(true)
		n = handleDeletion(p, key)
		return n, true
	case *node.Leaf:
		if bytes.Equal(key, p.Key) || len(key) == 0 {
			// Key exists. Delete it.
			return nil, true
		}
		// Key doesn't exist, return parent
		// without its generation changed
		return parent, false
	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", p, p, key))
	}
}

// handleDeletion is called when a value is deleted from a branch
// if the updated branch only has 1 child, it should be combined with that child
// if the updated branch only has a value, it should be turned into a leaf
func handleDeletion(p *node.Branch, key []byte) Node {
	// TODO try to remove key argument just use p.Key instead?
	var n Node = p
	length := lenCommonPrefix(p.Key, key)
	bitmap := p.ChildrenBitmap()

	// if branch has no children, just a value, turn it into a leaf
	if bitmap == 0 && p.Value != nil {
		n = node.NewLeaf(key[:length], p.Value, true, p.Generation)
	} else if p.NumChildren() == 1 && p.Value == nil {
		// there is only 1 child and no value, combine the child branch with this branch
		// find index of child
		var i int
		for i = 0; i < 16; i++ {
			bitmap = bitmap >> 1
			if bitmap == 0 {
				break
			}
		}

		child := p.Children[i]
		switch c := child.(type) {
		case *node.Leaf:
			key = append(append(p.Key, []byte{byte(i)}...), c.Key...)
			const dirty = true
			n = node.NewLeaf(
				key,
				c.Value,
				dirty,
				p.Generation,
			)
		case *node.Branch:
			br := new(node.Branch)
			br.Key = append(p.Key, append([]byte{byte(i)}, c.Key...)...)

			// adopt the grandchildren
			for i, grandchild := range c.Children {
				if grandchild != nil {
					br.Children[i] = grandchild
					// No need to copy and update the generation
					// of the grand children since they are not modified.
				}
			}

			br.Value = c.Value
			br.Generation = p.Generation
			n = br
		default:
			// TODO remove
			// do nothing
		}
		n.SetDirty(true)

	}
	return n
}

// lenCommonPrefix returns the length of the common prefix between two keys
func lenCommonPrefix(a, b []byte) int {
	var length, min = 0, len(a)

	if len(a) > len(b) {
		min = len(b)
	}

	for ; length < min; length++ {
		if a[length] != b[length] {
			break
		}
	}

	return length
}
