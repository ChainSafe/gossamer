// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type HashOut interface {
	comparable
	ToBytes() []byte
}

type NodeCodec[H HashOut] interface {
	HashedNullNode() H
	EmptyNode() []byte
	LeafNode(partialKey nibble.NibbleSlice, numberNibble uint, value Value) []byte
	BranchNodeNibbled(partialKey nibble.NibbleSlice, numberNibble uint, children [16]NodeHandle, value Value) []byte
	Decode(data []byte) (Node[H], error)
}

func EncodeNode[H HashOut](node Node[H], codec NodeCodec[H]) []byte {
	switch n := node.(type) {
	case Empty:
		return codec.EmptyNode()
	case Leaf:
		return codec.LeafNode(n.PartialKey, n.PartialKey.Len(), n.Value)
	case NibbledBranch:
		return codec.BranchNodeNibbled(n.PartialKey, n.PartialKey.Len(), n.Children, n.Value)
	default:
		panic(fmt.Sprintf("unknown node type %s", n.Type()))
	}
}
