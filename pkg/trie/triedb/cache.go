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
type CachedNodeValueTypes[H hash.Hash] interface {
	InlineCachedNodeValue[H] | NodeCachedNodeValue[H]
	CachedNodeValue[H]
}

// Value representation used in [CachedNode]
type CachedNodeValue[H any] interface {
	data() []byte // nil means there is no data
	dataHash() *H
	EncodedValue() codec.EncodedValue
}

type (
	// Value bytes as stored in a trie node and its hash.
	InlineCachedNodeValue[H hash.Hash] struct {
		Value []byte
		Hash  H
	}
	// Hash stored in a trie node.
	NodeCachedNodeValue[H hash.Hash] struct {
		Hash H
	}
)

func (vo InlineCachedNodeValue[H]) data() []byte { return vo.Value } //nolint:unused
func (vo NodeCachedNodeValue[H]) data() []byte   { return nil }      //nolint:unused
func (vo InlineCachedNodeValue[H]) dataHash() *H { return &vo.Hash } //nolint:unused
func (vo NodeCachedNodeValue[H]) dataHash() *H   { return &vo.Hash } //nolint:unused
func (vo InlineCachedNodeValue[H]) EncodedValue() codec.EncodedValue {
	return codec.InlineValue(vo.Value)
}
func (vo NodeCachedNodeValue[H]) EncodedValue() codec.EncodedValue { return codec.HashedValue[H](vo) }

func newCachedNodeValueFromEncodedValue[H hash.Hash, Hasher hash.Hasher[H]](
	encVal codec.EncodedValue) CachedNodeValue[H] {
	switch encVal := encVal.(type) {
	case codec.InlineValue:
		return InlineCachedNodeValue[H]{
			Value: encVal,
			Hash:  (*(new(Hasher))).Hash(encVal),
		}
	case codec.HashedValue[H]:
		return NodeCachedNodeValue[H](encVal)
	case nil:
		return nil
	default:
		panic("unreachable")
	}
}

// Cached version of [codec.MerkleValue] interface constraint.
type CachedNodeHandleTypes[H hash.Hash] interface {
	HashCachedNodeHandle[H] | InlineCachedNodeHandle[H]
}

// Cached version of [codec.MerkleValue].
type CachedNodeHandle interface {
	/// Returns [CachedNodeHandle] as a [ChildReference].
	ChildReference() ChildReference
}

type (
	HashCachedNodeHandle[H hash.Hash] struct {
		Hash H
	}
	InlineCachedNodeHandle[H hash.Hash] struct {
		CachedNode[H]
	}
)

func (nho HashCachedNodeHandle[H]) ChildReference() ChildReference {
	return HashChildReference[H](nho)
}
func (nho InlineCachedNodeHandle[H]) ChildReference() ChildReference {
	encoded := nho.CachedNode.encoded()
	store := (*new(H))
	if len(encoded) > store.Length() {
		panic("Invalid inline node handle")
	}
	return InlineChildReference(encoded)
}

func newCachedNodeHandleFromMerkleValue[H hash.Hash, Hasher hash.Hasher[H]](
	mv codec.MerkleValue,
) (CachedNodeHandle, error) {
	switch mv := mv.(type) {
	case codec.HashedNode[H]:
		return HashCachedNodeHandle[H](mv), nil
	case codec.InlineNode:
		buf := bytes.NewBuffer(mv)
		node, err := codec.Decode[H](buf)
		if err != nil {
			return nil, err
		}
		cachedNode, err := newCachedNodeFromNode[H, Hasher](node)
		if err != nil {
			return nil, err
		}
		return InlineCachedNodeHandle[H]{cachedNode}, nil
	default:
		panic("unreachable")
	}
}

type child[H any] struct {
	nibble *uint8
	CachedNodeHandle
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
		Value      CachedNodeValue[H]
	}
	// Branch node; has slice of child nodes (each possibly null)
	// and an optional value.
	BranchCachedNode[H any] struct {
		PartialKey nibbles.NibbleSlice
		Children   [codec.ChildrenCapacity]CachedNodeHandle // can be nil to represent no child
		Value      CachedNodeValue[H]
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
				nibble:           &nibble,
				CachedNodeHandle: ch,
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
			Value:      newCachedNodeValueFromEncodedValue[H, Hasher](n.Value),
		}, nil
	case codec.Branch:
		var children [codec.ChildrenCapacity]CachedNodeHandle
		for i, child := range n.Children {
			if child == nil {
				continue
			}
			var err error
			children[i], err = newCachedNodeHandleFromMerkleValue[H, Hasher](child)
			if err != nil {
				return nil, err
			}
		}
		return BranchCachedNode[H]{
			PartialKey: nibbles.NewNibbleSliceFromNibbles(n.PartialKey),
			Children:   children,
			Value:      newCachedNodeValueFromEncodedValue[H, Hasher](n.Value),
		}, nil
	default:
		panic("unreachable")
	}
}

// The values cached by [TrieCache].
type CachedValues[H any] interface {
	NonExistingCachedValue[H] | ExistingHashCachedValue[H] | ExistingCachedValue[H]
	CachedValue[H]
}

// A value cached by [TrieCache].
type CachedValue[H any] interface {
	data() []byte
	hash() *H
}

// Constructor for [CachedValue]
func NewCachedValue[H any, CV CachedValues[H]](cv CV) CachedValue[H] {
	return cv
}

// The value doesn't exist in the trie.
type NonExistingCachedValue[H any] struct{}

func (NonExistingCachedValue[H]) data() []byte { return nil } //nolint:unused
func (NonExistingCachedValue[H]) hash() *H     { return nil } //nolint:unused

// The hash is cached and not the data because it was not accessed.
type ExistingHashCachedValue[H any] struct {
	Hash H
}

func (ExistingHashCachedValue[H]) data() []byte  { return nil }        //nolint:unused
func (ehcv ExistingHashCachedValue[H]) hash() *H { return &ehcv.Hash } //nolint:unused

// The value exists in the trie.
type ExistingCachedValue[H any] struct {
	// The hash of the value.
	Hash H
	// The actual data of the value.
	Data []byte
}

func (ecv ExistingCachedValue[H]) data() []byte { return ecv.Data }  //nolint:unused
func (ecv ExistingCachedValue[H]) hash() *H     { return &ecv.Hash } //nolint:unused

// A cache that can be used to speed-up certain operations when accessing [TrieDB].
//
// For every lookup in the trie, every node is always fetched and decoded on the fly. Fetching and
// decoding a node always takes some time and can kill the performance of any application that is
// doing quite a lot of trie lookups. To circumvent this performance degradation, a cache can be
// used when looking up something in the trie. Any cache that should be used with the [TrieDB]
// needs to implement this interface.
//
// The interface consists of two cache levels, first the trie nodes cache and then the value cache.
// The trie nodes cache, as the name indicates, is for caching trie nodes as [CachedNode]. These
// trie nodes are referenced by their hash. The value cache is caching [CachedValue]'s and these
// are referenced by the key to look them up in the trie. As multiple different tries can have
// different values under the same key, it up to the cache implementation to ensure that the
// correct value is returned. As each trie has a different root, this root can be used to
// differentiate values under the same key.
type TrieCache[H hash.Hash] interface {
	// Lookup value for the given key.
	// Returns the nil if the key is unknown or otherwise the value is returned
	// [TrieCache.SetValue] is used to make the cache aware of data that is associated
	// to a key.
	//
	// NOTE: The cache can be used for different tries, aka with different roots. This means
	// that the cache implementation needs to take care of always returning the correct value
	// for the current trie root.
	GetValue(key []byte) CachedValue[H]
	// Cache the given value for the given key.
	//
	// NOTE: The cache can be used for different tries, aka with different roots. This means
	// that the cache implementation needs to take care of caching values for the current
	// trie root.
	SetValue(key []byte, value CachedValue[H])

	// Get or insert a [CachedNode].
	// The cache implementation should look up based on the given hash if the node is already
	// known. If the node is not yet known, the given fetchNode function can be used to fetch
	// the particular node.
	// Returns the [CachedNode] or an error that happened on fetching the node.
	GetOrInsertNode(hash H, fetchNode func() (CachedNode[H], error)) (CachedNode[H], error)

	// Get the [CachedNode] that corresponds to the given hash.
	GetNode(hash H) CachedNode[H]
}
