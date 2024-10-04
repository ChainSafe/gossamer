// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

const ChildrenCapacity = 16

// MerkleValue is a helper enum to differentiate between inline and hashed nodes
// https://spec.polkadot.network/chap-state#defn-merkle-value
type MerkleValue interface {
	IsHashed() bool
}

type (
	// InlineNode contains bytes of the encoded node data
	InlineNode []byte
	// HashedNode contains a hash used to lookup in db for encoded node data
	HashedNode[H any] struct{ Hash H }
)

func (InlineNode) IsHashed() bool    { return false }
func (HashedNode[H]) IsHashed() bool { return true }

// EncodedValue is a helper enum to differentiate between inline and hashed values
type EncodedValue interface {
	IsHashed() bool
	Write(writer io.Writer) error
}

type (
	// InlineValue contains bytes for the value in this node
	InlineValue []byte
	// HashedValue contains a hash used to lookup in db for real value
	HashedValue[H hash.Hash] struct {
		Hash H
	}
)

func (InlineValue) IsHashed() bool { return false }
func (v InlineValue) Write(writer io.Writer) error {
	encoder := scale.NewEncoder(writer)
	err := encoder.Encode(v)
	if err != nil {
		return fmt.Errorf("scale encoding storage value: %w", err)
	}
	return nil
}

func (HashedValue[H]) IsHashed() bool { return true }
func (v HashedValue[H]) Write(writer io.Writer) error {
	_, err := writer.Write(v.Hash.Bytes())
	if err != nil {
		return fmt.Errorf("writing hashed storage value: %w", err)
	}
	return nil
}

// EncodedNode is the object representation of a encoded node
type EncodedNode interface {
	GetPartialKey() *nibbles.Nibbles
	GetValue() EncodedValue
}

type (
	// Empty node
	Empty struct{}
	// Leaf always contains values
	Leaf struct {
		PartialKey nibbles.Nibbles
		Value      EncodedValue
	}
	// Branch could has or not has values
	Branch struct {
		PartialKey nibbles.Nibbles
		Children   [ChildrenCapacity]MerkleValue
		Value      EncodedValue
	}
)

func (Empty) GetPartialKey() *nibbles.Nibbles    { return nil }
func (Empty) GetValue() EncodedValue             { return nil }
func (l Leaf) GetPartialKey() *nibbles.Nibbles   { return &l.PartialKey }
func (l Leaf) GetValue() EncodedValue            { return l.Value }
func (b Branch) GetPartialKey() *nibbles.Nibbles { return &b.PartialKey }
func (b Branch) GetValue() EncodedValue          { return b.Value }

// NodeKind is an enum to represent the different types of nodes (Leaf, Branch, etc.)
type NodeKind int

const (
	LeafNode NodeKind = iota
	BranchWithoutValue
	BranchWithValue
	LeafWithHashedValue
	BranchWithHashedValue
)

func EncodeHeader(partialKey []byte, partialKeyLength uint, kind NodeKind, writer io.Writer) (err error) {
	if partialKeyLength > uint(maxPartialKeyLength) {
		panic(fmt.Sprintf("partial key length is too big: %d", partialKeyLength))
	}

	// Merge variant byte and partial key length together
	var nodeVariant variant

	switch kind {
	case LeafNode:
		nodeVariant = leafVariant
	case LeafWithHashedValue:
		nodeVariant = leafWithHashedValueVariant
	case BranchWithoutValue:
		nodeVariant = branchVariant
	case BranchWithValue:
		nodeVariant = branchWithValueVariant
	case BranchWithHashedValue:
		nodeVariant = branchWithHashedValueVariant
	}

	buffer := make([]byte, 1)
	buffer[0] = nodeVariant.bits
	partialKeyLengthMask := nodeVariant.partialKeyLengthHeaderMask()

	if partialKeyLength < uint(partialKeyLengthMask) {
		// Partial key length fits in header byte
		buffer[0] |= byte(partialKeyLength)
		_, err = writer.Write(buffer)
		if err != nil {
			return err
		}
	} else {
		// Partial key length does not fit in header byte only
		buffer[0] |= partialKeyLengthMask
		partialKeyLength -= uint(partialKeyLengthMask)
		_, err = writer.Write(buffer)
		if err != nil {
			return err
		}

		for {
			buffer[0] = 255
			if partialKeyLength < 255 {
				buffer[0] = byte(partialKeyLength)
			}

			_, err = writer.Write(buffer)
			if err != nil {
				return err
			}

			partialKeyLength -= uint(buffer[0])

			if buffer[0] < 255 {
				break
			}
		}
	}

	_, err = writer.Write(partialKey)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	return nil
}

type ValueOwnedTypes[H any] interface {
	ValueOwnedInline[H] | ValueOwnedNode[H]
	ValueOwned
}
type ValueOwned interface {
	isValueOwned()
}

type (
	// Value bytes as stored in a trie node and its hash.
	ValueOwnedInline[H any] struct {
		Value []byte
		Hash  H
	}
	// Hash byte slice as stored in a trie node.
	ValueOwnedNode[H any] struct {
		Hash H
	}
)

func (ValueOwnedInline[H]) isValueOwned() {}
func (ValueOwnedNode[H]) isValueOwned()   {}

var (
	_ ValueOwned = ValueOwnedInline[string]{}
	_ ValueOwned = ValueOwnedNode[string]{}
)

func ValueOwnedFromEncodedValue[H hash.Hash, Hasher hash.Hasher[H]](encVal EncodedValue) ValueOwned {
	switch encVal := encVal.(type) {
	case InlineValue:
		return ValueOwnedInline[H]{
			Value: encVal,
			Hash:  (*(new(Hasher))).Hash(encVal),
		}
	case HashedValue[H]:
		return ValueOwnedNode[H](encVal)
	default:
		panic("unreachable")
	}
}

type NodeHandleOwnedTypes[H any] interface {
	NodeHandleOwnedHash[H] | NodeHandleOwnedInline[H]
}

type NodeHandleOwned interface {
	isNodeHandleOwned()
}

type (
	NodeHandleOwnedHash[H any] struct {
		Hash H
	}
	NodeHandleOwnedInline[H any] struct {
		NodeOwned
	}
)

func (NodeHandleOwnedHash[H]) isNodeHandleOwned()   {}
func (NodeHandleOwnedInline[H]) isNodeHandleOwned() {}

var (
	_ NodeHandleOwned = NodeHandleOwnedHash[string]{}
	_ NodeHandleOwned = NodeHandleOwnedInline[string]{}
)

func NodeHandleOwnedFromMerkleValue[H hash.Hash, Hasher hash.Hasher[H]](mv MerkleValue) (NodeHandleOwned, error) {
	switch mv := mv.(type) {
	case HashedNode[H]:
		return NodeHandleOwnedHash[H](mv), nil
	case InlineNode:
		buf := bytes.NewBuffer(mv)
		node, err := Decode[H](buf)
		if err != nil {
			return nil, err
		}
		nodeOwned, err := NodeOwnedFromNode[H, Hasher](node)
		if err != nil {
			return nil, err
		}
		return NodeHandleOwnedInline[H]{nodeOwned}, nil
	default:
		panic("unreachable")
	}
}

type NodeOwnedTypes[H any] interface {
	NodeOwnedEmpty | NodeOwnedLeaf[H] | NodeOwnedBranch[H] | NodeOwnedValue[H]
	NodeOwned
}
type NodeOwned interface {
	isNodeOwned()
}

type (
	// Null trie node; could be an empty root or an empty branch entry.
	NodeOwnedEmpty struct{}
	// Leaf node; has key slice and value. Value may not be empty.
	NodeOwnedLeaf[H any] struct {
		PartialKey nibbles.Nibbles
		Value      ValueOwned
	}
	// Branch node; has slice of child nodes (each possibly null)
	// and an optional immediate node data.
	NodeOwnedBranch[H any] struct {
		PartialKey nibbles.Nibbles
		Children   [ChildrenCapacity]NodeHandleOwned // can be nil to represent no child
		Value      ValueOwned
	}
	// Node that represents a value.
	//
	// This variant is only constructed when working with a [`crate::TrieCache`]. It is only
	// used to cache a raw value.
	NodeOwnedValue[H any] struct {
		Value []byte
		Hash  H
	}
)

func (NodeOwnedEmpty) isNodeOwned()     {}
func (NodeOwnedLeaf[H]) isNodeOwned()   {}
func (NodeOwnedBranch[H]) isNodeOwned() {}
func (NodeOwnedValue[H]) isNodeOwned()  {}

var (
	_ NodeOwned = NodeOwnedEmpty{}
	_ NodeOwned = NodeOwnedLeaf[string]{}
	_ NodeOwned = NodeOwnedBranch[string]{}
	_ NodeOwned = NodeOwnedValue[string]{}
)

func NodeOwnedFromNode[H hash.Hash, Hasher hash.Hasher[H]](n EncodedNode) (NodeOwned, error) {
	switch n := n.(type) {
	case Empty:
		return NodeOwnedEmpty{}, nil
	case Leaf:
		return NodeOwnedLeaf[H]{
			PartialKey: n.PartialKey,
			Value:      ValueOwnedFromEncodedValue[H, Hasher](n.Value),
		}, nil
	case Branch:
		var childrenOwned [ChildrenCapacity]NodeHandleOwned
		for i, child := range n.Children {
			if child == nil {
				continue
			}
			var err error
			childrenOwned[i], err = NodeHandleOwnedFromMerkleValue[H, Hasher](child)
			if err != nil {
				return nil, err
			}
		}
		return NodeOwnedBranch[H]{
			PartialKey: n.PartialKey,
			Children:   childrenOwned,
			Value:      ValueOwnedFromEncodedValue[H, Hasher](n.Value),
		}, nil
	default:
		panic("unreachable")
	}
}
