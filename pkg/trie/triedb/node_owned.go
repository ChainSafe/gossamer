// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

// Value representation used in [CachedNode] interface constraint
type ValueOwnedTypes[H hash.Hash] interface {
	ValueOwnedInline[H] | ValueOwnedNode[H]
	ValueOwned[H]
}

// Value representation used in [CachedNode]
type ValueOwned[H any] interface {
	data() []byte // nil means there is no data
	dataHash() *H
	EncodedValue() codec.EncodedValue
}

type (
	// Value bytes as stored in a trie node and its hash.
	ValueOwnedInline[H hash.Hash] struct {
		Value []byte
		Hash  H
	}
	// Hash stored in a trie node.
	ValueOwnedNode[H hash.Hash] struct {
		Hash H
	}
)

func (vo ValueOwnedInline[H]) data() []byte                     { return vo.Value } //nolint:unused
func (vo ValueOwnedNode[H]) data() []byte                       { return nil }      //nolint:unused
func (vo ValueOwnedInline[H]) dataHash() *H                     { return &vo.Hash } //nolint:unused
func (vo ValueOwnedNode[H]) dataHash() *H                       { return &vo.Hash } //nolint:unused
func (vo ValueOwnedInline[H]) EncodedValue() codec.EncodedValue { return codec.InlineValue(vo.Value) }
func (vo ValueOwnedNode[H]) EncodedValue() codec.EncodedValue   { return codec.HashedValue[H](vo) }

func newValueOwnedFromEncodedValue[H hash.Hash, Hasher hash.Hasher[H]](encVal codec.EncodedValue) ValueOwned[H] {
	switch encVal := encVal.(type) {
	case codec.InlineValue:
		return ValueOwnedInline[H]{
			Value: encVal,
			Hash:  (*(new(Hasher))).Hash(encVal),
		}
	case codec.HashedValue[H]:
		return ValueOwnedNode[H](encVal)
	case nil:
		return nil
	default:
		panic("unreachable")
	}
}

// Cached version of [codec.MerkleValue] interface constraint.
type NodeHandleOwnedTypes[H hash.Hash] interface {
	NodeHandleOwnedHash[H] | NodeHandleOwnedInline[H]
}

// Cached version of [codec.MerkleValue].
type NodeHandleOwned interface {
	/// Returns [NodeHandleOwned] as a [ChildReference].
	ChildReference() ChildReference
}

type (
	NodeHandleOwnedHash[H hash.Hash] struct {
		Hash H
	}
	NodeHandleOwnedInline[H hash.Hash] struct {
		CachedNode[H]
	}
)

func (nho NodeHandleOwnedHash[H]) ChildReference() ChildReference {
	return HashChildReference[H](nho)
}
func (nho NodeHandleOwnedInline[H]) ChildReference() ChildReference {
	encoded := nho.CachedNode.encoded()
	store := (*new(H))
	if len(encoded) > store.Length() {
		panic("Invalid inline node handle")
	}
	return InlineChildReference(encoded)
}

func newNodeHandleOwnedFromMerkleValue[H hash.Hash, Hasher hash.Hasher[H]](
	mv codec.MerkleValue,
) (NodeHandleOwned, error) {
	switch mv := mv.(type) {
	case codec.HashedNode[H]:
		return NodeHandleOwnedHash[H](mv), nil
	case codec.InlineNode:
		buf := bytes.NewBuffer(mv)
		node, err := codec.Decode[H](buf)
		if err != nil {
			return nil, err
		}
		nodeOwned, err := newCachedNodeFromNode[H, Hasher](node)
		if err != nil {
			return nil, err
		}
		return NodeHandleOwnedInline[H]{nodeOwned}, nil
	default:
		panic("unreachable")
	}
}

type child[H any] struct {
	nibble *uint8
	NodeHandleOwned
}

// Cached nodes interface constraint.
type CachedNodeTypes[H hash.Hash] interface {
	EmptyCachedNode[H] | LeafCachedNode[H] | BranchCachedNode[H] | ValueCachedNode[H]
	CachedNode[H]
}

// Cached nodes.
type CachedNode[H any] interface {
	data() []byte // nil means there is no data
	dataHash() *H
	children() []child[H]
	partialKey() *nibbles.NibbleSlice
	encoded() []byte
}

type (
	// Empty trie node; could be an empty root or an empty branch entry.
	EmptyCachedNode[H hash.Hash] struct{}
	// Leaf node; has key slice and value. Value may not be empty.
	LeafCachedNode[H any] struct {
		PartialKey nibbles.NibbleSlice
		Value      ValueOwned[H]
	}
	// Branch node; has slice of child nodes (each possibly null)
	// and an optional value.
	BranchCachedNode[H any] struct {
		PartialKey nibbles.NibbleSlice
		Children   [codec.ChildrenCapacity]NodeHandleOwned // can be nil to represent no child
		Value      ValueOwned[H]
	}
	// Node that represents a value.
	//
	// This variant is only constructed when working with a [TrieCache]. It is only
	// used to cache a raw value.
	ValueCachedNode[H any] struct {
		Value []byte
		Hash  H
	}
)

func (EmptyCachedNode[H]) data() []byte   { return nil }             //nolint:unused
func (no LeafCachedNode[H]) data() []byte { return no.Value.data() } //nolint:unused
func (no BranchCachedNode[H]) data() []byte { //nolint:unused
	if no.Value != nil {
		return no.Value.data()
	}
	return nil
}
func (no ValueCachedNode[H]) data() []byte { return no.Value } //nolint:unused

func (EmptyCachedNode[H]) dataHash() *H   { return nil }                 //nolint:unused
func (no LeafCachedNode[H]) dataHash() *H { return no.Value.dataHash() } //nolint:unused
func (no BranchCachedNode[H]) dataHash() *H { //nolint:unused
	if no.Value != nil {
		return no.Value.dataHash()
	}
	return nil
}
func (no ValueCachedNode[H]) dataHash() *H { return &no.Hash } //nolint:unused

func (EmptyCachedNode[H]) children() []child[H]   { return nil } //nolint:unused
func (no LeafCachedNode[H]) children() []child[H] { return nil } //nolint:unused
func (no BranchCachedNode[H]) children() []child[H] { //nolint:unused
	r := []child[H]{}
	for i, ch := range no.Children {
		if ch != nil {
			nibble := uint8(i) //nolint:gosec
			r = append(r, child[H]{
				nibble:          &nibble,
				NodeHandleOwned: ch,
			})
		}
	}
	return r
}
func (no ValueCachedNode[H]) children() []child[H] { return nil } //nolint:unused

func (EmptyCachedNode[H]) partialKey() *nibbles.NibbleSlice     { return nil }            //nolint:unused
func (no LeafCachedNode[H]) partialKey() *nibbles.NibbleSlice   { return &no.PartialKey } //nolint:unused
func (no BranchCachedNode[H]) partialKey() *nibbles.NibbleSlice { return &no.PartialKey } //nolint:unused
func (no ValueCachedNode[H]) partialKey() *nibbles.NibbleSlice  { return nil }            //nolint:unused

func (EmptyCachedNode[H]) encoded() []byte { return []byte{EmptyTrieBytes} } //nolint:unused
func (no LeafCachedNode[H]) encoded() []byte { //nolint:unused
	encodingBuffer := bytes.NewBuffer(nil)
	err := NewEncodedLeaf(no.PartialKey.Right(), no.PartialKey.Len(), no.Value.EncodedValue(), encodingBuffer)
	if err != nil {
		panic(err)
	}
	return encodingBuffer.Bytes()
}
func (no BranchCachedNode[H]) encoded() []byte { //nolint:unused
	encodingBuffer := bytes.NewBuffer(nil)
	children := [16]ChildReference{}
	for i, ch := range no.Children {
		if ch == nil {
			continue
		}
		children[i] = ch.ChildReference()
	}
	var encodedVal codec.EncodedValue
	if no.Value != nil {
		encodedVal = no.Value.EncodedValue()
	}
	err := NewEncodedBranch(
		no.PartialKey.Right(),
		no.PartialKey.Len(),
		children,
		encodedVal,
		encodingBuffer)
	if err != nil {
		panic(err)
	}
	return encodingBuffer.Bytes()
}
func (no ValueCachedNode[H]) encoded() []byte { return no.Value } //nolint:unused

func newCachedNodeFromNode[H hash.Hash, Hasher hash.Hasher[H]](n codec.EncodedNode) (CachedNode[H], error) {
	switch n := n.(type) {
	case codec.Empty:
		return EmptyCachedNode[H]{}, nil
	case codec.Leaf:
		return LeafCachedNode[H]{
			PartialKey: nibbles.NewNibbleSliceFromNibbles(n.PartialKey),
			Value:      newValueOwnedFromEncodedValue[H, Hasher](n.Value),
		}, nil
	case codec.Branch:
		var childrenOwned [codec.ChildrenCapacity]NodeHandleOwned
		for i, child := range n.Children {
			if child == nil {
				continue
			}
			var err error
			childrenOwned[i], err = newNodeHandleOwnedFromMerkleValue[H, Hasher](child)
			if err != nil {
				return nil, err
			}
		}
		return BranchCachedNode[H]{
			PartialKey: nibbles.NewNibbleSliceFromNibbles(n.PartialKey),
			Children:   childrenOwned,
			Value:      newValueOwnedFromEncodedValue[H, Hasher](n.Value),
		}, nil
	default:
		panic("unreachable")
	}
}
