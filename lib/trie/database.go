// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
)

// DBGetter gets a value corresponding to the given key.
type DBGetter interface {
	Get(key []byte) (value []byte, err error)
}

// DBPutter puts a value at the given key and returns an error.
type DBPutter interface {
	Put(key []byte, value []byte) error
}

// NewBatcher creates a new database batch.
type NewBatcher interface {
	NewBatch() database.Batch
}

// Load reconstructs the trie from the database from the given root hash.
// It is used when restarting the node to load the current state trie.
func (t *Trie) Load(db DBGetter, rootHash common.Hash) error {
	if rootHash == EmptyHash {
		t.root = nil
		return nil
	}
	rootHashBytes := rootHash.ToBytes()

	encodedNode, err := db.Get(rootHashBytes)
	if err != nil {
		return fmt.Errorf("failed to find root key %s: %w", rootHash, err)
	}

	reader := bytes.NewReader(encodedNode)
	root, err := node.Decode(reader)
	if err != nil {
		return fmt.Errorf("cannot decode root node: %w", err)
	}

	t.root = root
	t.root.MerkleValue = rootHashBytes

	return t.loadNode(db, t.root)
}

func (t *Trie) loadNode(db DBGetter, n *Node) error {
	if n.Kind() != node.Branch {
		return nil
	}

	branch := n
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		merkleValue := child.MerkleValue

		if len(merkleValue) < 32 {
			// node has already been loaded inline
			// just set its encoding
			_, err := child.CalculateMerkleValue()
			if err != nil {
				return fmt.Errorf("merkle value: %w", err)
			}
			continue
		}

		nodeHash := merkleValue
		encodedNode, err := db.Get(nodeHash)
		if err != nil {
			return fmt.Errorf("cannot find child node key 0x%x in database: %w", nodeHash, err)
		}

		reader := bytes.NewReader(encodedNode)
		decodedNode, err := node.Decode(reader)
		if err != nil {
			return fmt.Errorf("decoding node with hash 0x%x: %w", nodeHash, err)
		}

		decodedNode.MerkleValue = nodeHash
		branch.Children[i] = decodedNode

		err = t.loadNode(db, decodedNode)
		if err != nil {
			return fmt.Errorf("loading child at index %d with node hash 0x%x: %w", i, nodeHash, err)
		}

		if decodedNode.Kind() == node.Branch {
			// Note 1: the node is fully loaded with all its descendants
			// count only after the database load above.
			// Note 2: direct child node is already counted as descendant
			// when it was read as a leaf with hash only in decodeBranch,
			// so we only add the descendants of the child branch to the
			// current branch.
			childBranchDescendants := decodedNode.Descendants
			branch.Descendants += childBranchDescendants
		}
	}

	for _, key := range t.GetKeysWithPrefix(ChildStorageKeyPrefix) {
		childTrie := NewEmptyTrie()
		value := t.Get(key)
		rootHash := common.BytesToHash(value)
		err := childTrie.Load(db, rootHash)
		if err != nil {
			return fmt.Errorf("failed to load child trie with root hash=%s: %w", rootHash, err)
		}

		hash, err := childTrie.Hash()
		if err != nil {
			return fmt.Errorf("cannot hash chilld trie at key 0x%x: %w", key, err)
		}
		t.childTries[hash] = childTrie
	}

	return nil
}

// PopulateNodeHashes writes the node hash values of the node given and of
// all its descendant nodes as keys to the nodeHashes map.
// It is assumed the node and its descendant nodes have their Merkle value already
// computed.
func PopulateNodeHashes(n *Node, nodeHashes map[common.Hash]struct{}) {
	if n == nil {
		return
	}

	switch {
	case len(n.MerkleValue) == 0:
		// TODO remove once lazy loading of nodes is implemented
		// https://github.com/ChainSafe/gossamer/issues/2838
		panic(fmt.Sprintf("node with partial key 0x%x has no Merkle value computed", n.PartialKey))
	case len(n.MerkleValue) < 32:
		// Inlined node where its Merkle value is its
		// encoding and not the encoding hash digest.
		return
	}

	nodeHash := common.NewHash(n.MerkleValue)
	nodeHashes[nodeHash] = struct{}{}

	if n.Kind() == node.Leaf {
		return
	}

	branch := n
	for _, child := range branch.Children {
		PopulateNodeHashes(child, nodeHashes)
	}
}

// recordAllDeleted records the node hashes of the given node and all its descendants.
// Note it does not record inlined nodes.
// It is assumed the node and its descendant nodes have their Merkle value already
// computed, or the function will panic.
func recordAllDeleted(n *Node, recorder DeltaRecorder) {
	if n == nil {
		return
	}

	if len(n.MerkleValue) == 0 {
		panic(fmt.Sprintf("node with key 0x%x has no Merkle value computed", n.PartialKey))
	}

	isInlined := len(n.MerkleValue) < 32
	if isInlined {
		return
	}

	nodeHash := common.NewHash(n.MerkleValue)
	recorder.RecordDeleted(nodeHash)

	if n.Kind() == node.Leaf {
		return
	}

	branch := n
	for _, child := range branch.Children {
		recordAllDeleted(child, recorder)
	}
}

// GetFromDB retrieves a value at the given key from the trie using the database.
// It recursively descends into the trie using the database starting
// from the root node until it reaches the node with the given key.
// It then reads the value from the database.
func GetFromDB(db DBGetter, rootHash common.Hash, key []byte) (
	value []byte, err error) {
	if rootHash == EmptyHash {
		return nil, nil
	}

	k := codec.KeyLEToNibbles(key)

	encodedRootNode, err := db.Get(rootHash[:])
	if err != nil {
		return nil, fmt.Errorf("cannot find root hash key %s: %w", rootHash, err)
	}

	reader := bytes.NewReader(encodedRootNode)
	rootNode, err := node.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot decode root node: %w", err)
	}

	return getFromDBAtNode(db, rootNode, k)
}

// getFromDBAtNode recursively searches through the trie and database
// for the value corresponding to a key.
// Note it does not copy the value so modifying the value bytes
// slice will modify the value of the node in the trie.
func getFromDBAtNode(db DBGetter, n *Node, key []byte) (
	value []byte, err error) {
	if n.Kind() == node.Leaf {
		if bytes.Equal(n.PartialKey, key) {
			return n.StorageValue, nil
		}
		return nil, nil
	}

	branch := n
	// Key is equal to the key of this branch or is empty
	if len(key) == 0 || bytes.Equal(branch.PartialKey, key) {
		return branch.StorageValue, nil
	}

	commonPrefixLength := lenCommonPrefix(branch.PartialKey, key)
	if len(key) < len(branch.PartialKey) && bytes.Equal(branch.PartialKey[:commonPrefixLength], key) {
		// The key to search is a prefix of the node key and is smaller than the node key.
		// Example: key to search: 0xabcd
		//          branch key:    0xabcdef
		return nil, nil
	}

	// childIndex is the nibble after the common prefix length in the key being searched.
	childIndex := key[commonPrefixLength]
	child := branch.Children[childIndex]
	if child == nil {
		return nil, nil
	}

	// Child can be either inlined or a hash pointer.
	childMerkleValue := child.MerkleValue
	if len(childMerkleValue) == 0 && child.Kind() == node.Leaf {
		return getFromDBAtNode(db, child, key[commonPrefixLength+1:])
	}

	encodedChild, err := db.Get(childMerkleValue)
	if err != nil {
		return nil, fmt.Errorf(
			"finding child node with hash 0x%x in database: %w",
			childMerkleValue, err)
	}

	reader := bytes.NewReader(encodedChild)
	decodedChild, err := node.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"decoding child node with hash 0x%x: %w",
			childMerkleValue, err)
	}

	return getFromDBAtNode(db, decodedChild, key[commonPrefixLength+1:])
	// Note: do not wrap error since it's called recursively.
}

// WriteDirty writes all dirty nodes to the database and sets them to clean
func (t *Trie) WriteDirty(db NewBatcher) error {
	batch := db.NewBatch()
	err := t.writeDirtyNode(batch, t.root)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Flush()
}

func (t *Trie) writeDirtyNode(db DBPutter, n *Node) (err error) {
	if n == nil || !n.Dirty {
		return nil
	}

	var encoding, merkleValue []byte
	if n == t.root {
		encoding, merkleValue, err = n.EncodeAndHashRoot()
	} else {
		encoding, merkleValue, err = n.EncodeAndHash()
	}
	if err != nil {
		return fmt.Errorf(
			"encoding and hashing node with Merkle value 0x%x: %w",
			n.MerkleValue, err)
	}

	if len(merkleValue) < 32 {
		// Merkle value is the node encoding which is less than 32 bytes.
		// That means this node encoding is inlined in its parent node encoding,
		// and so it is not needed to write it in the database.
		n.SetClean()
		return nil
	}

	nodeHash := merkleValue

	err = db.Put(nodeHash, encoding)
	if err != nil {
		return fmt.Errorf(
			"putting encoding of node with node hash 0x%x in database: %w",
			nodeHash, err)
	}

	if n.Kind() != node.Branch {
		n.SetClean()
		return nil
	}

	for _, child := range n.Children {
		if child == nil {
			continue
		}

		err = t.writeDirtyNode(db, child)
		if err != nil {
			// Note: do not wrap error since it's returned recursively.
			return err
		}
	}

	for _, childTrie := range t.childTries {
		if err := childTrie.writeDirtyNode(db, childTrie.root); err != nil {
			return fmt.Errorf("writing dirty node to database: %w", err)
		}
	}

	n.SetClean()

	return nil
}

// GetChangedNodeHashes returns the two sets of hashes for all nodes
// inserted and deleted in the state trie since the last snapshot.
// Returned inserted map is safe for mutation, but deleted is not safe for mutation.
func (t *Trie) GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error) {
	inserted = make(map[common.Hash]struct{})
	err = t.getInsertedNodeHashesAtNode(t.root, inserted)
	if err != nil {
		return nil, nil, fmt.Errorf("getting inserted node hashes: %w", err)
	}

	deleted = t.deltas.Deleted()

	return inserted, deleted, nil
}

func (t *Trie) getInsertedNodeHashesAtNode(n *Node, nodeHashes map[common.Hash]struct{}) (err error) {
	if n == nil || !n.Dirty {
		return nil
	}

	var merkleValue []byte
	if n == t.root {
		merkleValue, err = n.CalculateRootMerkleValue()
	} else {
		merkleValue, err = n.CalculateMerkleValue()
	}
	if err != nil {
		return fmt.Errorf("calculating Merkle value: %w", err)
	}

	if len(merkleValue) < 32 {
		// this is an inlined node and is encoded as part of its parent node.
		// Therefore it is not written to disk and the online pruner does not
		// need to track it. If the node encodes to less than 32B, it cannot have
		// non-inlined children so it's safe to stop here and not recurse further.
		return nil
	}

	nodeHash := common.NewHash(merkleValue)
	nodeHashes[nodeHash] = struct{}{}

	if n.Kind() != node.Branch {
		return nil
	}

	for _, child := range n.Children {
		if child == nil {
			continue
		}

		err := t.getInsertedNodeHashesAtNode(child, nodeHashes)
		if err != nil {
			// Note: do not wrap error since this is called recursively.
			return err
		}
	}

	return nil
}
