// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

// Value
type Value interface {
	Type() string
}

type (
	// InlineNodeValue if the value is inlined we can get the bytes and the hash of the value
	InlineValue struct {
		Bytes []byte
	}
	// HashedNodeValue is a trie node pointer to a hashed node
	NodeValue struct {
		Bytes []byte
	}
)

func (v InlineValue) Type() string { return "Inline" }
func (v NodeValue) Type() string   { return "Node" }

// Nodes
type Node interface {
	Type() string
}

type (
	// NodeEmptyNode represents an empty node
	Empty struct{}
	// NodeLeaf represents a leaf node
	Leaf struct {
		PartialKey nibble.NibbleSlice
		Value      Value
	}
	// NodeNibbledBranch represents a branch node
	NibbledBranch struct {
		PartialKey nibble.NibbleSlice
		Children   [nibble.NibbleLength]NodeHandle
		Value      Value
	}
)

func (n Empty) Type() string         { return "Empty" }
func (n Leaf) Type() string          { return "Leaf" }
func (n NibbledBranch) Type() string { return "NibbledBranch" }

// NodeOwned is a trie node
type NodeOwned[H hashdb.HashOut] interface {
	Type() string
}

type (
	// NodeEmptyNode represents an empty node
	NodeOwnedEmpty struct{}
	// NodeLeaf represents a leaf node
	NodeOwnedLeaf[H hashdb.HashOut] struct {
		PartialKey nibble.NibbleSlice
		Value      ValueOwned[H]
	}
	// NodeNibbledBranch represents a branch node
	NodeOwnedNibbledBranch[H hashdb.HashOut] struct {
		PartialKey      nibble.NibbleSlice
		EncodedChildren [nibble.NibbleLength]NodeHandleOwned[H]
		Value           ValueOwned[H]
	}
)

func (n NodeOwnedEmpty) Type() string            { return "Empty" }
func (n NodeOwnedLeaf[H]) Type() string          { return "Leaf" }
func (n NodeOwnedNibbledBranch[H]) Type() string { return "NibbledBranch" }

// Value is a trie node value
type ValueOwned[H hashdb.HashOut] interface {
	Type() string
	AsValue() Value
}
type (
	// InlineNodeValue if the value is inlined we can get the bytes and the hash of the value
	InlineValueOwned[H hashdb.HashOut] struct {
		bytes []byte
		hash  H
	}
	// HashedNodeValue is a trie node pointer to a hashed node
	NodeValueOwned[H hashdb.HashOut] struct {
		hash H
	}
)

func (v InlineValueOwned[H]) Type() string { return "Inline" }
func (v InlineValueOwned[H]) AsValue() Value {
	return InlineValue{Bytes: v.bytes}
}
func (v NodeValueOwned[H]) Type() string { return "Node" }
func (v NodeValueOwned[H]) AsValue() Value {
	return NodeValue{Bytes: v.hash.Bytes()}
}
