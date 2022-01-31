// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/chaindb"
)

var (
	ErrEmptyProof = errors.New("proof slice empty")
	ErrDecodeNode = errors.New("cannot decode node")
)

// Store stores each trie node in the database,
// where the key is the hash of the encoded node
// and the value is the encoded node.
// Generally, this will only be used for the genesis trie.
func (t *Trie) Store(db chaindb.Database) error {
	for _, v := range t.childTries {
		if err := v.Store(db); err != nil {
			return fmt.Errorf("failed to store child trie with root hash=0x%x in the db: %w", v.root.GetHash(), err)
		}
	}

	batch := db.NewBatch()
	err := t.store(batch, t.root)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Flush()
}

func (t *Trie) store(db chaindb.Batch, n Node) error {
	if n == nil {
		return nil
	}

	encoding, hash, err := n.EncodeAndHash()
	if err != nil {
		return err
	}

	err = db.Put(hash, encoding)
	if err != nil {
		return err
	}

	switch n.Type() {
	case node.BranchType, node.BranchWithValueType:
		branch := n.(*node.Branch)
		for _, child := range branch.Children {
			if child == nil {
				continue
			}

			err = t.store(db, child)
			if err != nil {
				return err
			}
		}
	}

	if n.IsDirty() {
		n.SetDirty(false)
	}

	return nil
}

// loadFromProof create a partial trie based on the proof slice, as it only contains nodes that are in the proof afaik.
func (t *Trie) loadFromProof(rawProof [][]byte, rootHash []byte) error {
	if len(rawProof) == 0 {
		return ErrEmptyProof
	}

	proofHashToNode := make(map[string]Node, len(rawProof))

	for i, rawNode := range rawProof {
		decodedNode, err := node.Decode(bytes.NewReader(rawNode))
		if err != nil {
			return fmt.Errorf("%w: at index %d: 0x%x",
				ErrDecodeNode, i, rawNode)
		}

		const dirty = false
		decodedNode.SetDirty(dirty)
		decodedNode.SetEncodingAndHash(rawNode, nil)

		_, hash, err := decodedNode.EncodeAndHash()
		if err != nil {
			return fmt.Errorf("cannot encode and hash node at index %d: %w", i, err)
		}

		proofHash := common.BytesToHex(hash)
		proofHashToNode[proofHash] = decodedNode

		if bytes.Equal(hash, rootHash) {
			// Found root in proof
			t.root = decodedNode
		}
	}

	t.loadProof(proofHashToNode, t.root)

	return nil
}

// loadProof is a recursive function that will create all the trie paths based
// on the mapped proofs slice starting at the root
func (t *Trie) loadProof(proofHashToNode map[string]Node, n Node) {
	switch n.Type() {
	case node.BranchType, node.BranchWithValueType:
	default:
		return
	}

	branch := n.(*node.Branch)

	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		proofHash := common.BytesToHex(child.GetHash())
		childOnMap, ok := proofHashToNode[proofHash]
		if !ok {
			continue
		}

		branch.Children[i] = childOnMap
		t.loadProof(proofHashToNode, childOnMap)
	}
}

// Load reconstructs the trie from the database from the given root hash.
// It is used when restarting the node to load the current state trie.
func (t *Trie) Load(db chaindb.Database, rootHash common.Hash) error {
	if rootHash == EmptyHash {
		t.root = nil
		return nil
	}

	rootHashBytes := rootHash[:]

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
	t.root.SetDirty(false)
	t.root.SetEncodingAndHash(encodedNode, rootHashBytes)

	return t.load(db, t.root)
}

func (t *Trie) load(db chaindb.Database, n Node) error {
	switch n.Type() {
	case node.BranchType, node.BranchWithValueType:
	default: // not a branch
		return nil
	}

	branch := n.(*node.Branch)

	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		hash := child.GetHash()
		encodedNode, err := db.Get(hash)
		if err != nil {
			return fmt.Errorf("cannot find child node key 0x%x in database: %w", hash, err)
		}

		reader := bytes.NewReader(encodedNode)
		decodedNode, err := node.Decode(reader)
		if err != nil {
			return fmt.Errorf("cannot decode node with hash 0x%x: %w", hash, err)
		}

		decodedNode.SetDirty(false)
		decodedNode.SetEncodingAndHash(encodedNode, hash)
		branch.Children[i] = decodedNode

		err = t.load(db, decodedNode)
		if err != nil {
			return fmt.Errorf("cannot load child at index %d with hash 0x%x: %w", i, hash, err)
		}
	}

	for _, key := range t.GetKeysWithPrefix(ChildStorageKeyPrefix) {
		childTrie := NewEmptyTrie()
		value := t.Get(key)
		err := childTrie.Load(db, common.NewHash(value))
		if err != nil {
			return fmt.Errorf("failed to load child trie with root hash=0x%x: %w", value, err)
		}

		err = t.PutChild(value, childTrie)
		if err != nil {
			return fmt.Errorf("failed to insert child trie with root hash=0x%x into main trie: %w",
				childTrie.root.GetHash(), err)
		}
	}

	return nil
}

// PopulateNodeHashes writes hashes of each children of the node given
// as keys to the map hashesSet.
func (t *Trie) PopulateNodeHashes(n Node, hashesSet map[common.Hash]struct{}) {
	switch n.Type() {
	case node.BranchType, node.BranchWithValueType:
	default:
		return
	}

	branch := n.(*node.Branch)

	for _, child := range branch.Children {
		if child == nil {
			continue
		}

		hash := common.BytesToHash(child.GetHash())
		hashesSet[hash] = struct{}{}

		t.PopulateNodeHashes(child, hashesSet)
	}
}

// PutInDB inserts a value in the trie at the key given.
// It writes the updated nodes from the changed node up to the root node
// to the database in a batch operation.
func (t *Trie) PutInDB(db chaindb.Database, key, value []byte) error {
	t.Put(key, value)
	return t.WriteDirty(db)
}

// DeleteFromDB deletes a value from the trie at the key given.
// It writes the updated nodes from the changed node up to the root node
// to the database in a batch operation.
func (t *Trie) DeleteFromDB(db chaindb.Database, key []byte) error {
	t.Delete(key)
	return t.WriteDirty(db)
}

// ClearPrefixFromDB deletes all nodes with keys starting the given prefix
// from the trie. It writes the updated nodes from the changed node up to
// the root node to the database in a batch operation.
// in a batch operation.
func (t *Trie) ClearPrefixFromDB(db chaindb.Database, prefix []byte) error {
	t.ClearPrefix(prefix)
	return t.WriteDirty(db)
}

// GetFromDB retrieves a value at the given key from the trie using the database.
// It recursively descends into the trie using the database starting
// from the root node until it reaches the node with the given key.
// It then reads the value from the database.
func GetFromDB(db chaindb.Database, rootHash common.Hash, key []byte) (
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

	return getFromDB(db, rootNode, k)
}

// getFromDB recursively searches through the trie and database
// for the value corresponding to a key.
// Note it does not copy the value so modifying the value bytes
// slice will modify the value of the node in the trie.
func getFromDB(db chaindb.Database, n Node, key []byte) (
	value []byte, err error) {
	leaf, ok := n.(*node.Leaf)
	if ok {
		if bytes.Equal(leaf.Key, key) {
			return leaf.Value, nil
		}
		return nil, nil
	}

	branch := n.(*node.Branch)
	// Key is equal to the key of this branch or is empty
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		return branch.Value, nil
	}

	commonPrefixLength := lenCommonPrefix(branch.Key, key)
	if len(key) < len(branch.Key) && bytes.Equal(branch.Key[:commonPrefixLength], key) {
		// The key to search is a prefix of the node key and is smaller than the node key.
		// Example: key to search: 0xabcd
		//          branch key:    0xabcdef
		return nil, nil
	}

	// childIndex is the nibble after the common prefix length in the key being searched.
	childIndex := key[commonPrefixLength]
	childWithHashOnly := branch.Children[childIndex]
	if childWithHashOnly == nil {
		return nil, nil
	}

	childHash := childWithHashOnly.GetHash()
	encodedChild, err := db.Get(childHash)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot find child with hash 0x%x in database: %w",
			childHash, err)
	}

	reader := bytes.NewReader(encodedChild)
	decodedChild, err := node.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot decode child node with hash 0x%x: %w",
			childHash, err)
	}

	return getFromDB(db, decodedChild, key[commonPrefixLength+1:])
	// Note: do not wrap error since it's called recursively.
}

// WriteDirty writes all dirty nodes to the database and sets them to clean
func (t *Trie) WriteDirty(db chaindb.Database) error {
	batch := db.NewBatch()
	err := t.writeDirty(batch, t.root)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Flush()
}

func (t *Trie) writeDirty(db chaindb.Batch, n Node) error {
	if n == nil || !n.IsDirty() {
		return nil
	}

	encoding, hash, err := n.EncodeAndHash()
	if err != nil {
		return fmt.Errorf(
			"cannot encode and hash node with hash 0x%x: %w",
			n.GetHash(), err)
	}

	if n == t.root {
		// hash root node even if its encoding is under 32 bytes
		encodingDigest, err := common.Blake2bHash(encoding)
		if err != nil {
			return fmt.Errorf("cannot hash root node encoding: %w", err)
		}

		hash = encodingDigest[:]
	}

	err = db.Put(hash, encoding)
	if err != nil {
		return fmt.Errorf(
			"cannot put encoding of node with hash 0x%x in database: %w",
			hash, err)
	}

	switch n.Type() {
	case node.BranchType, node.BranchWithValueType:
	default: // not a branch
		n.SetDirty(false)
		return nil
	}

	branch := n.(*node.Branch)

	for _, child := range branch.Children {
		if child == nil {
			continue
		}

		err = t.writeDirty(db, child)
		if err != nil {
			// Note: do not wrap error since it's returned recursively.
			return err
		}
	}

	for _, childTrie := range t.childTries {
		if err := childTrie.writeDirty(db, childTrie.root); err != nil {
			return fmt.Errorf("failed to write dirty node=0x%x to database: %w", childTrie.root.GetHash(), err)
		}
	}

	branch.SetDirty(false)

	return nil
}

// GetInsertedNodeHashes returns a set of hashes with all
// the hashes of all nodes that were inserted in the state trie
// since the last snapshot.
// We need to compute the hash values of each newly inserted node.
func (t *Trie) GetInsertedNodeHashes() (hashesSet map[common.Hash]struct{}, err error) {
	hashesSet = make(map[common.Hash]struct{})
	err = t.getInsertedNodeHashes(t.root, hashesSet)
	if err != nil {
		return nil, err
	}
	return hashesSet, nil
}

func (t *Trie) getInsertedNodeHashes(n Node, hashes map[common.Hash]struct{}) (err error) {
	// TODO pass map of hashes or slice as argument to avoid copying
	// and using more memory.
	if n == nil || !n.IsDirty() {
		return nil
	}

	encoding, hash, err := n.EncodeAndHash()
	if err != nil {
		return fmt.Errorf(
			"cannot encode and hash node with hash 0x%x: %w",
			n.GetHash(), err)
	}

	if n == t.root && len(encoding) < 32 {
		// hash root node even if its encoding is under 32 bytes
		encodingDigest, err := common.Blake2bHash(encoding)
		if err != nil {
			return fmt.Errorf("cannot hash root node encoding: %w", err)
		}

		hash = encodingDigest[:]
	}

	hashes[common.BytesToHash(hash)] = struct{}{}

	switch n.Type() {
	case node.BranchType, node.BranchWithValueType:
	default: // not a branch
		return nil
	}

	branch := n.(*node.Branch)

	for _, child := range branch.Children {
		if child == nil {
			continue
		}

		err := t.getInsertedNodeHashes(child, hashes)
		if err != nil {
			// Note: do not wrap error since this is called recursively.
			return err
		}
	}

	return nil
}

// GetDeletedNodeHashes returns a set of all the hashes of nodes that were
// deleted from the trie since the last snapshot was made.
// The returned set is a copy of the internal set to prevent data races.
func (t *Trie) GetDeletedNodeHashes() (hashesSet map[common.Hash]struct{}) {
	hashesSet = make(map[common.Hash]struct{}, len(t.deletedKeys))
	for k := range t.deletedKeys {
		hashesSet[k] = struct{}{}
	}
	return hashesSet
}
