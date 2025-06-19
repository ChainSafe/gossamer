// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	trieroot "github.com/ChainSafe/gossamer/internal/primitives/trie/trie-root"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// TrieStream is the codec implementation of [trieroot.TrieStream].
type TrieStream[Hasher hashdb.Hasher[H], H hashdb.Hash] struct {
	// Current node buffer.
	buffer []byte
}

func (*TrieStream[Hasher, H]) New() trieroot.TrieStream {
	return NewTrieStream[Hasher, H]()
}
func NewTrieStream[Hasher hashdb.Hasher[H], H hashdb.Hash]() *TrieStream[Hasher, H] {
	return &TrieStream[Hasher, H]{buffer: make([]byte, 0)}
}

func (ts *TrieStream[Hasher, H]) AppendEmptyData() {
	ts.buffer = append(ts.buffer, emptyTrie)
}
func (ts *TrieStream[Hasher, H]) AppendLeaf(key []byte, value trieroot.Value) {
	var kind nodeKind
	switch value.(type) {
	case trieroot.InlineValue:
		kind = nodeKindLeaf
	case trieroot.NodeValue:
		kind = nodeKindHashedValueLeaf
	default:
		panic("unreachable")
	}
	ts.buffer = append(ts.buffer, fuseNibblesNode(key, kind)...)
	switch value := value.(type) {
	case trieroot.InlineValue:
		length := scale.MustMarshal(len(value))
		ts.buffer = append(ts.buffer, length...)
		ts.buffer = append(ts.buffer, value...)
	case trieroot.NodeValue:
		ts.buffer = append(ts.buffer, value...)
	default:
		panic("unreachable")
	}
}
func (ts *TrieStream[Hasher, H]) BeginBranch(maybePartial []byte, maybeValue trieroot.Value, hasChildren []bool) {
	if maybePartial != nil {
		var kind nodeKind
		if maybeValue == nil {
			kind = nodeKindBranchNoValue
		} else {
			switch maybeValue.(type) {
			case trieroot.InlineValue:
				kind = nodeKindBranchWithValue
			case trieroot.NodeValue:
				kind = nodeKindHashedValueBranch
			default:
				panic("unreachable")
			}
		}

		ts.buffer = append(ts.buffer, fuseNibblesNode(maybePartial, kind)...)
		a, b := branchNodeBitMask(hasChildren)
		ts.buffer = append(ts.buffer, a, b)
	} else {
		panic("trie stream codec only for no extension trie")
	}
	if maybeValue == nil {
		return
	}
	switch value := maybeValue.(type) {
	case trieroot.InlineValue:
		length := scale.MustMarshal(len(value))
		ts.buffer = append(ts.buffer, length...)
		ts.buffer = append(ts.buffer, value...)
	case trieroot.NodeValue:
		ts.buffer = append(ts.buffer, value...)
	default:
		panic("unreachable")
	}
}

func (*TrieStream[Hasher, H]) AppendExtension(key []byte) {
	panic("trie stream codec only for no extension trie")
}

func (ts *TrieStream[Hasher, H]) AppendSubstream(other trieroot.TrieStream) {
	data := other.Out()
	if len(data) <= 31 {
		ts.buffer = append(ts.buffer, scale.MustMarshal(data)...)
	} else {
		hash := (*new(Hasher)).Hash(data)
		ts.buffer = append(ts.buffer, scale.MustMarshal(hash.Bytes())...)
	}
}

func (ts *TrieStream[Hasher, H]) Out() []byte {
	return ts.buffer
}

func (*TrieStream[Hasher, H]) AppendEmptyChild()              {}
func (*TrieStream[Hasher, H]) EndBranch(value trieroot.Value) {}

func branchNodeBitMask(hasChildren []bool) (uint8, uint8) {
	var (
		bitmap uint16
		cursor uint16 = 1
	)
	for _, v := range hasChildren {
		if v {
			bitmap |= cursor
		}
		cursor <<= 1
	}
	return uint8(bitmap % 256), uint8(bitmap / 256)
}

// Create a leaf/branch node, encoding a number of nibbles.
func fuseNibblesNode(nibbles []byte, kind nodeKind) []byte {
	// let size = nibbles.len();
	size := uint(len(nibbles))
	var start []byte
	switch kind {
	case nodeKindLeaf:
		start = sizeAndPrefixIterator(size, leafPrefixMask, 2)
	case nodeKindBranchNoValue:
		start = sizeAndPrefixIterator(size, branchWithoutMask, 2)
	case nodeKindBranchWithValue:
		start = sizeAndPrefixIterator(size, branchWithMask, 2)
	case nodeKindHashedValueLeaf:
		start = sizeAndPrefixIterator(size, altHashingLeafPrefixMask, 3)
	case nodeKindHashedValueBranch:
		start = sizeAndPrefixIterator(size, altHashingBranchWithMask, 4)
	}
	if len(nibbles)%2 == 1 {
		start = append(start, nibbles[0])
	}
	begin := len(nibbles) % 2
	for i := begin; i < len(nibbles); i += 2 {
		start = append(start, nibbles[i]<<4|nibbles[i+1])
	}
	return start
}
