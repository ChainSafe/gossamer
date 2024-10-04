// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

type TrieLookup[H hash.Hash, Hasher hash.Hasher[H]] struct {
	// db to query from
	db db.DBGetter
	// hash to start at
	hash H
	// optional cache to speed up the db lookups
	cache Cache
	// optional recorder for recording trie accesses
	recorder TrieRecorder
}

func NewTrieLookup[H hash.Hash, Hasher hash.Hasher[H]](
	db db.DBGetter,
	hash H,
	cache Cache,
	recorder TrieRecorder,
) TrieLookup[H, Hasher] {
	return TrieLookup[H, Hasher]{
		db:       db,
		hash:     hash,
		cache:    cache,
		recorder: recorder,
	}
}

func (l *TrieLookup[H, Hasher]) lookupNode(
	nibbleKey nibbles.Nibbles, fullKey []byte,
) (codec.EncodedNode, error) {
	// Start from root node and going downwards
	partialKey := nibbleKey.Clone()
	hash := l.hash
	var keyNibbles uint

	// Iterates through non inlined nodes
	for {
		// Get node from DB
		prefixedKey := append(nibbleKey.Mid(keyNibbles).Left().JoinedBytes(), hash.Bytes()...)
		nodeData, err := l.db.Get(prefixedKey)
		if err != nil {
			return nil, ErrIncompleteDB
		}

		l.recordAccess(EncodedNodeAccess[H]{Hash: hash, EncodedNode: nodeData})

	InlinedChildrenIterator:
		for {
			// Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := codec.Decode[H](reader)
			if err != nil {
				return nil, err
			}

			var nextNode codec.MerkleValue

			switch n := decodedNode.(type) {
			case codec.Empty:
				return nil, nil //nolint:nilnil
			case codec.Leaf:
				// We are in the node we were looking for
				if partialKey.Equal(n.PartialKey) {
					return n, nil
				}

				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})

				return nil, nil //nolint:nilnil
			case codec.Branch:
				nodePartialKey := n.PartialKey

				// This is unusual but could happen if for some reason one
				// branch has a hashed child node that points to a node that
				// doesn't share the prefix we are expecting
				if !partialKey.StartsWith(nodePartialKey) {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// We are in the node we were looking for
				if partialKey.Equal(n.PartialKey) {
					if n.Value != nil {
						return n, nil
					}

					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// This is not the node we were looking for but it might be in
				// one of its children
				childIdx := int(partialKey.At(nodePartialKey.Len()))
				nextNode = n.Children[childIdx]
				if nextNode == nil {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// Advance the partial key consuming the part we already checked
				partialKey = partialKey.Mid(nodePartialKey.Len() + 1)
				keyNibbles += nodePartialKey.Len() + 1
			}

			// Next node could be inlined or hashed (pointer to a node)
			// https://spec.polkadot.network/chap-state#defn-merkle-value
			switch merkleValue := nextNode.(type) {
			case codec.HashedNode[H]:
				// If it's hashed we set the hash to look for it in next loop
				hash = merkleValue.Hash
				break InlinedChildrenIterator
			case codec.InlineNode:
				// If it is inlined we just need to decode it in the next loop
				nodeData = merkleValue
			}
		}
	}
}

func (l *TrieLookup[H, Hasher]) lookupValue(
	fullKey []byte, keyNibbles nibbles.Nibbles,
) (value []byte, err error) {
	node, err := l.lookupNode(keyNibbles, fullKey)
	if err != nil {
		return nil, err
	}

	// node not found so we return nil
	if node == nil {
		return nil, nil
	}

	if nodeValue := node.GetValue(); nodeValue != nil {
		value, err = l.fetchValue(keyNibbles.OriginalDataPrefix(), fullKey, nodeValue)
		if err != nil {
			return nil, err
		}
		return value, nil
	}

	return nil, nil
}

// fetchValue gets the value from the node, if it is inlined we can return it
// directly. But if it is hashed (V1) we have to look up for its value in the DB
func (l *TrieLookup[H, Hasher]) fetchValue(
	prefix nibbles.Prefix, fullKey []byte, value codec.EncodedValue,
) ([]byte, error) {
	switch v := value.(type) {
	case codec.InlineValue:
		l.recordAccess(InlineValueAccess{FullKey: fullKey})
		return v, nil
	case codec.HashedValue[H]:
		prefixedKey := bytes.Join([][]byte{prefix.JoinedBytes(), v.Hash.Bytes()}, nil)

		nodeData, err := l.db.Get(prefixedKey)
		if err != nil {
			return nil, ErrIncompleteDB
		}

		l.recordAccess(ValueAccess[H]{Hash: v.Hash, FullKey: fullKey, Value: nodeData})

		return nodeData, nil
	default:
		panic("unreachable")
	}
}

func (l *TrieLookup[H, Hasher]) recordAccess(access TrieAccess) {
	if l.recorder != nil {
		l.recorder.Record(access)
	}
}
