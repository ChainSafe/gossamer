// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"errors"

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
type Node[H hashdb.HashOut] interface {
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

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandleOwned[H hashdb.HashOut] interface {
	Type() string
	AsChildReference(codec NodeCodec[H]) ChildReference[H]
}
type (
	NodeHandleOwnedHash[H hashdb.HashOut] struct {
		Hash H
	}
	NodeHandleOwnedInline[H hashdb.HashOut] struct {
		Node NodeOwned[H]
	}
)

func (h NodeHandleOwnedHash[H]) Type() string { return "Hash" }
func (h NodeHandleOwnedHash[H]) AsChildReference(codec NodeCodec[H]) ChildReference[H] {
	return ChildReferenceHash[H]{hash: h.Hash}
}
func (h NodeHandleOwnedInline[H]) Type() string { return "Inline" }
func (h NodeHandleOwnedInline[H]) AsChildReference(codec NodeCodec[H]) ChildReference[H] {
	encoded := EncodeNodeOwned(h.Node, codec)
	if len(encoded) > codec.Hasher().Length() {
		panic("Invalid inline node handle")
	}
	return ChildReferenceInline[H]{hash: codec.Hasher().FromBytes(encoded), length: uint(len(encoded))}
}

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandle interface {
	Type() string
}
type (
	Hash struct {
		Data []byte
	}
	Inline struct {
		Data []byte
	}
)

func (h Hash) Type() string   { return "Hash" }
func (h Inline) Type() string { return "Inline" }

func DecodeHash[H hashdb.HashOut](hasher hashdb.Hasher[H], data []byte) (H, error) {
	if len(data) != hasher.Length() {
		return hasher.FromBytes([]byte{}), errors.New("decoding hash")
	}
	return hasher.FromBytes(data), nil
}
