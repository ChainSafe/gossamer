package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

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

func ValueOwnedFromEncodedValue[H hash.Hash, Hasher hash.Hasher[H]](encVal codec.EncodedValue) ValueOwned {
	switch encVal := encVal.(type) {
	case codec.InlineValue:
		return ValueOwnedInline[H]{
			Value: encVal,
			Hash:  (*(new(Hasher))).Hash(encVal),
		}
	case codec.HashedValue[H]:
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

func NodeHandleOwnedFromMerkleValue[H hash.Hash, Hasher hash.Hasher[H]](mv codec.MerkleValue) (NodeHandleOwned, error) {
	switch mv := mv.(type) {
	case codec.HashedNode[H]:
		return NodeHandleOwnedHash[H](mv), nil
	case codec.InlineNode:
		buf := bytes.NewBuffer(mv)
		node, err := codec.Decode[H](buf)
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
		Children   [codec.ChildrenCapacity]NodeHandleOwned // can be nil to represent no child
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

func NodeOwnedFromNode[H hash.Hash, Hasher hash.Hasher[H]](n codec.EncodedNode) (NodeOwned, error) {
	switch n := n.(type) {
	case codec.Empty:
		return NodeOwnedEmpty{}, nil
	case codec.Leaf:
		return NodeOwnedLeaf[H]{
			PartialKey: n.PartialKey,
			Value:      ValueOwnedFromEncodedValue[H, Hasher](n.Value),
		}, nil
	case codec.Branch:
		var childrenOwned [codec.ChildrenCapacity]NodeHandleOwned
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
