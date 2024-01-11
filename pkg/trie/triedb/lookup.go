// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

var EmptyValue = []byte{}

type Lookup[Hash node.HashOut] struct {
	db       hashdb.HashDB[Hash, DBValue]
	hash     Hash
	cache    TrieCache[Hash]
	recorder TrieRecorder[Hash]
	layout   TrieLayout[Hash]
}

func NewLookup[H node.HashOut](
	db hashdb.HashDB[H, DBValue], hash H, cache TrieCache[H], recorder TrieRecorder[H]) *Lookup[H] {
	return &Lookup[H]{
		db:       db,
		hash:     hash,
		cache:    cache,
		recorder: recorder,
	}
}

func (l Lookup[H]) Lookup(nibbleKey *nibble.NibbleSlice) ([]byte, error) {
	return l.lookupWithoutCache(nibbleKey)
}

func (l Lookup[H]) record(access TrieAccess[H]) {
	if l.recorder != nil {
		l.recorder.record(access)
	}
}

func (l Lookup[H]) lookupWithoutCache(nibbleKey *nibble.NibbleSlice) ([]byte, error) {
	partial := nibbleKey
	hash := l.hash
	keyNibbles := 0

	depth := 0

	for {
		// Get node from DB
		nodeData := l.db.Get(hash, nibbleKey.Mid(keyNibbles).Left())
		if nodeData == nil {
			if depth == 0 {
				return nil, ErrInvalidStateRoot
			} else {
				return nil, ErrIncompleteDB
			}
		}

		l.record(TrieAccessEncodedNode[H]{
			hash:        hash,
			encodedNode: *nodeData,
		})

		// Iterates children
		for {
			// Decode node
			decodedNode, err := l.layout.Codec().Decode(*nodeData)
			if err != nil {
				return nil, DecoderError
			}

			var nextNode node.NodeHandle = nil

			switch node := decodedNode.(type) {
			case node.Empty:
				return EmptyValue, nil
			case node.Leaf:
				// If leaf and matches return value
				if partial.Eq(node.PartialKey) {
					return l.loadValue(node.Value, nibbleKey.OriginalDataAsPrefix())
				}
				return EmptyValue, nil
			case node.NibbledBranch:
				slice := node.PartialKey
				children := node.Children
				value := node.Value
				// Get next node
				if !partial.StartsWith(slice) {
					return EmptyValue, nil
				}

				if partial.Len() == slice.Len() {
					if value != nil {
						return l.loadValue(value, nibbleKey.OriginalDataAsPrefix())
					}
				}

				nextNode = children[partial.At(slice.Len())]
				if nextNode == nil {
					return EmptyValue, nil
				}

				partial = partial.Mid(slice.Len() + 1)
				keyNibbles += slice.Len() + 1
			}

			switch n := nextNode.(type) {
			case node.Hash:
				nextHash := node.DecodeHash(n.Value, l.layout.Hasher())
				if nextHash == nil {
					return nil, InvalidHash
				}
				hash = *nextHash
			case node.Inline:
				nodeData = &n.Value
			}
		}
		depth++
	}
}

func (l Lookup[H]) loadValue(value node.Value, prefix nibble.Prefix) ([]byte, error) {
	switch v := value.(type) {
	case node.InlineValue:
		return v.Bytes, nil
	case node.NodeValue:
		hash := l.layout.Hasher().FromBytes(v.Bytes)
		bytes := l.db.Get(hash, prefix)
		if bytes == nil {
			return nil, ErrIncompleteDB
		}
		return *bytes, nil
	default:
		panic("unknown value type")
	}
}
