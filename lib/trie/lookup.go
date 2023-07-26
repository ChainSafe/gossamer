package trie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/node"
)

var ErrInvalidStateRoot = errors.New("invalid state root")
var ErrIncompleteDB = errors.New("incomplete database")

var EmptyValue = []byte{}

type Lookup struct {
	db   HashDB
	hash []byte
	//TODO: implement cache and recorder
}

func NewLookup(db HashDB, hash []byte) *Lookup {
	return &Lookup{db, hash}
}

func (l Lookup) Lookup(key []byte, nibbleKey *NibbleSlice) ([]byte, error) {
	return l.lookupWithoutCache(nibbleKey, key)
}

func (l Lookup) lookupWithoutCache(nibbleKey *NibbleSlice, fullKey []byte) ([]byte, error) {
	partial := nibbleKey
	hash := l.hash
	keyNibbles := uint(0)

	depth := 0

	for {
		//Get node from DB
		nodeData, err := l.db.GetWithPrefix(hash, *nibbleKey.mid(keyNibbles).left())
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
			decodedNode, err := node.Decode(reader)
			if err != nil {
				return nil, fmt.Errorf("decoding node error %s", err.Error())
			}

			//Empty Node
			if decodedNode == node.EmptyNode {
				return EmptyValue, nil
			}

			var nextNode *Node = nil

			switch decodedNode.Kind() {
			case node.Leaf:
				//If leaf and matches return value
				if bytes.Equal(decodedNode.PartialKey, partial.data) {
					l.loadValue(decodedNode, nibbleKey.originalDataAsPrefix(), fullKey)
				}
				return EmptyValue, nil
			//Nibbled branch
			case node.Branch:
				//Get next node
				slice := NewNibbleSlice(decodedNode.PartialKey)
				if !partial.startsWith(slice) {
					return EmptyValue, nil
				}

				if partial.len() == slice.len() {
					if len(decodedNode.StorageValue) > 0 {
						return l.loadValue(decodedNode, nibbleKey.originalDataAsPrefix(), fullKey)
					}
				}

				nextNode := decodedNode.ChildAt(slice.len())
				if nextNode == node.EmptyNode {
					return EmptyValue, nil
				}

				partial = partial.mid(slice.len() + 1)
				keyNibbles += slice.len() + 1
			}

			if nextNode.IsHashed() {
				hash = decodedNode.StorageValue
				break
			} else {
				nodeData = decodedNode.StorageValue
			}
		}
	}
}

func (l Lookup) loadValue(node *Node, prefix Prefix, fullKey []byte) ([]byte, error) {
	if !node.IsHashed() {
		return node.StorageValue, nil
	}

	return l.db.GetWithPrefix(node.StorageValue, prefix)
}
