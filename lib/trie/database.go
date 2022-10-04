// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/chaindb"
)

// Database is an interface to get values from a
// key value database.
type Database interface {
	Get(key []byte) (value []byte, err error)
}

// Store stores each trie node in the database,
// where the key is the hash of the encoded node
// and the value is the encoded node.
// Generally, this will only be used for the genesis trie.
func (t *Trie) Store(db chaindb.Database) error {
	for _, v := range t.childTries {
		if err := v.Store(db); err != nil {
			return fmt.Errorf("failed to store child trie with root hash=0x%x in the db: %w", v.root.MerkleValue, err)
		}
	}

	batch := db.NewBatch()
	err := t.storeNode(batch, t.root)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Flush()
}

func (t *Trie) storeNode(db chaindb.Batch, n *Node) (err error) {
	if n == nil {
		return nil
	}

	var encoding, hash []byte
	if n == t.root {
		encoding, hash, err = n.EncodeAndHashRoot()
	} else {
		encoding, hash, err = n.EncodeAndHash()
	}
	if err != nil {
		return err
	}

	err = db.Put(hash, encoding)
	if err != nil {
		return err
	}

	if n.Kind() == node.Branch {
		for _, child := range n.Children {
			if child == nil {
				continue
			}

			err = t.storeNode(db, child)
			if err != nil {
				return err
			}
		}
	}

	if n.Dirty {
		n.SetClean()
	}

	return nil
}

// Load reconstructs the trie from the database from the given root hash.
// It is used when restarting the node to load the current state trie.
func (t *Trie) Load(db Database, rootHash common.Hash) error {
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
	t.root.SetClean()
	t.root.Encoding = encodedNode
	t.root.MerkleValue = rootHashBytes

	return t.loadNode(db, t.root)
}

func (t *Trie) loadNode(db Database, n *Node) error {
	if n.Kind() != node.Branch {
		return nil
	}

	branch := n
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		merkleValue := child.MerkleValue

		if len(merkleValue) == 0 {
			// node has already been loaded inline
			// just set encoding + hash digest
			_, err := child.CalculateMerkleValue()
			if err != nil {
				return fmt.Errorf("merkle value: %w", err)
			}
			child.SetClean()
			continue
		}

		encodedNode, err := db.Get(merkleValue)
		if err != nil {
			return fmt.Errorf("cannot find child node key 0x%x in database: %w", merkleValue, err)
		}

		reader := bytes.NewReader(encodedNode)
		decodedNode, err := node.Decode(reader)
		if err != nil {
			return fmt.Errorf("decoding node with Merkle value 0x%x: %w", merkleValue, err)
		}

		decodedNode.SetClean()
		decodedNode.Encoding = encodedNode
		decodedNode.MerkleValue = merkleValue
		branch.Children[i] = decodedNode

		err = t.loadNode(db, decodedNode)
		if err != nil {
			return fmt.Errorf("loading child at index %d with Merkle value 0x%x: %w", i, merkleValue, err)
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

// PopulateMerkleValues writes the Merkle values of the node given and of
// all its descendant nodes as keys to the map merkleValues.
// It is assumed the node and its descendant nodes have their Merkle value already
// computed.
func PopulateMerkleValues(n *Node, merkleValues map[string]struct{}) {
	if n == nil {
		return
	}

	if len(n.MerkleValue) == 0 {
		panic(fmt.Sprintf("node with key 0x%x has no Merkle value computed", n.Key))
	}

	merkleValues[string(n.MerkleValue)] = struct{}{}

	if n.Kind() == node.Leaf {
		return
	}

	branch := n
	for _, child := range branch.Children {
		PopulateMerkleValues(child, merkleValues)
	}
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

	return getFromDBAtNode(db, rootNode, k)
}

// getFromDBAtNode recursively searches through the trie and database
// for the value corresponding to a key.
// Note it does not copy the value so modifying the value bytes
// slice will modify the value of the node in the trie.
func getFromDBAtNode(db chaindb.Database, n *Node, key []byte) (
	value []byte, err error) {
	if n.Kind() == node.Leaf {
		if bytes.Equal(n.Key, key) {
			return n.SubValue, nil
		}
		return nil, nil
	}

	branch := n
	// Key is equal to the key of this branch or is empty
	if len(key) == 0 || bytes.Equal(branch.Key, key) {
		return branch.SubValue, nil
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
			"finding child node with Merkle value 0x%x in database: %w",
			childMerkleValue, err)
	}

	reader := bytes.NewReader(encodedChild)
	decodedChild, err := node.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"decoding child node with Merkle value 0x%x: %w",
			childMerkleValue, err)
	}

	return getFromDBAtNode(db, decodedChild, key[commonPrefixLength+1:])
	// Note: do not wrap error since it's called recursively.
}

// WriteDirty writes all dirty nodes to the database and sets them to clean
func (t *Trie) WriteDirty(db chaindb.Database) error {
	batch := db.NewBatch()
	err := t.writeDirtyNode(batch, t.root)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Flush()
}

func (t *Trie) writeDirtyNode(db chaindb.Batch, n *Node) (err error) {
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

	err = db.Put(merkleValue, encoding)
	if err != nil {
		return fmt.Errorf(
			"putting encoding of node with Merkle value 0x%x in database: %w",
			merkleValue, err)
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

// GetInsertedMerkleValues returns the set of node Merkle values
// for each node that was inserted in the state trie since the last snapshot.
func (t *Trie) GetInsertedMerkleValues() (merkleValues map[string]struct{}, err error) {
	merkleValues = make(map[string]struct{})
	err = t.getInsertedNodeHashesAtNode(t.root, merkleValues)
	if err != nil {
		return nil, err
	}
	return merkleValues, nil
}

func (t *Trie) getInsertedNodeHashesAtNode(n *Node, merkleValues map[string]struct{}) (err error) {
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
		return fmt.Errorf(
			"encoding and hashing node with Merkle value 0x%x: %w",
			n.MerkleValue, err)
	}

	merkleValues[string(merkleValue)] = struct{}{}

	if n.Kind() != node.Branch {
		return nil
	}

	for _, child := range n.Children {
		if child == nil {
			continue
		}

		err := t.getInsertedNodeHashesAtNode(child, merkleValues)
		if err != nil {
			// Note: do not wrap error since this is called recursively.
			return err
		}
	}

	return nil
}

// GetDeletedMerkleValues returns a set of all the node Merkle values for each
// node that was deleted from the trie since the last snapshot was made.
// The returned set is a copy of the internal set to prevent data corruption.
func (t *Trie) GetDeletedMerkleValues() (merkleValues map[string]struct{}) {
	merkleValues = make(map[string]struct{}, len(t.deletedMerkleValues))
	for k := range t.deletedMerkleValues {
		merkleValues[k] = struct{}{}
	}
	return merkleValues
}
