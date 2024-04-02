// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
)

var ErrInvalidStateRoot = errors.New("invalid state root")
var ErrIncompleteDB = errors.New("incomplete database")

var EmptyValue = []byte{}

type Lookup struct {
	db   db.DBGetter
	hash []byte
}

func NewLookup(db db.DBGetter, hash []byte) *Lookup {
	return &Lookup{db, hash}
}

func (l Lookup) Lookup(nibbleKey []byte) ([]byte, error) {
	return l.lookupWithoutCache(nibbleKey)
}

func (l Lookup) lookupWithoutCache(nibbleKey []byte) ([]byte, error) {
	partial := nibbleKey
	hash := l.hash
	keyNibbles := 0

	depth := 0

	for {
		// Get node from DB
		nodeData, err := l.db.Get(hash)

		if err != nil {
			if depth == 0 {
				return nil, ErrInvalidStateRoot
			}
			return nil, ErrIncompleteDB
		}

		// Iterates children
		for {
			// Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := node.Decode(reader)
			if err != nil {
				return nil, fmt.Errorf("decoding node error %s", err.Error())
			}

			// Empty Node
			if decodedNode == nil {
				return EmptyValue, nil
			}

			var nextNode *node.Node

			switch decodedNode.Kind() {
			case node.Leaf:
				// If leaf and matches return value
				if bytes.Equal(partial, decodedNode.PartialKey) {
					return l.loadValue(decodedNode.StorageValue)
				}
				return EmptyValue, nil
			// Nibbled branch
			case node.Branch:
				// Get next node
				slice := decodedNode.PartialKey

				if !bytes.HasPrefix(partial, slice) {
					return EmptyValue, nil
				}

				if len(partial) == len(slice) {
					if decodedNode.StorageValue != nil {
						return l.loadValue(decodedNode.StorageValue)
					}
				}

				nextNode = decodedNode.Children[partial[len(slice)]]
				if nextNode == nil {
					return EmptyValue, nil
				}

				partial = partial[len(slice)+1:]
				keyNibbles += len(slice) + 1
			}

			if nextNode.IsHashedValue {
				hash = nextNode.StorageValue
				break
			}

			nodeData = nextNode.StorageValue
		}
		depth++
	}
}

// TODO: change our nodes to use *NodeValue type instead of using []byte and
// stop decoding the value in the Decode method if it is a hashed reference to
// a different node
func (l Lookup) loadValue(value []byte) ([]byte, error) {
	// Since we are already decoding the node value when it is a reference this
	// function is trivial
	return value, nil

	// I'll leave the code below for reference regarding the right way to do it
	// But we need to change the node struct to use *NodeValue instead of []byte
	// And the node decode to not decode the value if it is a reference

	/*
		if value == nil {
			return nil, fmt.Errorf("trying to load value from nil node")
		}
		if !value.Hashed {
			return value.Data, nil
		}

		return l.db.Get(value.Data)
	*/
}
