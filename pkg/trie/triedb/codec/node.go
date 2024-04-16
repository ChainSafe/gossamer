// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

const ChildrenCapacity = 16

// MerkleValue is a helper enum to differentiate between inline and hashed nodes
// https://spec.polkadot.network/chap-state#defn-merkle-value
type MerkleValue interface {
	isMerkleValue()
	IsHashed() bool
}

type (
	// Value bytes as stored in a trie node
	InlineNode struct {
		Data []byte
	}
	// Containing a hash used to lookup in db for real value
	HashedNode struct {
		Data []byte
	}
)

func (InlineNode) isMerkleValue() {}
func (InlineNode) IsHashed() bool { return false }
func (HashedNode) isMerkleValue() {}
func (HashedNode) IsHashed() bool { return true }

func NewInlineNode(data []byte) MerkleValue {
	return InlineNode{Data: data}
}

func NewHashedNode(data []byte) MerkleValue {
	return HashedNode{Data: data}
}

// NodeValue is a helper enum to differentiate between inline and hashed values
type NodeValue interface {
	isNodeValue()
}

type (
	// Value bytes as stored in a trie node
	InlineValue struct {
		Data []byte
	}
	// Containing a hash used to lookup in db for real value
	HashedValue struct {
		Data []byte
	}
)

func (InlineValue) isNodeValue() {}
func (HashedValue) isNodeValue() {}

func NewInlineValue(data []byte) NodeValue {
	return InlineValue{Data: data}
}

func NewHashedValue(data []byte) NodeValue {
	return HashedValue{Data: data}
}

// Node is the representation of a decoded node
type Node interface {
	isNode()
}

type (
	// Empty node
	Empty struct{}
	// Leaf always contains values
	Leaf struct {
		PartialKey []byte
		Value      NodeValue
	}
	// Branch could has or not has values
	Branch struct {
		PartialKey []byte
		Children   [16]MerkleValue
		Value      NodeValue
	}
)

func (Empty) isNode()  {}
func (Leaf) isNode()   {}
func (Branch) isNode() {}
