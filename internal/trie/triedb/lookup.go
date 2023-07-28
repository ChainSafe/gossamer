// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/hashdb"
	"github.com/ChainSafe/gossamer/internal/trie/triedb/nibble"
)

var ErrInvalidStateRoot = errors.New("invalid state root")
var ErrIncompleteDB = errors.New("incomplete database")

var EmptyValue = []byte{}

type Lookup struct {
	db   hashdb.HashDB
	hash []byte
	//TODO: implement cache and recorder
}

func NewLookup(db hashdb.HashDB, hash []byte) *Lookup {
	return &Lookup{db, hash}
}

func (l Lookup) Lookup(key []byte, nibbleKey *nibble.NibbleSlice) ([]byte, error) {
	return l.lookupWithoutCache(nibbleKey)
}

func (l Lookup) lookupWithoutCache(nibbleKey *nibble.NibbleSlice) ([]byte, error) {
	partial := nibbleKey
	hash := l.hash
	keyNibbles := uint(0)

	depth := 0

	for {
		//Get node from DB
		logger.Errorf("Hash: %x", hash)
		logger.Errorf("Partial: %v", partial)

		nodeData, err := l.db.Get(hash)

		logger.Errorf("Node data: %x", nodeData)

		if err != nil {
			if depth == 0 {
				return nil, ErrInvalidStateRoot
			}
			return nil, ErrIncompleteDB
		}

		//Iterates children
		for {
			//Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := Decode(reader)
			if err != nil {
				return nil, fmt.Errorf("decoding node error %s", err.Error())
			}

			logger.Errorf("Slice: %v", decodedNode.Slice)

			//logger.Errorf("Decoded node: %v", decodedNode)

			//Empty Node
			if decodedNode.Type == Empty {
				return EmptyValue, nil
			}

			var nextNode *NodeHandle = nil

			switch decodedNode.Type {
			case Leaf:
				logger.Errorf("Leaf")
				//If leaf and matches return value
				if bytes.Equal(decodedNode.Slice.Data(), partial.Data()) {
					return l.loadValue(decodedNode.Value, nibbleKey.OriginalDataAsPrefix())
				}
				return EmptyValue, nil
			//Nibbled branch
			case NibbledBranch:
				logger.Errorf("NibbledBranch")
				//Get next node
				slice := decodedNode.Slice

				if !partial.StartsWith(&slice) {
					logger.Errorf("!partial.StartsWith(&slice)")
					return EmptyValue, nil
				}

				if partial.Len() == slice.Len() {
					if decodedNode.Value != nil {
						return l.loadValue(decodedNode.Value, nibbleKey.OriginalDataAsPrefix())
					}
				}

				logger.Errorf("idx: %d", partial.At(slice.Len()))

				nextNode = decodedNode.Children[partial.At(slice.Len())]
				if nextNode == nil {
					return EmptyValue, nil
				}

				partial = partial.Mid(slice.Len() + 1)
				keyNibbles += slice.Len() + 1
			}

			if nextNode.Hashed {
				hash = nextNode.Data
				break
			} else {
				nodeData = nextNode.Data
			}
		}
	}
}

func (l Lookup) loadValue(value *NodeValue, prefix hashdb.Prefix) ([]byte, error) {
	if value != nil {
		return nil, fmt.Errorf("trying to load value from nil node")
	}
	if !value.Hashed {
		return value.Data, nil
	}

	return l.db.Get(value.Data)
}
