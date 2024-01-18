// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package reference_trie

import (
	"errors"
	"io"

	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/test_support/keccak_hasher"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

const firstPrefix = 0b_00 << 6
const leafPrefixMask = 0b_01 << 6
const branchWithoutValueMask = 0b_10 << 6
const branchWithValueMask = 0b_11 << 6
const emptyTrie = firstPrefix | (0b_00 << 4)
const leafWithHashedValuePrefixMask = firstPrefix | (0b_1 << 5)
const branchWithHashedValuePrefixMask = firstPrefix | (0b_1 << 4)
const escapeCompactHeader = emptyTrie | 0b_00_01

var hasher = keccak_hasher.NewKeccakHasher()

type byteSliceInput struct {
	data   []byte
	offset int
}

func NewByteSliceInput(data []byte) byteSliceInput {
	return byteSliceInput{data, 0}
}

func (self byteSliceInput) Take(count int) (node.BytesRange, error) {
	if self.offset+count > len(self.data) {
		return node.BytesRange{}, errors.New("out of data")
	}

	res := node.BytesRange{self.offset, self.offset + count}
	self.offset += count
	return res, nil
}

type NodeHeader interface {
	Type() string
}

type (
	NullNodeHeader   struct{}
	BranchNodeHeader struct {
		hasValue    bool
		nibbleCount int
	}
	LeafNodeHeader struct {
		nibbleCount int
	}
	HashedValueBranchNodeHeader struct {
		nibbleCount int
	}
	HashedValueLeaf struct {
		nibbleCount int
	}
)

func (NullNodeHeader) Type() string              { return "Null" }
func (BranchNodeHeader) Type() string            { return "Branch" }
func (LeafNodeHeader) Type() string              { return "Leaf" }
func (HashedValueBranchNodeHeader) Type() string { return "HashedValueBranch" }
func (HashedValueLeaf) Type() string             { return "HashedValueLeaf" }

func headerContainsHashedValues(header NodeHeader) bool {
	switch header.(type) {
	case HashedValueBranchNodeHeader, HashedValueLeaf:
		return true
	default:
		return false
	}
}

// Decode nibble count from stream input and header byte
func decodeSize(first byte, input io.Reader, prefixMask int) (int, error) {
	maxValue := byte(255) >> prefixMask
	result := (first & maxValue)
	if result < maxValue {
		return int(result), nil
	}
	result -= 1
	for {
		b := make([]byte, 1)
		_, err := input.Read(b)
		if err != nil {
			return -1, err
		}
		n := int(b[0])
		if n < 255 {
			return int(result) + n + 1, nil
		}
		result += 255
	}
}

// DecodeHeader decodes a node header from a stream input
func DecodeHeader(input io.Reader) (NodeHeader, error) {
	b := make([]byte, 1)
	_, err := input.Read(b)
	if err != nil {
		return nil, err
	}
	i := b[0]

	if i == emptyTrie {
		return NullNodeHeader{}, nil
	}

	mask := i & (0b11 << 6)

	var (
		size int
		node NodeHeader
	)

	switch mask {
	case leafPrefixMask:
		size, err = decodeSize(i, input, 2)
		node = LeafNodeHeader{size}
	case branchWithValueMask:
		size, err = decodeSize(i, input, 2)
		node = BranchNodeHeader{true, size}
	case branchWithoutValueMask:
		size, err = decodeSize(i, input, 2)
		node = BranchNodeHeader{false, size}
	case emptyTrie:
		if i&(0b111<<5) == leafWithHashedValuePrefixMask {
			size, err = decodeSize(i, input, 3)
			node = HashedValueLeaf{size}
		} else if i&(0b1111<<4) == branchWithHashedValuePrefixMask {
			size, err = decodeSize(i, input, 4)
			node = HashedValueBranchNodeHeader{size}
		} else {
			err = errors.New("invalid header")
		}
	default:
		panic("unreachable")
	}

	if err != nil {
		return nil, err
	}
	return node, err
}

// NodeCodec is the node codec configuration used in substrate
type NodeCodec[H hashdb.HashOut] struct {
	hasher hashdb.Hasher[H]
}

// HashedNullNode returns the hash of an empty node
func (self NodeCodec[H]) HashedNullNode() H {
	return self.hasher.Hash(self.EmptyNode())
}

// Hasher returns the hasher used for this codec
func (self NodeCodec[H]) Hasher() hashdb.Hasher[H] {
	return self.hasher
}

// EmptyNode returns an empty node
func (self NodeCodec[H]) EmptyNode() []byte {
	return []byte{emptyTrie}
}

// LeafNode encodes a leaf node
func (self NodeCodec[H]) LeafNode(partialKey nibble.NibbleSlice, numberNibble int, value node.Value) []byte {
	panic("Implement me")
}

// BranchNodeNibbled encodes a branch node
func (self NodeCodec[H]) BranchNodeNibbled(
	partialKey nibble.NibbleSlice,
	numberNibble int,
	children [16]node.ChildReference[H],
	value *node.Value,
) []byte {
	panic("Implement me")
}

func (self NodeCodec[H]) decodePlan(data []byte) (node.NodePlan, error) {
	//input := NewByteSliceInput(data)

	return nil, nil
}

// Decode decodes bytes to a Node
func (self NodeCodec[H]) Decode(data []byte) (node.Node, error) {
	plan, err := self.decodePlan(data)
	if err != nil {
		return nil, err
	}
	return plan.Build(data), nil
}

func NewNodeCodecForKeccak() NodeCodec[keccak_hasher.KeccakHash] {
	return NodeCodec[keccak_hasher.KeccakHash]{hasher}
}

var SubstrateNodeCodec node.NodeCodec[keccak_hasher.KeccakHash] = NodeCodec[keccak_hasher.KeccakHash]{}

type LayoutV0[H hashdb.HashOut] struct {
	codec node.NodeCodec[H]
}

func (l LayoutV0[H]) AllowEmpty() bool {
	return true
}

func (l LayoutV0[H]) MaxInlineValue() *uint {
	return nil
}

func (l LayoutV0[H]) Codec() node.NodeCodec[H] {
	return l.codec
}

var V0Layout triedb.TrieLayout[keccak_hasher.KeccakHash] = LayoutV0[keccak_hasher.KeccakHash]{NewNodeCodecForKeccak()}
