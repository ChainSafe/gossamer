// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/chaindb"
)

// Store stores each trie node in the database, where the key is the hash of the encoded node and the value is the encoded node.
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

func (t *Trie) store(db chaindb.Batch, curr node) error {
	if curr == nil {
		return nil
	}

	enc, hash, err := curr.encodeAndHash()
	if err != nil {
		return err
	}

	err = db.Put(hash, enc)
	if err != nil {
		return err
	}

	switch c := curr.(type) {
	case *branch:
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

	if curr.isDirty() {
		curr.setDirty(false)
	}

	return nil
}

// Load reconstructs the trie from the database from the given root hash. Used when restarting the node to load the current state trie.
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

	t.root.setDirty(false)
	t.root.setEncodingAndHash(enc, root[:])

	return t.load(db, t.root)
}

func (t *Trie) load(db chaindb.Database, curr node) error {
	switch c := curr.(type) {
	case *branch:
		for i, child := range c.children {
			if child == nil {
				continue
			}

			hash := child.getHash()
			enc, err := db.Get(hash)
			if err != nil {
				return fmt.Errorf("failed to find node key=%x index=%d: %w", child.(*leaf).hash, i, err)
			}

			child, err = decodeBytes(enc)
			if err != nil {
				return err
			}

			child.setDirty(false)
			child.setEncodingAndHash(enc, hash)

			c.children[i] = child
			err = t.load(db, child)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PutInDB puts a value into the trie and writes the updates nodes the database. Since it needs to write all the nodes from the changed node up to the root, it writes these in a batch operation.
func (t *Trie) PutInDB(db chaindb.Database, key, value []byte) error {
	t.Put(key, value)
	return t.WriteDirty(db)
}

// DeleteFromDB deletes a value from the trie and writes the updated nodes the database. Since it needs to write all the nodes from the changed node up to the root, it writes these in a batch operation.
func (t *Trie) DeleteFromDB(db chaindb.Database, key []byte) error {
	err := t.Delete(key)
	if err != nil {
		return err
	}

	return t.WriteDirty(db)
}

// ClearPrefixFromDB deletes all keys with the given prefix from the trie and writes the updated nodes the database. Since it needs to write all the nodes from the changed node up to the root, it writes these in a batch operation.
func (t *Trie) ClearPrefixFromDB(db chaindb.Database, prefix []byte) error {
	t.ClearPrefix(prefix)
	return t.WriteDirty(db)
}

// GetFromDB retrieves a value from the trie using the database. It recursively descends into the trie using the database starting from the root node until it reaches the node with the given key. It then reads the value from the database.
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

func getFromDB(db chaindb.Database, parent node, key []byte) ([]byte, error) {
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

func (t *Trie) writeDirty(db chaindb.Batch, curr node) error {
	if curr == nil || !curr.isDirty() {
		return nil
	}

	enc, hash, err := curr.encodeAndHash()
	if err != nil {
		return err
	}

	// always hash root even if encoding is under 32 bytes
	if curr == t.root {
		h, err := common.Blake2bHash(enc) //nolint
		if err != nil {
			return err
		}

		hash = h[:]
	}

	err = db.Put(hash, enc)
	if err != nil {
		return err
	}

	switch c := curr.(type) {
	case *branch:
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

	curr.setDirty(false)
	return nil
}
