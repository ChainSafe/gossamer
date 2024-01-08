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
type Node[H HashOut] interface {
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
type NodeOwned[H HashOut] interface {
	Type() string
}

type (
	// NodeEmptyNode represents an empty node
	NodeOwnedEmpty struct{}
	// NodeLeaf represents a leaf node
	NodeOwnedLeaf[H HashOut] struct {
		partialKey nibble.NibbleSlice
		value      ValueOwned[H]
	}
	// NodeNibbledBranch represents a branch node
	NodeOwnedNibbledBranch[H HashOut] struct {
		partialKey nibble.NibbleSlice
		children   [nibble.NibbleLength]NodeHandleOwned[H]
		value      ValueOwned[H]
	}
)

func (n NodeOwnedEmpty) Type() string            { return "Empty" }
func (n NodeOwnedLeaf[H]) Type() string          { return "Leaf" }
func (n NodeOwnedNibbledBranch[H]) Type() string { return "NibbledBranch" }

// Value is a trie node value
type ValueOwned[H HashOut] interface {
	Type() string
}
type (
	// InlineNodeValue if the value is inlined we can get the bytes and the hash of the value
	InlineValueOwned[H HashOut] struct {
		bytes []byte
		hash  H
	}
	// HashedNodeValue is a trie node pointer to a hashed node
	NodeValueOwned[H comparable] struct {
		hash H
	}
)

func (v InlineValueOwned[H]) Type() string { return "Inline" }
func (v NodeValueOwned[H]) Type() string   { return "Node" }

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandleOwned[H HashOut] interface {
	Type() string
}
type (
	NodeHandleOwnedHash[H HashOut] struct {
		ValueOwned H
	}
	NodeHandleOwnedInline[H HashOut] struct {
		node NodeOwned[H]
	}
)

func (h NodeHandleOwnedHash[H]) Type() string   { return "Hash" }
func (h NodeHandleOwnedInline[H]) Type() string { return "Inline" }

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandle interface {
	Type() string
}
type (
	Hash struct {
		Value []byte
	}
	Inline struct {
		Value []byte
	}
)

func (h Hash) Type() string   { return "Hash" }
func (h Inline) Type() string { return "Inline" }

func DecodeHash[H HashOut](data []byte, hasher hashdb.Hasher[H]) *H {
	if len(data) != hasher.Length() {
		return nil
	}
	hash := hasher.FromBytes(data)
	return &hash
}
