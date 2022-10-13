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
	generation uint64
	root       *Node
	childTries map[common.Hash]*Trie
	// deletedMerkleValues are the node Merkle values that were deleted
	// from this trie since the last snapshot. These are used by the online
	// pruner to detect with database keys (trie node Merkle values) can
	// be deleted.
	deletedMerkleValues map[string]struct{}
}

// NewEmptyTrie creates a trie with a nil root
func NewEmptyTrie() *Trie {
	return NewTrie(nil)
}

// NewTrie creates a trie with an existing root node
func NewTrie(root *Node) *Trie {
	return &Trie{
		root:                root,
		childTries:          make(map[common.Hash]*Trie),
		generation:          0, // Initially zero but increases after every snapshot.
		deletedMerkleValues: make(map[string]struct{}),
	}
}

// Snapshot creates a copy of the trie.
// Note it does not deep copy the trie, but will
// copy on write as modifications are done on this new trie.
// It does a snapshot of all child tries as well, and resets
// the set of deleted hashes.
func (t *Trie) Snapshot() (newTrie *Trie) {
	childTries := make(map[common.Hash]*Trie, len(t.childTries))
	rootCopySettings := node.DefaultCopySettings
	rootCopySettings.CopyCached = true
	for rootHash, childTrie := range t.childTries {
		childTries[rootHash] = &Trie{
			generation:          childTrie.generation + 1,
			root:                childTrie.root.Copy(rootCopySettings),
			deletedMerkleValues: make(map[string]struct{}),
		}
	}

	return &Trie{
		generation:          t.generation + 1,
		root:                t.root,
		childTries:          childTries,
		deletedMerkleValues: make(map[string]struct{}),
	}
}

// handleTrackedDeltas sets the pending deleted Merkle values in
// the trie deleted merkle values set if and only if success is true.
func (t *Trie) handleTrackedDeltas(success bool, pendingDeletedMerkleValues map[string]struct{}) {
	if !success {
		return
	}

	for merkleValue := range pendingDeletedMerkleValues {
		t.deletedMerkleValues[merkleValue] = struct{}{}
	}
}

func (t *Trie) prepForMutation(currentNode *Node,
	copySettings node.CopySettings,
	pendingDeletedMerkleValues map[string]struct{}) (
	newNode *Node, err error) {
	if currentNode.Generation == t.generation {
		// no need to track deleted node, deep copy the node and
		// update the node generation.
		newNode = currentNode
	} else {
		isRoot := currentNode == t.root
		err = registerDeletedMerkleValue(currentNode, isRoot,
			pendingDeletedMerkleValues)
		if err != nil {
			return nil, fmt.Errorf("registering deleted node: %w", err)
		}
		newNode = currentNode.Copy(copySettings)
		newNode.Generation = t.generation
	}
	newNode.SetDirty()
	return newNode, nil
}

func registerDeletedMerkleValue(node *Node, isRoot bool,
	pendingDeletedMerkleValues map[string]struct{}) (err error) {
	err = ensureMerkleValueIsCalculated(node, isRoot)
	if err != nil {
		return fmt.Errorf("ensuring Merkle value is calculated: %w", err)
	}

	if len(node.MerkleValue) < 32 {
		// Merkle values which are less than 32 bytes are inlined
		// in the parent branch and are not stored on disk, so there
		// is no need to track their deletion for the online pruning.
		return nil
	}

	if !node.Dirty {
		// Only register deleted nodes that were not previously modified
		// since the last trie snapshot.
		pendingDeletedMerkleValues[string(node.MerkleValue)] = struct{}{}
	}

	return nil
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

	if t.deletedMerkleValues != nil {
		trieCopy.deletedMerkleValues = make(map[string]struct{}, len(t.deletedMerkleValues))
		for k := range t.deletedMerkleValues {
			trieCopy.deletedMerkleValues[k] = struct{}{}
		}
	}

	if t.childTries != nil {
		trieCopy.childTries = make(map[common.Hash]*Trie, len(t.childTries))
		for hash, trie := range t.childTries {
			trieCopy.childTries[hash] = trie.DeepCopy()
		}
	}

	if t.root != nil {
		copySettings := node.DeepCopySettings
		trieCopy.root = t.root.Copy(copySettings)
	}

	return trieCopy
}

// RootNode returns a copy of the root node of the trie.
func (t *Trie) RootNode() *Node {
	copySettings := node.DefaultCopySettings
	copySettings.CopyCached = true
	return t.root.Copy(copySettings)
}

// encodeRoot writes the encoding of the root node to the buffer.
func encodeRoot(root *Node, buffer node.Buffer) (err error) {
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

func entries(parent *Node, prefix []byte, kv map[string][]byte) map[string][]byte {
	if parent == nil {
		return kv
	}

	if parent.Kind() == node.Leaf {
		parentKey := parent.Key
		fullKeyNibbles := concatenateSlices(prefix, parentKey)
		keyLE := string(codec.NibblesToKeyLE(fullKeyNibbles))
		kv[keyLE] = parent.SubValue
		return kv
	}

	branch := parent
	if branch.SubValue != nil {
		fullKeyNibbles := concatenateSlices(prefix, branch.Key)
		keyLE := string(codec.NibblesToKeyLE(fullKeyNibbles))
		kv[keyLE] = branch.SubValue
	}

	for i, child := range branch.Children {
		childPrefix := concatenateSlices(prefix, branch.Key, intToByteSlice(i))
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

func findNextKey(parent *Node, prefix, searchKey []byte) (nextKey []byte) {
	if parent == nil {
		return nil
	}

	if parent.Kind() == node.Leaf {
		return findNextKeyLeaf(parent, prefix, searchKey)
	}
	return findNextKeyBranch(parent, prefix, searchKey)
}

func findNextKeyLeaf(leaf *Node, prefix, searchKey []byte) (nextKey []byte) {
	parentLeafKey := leaf.Key
	fullKey := concatenateSlices(prefix, parentLeafKey)

	if keyIsLexicographicallyBigger(searchKey, fullKey) {
		return nil
	}

	return fullKey
}

func findNextKeyBranch(parentBranch *Node, prefix, searchKey []byte) (nextKey []byte) {
	fullKey := concatenateSlices(prefix, parentBranch.Key)

	if bytes.Equal(searchKey, fullKey) {
		const startChildIndex = 0
		return findNextKeyChild(parentBranch.Children, startChildIndex, fullKey, searchKey)
	}

	if keyIsLexicographicallyBigger(searchKey, fullKey) {
		if len(searchKey) < len(fullKey) {
			return nil
		} else if len(searchKey) > len(fullKey) {
			startChildIndex := searchKey[len(fullKey)]
			return findNextKeyChild(parentBranch.Children,
				startChildIndex, fullKey, searchKey)
		}
	}

	// search key is smaller than full key
	if parentBranch.SubValue != nil {
		return fullKey
	}
	const startChildIndex = 0
	return findNextKeyChild(parentBranch.Children, startChildIndex,
		fullKey, searchKey)
}

func keyIsLexicographicallyBigger(key, key2 []byte) (bigger bool) {
	if len(key) < len(key2) {
		return bytes.Compare(key, key2[:len(key)]) == 1
	}
	return bytes.Compare(key[:len(key2)], key2) != -1
}

// findNextKeyChild searches for a next key in the children
// given and returns a next key or nil if no next key is found.
func findNextKeyChild(children []*Node, startIndex byte,
	fullKey, key []byte) (nextKey []byte) {
	for i := startIndex; i < node.ChildrenCapacity; i++ {
		child := children[i]
		if child == nil {
			continue
		}

		childFullKey := concatenateSlices(fullKey, []byte{i})
		next := findNextKey(child, childFullKey, key)
		if len(next) > 0 {
			return next
		}
	}

	return nil
}

// Put inserts a value into the trie at the
// key specified in little Endian format.
func (t *Trie) Put(keyLE, value []byte) (err error) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()
	return t.insertKeyLE(keyLE, value, pendingDeletedMerkleValues)
}

func (t *Trie) insertKeyLE(keyLE, value []byte,
	deletedMerkleValues map[string]struct{}) (err error) {
	nibblesKey := codec.KeyLEToNibbles(keyLE)
	root, _, _, err := t.insert(t.root, nibblesKey, value, deletedMerkleValues)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

// insert inserts a value in the trie at the key specified.
// It may create one or more new nodes or update an existing node.
func (t *Trie) insert(parent *Node, key, value []byte,
	deletedMerkleValues map[string]struct{}) (newParent *Node,
	mutated bool, nodesCreated uint32, err error) {
	if parent == nil {
		mutated = true
		nodesCreated = 1
		return &Node{
			Key:        key,
			SubValue:   value,
			Generation: t.generation,
			Dirty:      true,
		}, mutated, nodesCreated, nil
	}

	// TODO ensure all values have dirty set to true

	if parent.Kind() == node.Branch {
		newParent, mutated, nodesCreated, err = t.insertInBranch(
			parent, key, value, deletedMerkleValues)
		if err != nil {
			// `insertInBranch` may call `insert` so do not wrap the
			// error since this may be a deep recursive call.
			return nil, false, 0, err
		}
		return newParent, mutated, nodesCreated, nil
	}

	newParent, mutated, nodesCreated, err = t.insertInLeaf(
		parent, key, value, deletedMerkleValues)
	if err != nil {
		return nil, false, 0, fmt.Errorf("inserting in leaf: %w", err)
	}

	return newParent, mutated, nodesCreated, nil
}

func (t *Trie) insertInLeaf(parentLeaf *Node, key, value []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, mutated bool, nodesCreated uint32, err error) {
	if bytes.Equal(parentLeaf.Key, key) {
		nodesCreated = 0
		if parentLeaf.SubValueEqual(value) {
			mutated = false
			return parentLeaf, mutated, nodesCreated, nil
		}

		copySettings := node.DefaultCopySettings
		copySettings.CopyValue = false
		parentLeaf, err = t.prepForMutation(parentLeaf, copySettings, deletedMerkleValues)
		if err != nil {
			return nil, false, 0, fmt.Errorf("preparing leaf for mutation: %w", err)
		}

		parentLeaf.SubValue = value
		mutated = true
		return parentLeaf, mutated, nodesCreated, nil
	}

	commonPrefixLength := lenCommonPrefix(key, parentLeaf.Key)

	// Convert the current leaf parent into a branch parent
	mutated = true
	newBranchParent := &Node{
		Key:        key[:commonPrefixLength],
		Generation: t.generation,
		Children:   make([]*node.Node, node.ChildrenCapacity),
		Dirty:      true,
	}
	parentLeafKey := parentLeaf.Key

	if len(key) == commonPrefixLength {
		// key is included in parent leaf key
		newBranchParent.SubValue = value

		if len(key) < len(parentLeafKey) {
			// Move the current leaf parent as a child to the new branch.
			copySettings := node.DefaultCopySettings
			childIndex := parentLeafKey[commonPrefixLength]
			newParentLeafKey := parentLeaf.Key[commonPrefixLength+1:]
			if !bytes.Equal(parentLeaf.Key, newParentLeafKey) {
				parentLeaf, err = t.prepForMutation(parentLeaf, copySettings, deletedMerkleValues)
				if err != nil {
					return nil, false, 0, fmt.Errorf("preparing leaf for mutation: %w", err)
				}
				parentLeaf.Key = newParentLeafKey
			}
			newBranchParent.Children[childIndex] = parentLeaf
			newBranchParent.Descendants++
			nodesCreated++
		}

		return newBranchParent, mutated, nodesCreated, nil
	}

	if len(parentLeaf.Key) == commonPrefixLength {
		// the key of the parent leaf is at this new branch
		newBranchParent.SubValue = parentLeaf.SubValue
	} else {
		// make the leaf a child of the new branch
		copySettings := node.DefaultCopySettings
		childIndex := parentLeafKey[commonPrefixLength]
		newParentLeafKey := parentLeaf.Key[commonPrefixLength+1:]
		if !bytes.Equal(parentLeaf.Key, newParentLeafKey) {
			parentLeaf, err = t.prepForMutation(parentLeaf, copySettings, deletedMerkleValues)
			if err != nil {
				return nil, false, 0, fmt.Errorf("preparing leaf for mutation: %w", err)
			}
			parentLeaf.Key = newParentLeafKey
		}
		newBranchParent.Children[childIndex] = parentLeaf
		newBranchParent.Descendants++
		nodesCreated++
	}
	childIndex := key[commonPrefixLength]
	newBranchParent.Children[childIndex] = &Node{
		Key:        key[commonPrefixLength+1:],
		SubValue:   value,
		Generation: t.generation,
		Dirty:      true,
	}
	newBranchParent.Descendants++
	nodesCreated++

	return newBranchParent, mutated, nodesCreated, nil
}

func (t *Trie) insertInBranch(parentBranch *Node, key, value []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, mutated bool, nodesCreated uint32, err error) {
	copySettings := node.DefaultCopySettings

	if bytes.Equal(key, parentBranch.Key) {
		if parentBranch.SubValueEqual(value) {
			mutated = false
			return parentBranch, mutated, 0, nil
		}
		parentBranch, err = t.prepForMutation(parentBranch, copySettings, deletedMerkleValues)
		if err != nil {
			return nil, false, 0, fmt.Errorf("preparing branch for mutation: %w", err)
		}
		parentBranch.SubValue = value
		mutated = true
		return parentBranch, mutated, 0, nil
	}

	if bytes.HasPrefix(key, parentBranch.Key) {
		// key is included in parent branch key
		commonPrefixLength := lenCommonPrefix(key, parentBranch.Key)
		childIndex := key[commonPrefixLength]
		remainingKey := key[commonPrefixLength+1:]
		child := parentBranch.Children[childIndex]

		if child == nil {
			child = &Node{
				Key:        remainingKey,
				SubValue:   value,
				Generation: t.generation,
				Dirty:      true,
			}
			nodesCreated = 1
			parentBranch, err = t.prepForMutation(parentBranch, copySettings, deletedMerkleValues)
			if err != nil {
				return nil, false, 0, fmt.Errorf("preparing branch for mutation: %w", err)
			}
			parentBranch.Children[childIndex] = child
			parentBranch.Descendants += nodesCreated
			mutated = true
			return parentBranch, mutated, nodesCreated, nil
		}

		child, mutated, nodesCreated, err = t.insert(child, remainingKey, value, deletedMerkleValues)
		if err != nil {
			// do not wrap error since `insert` may call `insertInBranch` recursively
			return nil, false, 0, err
		} else if !mutated {
			return parentBranch, mutated, 0, nil
		}

		parentBranch, err = t.prepForMutation(parentBranch, copySettings, deletedMerkleValues)
		if err != nil {
			return nil, false, 0, fmt.Errorf("preparing branch for mutation: %w", err)
		}

		parentBranch.Children[childIndex] = child
		parentBranch.Descendants += nodesCreated
		return parentBranch, mutated, nodesCreated, nil
	}

	// we need to branch out at the point where the keys diverge
	// update partial keys, new branch has key up to matching length
	mutated = true
	nodesCreated = 1
	commonPrefixLength := lenCommonPrefix(key, parentBranch.Key)
	newParentBranch := &Node{
		Key:        key[:commonPrefixLength],
		Generation: t.generation,
		Children:   make([]*node.Node, node.ChildrenCapacity),
		Dirty:      true,
	}

	oldParentIndex := parentBranch.Key[commonPrefixLength]
	remainingOldParentKey := parentBranch.Key[commonPrefixLength+1:]

	// Note: parentBranch.Key != remainingOldParentKey
	parentBranch, err = t.prepForMutation(parentBranch, copySettings, deletedMerkleValues)
	if err != nil {
		return nil, false, 0, fmt.Errorf("preparing branch for mutation: %w", err)
	}

	parentBranch.Key = remainingOldParentKey
	newParentBranch.Children[oldParentIndex] = parentBranch
	newParentBranch.Descendants += 1 + parentBranch.Descendants

	if len(key) <= commonPrefixLength {
		newParentBranch.SubValue = value
	} else {
		childIndex := key[commonPrefixLength]
		remainingKey := key[commonPrefixLength+1:]
		var additionalNodesCreated uint32
		newParentBranch.Children[childIndex], _, additionalNodesCreated, err = t.insert(
			nil, remainingKey, value, deletedMerkleValues)
		if err != nil {
			// do not wrap error since `insert` may call `insertInBranch` recursively
			return nil, false, 0, err
		}

		nodesCreated += additionalNodesCreated
		newParentBranch.Descendants += additionalNodesCreated
	}

	return newParentBranch, mutated, nodesCreated, nil
}

// LoadFromMap loads the given data mapping of key to value into a new empty trie.
// The keys are in hexadecimal little Endian encoding and the values
// are hexadecimal encoded.
func LoadFromMap(data map[string]string) (trie Trie, err error) {
	trie = *NewEmptyTrie()

	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		trie.handleTrackedDeltas(err == nil, pendingDeletedMerkleValues)
	}()

	for key, value := range data {
		keyLEBytes, err := common.HexToBytes(key)
		if err != nil {
			return Trie{}, fmt.Errorf("cannot convert key hex to bytes: %w", err)
		}

		valueBytes, err := common.HexToBytes(value)
		if err != nil {
			return Trie{}, fmt.Errorf("cannot convert value hex to bytes: %w", err)
		}

		err = trie.insertKeyLE(keyLEBytes, valueBytes, pendingDeletedMerkleValues)
		if err != nil {
			return Trie{}, fmt.Errorf("inserting key value pair in trie: %w", err)
		}
	}

	return trie, nil
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
func getKeysWithPrefix(parent *Node, prefix, key []byte,
	keysLE [][]byte) (newKeysLE [][]byte) {
	if parent == nil {
		return keysLE
	}

	if parent.Kind() == node.Leaf {
		return getKeysWithPrefixFromLeaf(parent, prefix, key, keysLE)
	}

	return getKeysWithPrefixFromBranch(parent, prefix, key, keysLE)
}

func getKeysWithPrefixFromLeaf(parent *Node, prefix, key []byte,
	keysLE [][]byte) (newKeysLE [][]byte) {
	if len(key) == 0 || bytes.HasPrefix(parent.Key, key) {
		fullKeyLE := makeFullKeyLE(prefix, parent.Key)
		keysLE = append(keysLE, fullKeyLE)
	}
	return keysLE
}

func getKeysWithPrefixFromBranch(parent *Node, prefix, key []byte,
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
func addAllKeys(parent *Node, prefix []byte, keysLE [][]byte) (newKeysLE [][]byte) {
	if parent == nil {
		return keysLE
	}

	if parent.Kind() == node.Leaf {
		keyLE := makeFullKeyLE(prefix, parent.Key)
		keysLE = append(keysLE, keyLE)
		return keysLE
	}

	if parent.SubValue != nil {
		keyLE := makeFullKeyLE(prefix, parent.Key)
		keysLE = append(keysLE, keyLE)
	}

	for i, child := range parent.Children {
		childPrefix := makeChildPrefix(prefix, parent.Key, i)
		keysLE = addAllKeys(child, childPrefix, keysLE)
	}

	return keysLE
}

func makeFullKeyLE(prefix, nodeKey []byte) (fullKeyLE []byte) {
	fullKey := concatenateSlices(prefix, nodeKey)
	fullKeyLE = codec.NibblesToKeyLE(fullKey)
	return fullKeyLE
}

func makeChildPrefix(branchPrefix, branchKey []byte,
	childIndex int) (childPrefix []byte) {
	childPrefix = concatenateSlices(branchPrefix, branchKey, intToByteSlice(childIndex))
	return childPrefix
}

// Get returns the value in the node of the trie
// which matches its key with the key given.
// Note the key argument is given in little Endian format.
func (t *Trie) Get(keyLE []byte) (value []byte) {
	keyNibbles := codec.KeyLEToNibbles(keyLE)
	return retrieve(t.root, keyNibbles)
}

func retrieve(parent *Node, key []byte) (value []byte) {
	if parent == nil {
		return nil
	}

	if parent.Kind() == node.Leaf {
		return retrieveFromLeaf(parent, key)
	}
	return retrieveFromBranch(parent, key)
}

func retrieveFromLeaf(leaf *Node, key []byte) (value []byte) {
	if bytes.Equal(leaf.Key, key) {
		return leaf.SubValue
	}
	return nil
}

func retrieveFromBranch(branch *Node, key []byte) (value []byte) {
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		return branch.SubValue
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
func (t *Trie) ClearPrefixLimit(prefixLE []byte, limit uint32) (
	deleted uint32, allDeleted bool, err error) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()

	if limit == 0 {
		return 0, false, nil
	}

	prefix := codec.KeyLEToNibbles(prefixLE)
	prefix = bytes.TrimSuffix(prefix, []byte{0})

	root, deleted, _, allDeleted, err := t.clearPrefixLimitAtNode(
		t.root, prefix, limit, pendingDeletedMerkleValues)
	if err != nil {
		// Note: no need to wrap the error really since the private function has
		// the same name as the exported function `ClearPrefixLimit`.
		return 0, false, err
	}
	t.root = root

	return deleted, allDeleted, nil
}

// clearPrefixLimitAtNode deletes the keys having the prefix until the value deletion limit is reached.
// It returns the updated node newParent, the number of deleted values valuesDeleted and the
// allDeleted boolean indicating if there is no key left with the prefix.
func (t *Trie) clearPrefixLimitAtNode(parent *Node, prefix []byte,
	limit uint32, deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, allDeleted bool, err error) {
	if parent == nil {
		return nil, 0, 0, true, nil
	}

	if parent.Kind() == node.Leaf {
		// if prefix is not found, it's also all deleted.
		// TODO check this is the same behaviour as in substrate
		const allDeleted = true
		if bytes.HasPrefix(parent.Key, prefix) {
			isRoot := parent == t.root
			err = registerDeletedMerkleValue(parent, isRoot, deletedMerkleValues)
			if err != nil {
				return nil, 0, 0, false,
					fmt.Errorf("registering deleted Merkle value: %w", err)
			}

			valuesDeleted, nodesRemoved = 1, 1
			return nil, valuesDeleted, nodesRemoved, allDeleted, nil
		}
		return parent, 0, 0, allDeleted, nil
	}

	// Note: `clearPrefixLimitBranch` may call `clearPrefixLimitAtNode` so do not wrap
	// the error since that could be a deep recursive call.
	return t.clearPrefixLimitBranch(parent, prefix, limit, deletedMerkleValues)
}

func (t *Trie) clearPrefixLimitBranch(branch *Node, prefix []byte, limit uint32,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, allDeleted bool, err error) {
	newParent = branch

	if bytes.HasPrefix(branch.Key, prefix) {
		newParent, valuesDeleted, nodesRemoved, err = t.deleteNodesLimit(
			branch, limit, deletedMerkleValues)
		if err != nil {
			return nil, 0, 0, false, fmt.Errorf("deleting nodes: %w", err)
		}
		allDeleted = newParent == nil
		return newParent, valuesDeleted, nodesRemoved, allDeleted, nil
	}

	if len(prefix) == len(branch.Key)+1 &&
		bytes.HasPrefix(branch.Key, prefix[:len(prefix)-1]) {
		// Prefix is one the children of the branch
		return t.clearPrefixLimitChild(branch, prefix, limit, deletedMerkleValues)
	}

	noPrefixForNode := len(prefix) <= len(branch.Key) ||
		lenCommonPrefix(branch.Key, prefix) < len(branch.Key)
	if noPrefixForNode {
		valuesDeleted, nodesRemoved = 0, 0
		allDeleted = true
		return newParent, valuesDeleted, nodesRemoved, allDeleted, nil
	}

	childIndex := prefix[len(branch.Key)]
	childPrefix := prefix[len(branch.Key)+1:]
	child := branch.Children[childIndex]

	child, valuesDeleted, nodesRemoved, allDeleted, err = t.clearPrefixLimitAtNode(
		child, childPrefix, limit, deletedMerkleValues)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("clearing prefix limit at node: %w", err)
	} else if valuesDeleted == 0 {
		return branch, valuesDeleted, nodesRemoved, allDeleted, nil
	}

	copySettings := node.DefaultCopySettings
	branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("preparing branch for mutation: %w", err)
	}

	branch.Children[childIndex] = child
	branch.Descendants -= nodesRemoved
	newParent, branchChildMerged, err := handleDeletion(branch, prefix, deletedMerkleValues)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("handling deletion: %w", err)
	}

	if branchChildMerged {
		nodesRemoved++
	}

	return newParent, valuesDeleted, nodesRemoved, allDeleted, nil
}

func (t *Trie) clearPrefixLimitChild(branch *Node, prefix []byte, limit uint32,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, allDeleted bool, err error) {
	newParent = branch

	childIndex := prefix[len(branch.Key)]
	child := branch.Children[childIndex]

	if child == nil {
		const valuesDeleted, nodesRemoved = 0, 0
		// TODO ensure this is the same behaviour as in substrate
		allDeleted = true
		return newParent, valuesDeleted, nodesRemoved, allDeleted, nil
	}

	child, valuesDeleted, nodesRemoved, err = t.deleteNodesLimit(
		child, limit, deletedMerkleValues)
	if err != nil {
		// Note: do not wrap error since this is recursive.
		return nil, 0, 0, false, err
	}

	if valuesDeleted == 0 {
		allDeleted = branch.Children[childIndex] == nil
		return branch, valuesDeleted, nodesRemoved, allDeleted, nil
	}

	copySettings := node.DefaultCopySettings
	branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("preparing branch for mutation: %w", err)
	}

	branch.Children[childIndex] = child
	branch.Descendants -= nodesRemoved

	newParent, branchChildMerged, err := handleDeletion(branch, prefix, deletedMerkleValues)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("handling deletion: %w", err)
	}

	if branchChildMerged {
		nodesRemoved++
	}

	allDeleted = branch.Children[childIndex] == nil
	return newParent, valuesDeleted, nodesRemoved, allDeleted, nil
}

func (t *Trie) deleteNodesLimit(parent *Node, limit uint32,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, err error) {
	if limit == 0 {
		valuesDeleted, nodesRemoved = 0, 0
		return parent, valuesDeleted, nodesRemoved, nil
	}

	if parent == nil {
		valuesDeleted, nodesRemoved = 0, 0
		return nil, valuesDeleted, nodesRemoved, nil
	}

	if parent.Kind() == node.Leaf {
		isRoot := parent == t.root
		err = registerDeletedMerkleValue(parent, isRoot, deletedMerkleValues)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("registering deleted merkle value: %w", err)
		}
		valuesDeleted, nodesRemoved = 1, 1
		return nil, valuesDeleted, nodesRemoved, nil
	}

	branch := parent

	nilChildren := node.ChildrenCapacity - branch.NumChildren()
	if nilChildren == node.ChildrenCapacity {
		panic("got branch with all nil children")
	}

	// Note: there is at least one non-nil child and the limit isn't zero,
	// therefore it is safe to prepare the branch for mutation.
	copySettings := node.DefaultCopySettings
	branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("preparing branch for mutation: %w", err)
	}

	var newDeleted, newNodesRemoved uint32
	var branchChildMerged bool
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		branch.Children[i], newDeleted, newNodesRemoved, err = t.deleteNodesLimit(
			child, limit, deletedMerkleValues)
		if err != nil {
			// `deleteNodesLimit` is recursive, so do not wrap error.
			return nil, 0, 0, err
		}

		if branch.Children[i] == nil {
			nilChildren++
		}
		limit -= newDeleted
		valuesDeleted += newDeleted
		nodesRemoved += newNodesRemoved
		branch.Descendants -= newNodesRemoved

		newParent, branchChildMerged, err = handleDeletion(branch, branch.Key, deletedMerkleValues)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("handling deletion: %w", err)
		}

		if branchChildMerged {
			nodesRemoved++
		}

		if nilChildren == node.ChildrenCapacity &&
			branch.SubValue == nil {
			return nil, valuesDeleted, nodesRemoved, nil
		}

		if limit == 0 {
			return newParent, valuesDeleted, nodesRemoved, nil
		}
	}

	nodesRemoved++
	if branch.SubValue != nil {
		valuesDeleted++
	}

	return nil, valuesDeleted, nodesRemoved, nil
}

// ClearPrefix deletes all nodes in the trie for which the key contains the
// prefix given in little Endian format.
func (t *Trie) ClearPrefix(prefixLE []byte) (err error) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()

	if len(prefixLE) == 0 {
		const isRoot = true
		err = ensureMerkleValueIsCalculated(t.root, isRoot)
		if err != nil {
			return fmt.Errorf("ensuring Merkle values are calculated: %w", err)
		}

		PopulateNodeHashes(t.root, pendingDeletedMerkleValues)
		t.root = nil
		return nil
	}

	prefix := codec.KeyLEToNibbles(prefixLE)
	prefix = bytes.TrimSuffix(prefix, []byte{0})

	root, _, err := t.clearPrefixAtNode(t.root, prefix, pendingDeletedMerkleValues)
	if err != nil {
		return fmt.Errorf("clearing prefix at root node: %w", err)
	}
	t.root = root

	return nil
}

func (t *Trie) clearPrefixAtNode(parent *Node, prefix []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, nodesRemoved uint32, err error) {
	if parent == nil {
		const nodesRemoved = 0
		return nil, nodesRemoved, nil
	}

	if bytes.HasPrefix(parent.Key, prefix) {
		isRoot := parent == t.root
		err = ensureMerkleValueIsCalculated(parent, isRoot)
		if err != nil {
			nodesRemoved = 0
			return parent, nodesRemoved, fmt.Errorf("ensuring Merkle values are calculated: %w", err)
		}

		PopulateNodeHashes(parent, deletedMerkleValues)
		nodesRemoved = 1 + parent.Descendants
		return nil, nodesRemoved, nil
	}

	if parent.Kind() == node.Leaf {
		const nodesRemoved = 0
		return parent, nodesRemoved, nil
	}

	branch := parent
	if len(prefix) == len(branch.Key)+1 &&
		bytes.HasPrefix(branch.Key, prefix[:len(prefix)-1]) {
		// Prefix is one of the children of the branch
		childIndex := prefix[len(branch.Key)]
		child := branch.Children[childIndex]

		if child == nil {
			const nodesRemoved = 0
			return parent, nodesRemoved, nil
		}

		nodesRemoved = 1 + child.Descendants
		copySettings := node.DefaultCopySettings
		branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
		if err != nil {
			return nil, 0, fmt.Errorf("preparing branch for mutation: %w", err)
		}

		const isRoot = false // child so it cannot be the root
		err = registerDeletedMerkleValue(child, isRoot, deletedMerkleValues)
		if err != nil {
			return nil, 0, fmt.Errorf("registering deleted merkle value for child: %w", err)
		}

		branch.Children[childIndex] = nil
		branch.Descendants -= nodesRemoved
		var branchChildMerged bool
		newParent, branchChildMerged, err = handleDeletion(branch, prefix, deletedMerkleValues)
		if err != nil {
			return nil, 0, fmt.Errorf("handling deletion: %w", err)
		}

		if branchChildMerged {
			nodesRemoved++
		}
		return newParent, nodesRemoved, nil
	}

	noPrefixForNode := len(prefix) <= len(branch.Key) ||
		lenCommonPrefix(branch.Key, prefix) < len(branch.Key)
	if noPrefixForNode {
		const nodesRemoved = 0
		return parent, nodesRemoved, nil
	}

	childIndex := prefix[len(branch.Key)]
	childPrefix := prefix[len(branch.Key)+1:]
	child := branch.Children[childIndex]

	child, nodesRemoved, err = t.clearPrefixAtNode(child, childPrefix, deletedMerkleValues)
	if err != nil {
		nodesRemoved = 0
		// Note: do not wrap error since this is recursive
		return parent, nodesRemoved, err
	} else if nodesRemoved == 0 {
		return parent, nodesRemoved, nil
	}

	copySettings := node.DefaultCopySettings
	branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
	if err != nil {
		return nil, 0, fmt.Errorf("preparing branch for mutation: %w", err)
	}

	branch.Descendants -= nodesRemoved
	branch.Children[childIndex] = child
	newParent, branchChildMerged, err := handleDeletion(branch, prefix, deletedMerkleValues)
	if err != nil {
		return nil, 0, fmt.Errorf("handling deletion: %w", err)
	}

	if branchChildMerged {
		nodesRemoved++
	}

	return newParent, nodesRemoved, nil
}

// Delete removes the node of the trie with the key
// matching the key given in little Endian format.
// If no node is found at this key, nothing is deleted.
func (t *Trie) Delete(keyLE []byte) (err error) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()

	key := codec.KeyLEToNibbles(keyLE)
	root, _, _, err := t.deleteAtNode(t.root, key, pendingDeletedMerkleValues)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

func (t *Trie) deleteAtNode(parent *Node, key []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, deleted bool, nodesRemoved uint32, err error) {
	if parent == nil {
		const nodesRemoved = 0
		return nil, false, nodesRemoved, nil
	}

	if parent.Kind() == node.Leaf {
		newParent, err = t.deleteLeaf(parent, key, deletedMerkleValues)
		if err != nil {
			return nil, false, 0, fmt.Errorf("deleting leaf: %w", err)
		}

		if newParent == nil {
			const nodesRemoved = 1
			return nil, true, nodesRemoved, nil
		}
		const nodesRemoved = 0
		return parent, false, nodesRemoved, nil
	}

	newParent, deleted, nodesRemoved, err = t.deleteBranch(parent, key, deletedMerkleValues)
	if err != nil {
		return nil, false, 0, fmt.Errorf("deleting branch: %w", err)
	}

	return newParent, deleted, nodesRemoved, nil
}

func (t *Trie) deleteLeaf(parent *Node, key []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, err error) {
	if len(key) > 0 && !bytes.Equal(key, parent.Key) {
		return parent, nil
	}

	newParent = nil

	isRoot := parent == t.root
	err = registerDeletedMerkleValue(parent, isRoot, deletedMerkleValues)
	if err != nil {
		return nil, fmt.Errorf("registering deleted merkle value: %w", err)
	}

	return newParent, nil
}

func (t *Trie) deleteBranch(branch *Node, key []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, deleted bool, nodesRemoved uint32, err error) {
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		copySettings := node.DefaultCopySettings
		copySettings.CopyValue = false
		branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
		if err != nil {
			return nil, false, 0, fmt.Errorf("preparing branch for mutation: %w", err)
		}

		// we need to set to nil if the branch has the same generation
		// as the current trie.
		branch.SubValue = nil
		deleted = true
		var branchChildMerged bool
		newParent, branchChildMerged, err = handleDeletion(branch, key, deletedMerkleValues)
		if err != nil {
			return nil, false, 0, fmt.Errorf("handling deletion: %w", err)
		}

		if branchChildMerged {
			nodesRemoved = 1
		}
		return newParent, deleted, nodesRemoved, nil
	}

	commonPrefixLength := lenCommonPrefix(branch.Key, key)
	keyDoesNotExist := commonPrefixLength == len(key)
	if keyDoesNotExist {
		return branch, false, 0, nil
	}
	childIndex := key[commonPrefixLength]
	childKey := key[commonPrefixLength+1:]
	child := branch.Children[childIndex]

	newChild, deleted, nodesRemoved, err := t.deleteAtNode(child, childKey, deletedMerkleValues)
	if err != nil {
		// deleteAtNode may call deleteBranch so don't wrap the error
		// since this may be a recursive call.
		return nil, false, 0, err
	}

	if !deleted {
		const nodesRemoved = 0
		return branch, false, nodesRemoved, nil
	}

	copySettings := node.DefaultCopySettings
	branch, err = t.prepForMutation(branch, copySettings, deletedMerkleValues)
	if err != nil {
		return nil, false, 0, fmt.Errorf("preparing branch for mutation: %w", err)
	}

	branch.Descendants -= nodesRemoved
	branch.Children[childIndex] = newChild

	newParent, branchChildMerged, err := handleDeletion(branch, key, deletedMerkleValues)
	if err != nil {
		return nil, false, 0, fmt.Errorf("handling deletion: %w", err)
	}

	if branchChildMerged {
		nodesRemoved++
	}

	return newParent, true, nodesRemoved, nil
}

// handleDeletion is called when a value is deleted from a branch to handle
// the eventual mutation of the branch depending on its children.
// If the branch has no value and a single child, it will be combined with this child.
// In this first case, branchChildMerged is returned as true to keep track of the removal
// of one node in callers.
// If the branch has a value and no child, it will be changed into a leaf.
func handleDeletion(branch *Node, key []byte,
	deletedMerkleValues map[string]struct{}) (
	newNode *Node, branchChildMerged bool, err error) {
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
		const branchChildMerged = false
		return branch, branchChildMerged, nil
	case childrenCount == 0 && branch.SubValue != nil:
		// The branch passed to handleDeletion is always a modified branch
		// so the original branch Merkle value is already tracked in the deleted
		// Merkle values map.
		const branchChildMerged = false
		commonPrefixLength := lenCommonPrefix(branch.Key, key)
		return &Node{
			Key:        key[:commonPrefixLength],
			SubValue:   branch.SubValue,
			Dirty:      true,
			Generation: branch.Generation,
		}, branchChildMerged, nil
	case childrenCount == 1 && branch.SubValue == nil:
		// The branch passed to handleDeletion is always a modified branch
		// so the original branch Merkle value is already tracked in the deleted
		// Merkle values map.
		const branchChildMerged = true
		childIndex := firstChildIndex
		child := branch.Children[firstChildIndex]
		const isRoot = false // child so it cannot be the root node
		err = registerDeletedMerkleValue(child, isRoot, deletedMerkleValues)
		if err != nil {
			return nil, false, fmt.Errorf("registering deleted merkle value: %w", err)
		}

		if child.Kind() == node.Leaf {
			newLeafKey := concatenateSlices(branch.Key, intToByteSlice(childIndex), child.Key)
			return &Node{
				Key:        newLeafKey,
				SubValue:   child.SubValue,
				Dirty:      true,
				Generation: branch.Generation,
			}, branchChildMerged, nil
		}

		childBranch := child
		newBranchKey := concatenateSlices(branch.Key, intToByteSlice(childIndex), childBranch.Key)
		newBranch := &Node{
			Key:        newBranchKey,
			SubValue:   childBranch.SubValue,
			Generation: branch.Generation,
			Children:   make([]*node.Node, node.ChildrenCapacity),
			Dirty:      true,
			// this is the descendants of the original branch minus one
			Descendants: childBranch.Descendants,
		}

		// Adopt the grand-children
		for i, grandChild := range childBranch.Children {
			if grandChild != nil {
				newBranch.Children[i] = grandChild
				// No need to copy and update the generation
				// of the grand children since they are not modified.
			}
		}

		return newBranch, branchChildMerged, nil
	}
}

// ensureMerkleValueIsCalculated is used before calling PopulateMerkleValues
// to ensure the parent node and all its descendant nodes have their Merkle
// value computed and ready to be used. This has a close to zero performance
// impact if the parent node Merkle value is already computed.
func ensureMerkleValueIsCalculated(parent *Node, isRoot bool) (err error) {
	if parent == nil {
		return nil
	}

	if isRoot {
		_, err = parent.CalculateRootMerkleValue()
		if err != nil {
			return fmt.Errorf("calculating Merkle value of root node: %w", err)
		}
	} else {
		_, err = parent.CalculateMerkleValue()
		if err != nil {
			return fmt.Errorf("calculating Merkle value of node: %w", err)
		}
	}

	return nil
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

func concatenateSlices(sliceOne, sliceTwo []byte, otherSlices ...[]byte) (concatenated []byte) {
	allNil := sliceOne == nil && sliceTwo == nil
	totalLength := len(sliceOne) + len(sliceTwo)

	for _, otherSlice := range otherSlices {
		allNil = allNil && otherSlice == nil
		totalLength += len(otherSlice)
	}

	if allNil {
		// Return a nil slice instead of an an empty slice
		// if all slices are nil.
		return nil
	}

	concatenated = make([]byte, 0, totalLength)

	concatenated = append(concatenated, sliceOne...)
	concatenated = append(concatenated, sliceTwo...)
	for _, otherSlice := range otherSlices {
		concatenated = append(concatenated, otherSlice...)
	}

	return concatenated
}

func intToByteSlice(n int) (slice []byte) {
	return []byte{byte(n)}
}
