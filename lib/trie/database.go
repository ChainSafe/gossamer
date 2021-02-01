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
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/chaindb"
)

// Store stores each trie node in the database, where the key is the hash of the encoded node and the value is the encoded node.
// Generally, this will only be used for the genesis trie.
func (t *Trie) Store(db chaindb.Database) error {
	return t.store(db, t.root)
}

func (t *Trie) store(db chaindb.Database, curr node) error {
	enc, hash, err := curr.encodeAndHash()
	if err != nil {
		return err
	}

	err = db.Put(hash, enc)
	if err != nil {
		return err
	}
	fmt.Printf("stored node hash=%x\n", hash)

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

	return nil
}

// Load reconstructs the trie from the database from the given root hash. Used when restarting the node to load the current state trie.
func (t *Trie) Load(db chaindb.Database, root common.Hash) error {
	enc, err := db.Get(root[:])
	if err != nil {
		return fmt.Errorf("failed to find root key=%s: %w", root, err)
	}

	t.root, err = decodeBytes(enc)
	if err != nil {
		return err
	}

	t.root, err = t.load(db, t.root)
	return err
}

func (t *Trie) load(db chaindb.Database, curr node) (node, error) {
	switch c := curr.(type) {
	case *branch:
		for i, child := range c.children {
			if child == nil {
				continue
			}

			enc, err := db.Get(child.(*leaf).hash)
			if err != nil {
				return nil, fmt.Errorf("failed to find node key=%x: %w", child.(*leaf).hash, err)
			}

			child, err = decodeBytes(enc)
			if err != nil {
				return nil, err
			}

			c.children[i], err = t.load(db, child)
		}
	case *leaf:
		return c, nil
	default:
		return nil, nil
	}

	return nil, nil
}

// PutInDB puts a value into the trie and writes the updates nodes the database. Since it needs to write all the nodes from the changed node up to the root, it writes these in a batch operation.
func (t *Trie) PutInDB(db chaindb.Database, key, value []byte) error {
	return nil
}

// DeleteFromDB deletes a value from the trie and writes the updated nodes the database. Since it needs to write all the nodes from the changed node up to the root, it writes these in a batch operation.
func (t *Trie) DeleteFromDB(db chaindb.Database, key []byte) error {
	return nil
}

// ClearPrefixFromDB deletes all keys with the given prefix from the trie and writes the updated nodes the database. Since it needs to write all the nodes from the changed node up to the root, it writes these in a batch operation.
func (t *Trie) ClearPrefixFromDB(db chaindb.Database, key []byte) error {
	return nil
}

// GetFromDB retrieves a value from the trie using the database. It recursively descends into the trie using the database starting from the root node until it reaches the node with the given key. It then reads the value from the database.
func (t *Trie) GetFromDB(db chaindb.Database, key []byte) ([]byte, error) {
	return nil, nil
}
