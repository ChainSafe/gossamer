// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

var EmptyValue = []byte{}

type Lookup[Hash HashOut] struct {
	db       hashdb.HashDB[Hash, DBValue]
	hash     Hash
	cache    TrieCache[Hash]
	recorder TrieRecorder[Hash]
	layout   TrieLayout[Hash]
}

func NewLookup[H HashOut](
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
	keyNibbles := uint(0)

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

			var nextNode NodeHandle = nil

			switch node := decodedNode.(type) {
			case Empty:
				return EmptyValue, nil
			case Leaf:
				// If leaf and matches return value
				if partial.Eq(&node.partialKey) {
					return l.loadValue(node.value, nibbleKey.OriginalDataAsPrefix())
				}
				return EmptyValue, nil
			case NibbledBranch:
				slice := node.partialKey
				children := node.children
				value := node.value
				// Get next node
				if !partial.StartsWith(&slice) {
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

			switch node := nextNode.(type) {
			case Hash:
				nextHash := DecodeHash(node.value, l.layout.Hasher())
				if nextHash == nil {
					return nil, InvalidHash
				}
				hash = *nextHash
				break
			case Inline:
				nodeData = &node.value
			}
		}
		depth++
	}
}

func (l Lookup[H]) loadValue(value Value, prefix hashdb.Prefix) ([]byte, error) {
	switch v := value.(type) {
	case InlineValue:
		return v.bytes, nil
	case NodeValue:
		hash := l.layout.Hasher().FromBytes(v.bytes)
		bytes := l.db.Get(hash, prefix)
		if bytes == nil {
			return nil, ErrIncompleteDB
		}
		return *bytes, nil
	default:
		panic("unknown value type")
	}
}
