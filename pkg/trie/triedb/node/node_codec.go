// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type ChildReference[H HashOut] interface {
	Type() string
}

type (
	ChildReferenceHash[H HashOut] struct {
		hash H
	}
	ChildReferenceInline[H HashOut] struct {
		hash   H
		length uint
	}
)

func (c ChildReferenceHash[H]) Type() string   { return "Hash" }
func (c ChildReferenceInline[H]) Type() string { return "Inline" }

type HashOut interface {
	comparable
	ToBytes() []byte
}

type NodeCodec[H HashOut] interface {
	HashedNullNode() H
	Hasher() hashdb.Hasher[H]
	EmptyNode() []byte
	LeafNode(partialKey nibble.NibbleSlice, numberNibble uint, value Value) []byte
	BranchNodeNibbled(partialKey nibble.NibbleSlice, numberNibble uint, children [16]ChildReference[H], value *Value) []byte
	Decode(data []byte) (Node[H], error)
}

func EncodeNodeOwned[H HashOut](node NodeOwned[H], codec NodeCodec[H]) []byte {
	switch n := node.(type) {
	case NodeOwnedEmpty:
		return codec.EmptyNode()
	case NodeOwnedLeaf[H]:
		return codec.LeafNode(n.PartialKey, n.PartialKey.Len(), n.Value)
	case NodeOwnedNibbledBranch[H]:
		var value = n.Value.AsValue()

		var children [16]ChildReference[H]
		for i, c := range n.EncodedChildren {
			if c != nil {
				children[i] = c.AsChildReference(codec)
			}
		}
		return codec.BranchNodeNibbled(n.PartialKey, n.PartialKey.Len(), children, &value)
	default:
		panic(fmt.Sprintf("unknown node type %s", n.Type()))
	}
}
