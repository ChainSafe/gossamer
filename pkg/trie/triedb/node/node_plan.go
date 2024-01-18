// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"

type BytesRange struct {
	Start int
	End   int
}

// A `NibbleSlicePlan` is a blueprint for decoding a nibble slice from a byte slice
type NibbleSlicePlan struct {
	bytes  BytesRange
	offset int
}

func NewNibbleSlicePlan(bytes BytesRange, offset int) NibbleSlicePlan {
	return NibbleSlicePlan{bytes, offset}
}

func (self NibbleSlicePlan) Len() int {
	return (self.bytes.End-self.bytes.Start)*nibble.NibblePerByte - self.offset
}

func (self NibbleSlicePlan) Build(data []byte) nibble.NibbleSlice {
	return *nibble.NewNibbleSliceWithOffset(data[self.bytes.Start:self.bytes.End], self.offset)
}

// A `NodeHandlePlan` is a decoding plan for constructing a `NodeHandle` from an encoded trie
type NodeHandlePlan interface {
	Type() string
	Build(data []byte) NodeHandle
}

type (
	NodeHandlePlanHash struct {
		bytes BytesRange
	}
	NodeHandlePlanInline struct {
		bytes BytesRange
	}
)

func (NodeHandlePlanHash) Type() string { return "Hash" }
func (n NodeHandlePlanHash) Build(data []byte) NodeHandle {
	return Hash{data[n.bytes.Start:n.bytes.End]}
}
func (NodeHandlePlanInline) Type() string { return "Inline" }
func (n NodeHandlePlanInline) Build(data []byte) NodeHandle {
	return Inline{data[n.bytes.Start:n.bytes.End]}
}

// Plan for value representation in `NodePlan`.
type ValuePlan interface {
	Type() string
	Build(data []byte) Value
}

type (
	// Range for byte representation in encoded node.
	ValuePlanInline struct {
		bytes BytesRange
	}
	// Range for hash in encoded node and original
	ValuePlanNode struct {
		bytes BytesRange
	}
)

func (ValuePlanInline) Type() string { return "Inline" }
func (n ValuePlanInline) Build(data []byte) Value {
	return InlineValue{data[n.bytes.Start:n.bytes.End]}
}
func (ValuePlanNode) Type() string { return "Node" }
func (n ValuePlanNode) Build(data []byte) Value {
	return NodeValue{data[n.bytes.Start:n.bytes.End]}
}

type NodePlan interface {
	Type() string
	Build(data []byte) Node
}

type (
	// Null trie node; could be an empty root or an empty branch entry
	NodePlanEmptyNode struct{}
	// Leaf node, has a partial key plan and value
	NodePlanLeafNode struct {
		partial NibbleSlicePlan
		value   ValuePlan
	}
	// Branch node
	NodePlanNibbledBranchNode struct {
		partial  NibbleSlicePlan
		value    ValuePlan
		children [nibble.NibbleLength]NodeHandlePlan
	}
)

func (NodePlanEmptyNode) Type() string { return "Empty" }
func (self NodePlanEmptyNode) Build(data []byte) Node {
	return Empty{}
}
func (NodePlanLeafNode) Type() string { return "Leaf" }
func (self NodePlanLeafNode) Build(data []byte) Node {
	return Leaf{
		PartialKey: self.partial.Build(data),
		Value:      self.value.Build(data),
	}
}
func (NodePlanNibbledBranchNode) Type() string { return "NibbledBranch" }
func (self NodePlanNibbledBranchNode) Build(data []byte) Node {
	children := [nibble.NibbleLength]NodeHandle{}
	for i := 0; i < nibble.NibbleLength; i++ {
		if self.children[i] != nil {
			children[i] = self.children[i].Build(data)
		}
	}
	var value Value
	if self.value != nil {
		value = self.value.Build(data)
	}
	return NibbledBranch{
		PartialKey: self.partial.Build(data),
		Children:   children,
		Value:      value,
	}
}
