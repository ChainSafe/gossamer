package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

type ValueOwnedTypes[H hash.Hash] interface {
	ValueOwnedInline[H] | ValueOwnedNode[H]
	ValueOwned[H]
}
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
	// Hash byte slice as stored in a trie node.
	ValueOwnedNode[H hash.Hash] struct {
		Hash H
	}
)

func (vo ValueOwnedInline[H]) data() []byte                     { return vo.Value }
func (vo ValueOwnedNode[H]) data() []byte                       { return nil }
func (vo ValueOwnedInline[H]) dataHash() *H                     { return &vo.Hash }
func (vo ValueOwnedNode[H]) dataHash() *H                       { return &vo.Hash }
func (vo ValueOwnedInline[H]) EncodedValue() codec.EncodedValue { return codec.InlineValue(vo.Value) }
func (vo ValueOwnedNode[H]) EncodedValue() codec.EncodedValue {
	return codec.HashedValue[H]{Hash: vo.Hash}
}

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

type NodeHandleOwnedTypes[H hash.Hash] interface {
	NodeHandleOwnedHash[H] | NodeHandleOwnedInline[H]
}

type NodeHandleOwned interface {
	ChildReference() ChildReference
	isNodeHandleOwned()
}

type (
	NodeHandleOwnedHash[H hash.Hash] struct {
		Hash H
	}
	NodeHandleOwnedInline[H hash.Hash] struct {
		NodeOwned[H]
	}
)

func (NodeHandleOwnedHash[H]) isNodeHandleOwned()   {}
func (NodeHandleOwnedInline[H]) isNodeHandleOwned() {}
func (nho NodeHandleOwnedHash[H]) ChildReference() ChildReference {
	return HashChildReference[H]{Hash: nho.Hash}
}
func (nho NodeHandleOwnedInline[H]) ChildReference() ChildReference {
	encoded := nho.NodeOwned.encoded()
	store := (*new(H))
	if len(encoded) > store.Length() {
		panic("Invalid inline node handle")
	}
	return InlineChildReference(encoded)
}

func newNodeHandleOwnedFromMerkleValue[H hash.Hash, Hasher hash.Hasher[H]](mv codec.MerkleValue) (NodeHandleOwned, error) {
	switch mv := mv.(type) {
	case codec.HashedNode[H]:
		return NodeHandleOwnedHash[H](mv), nil
	case codec.InlineNode:
		buf := bytes.NewBuffer(mv)
		node, err := codec.Decode[H](buf)
		if err != nil {
			return nil, err
		}
		nodeOwned, err := newNodeOwnedFromNode[H, Hasher](node)
		if err != nil {
			return nil, err
		}
		return NodeHandleOwnedInline[H]{nodeOwned}, nil
	default:
		panic("unreachable")
	}
}

type NodeOwnedTypes[H hash.Hash] interface {
	NodeOwnedEmpty[H] | NodeOwnedLeaf[H] | NodeOwnedBranch[H] | NodeOwnedValue[H]
	NodeOwned[H]
}

type child[H any] struct {
	nibble *uint8
	NodeHandleOwned
}
type NodeOwned[H any] interface {
	// isNodeOwned()
	data() []byte // nil means there is no data
	dataHash() *H
	children() []child[H]
	partialKey() *nibbles.NibbleSlice
	encoded() []byte
}

type (
	// Null trie node; could be an empty root or an empty branch entry.
	NodeOwnedEmpty[H hash.Hash] struct{}
	// Leaf node; has key slice and value. Value may not be empty.
	NodeOwnedLeaf[H any] struct {
		PartialKey nibbles.NibbleSlice
		Value      ValueOwned[H]
	}
	// Branch node; has slice of child nodes (each possibly null)
	// and an optional immediate node data.
	NodeOwnedBranch[H any] struct {
		PartialKey nibbles.NibbleSlice
		Children   [codec.ChildrenCapacity]NodeHandleOwned // can be nil to represent no child
		Value      ValueOwned[H]
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

func (NodeOwnedEmpty[H]) data() []byte   { return nil }
func (no NodeOwnedLeaf[H]) data() []byte { return no.Value.data() }
func (no NodeOwnedBranch[H]) data() []byte {
	if no.Value != nil {
		return no.Value.data()
	}
	return nil
}
func (no NodeOwnedValue[H]) data() []byte { return no.Value }

func (NodeOwnedEmpty[H]) dataHash() *H   { return nil }
func (no NodeOwnedLeaf[H]) dataHash() *H { return no.Value.dataHash() }
func (no NodeOwnedBranch[H]) dataHash() *H {
	if no.Value != nil {
		return no.Value.dataHash()
	}
	return nil
}
func (no NodeOwnedValue[H]) dataHash() *H { return &no.Hash }

func (NodeOwnedEmpty[H]) children() []child[H]   { return nil }
func (no NodeOwnedLeaf[H]) children() []child[H] { return nil }
func (no NodeOwnedBranch[H]) children() []child[H] {
	r := []child[H]{}
	for i, ch := range no.Children {
		if ch != nil {
			nibble := uint8(i)
			r = append(r, child[H]{
				nibble:          &nibble,
				NodeHandleOwned: ch,
			})
		}
	}
	return r
}
func (no NodeOwnedValue[H]) children() []child[H] { return nil }

func (NodeOwnedEmpty[H]) partialKey() *nibbles.NibbleSlice     { return nil }
func (no NodeOwnedLeaf[H]) partialKey() *nibbles.NibbleSlice   { return &no.PartialKey }
func (no NodeOwnedBranch[H]) partialKey() *nibbles.NibbleSlice { return &no.PartialKey }
func (no NodeOwnedValue[H]) partialKey() *nibbles.NibbleSlice  { return nil }

func (NodeOwnedEmpty[H]) encoded() []byte {
	return []byte{EmptyTrieBytes}
}
func (no NodeOwnedLeaf[H]) encoded() []byte {
	encodingBuffer := bytes.NewBuffer(nil)
	err := NewEncodedLeaf(no.PartialKey.Right(), no.PartialKey.Len(), no.Value.EncodedValue(), encodingBuffer)
	if err != nil {
		panic(err)
	}
	return encodingBuffer.Bytes()
}
func (no NodeOwnedBranch[H]) encoded() []byte {
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
func (no NodeOwnedValue[H]) encoded() []byte { return no.Value }

func newNodeOwnedFromNode[H hash.Hash, Hasher hash.Hasher[H]](n codec.EncodedNode) (NodeOwned[H], error) {
	switch n := n.(type) {
	case codec.Empty:
		return NodeOwnedEmpty[H]{}, nil
	case codec.Leaf:
		return NodeOwnedLeaf[H]{
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
		return NodeOwnedBranch[H]{
			PartialKey: nibbles.NewNibbleSliceFromNibbles(n.PartialKey),
			Children:   childrenOwned,
			Value:      newValueOwnedFromEncodedValue[H, Hasher](n.Value),
		}, nil
	default:
		panic("unreachable")
	}
}
