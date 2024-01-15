// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package reference_trie

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/test_support/keccak_hasher"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

const FirstPrefix = 0b_00 << 6
const EmptyTree = FirstPrefix | (0b_00 << 4)

var hasher = keccak_hasher.NewKeccakHasher()

type NodeCodec[H hashdb.HashOut] struct {
	hasher hashdb.Hasher[H]
}

func (self NodeCodec[H]) HashedNullNode() H {
	return self.hasher.Hash(self.EmptyNode())
}

func (self NodeCodec[H]) Hasher() hashdb.Hasher[keccak_hasher.KeccakHash] {
	return hasher
}

func (self NodeCodec[H]) EmptyNode() []byte {
	return []byte{EmptyTree}
}

func (self NodeCodec[H]) LeafNode(partialKey nibble.NibbleSlice, numberNibble int, value node.Value) []byte {
	panic("Implement me")
}

func (self NodeCodec[H]) BranchNodeNibbled(
	partialKey nibble.NibbleSlice,
	numberNibble int,
	children [16]node.ChildReference[H],
	value *node.Value,
) []byte {
	panic("Implement me")
}

func (self NodeCodec[H]) Decode(data []byte) (node.Node[H], error) {
	panic("Implement me")
}

var _ node.NodeCodec[keccak_hasher.KeccakHash] = NodeCodec[keccak_hasher.KeccakHash]{}

type LayoutV0[H hashdb.HashOut] struct{}

func (l LayoutV0[H]) AllowEmpty() bool {
	return true
}

func (l LayoutV0[H]) MaxInlineValue() *uint {
	return nil
}

func (l LayoutV0[H]) Codec() node.NodeCodec[H] {
	panic("Implement me")
}

var V0Layout triedb.TrieLayout[hashdb.HashOut] = LayoutV0[hashdb.HashOut]{}
