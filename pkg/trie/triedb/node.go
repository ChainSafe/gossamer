// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

type nodeValue interface {
	equal(other nodeValue) bool
}

type (
	// inline is an inlined value representation
	inline []byte

	// valueRef is a reference to a value stored in the db
	valueRef[H hash.Hash] struct {
		hash H
	}

	// newValueRef is a value that will be stored in the db
	newValueRef[H hash.Hash] struct {
		hash H
		data []byte
	}
)

// newEncodedValue creates an EncodedValue from a nodeValue
func newEncodedValue[H hash.Hash](
	value nodeValue, partial *nibbles.Nibbles, childF onChildStoreFn,
) (codec.EncodedValue, error) {
	switch v := value.(type) {
	case inline:
		return codec.InlineValue(v), nil
	case valueRef[H]:
		return codec.HashedValue[H]{Hash: v.hash}, nil
	case newValueRef[H]:
		// Store value in db
		childRef, err := childF(newNodeToEncode{value: v.data}, partial, nil)
		if err != nil {
			return nil, err
		}

		// Check and get new new value hash
		switch cr := childRef.(type) {
		case HashChildReference[H]:
			empty := *new(H)
			if cr.Hash == empty {
				panic("new external value are always added before encoding a node")
			}

			if v.hash != empty {
				if v.hash != cr.Hash {
					panic("hash mismatch")
				}
			} else {
				v.hash = cr.Hash
			}
		default:
			panic("value node can never be inlined")
		}

		return codec.HashedValue[H]{Hash: v.hash}, nil
	default:
		panic("unreachable")
	}
}

func (n inline) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case inline:
		return bytes.Equal(n, otherValue)
	default:
		return false
	}
}

func (vr valueRef[H]) getHash() H { return vr.hash }
func (vr valueRef[H]) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case valueRef[H]:
		return vr.hash == otherValue.hash
	default:
		return false
	}
}

func (vr newValueRef[H]) getHash() H {
	return vr.hash
}
func (vr newValueRef[H]) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case newValueRef[H]:
		return vr.hash == otherValue.hash
	default:
		return false
	}
}

func NewValue[H hash.Hash](data []byte, threshold int) nodeValue {
	if len(data) >= threshold {
		return newValueRef[H]{
			hash: *new(H),
			data: data,
		}
	}

	return inline(data)
}

func NewValueFromEncoded[H hash.Hash](encodedValue codec.EncodedValue) nodeValue {
	switch v := encodedValue.(type) {
	case codec.InlineValue:
		return inline(v)
	case codec.HashedValue[H]:
		return valueRef[H]{v.Hash}
	}

	return nil
}

func newValueFromCachedNodeValue[H hash.Hash](val CachedNodeValue[H]) nodeValue {
	switch val := val.(type) {
	case InlineCachedNodeValue[H]:
		return inline(val.Value)
	case NodeCachedNodeValue[H]:
		return valueRef[H]{val.Hash}
	default:
		panic("unreachable")
	}
}

func inMemoryFetchedValue[H hash.Hash](value nodeValue, prefix []byte, db db.DBGetter) ([]byte, error) {
	switch v := value.(type) {
	case inline:
		return v, nil
	case newValueRef[H]:
		return v.data, nil
	case valueRef[H]:
		prefixedKey := bytes.Join([][]byte{prefix, v.hash.Bytes()}, nil)
		value, err := db.Get(prefixedKey)
		if err != nil {
			return nil, err
		}
		if value != nil {
			return value, nil
		}
		return value, ErrIncompleteDB
	default:
		panic("unreachable")
	}
}

type NodeTypes[H hash.Hash] interface {
	Empty | Leaf[H] | Branch[H]
	Node
}
type Node interface {
	getPartialKey() *nodeKey
}

type nodeKey = nibbles.NodeKey

type (
	Empty             struct{}
	Leaf[H hash.Hash] struct {
		partialKey nodeKey
		value      nodeValue
	}
	Branch[H hash.Hash] struct {
		partialKey nodeKey
		children   [codec.ChildrenCapacity]NodeHandle
		value      nodeValue
	}
)

func (Empty) getPartialKey() *nodeKey       { return nil }
func (n Leaf[H]) getPartialKey() *nodeKey   { return &n.partialKey }
func (n Branch[H]) getPartialKey() *nodeKey { return &n.partialKey }

// Create a new node from the encoded data, decoding this data into a codec.Node
// and mapping that with this node type
func newNodeFromEncoded[H hash.Hash](nodeHash H, data []byte, storage *nodeStorage[H]) (Node, error) {
	reader := bytes.NewReader(data)
	encodedNode, err := codec.Decode[H](reader)
	if err != nil {
		return nil, err
	}

	switch encoded := encodedNode.(type) {
	case codec.Empty:
		return Empty{}, nil
	case codec.Leaf:
		return Leaf[H]{
			partialKey: encoded.PartialKey.NodeKey(),
			value:      NewValueFromEncoded[H](encoded.Value),
		}, nil
	case codec.Branch:
		key := encoded.PartialKey
		encodedChildren := encoded.Children
		value := encoded.Value

		child := func(i int) (NodeHandle, error) {
			if encodedChildren[i] != nil {
				newChild, err := newNodeHandleFromMerkleValue[H](nodeHash, encodedChildren[i], storage)
				if err != nil {
					return nil, err
				}
				return newChild, nil
			}
			return nil, nil //nolint:nilnil
		}

		children := [codec.ChildrenCapacity]NodeHandle{}
		for i := 0; i < len(children); i++ {
			child, err := child(i)
			if err != nil {
				return nil, err
			}
			children[i] = child
		}

		return Branch[H]{partialKey: key.NodeKey(), children: children, value: NewValueFromEncoded[H](value)}, nil
	default:
		panic("unreachable")
	}
}

func newNodeFromCachedNode[H hash.Hash](
	nodeOwned CachedNode[H], storage *nodeStorage[H],
) Node {
	switch nodeOwned := nodeOwned.(type) {
	case EmptyCachedNode[H]:
		return Empty{}
	case LeafCachedNode[H]:
		leaf := nodeOwned
		return Leaf[H]{
			partialKey: leaf.PartialKey.NodeKey(),
			value:      newValueFromCachedNodeValue[H](leaf.Value),
		}
	case BranchCachedNode[H]:
		k := nodeOwned.PartialKey
		encodedChildren := nodeOwned.Children
		val := nodeOwned.Value

		child := func(i uint) NodeHandle {
			if encodedChildren[i] != nil {
				newChild := newNodeHandleFromNodeHandleOwned(encodedChildren[i], storage)
				return newChild
			}
			return nil
		}

		children := [codec.ChildrenCapacity]NodeHandle{}
		for i := uint(0); i < codec.ChildrenCapacity; i++ {
			children[i] = child(i)
		}
		var value nodeValue
		if val != nil {
			value = newValueFromCachedNodeValue(val)
		}
		return Branch[H]{
			partialKey: k.NodeKey(),
			children:   children,
			value:      value,
		}
	case ValueCachedNode[H]:
		panic("ValueCachedNode can only be returned for the hash of a value")
	default:
		panic("unreachable")
	}
}

type nodeToEncode interface {
	isNodeToEncode()
}

type (
	newNodeToEncode struct {
		value []byte
	}
	trieNodeToEncode struct {
		child NodeHandle
	}
)

func (newNodeToEncode) isNodeToEncode()  {}
func (trieNodeToEncode) isNodeToEncode() {}

// ChildReferences is a slice of ChildReference
type ChildReferences [codec.ChildrenCapacity]ChildReference

// ChildReference is a reference to a child node
type ChildReference interface {
	getNodeData() []byte
}

type (
	// HashChildReference is a reference to a child node that is not inlined
	HashChildReference[H hash.Hash] struct{ Hash H }
	// InlineChildReference is a reference to an inlined child node
	InlineChildReference []byte
)

func (h HashChildReference[H]) getNodeData() []byte {
	return h.Hash.Bytes()
}
func (i InlineChildReference) getNodeData() []byte {
	return i
}

type onChildStoreFn = func(node nodeToEncode, partialKey *nibbles.Nibbles, childIndex *byte) (ChildReference, error)

const EmptyTrieBytes = byte(0)

// newEncodedNode creates a new encoded node from a node and a child store function and return its bytes
func newEncodedNode[H hash.Hash](node Node, childF onChildStoreFn) (encodedNode []byte, err error) {
	encodingBuffer := bytes.NewBuffer(nil)

	switch n := node.(type) {
	case Empty:
		return []byte{EmptyTrieBytes}, nil
	case Leaf[H]:
		partialKey := nibbles.NewNibbles(n.partialKey.Data, n.partialKey.Offset)
		value, err := newEncodedValue[H](n.value, &partialKey, childF)
		if err != nil {
			return nil, err
		}
		right := partialKey.Right()
		len := partialKey.Len()
		err = NewEncodedLeaf(right, len, value, encodingBuffer)
		if err != nil {
			return nil, err
		}
	case Branch[H]:
		partialKey := nibbles.NewNibbles(n.partialKey.Data, n.partialKey.Offset)
		var value codec.EncodedValue
		if n.value != nil {
			value, err = newEncodedValue[H](n.value, &partialKey, childF)
			if err != nil {
				return nil, err
			}
		}

		var children [codec.ChildrenCapacity]ChildReference
		for i, child := range n.children {
			if child == nil {
				continue
			}

			childIndex := byte(i)

			children[i], err = childF(trieNodeToEncode{child}, &partialKey, &childIndex)
			if err != nil {
				return nil, err
			}
		}

		err := NewEncodedBranch(partialKey.Right(), partialKey.Len(), children, value, encodingBuffer)
		if err != nil {
			return nil, err
		}
	default:
		panic("unreachable")
	}

	return encodingBuffer.Bytes(), nil
}

// NewEncodedLeaf creates a new encoded leaf node and writes it to the writer
func NewEncodedLeaf(partialKey []byte, numberNibble uint, value codec.EncodedValue, writer io.Writer) error {
	// Write encoded header
	if value.IsHashed() {
		err := codec.EncodeHeader(partialKey, numberNibble, codec.LeafWithHashedValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for leaf with hashed value: %w", err)
		}
	} else {
		err := codec.EncodeHeader(partialKey, numberNibble, codec.LeafNode, writer)
		if err != nil {
			return fmt.Errorf("encoding header for leaf node value: %w", err)
		}
	}

	// Write encoded value
	err := value.Write(writer)
	if err != nil {
		return fmt.Errorf("writing leaf value: %w", err)
	}
	return nil
}

// NewEncodedBranch creates a new encoded branch node and writes it to the writer
func NewEncodedBranch(
	partialKey []byte,
	numberNibbles uint,
	children [codec.ChildrenCapacity]ChildReference,
	value codec.EncodedValue,
	writer io.Writer,
) error {
	// Write encoded header
	if value == nil {
		err := codec.EncodeHeader(partialKey, numberNibbles, codec.BranchWithoutValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for branch without value: %w", err)
		}
	} else if value.IsHashed() {
		err := codec.EncodeHeader(partialKey, numberNibbles, codec.BranchWithHashedValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for branch with hashed value: %w", err)
		}
	} else {
		err := codec.EncodeHeader(partialKey, numberNibbles, codec.BranchWithValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for branch with value: %w", err)
		}
	}

	// Write bitmap
	var bitmap uint16
	for i := range children {
		if children[i] == nil {
			continue
		}
		bitmap |= 1 << uint(i)
	}
	encoder := scale.NewEncoder(writer)
	err := encoder.Encode(bitmap)
	if err != nil {
		return fmt.Errorf("writing branch bitmap: %w", err)
	}

	// Write encoded value
	if value != nil {
		err := value.Write(writer)
		if err != nil {
			return fmt.Errorf("writing branch value: %w", err)
		}
	}

	// Write children
	for _, child := range children {
		if child != nil {
			encoder := scale.NewEncoder(writer)
			err := encoder.Encode(child.getNodeData())
			if err != nil {
				return fmt.Errorf("encoding hash child reference: %w", err)
			}
		}
	}

	return nil
}
