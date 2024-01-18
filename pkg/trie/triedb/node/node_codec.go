// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type ChildReference[H hashdb.HashOut] interface {
	Type() string
}

type (
	ChildReferenceHash[H hashdb.HashOut] struct {
		hash H
	}
	ChildReferenceInline[H hashdb.HashOut] struct {
		hash   H
		length uint
	}
)

func (c ChildReferenceHash[H]) Type() string   { return "Hash" }
func (c ChildReferenceInline[H]) Type() string { return "Inline" }

type NodeCodec[H hashdb.HashOut] interface {
	HashedNullNode() H
	Hasher() hashdb.Hasher[H]
	EmptyNode() []byte
	LeafNode(partialKey nibble.NibbleSlice, numberNibble int, value Value) []byte
	BranchNodeNibbled(
		partialKey nibble.NibbleSlice,
		numberNibble int,
		children [16]ChildReference[H],
		value *Value,
	) []byte
	Decode(data []byte) (Node, error)
}

func EncodeNodeOwned[H hashdb.HashOut](node NodeOwned[H], codec NodeCodec[H]) []byte {
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
