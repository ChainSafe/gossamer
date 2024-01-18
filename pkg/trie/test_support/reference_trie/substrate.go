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

func (self NodeCodec[H]) Hasher() hashdb.Hasher[H] {
	return self.hasher
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

func (self NodeCodec[H]) decodePlan(data []byte) (node.NodePlan, error) {
	panic("Implement me")
}

func (self NodeCodec[H]) Decode(data []byte) (node.Node, error) {
	plan, err := self.decodePlan(data)
	if err != nil {
		return nil, err
	}
	return plan.Build(data), nil
}

func NewNodeCodecForKeccak() NodeCodec[keccak_hasher.KeccakHash] {
	return NodeCodec[keccak_hasher.KeccakHash]{hasher}
}

var _ node.NodeCodec[keccak_hasher.KeccakHash] = NodeCodec[keccak_hasher.KeccakHash]{}

type LayoutV0[H hashdb.HashOut] struct {
	codec node.NodeCodec[H]
}

func (l LayoutV0[H]) AllowEmpty() bool {
	return true
}

func (l LayoutV0[H]) MaxInlineValue() *uint {
	return nil
}

func (l LayoutV0[H]) Codec() node.NodeCodec[H] {
	return l.codec
}

var V0Layout triedb.TrieLayout[keccak_hasher.KeccakHash] = LayoutV0[keccak_hasher.KeccakHash]{NewNodeCodecForKeccak()}
