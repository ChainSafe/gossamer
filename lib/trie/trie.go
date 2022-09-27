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

func (t *Trie) prepLeafForMutation(currentLeaf *Node,
	copySettings node.CopySettings,
	pendingDeletedMerkleValues map[string]struct{}) (newLeaf *Node) {
	if currentLeaf.Generation == t.generation {
		// no need to deep copy and update generation
		// of current leaf.
		newLeaf = currentLeaf
	} else {
		newLeaf = updateGeneration(currentLeaf, t.generation, pendingDeletedMerkleValues, copySettings)
	}
	newLeaf.SetDirty()
	return newLeaf
}

func (t *Trie) prepBranchForMutation(currentBranch *Node,
	copySettings node.CopySettings,
	pendingDeletedMerkleValues map[string]struct{}) (newBranch *Node) {
	if currentBranch.Generation == t.generation {
		// no need to deep copy and update generation
		// of current branch.
		newBranch = currentBranch
	} else {
		newBranch = updateGeneration(currentBranch, t.generation, pendingDeletedMerkleValues, copySettings)
	}
	newBranch.SetDirty()
	return newBranch
}

// updateGeneration is called when the currentNode is from
// an older trie generation (snapshot) so we deep copy the
// node and update the generation on the newer copy.
func updateGeneration(currentNode *Node, trieGeneration uint64,
	deletedMerkleValues map[string]struct{}, copySettings node.CopySettings) (
	newNode *Node) {
	newNode = currentNode.Copy(copySettings)
	newNode.Generation = trieGeneration

	// The hash of the node from a previous snapshotted trie
	// is usually already computed.
	deletedMerkleValue := currentNode.MerkleValue
	if len(deletedMerkleValue) > 0 {
		deletedMerkleValueString := string(deletedMerkleValue)
		deletedMerkleValues[deletedMerkleValueString] = struct{}{}
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
func (t *Trie) Put(keyLE, value []byte) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()
	t.insertKeyLE(keyLE, value, pendingDeletedMerkleValues)
}

func (t *Trie) insertKeyLE(keyLE, value []byte, deletedMerkleValues map[string]struct{}) {
	nibblesKey := codec.KeyLEToNibbles(keyLE)
	t.root, _, _ = t.insert(t.root, nibblesKey, value, deletedMerkleValues)
}

// insert inserts a value in the trie at the key specified.
// It may create one or more new nodes or update an existing node.
func (t *Trie) insert(parent *Node, key, value []byte,
	deletedMerkleValues map[string]struct{}) (newParent *Node,
	mutated bool, nodesCreated uint32) {
	if parent == nil {
		mutated = true
		nodesCreated = 1
		return &Node{
			Key:        key,
			SubValue:   value,
			Generation: t.generation,
			Dirty:      true,
		}, mutated, nodesCreated
	}

	// TODO ensure all values have dirty set to true

	if parent.Kind() == node.Branch {
		return t.insertInBranch(parent, key, value, deletedMerkleValues)
	}
	return t.insertInLeaf(parent, key, value, deletedMerkleValues)
}

func (t *Trie) insertInLeaf(parentLeaf *Node, key, value []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, mutated bool, nodesCreated uint32) {
	if bytes.Equal(parentLeaf.Key, key) {
		nodesCreated = 0
		if parentLeaf.SubValueEqual(value) {
			mutated = false
			return parentLeaf, mutated, nodesCreated
		}

		copySettings := node.DefaultCopySettings
		copySettings.CopyValue = false
		parentLeaf = t.prepLeafForMutation(parentLeaf, copySettings, deletedMerkleValues)
		parentLeaf.SubValue = value
		mutated = true
		return parentLeaf, mutated, nodesCreated
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
				parentLeaf = t.prepLeafForMutation(parentLeaf, copySettings, deletedMerkleValues)
				parentLeaf.Key = newParentLeafKey
			}
			newBranchParent.Children[childIndex] = parentLeaf
			newBranchParent.Descendants++
			nodesCreated++
		}

		return newBranchParent, mutated, nodesCreated
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
			parentLeaf = t.prepLeafForMutation(parentLeaf, copySettings, deletedMerkleValues)
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

	return newBranchParent, mutated, nodesCreated
}

func (t *Trie) insertInBranch(parentBranch *Node, key, value []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, mutated bool, nodesCreated uint32) {
	copySettings := node.DefaultCopySettings

	if bytes.Equal(key, parentBranch.Key) {
		if parentBranch.SubValueEqual(value) {
			mutated = false
			return parentBranch, mutated, 0
		}
		parentBranch = t.prepBranchForMutation(parentBranch, copySettings, deletedMerkleValues)
		parentBranch.SubValue = value
		mutated = true
		return parentBranch, mutated, 0
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
			parentBranch = t.prepBranchForMutation(parentBranch, copySettings, deletedMerkleValues)
			parentBranch.Children[childIndex] = child
			parentBranch.Descendants += nodesCreated
			mutated = true
			return parentBranch, mutated, nodesCreated
		}

		child, mutated, nodesCreated = t.insert(child, remainingKey, value, deletedMerkleValues)
		if !mutated {
			return parentBranch, mutated, 0
		}

		parentBranch = t.prepBranchForMutation(parentBranch, copySettings, deletedMerkleValues)
		parentBranch.Children[childIndex] = child
		parentBranch.Descendants += nodesCreated
		return parentBranch, mutated, nodesCreated
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
	parentBranch = t.prepBranchForMutation(parentBranch, copySettings, deletedMerkleValues)
	parentBranch.Key = remainingOldParentKey
	newParentBranch.Children[oldParentIndex] = parentBranch
	newParentBranch.Descendants += 1 + parentBranch.Descendants

	if len(key) <= commonPrefixLength {
		newParentBranch.SubValue = value
	} else {
		childIndex := key[commonPrefixLength]
		remainingKey := key[commonPrefixLength+1:]
		var additionalNodesCreated uint32
		newParentBranch.Children[childIndex], _, additionalNodesCreated = t.insert(
			nil, remainingKey, value, deletedMerkleValues)
		nodesCreated += additionalNodesCreated
		newParentBranch.Descendants += additionalNodesCreated
	}

	return newParentBranch, mutated, nodesCreated
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

		trie.insertKeyLE(keyLEBytes, valueBytes, pendingDeletedMerkleValues)
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
func (t *Trie) ClearPrefixLimit(prefixLE []byte, limit uint32) (deleted uint32, allDeleted bool) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()

	if limit == 0 {
		return 0, false
	}

	prefix := codec.KeyLEToNibbles(prefixLE)
	prefix = bytes.TrimSuffix(prefix, []byte{0})

	t.root, deleted, _, allDeleted = t.clearPrefixLimitAtNode(
		t.root, prefix, limit, pendingDeletedMerkleValues)
	return deleted, allDeleted
}

// clearPrefixLimitAtNode deletes the keys having the prefix until the value deletion limit is reached.
// It returns the updated node newParent, the number of deleted values valuesDeleted and the
// allDeleted boolean indicating if there is no key left with the prefix.
func (t *Trie) clearPrefixLimitAtNode(parent *Node, prefix []byte,
	limit uint32, deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, allDeleted bool) {
	if parent == nil {
		return nil, 0, 0, true
	}

	if parent.Kind() == node.Leaf {
		// if prefix is not found, it's also all deleted.
		// TODO check this is the same behaviour as in substrate
		const allDeleted = true
		if bytes.HasPrefix(parent.Key, prefix) {
			valuesDeleted, nodesRemoved = 1, 1
			return nil, valuesDeleted, nodesRemoved, allDeleted
		}
		return parent, 0, 0, allDeleted
	}

	return t.clearPrefixLimitBranch(parent, prefix, limit, deletedMerkleValues)
}

func (t *Trie) clearPrefixLimitBranch(branch *Node, prefix []byte, limit uint32,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, allDeleted bool) {
	newParent = branch

	if bytes.HasPrefix(branch.Key, prefix) {
		newParent, valuesDeleted, nodesRemoved = t.deleteNodesLimit(
			branch, limit, deletedMerkleValues)
		allDeleted = newParent == nil
		return newParent, valuesDeleted, nodesRemoved, allDeleted
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
		return newParent, valuesDeleted, nodesRemoved, allDeleted
	}

	childIndex := prefix[len(branch.Key)]
	childPrefix := prefix[len(branch.Key)+1:]
	child := branch.Children[childIndex]

	child, valuesDeleted, nodesRemoved, allDeleted = t.clearPrefixLimitAtNode(
		child, childPrefix, limit, deletedMerkleValues)
	if valuesDeleted == 0 {
		return branch, valuesDeleted, nodesRemoved, allDeleted
	}

	copySettings := node.DefaultCopySettings
	branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)
	branch.Children[childIndex] = child
	branch.Descendants -= nodesRemoved
	newParent, branchChildMerged := handleDeletion(branch, prefix)
	if branchChildMerged {
		nodesRemoved++
	}

	return newParent, valuesDeleted, nodesRemoved, allDeleted
}

func (t *Trie) clearPrefixLimitChild(branch *Node, prefix []byte, limit uint32,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32, allDeleted bool) {
	newParent = branch

	childIndex := prefix[len(branch.Key)]
	child := branch.Children[childIndex]

	if child == nil {
		const valuesDeleted, nodesRemoved = 0, 0
		// TODO ensure this is the same behaviour as in substrate
		allDeleted = true
		return newParent, valuesDeleted, nodesRemoved, allDeleted
	}

	child, valuesDeleted, nodesRemoved = t.deleteNodesLimit(
		child, limit, deletedMerkleValues)
	if valuesDeleted == 0 {
		allDeleted = branch.Children[childIndex] == nil
		return branch, valuesDeleted, nodesRemoved, allDeleted
	}

	copySettings := node.DefaultCopySettings
	branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)
	branch.Children[childIndex] = child
	branch.Descendants -= nodesRemoved

	newParent, branchChildMerged := handleDeletion(branch, prefix)
	if branchChildMerged {
		nodesRemoved++
	}

	allDeleted = branch.Children[childIndex] == nil
	return newParent, valuesDeleted, nodesRemoved, allDeleted
}

func (t *Trie) deleteNodesLimit(parent *Node, limit uint32,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, valuesDeleted, nodesRemoved uint32) {
	if limit == 0 {
		valuesDeleted, nodesRemoved = 0, 0
		return parent, valuesDeleted, nodesRemoved
	}

	if parent == nil {
		valuesDeleted, nodesRemoved = 0, 0
		return nil, valuesDeleted, nodesRemoved
	}

	if parent.Kind() == node.Leaf {
		valuesDeleted, nodesRemoved = 1, 1
		return nil, valuesDeleted, nodesRemoved
	}

	branch := parent

	nilChildren := node.ChildrenCapacity - branch.NumChildren()

	var newDeleted, newNodesRemoved uint32
	var branchChildMerged bool
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		copySettings := node.DefaultCopySettings
		branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)

		branch.Children[i], newDeleted, newNodesRemoved = t.deleteNodesLimit(
			child, limit, deletedMerkleValues)
		// Note: newDeleted can never be zero here since the limit isn't zero
		// and the child is not nil. Therefore it is safe to prepare the branch
		// for mutation right before this call.
		if branch.Children[i] == nil {
			nilChildren++
		}
		limit -= newDeleted
		valuesDeleted += newDeleted
		nodesRemoved += newNodesRemoved
		branch.Descendants -= newNodesRemoved

		branch.SetDirty()

		newParent, branchChildMerged = handleDeletion(branch, branch.Key)
		if branchChildMerged {
			nodesRemoved++
		}

		if nilChildren == node.ChildrenCapacity &&
			branch.SubValue == nil {
			return nil, valuesDeleted, nodesRemoved
		}

		if limit == 0 {
			return newParent, valuesDeleted, nodesRemoved
		}
	}

	nodesRemoved++
	if branch.SubValue != nil {
		valuesDeleted++
	}

	return nil, valuesDeleted, nodesRemoved
}

// ClearPrefix deletes all nodes in the trie for which the key contains the
// prefix given in little Endian format.
func (t *Trie) ClearPrefix(prefixLE []byte) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()

	if len(prefixLE) == 0 {
		t.root = nil
		return
	}

	prefix := codec.KeyLEToNibbles(prefixLE)
	prefix = bytes.TrimSuffix(prefix, []byte{0})

	t.root, _ = t.clearPrefixAtNode(t.root, prefix, pendingDeletedMerkleValues)
}

func (t *Trie) clearPrefixAtNode(parent *Node, prefix []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, nodesRemoved uint32) {
	if parent == nil {
		const nodesRemoved = 0
		return nil, nodesRemoved
	}

	if bytes.HasPrefix(parent.Key, prefix) {
		nodesRemoved = 1 + parent.Descendants
		return nil, nodesRemoved
	}

	if parent.Kind() == node.Leaf {
		const nodesRemoved = 0
		return parent, nodesRemoved
	}

	branch := parent
	if len(prefix) == len(branch.Key)+1 &&
		bytes.HasPrefix(branch.Key, prefix[:len(prefix)-1]) {
		// Prefix is one of the children of the branch
		childIndex := prefix[len(branch.Key)]
		child := branch.Children[childIndex]

		if child == nil {
			const nodesRemoved = 0
			return parent, nodesRemoved
		}

		nodesRemoved = 1 + child.Descendants
		copySettings := node.DefaultCopySettings
		branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)
		branch.Children[childIndex] = nil
		branch.Descendants -= nodesRemoved
		var branchChildMerged bool
		newParent, branchChildMerged = handleDeletion(branch, prefix)
		if branchChildMerged {
			nodesRemoved++
		}
		return newParent, nodesRemoved
	}

	noPrefixForNode := len(prefix) <= len(branch.Key) ||
		lenCommonPrefix(branch.Key, prefix) < len(branch.Key)
	if noPrefixForNode {
		const nodesRemoved = 0
		return parent, nodesRemoved
	}

	childIndex := prefix[len(branch.Key)]
	childPrefix := prefix[len(branch.Key)+1:]
	child := branch.Children[childIndex]

	child, nodesRemoved = t.clearPrefixAtNode(child, childPrefix, deletedMerkleValues)
	if nodesRemoved == 0 {
		return parent, nodesRemoved
	}

	copySettings := node.DefaultCopySettings
	branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)
	branch.Descendants -= nodesRemoved
	branch.Children[childIndex] = child
	newParent, branchChildMerged := handleDeletion(branch, prefix)
	if branchChildMerged {
		nodesRemoved++
	}

	return newParent, nodesRemoved
}

// Delete removes the node of the trie with the key
// matching the key given in little Endian format.
// If no node is found at this key, nothing is deleted.
func (t *Trie) Delete(keyLE []byte) {
	pendingDeletedMerkleValues := make(map[string]struct{})
	defer func() {
		const success = true
		t.handleTrackedDeltas(success, pendingDeletedMerkleValues)
	}()

	key := codec.KeyLEToNibbles(keyLE)
	t.root, _, _ = t.deleteAtNode(t.root, key, pendingDeletedMerkleValues)
}

func (t *Trie) deleteAtNode(parent *Node, key []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, deleted bool, nodesRemoved uint32) {
	if parent == nil {
		const nodesRemoved = 0
		return nil, false, nodesRemoved
	}

	if parent.Kind() == node.Leaf {
		if deleteLeaf(parent, key) == nil {
			const nodesRemoved = 1
			return nil, true, nodesRemoved
		}
		const nodesRemoved = 0
		return parent, false, nodesRemoved
	}
	return t.deleteBranch(parent, key, deletedMerkleValues)
}

func deleteLeaf(parent *Node, key []byte) (newParent *Node) {
	if len(key) == 0 || bytes.Equal(key, parent.Key) {
		return nil
	}
	return parent
}

func (t *Trie) deleteBranch(branch *Node, key []byte,
	deletedMerkleValues map[string]struct{}) (
	newParent *Node, deleted bool, nodesRemoved uint32) {
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		copySettings := node.DefaultCopySettings
		copySettings.CopyValue = false
		branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)
		// we need to set to nil if the branch has the same generation
		// as the current trie.
		branch.SubValue = nil
		deleted = true
		var branchChildMerged bool
		newParent, branchChildMerged = handleDeletion(branch, key)
		if branchChildMerged {
			nodesRemoved = 1
		}
		return newParent, deleted, nodesRemoved
	}

	commonPrefixLength := lenCommonPrefix(branch.Key, key)
	keyDoesNotExist := commonPrefixLength == len(key)
	if keyDoesNotExist {
		return branch, false, 0
	}
	childIndex := key[commonPrefixLength]
	childKey := key[commonPrefixLength+1:]
	child := branch.Children[childIndex]

	newChild, deleted, nodesRemoved := t.deleteAtNode(child, childKey, deletedMerkleValues)
	if !deleted {
		const nodesRemoved = 0
		return branch, false, nodesRemoved
	}

	copySettings := node.DefaultCopySettings
	branch = t.prepBranchForMutation(branch, copySettings, deletedMerkleValues)
	branch.Descendants -= nodesRemoved
	branch.Children[childIndex] = newChild

	newParent, branchChildMerged := handleDeletion(branch, key)
	if branchChildMerged {
		nodesRemoved++
	}

	return newParent, true, nodesRemoved
}

// handleDeletion is called when a value is deleted from a branch to handle
// the eventual mutation of the branch depending on its children.
// If the branch has no value and a single child, it will be combined with this child.
// In this first case, branchChildMerged is returned as true to keep track of the removal
// of one node in callers.
// If the branch has a value and no child, it will be changed into a leaf.
func handleDeletion(branch *Node, key []byte) (newNode *Node, branchChildMerged bool) {
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
		return branch, branchChildMerged
	case childrenCount == 0 && branch.SubValue != nil:
		const branchChildMerged = false
		commonPrefixLength := lenCommonPrefix(branch.Key, key)
		return &Node{
			Key:        key[:commonPrefixLength],
			SubValue:   branch.SubValue,
			Dirty:      true,
			Generation: branch.Generation,
		}, branchChildMerged
	case childrenCount == 1 && branch.SubValue == nil:
		const branchChildMerged = true
		childIndex := firstChildIndex
		child := branch.Children[firstChildIndex]

		if child.Kind() == node.Leaf {
			newLeafKey := concatenateSlices(branch.Key, intToByteSlice(childIndex), child.Key)
			return &Node{
				Key:        newLeafKey,
				SubValue:   child.SubValue,
				Dirty:      true,
				Generation: branch.Generation,
			}, branchChildMerged
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

		return newBranch, branchChildMerged
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
