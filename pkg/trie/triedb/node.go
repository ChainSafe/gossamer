package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

// Nodes

// Node is a trie node
type Node[H HashOut] interface {
	Type() string
}

type (
	// NodeEmptyNode represents an empty node
	Empty struct{}
	// NodeLeaf represents a leaf node
	Leaf[H HashOut] struct {
		partialKey nibble.NibbleVec
		value      Value[H]
	}
	// NodeNibbledBranch represents a branch node
	NibbledBranch[H HashOut] struct {
		partialKey nibble.NibbleVec
		childs     [nibble.NibbleLength]NodeHandle[H]
		value      Value[H]
	}
)

func (n Empty) Type() string            { return "Empty" }
func (n Leaf[H]) Type() string          { return "Leaf" }
func (n NibbledBranch[H]) Type() string { return "NibbledBranch" }

// Value is a trie node value
type Value[H HashOut] interface {
	Type() string
	Hash() H
	Value() []byte
}
type (
	// InlineNodeValue if the value is inlined we can get the bytes and the hash of the value
	InlineValue[H HashOut] struct {
		bytes []byte
		hash  H
	}
	// HashedNodeValue is a trie node pointer to a hashed node
	HashedValue[H comparable] struct {
		hash H
	}
)

func (v InlineValue[H]) Type() string  { return "Inline" }
func (v InlineValue[H]) Hash() H       { return v.hash }
func (v InlineValue[H]) Value() []byte { return v.bytes }
func (v HashedValue[H]) Type() string  { return "Node" }
func (v HashedValue[H]) Hash() H       { return v.hash }
func (v HashedValue[H]) Value() []byte { return nil }

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandle[H HashOut] interface {
	Type() string
}
type (
	HashNodeHandle[H HashOut] struct {
		value H
	}
	InlineNodeHandle[H HashOut] struct {
		node Node[H]
	}
)

func (h HashNodeHandle[H]) Type() string   { return "Hash" }
func (h InlineNodeHandle[H]) Type() string { return "Inline" }
