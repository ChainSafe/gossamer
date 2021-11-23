// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/chaindb"
)

// ErrEmptyProof indicates the proof slice is empty
var ErrEmptyProof = errors.New("proof slice empty")

// Store stores each trie node in the database,
// where the key is the hash of the encoded node
// and the value is the encoded node.
// Generally, this will only be used for the genesis trie.
func (t *Trie) Store(db chaindb.Database) error {
	batch := db.NewBatch()
	err := t.store(batch, t.root)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Flush()
}

func (t *Trie) store(db chaindb.Batch, curr Node) error {
	if curr == nil {
		return nil
	}

	enc, hash, err := curr.EncodeAndHash()
	if err != nil {
		return err
	}

	err = db.Put(hash, enc)
	if err != nil {
		return err
	}

	if c, ok := curr.(*branch); ok {
		for _, child := range c.children {
			if child == nil {
				continue
			}

			err = t.store(db, child)
			if err != nil {
				return err
			}
		}
	}

	if curr.IsDirty() {
		curr.SetDirty(false)
	}

	return nil
}

// LoadFromProof create a partial trie based on the proof slice, as it only contains nodes that are in the proof afaik.
func (t *Trie) LoadFromProof(proof [][]byte, root []byte) error {
	if len(proof) == 0 {
		return ErrEmptyProof
	}

	mappedNodes := make(map[string]Node, len(proof))

	// map all the proofs hash -> decoded node
	// and takes the loop to indentify the root node
	for _, rawNode := range proof {
		decNode, err := decodeBytes(rawNode)
		if err != nil {
			return err
		}

		decNode.SetDirty(false)
		decNode.SetEncodingAndHash(rawNode, nil)

		_, computedRoot, err := decNode.EncodeAndHash()
		if err != nil {
			return err
		}

		mappedNodes[common.BytesToHex(computedRoot)] = decNode

		if bytes.Equal(computedRoot, root) {
			t.root = decNode
		}
	}

	t.loadProof(mappedNodes, t.root)
	return nil
}

// loadProof is a recursive function that will create all the trie paths based
// on the mapped proofs slice starting by the root
func (t *Trie) loadProof(proof map[string]Node, curr Node) {
	c, ok := curr.(*branch)
	if !ok {
		return
	}

	for i, child := range c.children {
		if child == nil {
			continue
		}

		proofNode, ok := proof[common.BytesToHex(child.GetHash())]
		if !ok {
			continue
		}

		c.children[i] = proofNode
		t.loadProof(proof, proofNode)
	}
}

// Load reconstructs the trie from the database from the given root hash.
// It is used when restarting the node to load the current state trie.
func (t *Trie) Load(db chaindb.Database, root common.Hash) error {
	if root == EmptyHash {
		t.root = nil
		return nil
	}

	enc, err := db.Get(root[:])
	if err != nil {
		return fmt.Errorf("failed to find root key=%s: %w", root, err)
	}

	t.root, err = decodeBytes(enc)
	if err != nil {
		return err
	}

	t.root.SetDirty(false)
	t.root.SetEncodingAndHash(enc, root[:])

	return t.load(db, t.root)
}

func (t *Trie) load(db chaindb.Database, curr Node) error {
	if c, ok := curr.(*branch); ok {
		for i, child := range c.children {
			if child == nil {
				continue
			}

			hash := child.GetHash()
			enc, err := db.Get(hash)
			if err != nil {
				return fmt.Errorf("failed to find node key=%x index=%d: %w", child.(*leaf).hash, i, err)
			}

			child, err = decodeBytes(enc)
			if err != nil {
				return err
			}

			child.SetDirty(false)
			child.SetEncodingAndHash(enc, hash)

			c.children[i] = child
			err = t.load(db, child)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetNodeHashes return hash of each key of the trie.
func (t *Trie) GetNodeHashes(curr Node, keys map[common.Hash]struct{}) error {
	if c, ok := curr.(*branch); ok {
		for _, child := range c.children {
			if child == nil {
				continue
			}

			hash := child.GetHash()
			keys[common.BytesToHash(hash)] = struct{}{}

			err := t.GetNodeHashes(child, keys)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PutInDB puts a value into the trie and writes the updates nodes the database.
// Since it needs to write all the nodes from the changed node up to the root,
// it writes these in a batch operation.
func (t *Trie) PutInDB(db chaindb.Database, key, value []byte) error {
	t.Put(key, value)
	return t.WriteDirty(db)
}

// DeleteFromDB deletes a value from the trie and writes the updated nodes the database.
// Since it needs to write all the nodes from the changed node up to the root,
// it writes these in a batch operation.
func (t *Trie) DeleteFromDB(db chaindb.Database, key []byte) error {
	t.Delete(key)
	return t.WriteDirty(db)
}

// ClearPrefixFromDB deletes all keys with the given prefix from the trie
// and writes the updated nodes the database. Since it needs to write all
//  the nodes from the changed node up to the root, it writes these
// in a batch operation.
func (t *Trie) ClearPrefixFromDB(db chaindb.Database, prefix []byte) error {
	t.ClearPrefix(prefix)
	return t.WriteDirty(db)
}

// GetFromDB retrieves a value from the trie using the database.
// It recursively descends into the trie using the database starting
// from the root node until it reaches the node with the given key.
// It then reads the value from the database.
func GetFromDB(db chaindb.Database, root common.Hash, key []byte) ([]byte, error) {
	if root == EmptyHash {
		return nil, nil
	}

	k := keyToNibbles(key)

	enc, err := db.Get(root[:])
	if err != nil {
		return nil, fmt.Errorf("failed to find root key=%s: %w", root, err)
	}

	rootNode, err := decodeBytes(enc)
	if err != nil {
		return nil, err
	}

	return getFromDB(db, rootNode, k)
}

func getFromDB(db chaindb.Database, parent Node, key []byte) ([]byte, error) {
	var value []byte

	switch p := parent.(type) {
	case *branch:
		length := lenCommonPrefix(p.key, key)

		// found the value at this node
		if bytes.Equal(p.key, key) || len(key) == 0 {
			return p.value, nil
		}

		// did not find value
		if bytes.Equal(p.key[:length], key) && len(key) < len(p.key) {
			return nil, nil
		}

		if p.children[key[length]] == nil {
			return nil, nil
		}

		// load child with potential value
		enc, err := db.Get(p.children[key[length]].(*leaf).hash)
		if err != nil {
			return nil, fmt.Errorf("failed to find node in database: %w", err)
		}

		child, err := decodeBytes(enc)
		if err != nil {
			return nil, err
		}

		value, err = getFromDB(db, child, key[length+1:])
		if err != nil {
			return nil, err
		}
	case *leaf:
		if bytes.Equal(p.key, key) {
			return p.value, nil
		}
	case nil:
		return nil, nil

	}
	return value, nil
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

func (t *Trie) writeDirty(db chaindb.Batch, curr Node) error {
	if curr == nil || !curr.IsDirty() {
		return nil
	}

	enc, hash, err := curr.EncodeAndHash()
	if err != nil {
		return err
	}

	// always hash root even if encoding is under 32 bytes
	if curr == t.root {
		h, err := common.Blake2bHash(enc)
		if err != nil {
			return err
		}

		hash = h[:]
	}

	err = db.Put(hash, enc)
	if err != nil {
		return err
	}

	if c, ok := curr.(*branch); ok {
		for _, child := range c.children {
			if child == nil {
				continue
			}

			err = t.writeDirty(db, child)
			if err != nil {
				return err
			}
		}
	}

	curr.SetDirty(false)
	return nil
}

// GetInsertedNodeHashes returns the hash of nodes that are inserted into state trie since last snapshot is called
// Since inserted nodes are newly created we need to compute their hash values.
func (t *Trie) GetInsertedNodeHashes() ([]common.Hash, error) {
	return t.getInsertedNodeHashes(t.root)
}

func (t *Trie) getInsertedNodeHashes(curr Node) ([]common.Hash, error) {
	var nodeHashes []common.Hash
	if curr == nil || !curr.IsDirty() {
		return nil, nil
	}

	enc, hash, err := curr.EncodeAndHash()
	if err != nil {
		return nil, err
	}

	if curr == t.root && len(enc) < 32 {
		h, err := common.Blake2bHash(enc)
		if err != nil {
			return nil, err
		}

		hash = h[:]
	}

	nodeHash := common.BytesToHash(hash)
	nodeHashes = append(nodeHashes, nodeHash)

	if c, ok := curr.(*branch); ok {
		for _, child := range c.children {
			if child == nil {
				continue
			}
			nodes, err := t.getInsertedNodeHashes(child)
			if err != nil {
				return nil, err
			}
			nodeHashes = append(nodeHashes, nodes...)
		}
	}

	return nodeHashes, nil
}

// GetDeletedNodeHash returns the hash of nodes that are deleted from state trie since last snapshot is called
func (t *Trie) GetDeletedNodeHash() []common.Hash {
	return t.deletedKeys
}
