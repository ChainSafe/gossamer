// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

type nodeValue interface {
	getHash() common.Hash
	equal(other nodeValue) bool
}

type (
	// inline is an inlined value representation
	inline []byte

	// valueRef is a reference to a value stored in the db
	valueRef common.Hash

	// newValueRef is a value that will be stored in the db
	newValueRef struct {
		hash common.Hash
		data []byte
	}
)

// newEncodedValue creates an EncodedValue from a nodeValue
func newEncodedValue(value nodeValue, partial []byte, childF onChildStoreFn) (codec.EncodedValue, error) {
	switch v := value.(type) {
	case inline:
		return codec.InlineValue(v), nil
	case valueRef:
		return codec.HashedValue(v), nil
	case newValueRef:
		// Store value in db
		childRef, err := childF(newNodeToEncode{partialKey: partial, value: v.data}, partial, nil)
		if err != nil {
			return nil, err
		}

		// Check and get new new value hash
		switch cr := childRef.(type) {
		case HashChildReference:
			if common.Hash(cr) == common.EmptyHash {
				panic("new external value are always added before encoding a node")
			}

			if v.hash != common.EmptyHash {
				if v.hash != common.Hash(cr) {
					panic("hash mismatch")
				}
			} else {
				v.hash = common.Hash(cr)
			}
		default:
			panic("value node can never be inlined")
		}

		return codec.HashedValue(v.hash), nil
	default:
		panic("unreachable")
	}
}

func (inline) getHash() common.Hash { return common.EmptyHash }
func (n inline) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case inline:
		return bytes.Equal(n, otherValue)
	default:
		return false
	}
}
func (vr valueRef) getHash() common.Hash { return common.Hash(vr) }
func (vr valueRef) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case valueRef:
		return vr == otherValue
	default:
		return false
	}
}

func (vr newValueRef) getHash() common.Hash {
	return vr.hash
}
func (vr newValueRef) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case newValueRef:
		return vr.hash == otherValue.hash
	default:
		return false
	}
}

func NewValue(data []byte, threshold int) nodeValue {
	if len(data) >= threshold {
		return newValueRef{data: data}
	}

	return inline(data)
}

func NewValueFromEncoded(prefix []byte, encodedValue codec.EncodedValue) nodeValue {
	switch v := encodedValue.(type) {
	case codec.InlineValue:
		return inline(v)
	case codec.HashedValue:
		prefixedKey := bytes.Join([][]byte{prefix, v[:]}, nil)
		return valueRef(common.NewHash(prefixedKey))
	}

	return nil
}

func inMemoryFetchedValue(value nodeValue, prefix []byte, db db.DBGetter) ([]byte, error) {
	switch v := value.(type) {
	case inline:
		return v, nil
	case newValueRef:
		return v.data, nil
	case valueRef:
		prefixedKey := bytes.Join([][]byte{prefix, v[:]}, nil)
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

type Node interface {
	getPartialKey() []byte
}

type (
	Empty struct{}
	Leaf  struct {
		partialKey []byte
		value      nodeValue
	}
	Branch struct {
		partialKey []byte
		children   [codec.ChildrenCapacity]nodeHandle
		value      nodeValue
	}
)

func (Empty) getPartialKey() []byte    { return nil }
func (n Leaf) getPartialKey() []byte   { return n.partialKey }
func (n Branch) getPartialKey() []byte { return n.partialKey }

// Create a new node from the encoded data, decoding this data into a codec.Node
// and mapping that with this node type
func newNodeFromEncoded(nodeHash common.Hash, data []byte, storage nodeStorage) (Node, error) {
	reader := bytes.NewReader(data)
	encodedNode, err := codec.Decode(reader)
	if err != nil {
		return nil, err
	}

	switch encoded := encodedNode.(type) {
	case codec.Empty:
		return Empty{}, nil
	case codec.Leaf:
		return Leaf{partialKey: encoded.PartialKey, value: NewValueFromEncoded(encoded.PartialKey, encoded.Value)}, nil
	case codec.Branch:
		key := encoded.PartialKey
		encodedChildren := encoded.Children
		value := encoded.Value

		child := func(i int) (nodeHandle, error) {
			if encodedChildren[i] != nil {
				newChild, err := newFromEncodedMerkleValue(nodeHash, encodedChildren[i], storage)
				if err != nil {
					return nil, err
				}
				return newChild, nil
			}
			return nil, nil //nolint:nilnil
		}

		children := [codec.ChildrenCapacity]nodeHandle{}
		for i := 0; i < len(children); i++ {
			child, err := child(i)
			if err != nil {
				return nil, err
			}
			children[i] = child
		}

		return Branch{partialKey: key, children: children, value: NewValueFromEncoded(encoded.PartialKey, value)}, nil
	default:
		panic("unreachable")
	}
}

type nodeToEncode interface {
	isNodeToEncode()
}

type (
	newNodeToEncode struct {
		partialKey []byte
		value      []byte
	}
	trieNodeToEncode struct {
		child nodeHandle
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
	HashChildReference common.Hash
	// InlineChildReference is a reference to an inlined child node
	InlineChildReference []byte
)

func (h HashChildReference) getNodeData() []byte {
	return h[:]
}
func (i InlineChildReference) getNodeData() []byte {
	return i
}

type onChildStoreFn = func(node nodeToEncode, partialKey []byte, childIndex *byte) (ChildReference, error)

const EmptyTrieBytes = byte(0)

// newEncodedNode creates a new encoded node from a node and a child store function and return its bytes
func newEncodedNode(node Node, childF onChildStoreFn) (encodedNode []byte, err error) {
	encodingBuffer := bytes.NewBuffer(nil)

	switch n := node.(type) {
	case Empty:
		return []byte{EmptyTrieBytes}, nil
	case Leaf:
		pr := n.partialKey
		value, err := newEncodedValue(n.value, pr, childF)
		if err != nil {
			return nil, err
		}

		err = NewEncodedLeaf(pr, value, encodingBuffer)
		if err != nil {
			return nil, err
		}
	case Branch:
		var value codec.EncodedValue
		if n.value != nil {
			value, err = newEncodedValue(n.value, n.partialKey, childF)
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
			children[i], err = childF(trieNodeToEncode{child}, n.partialKey, &childIndex)
			if err != nil {
				return nil, err
			}
		}

		err := NewEncodedBranch(n.partialKey, children, value, encodingBuffer)
		if err != nil {
			return nil, err
		}
	default:
		panic("unreachable")
	}

	return encodingBuffer.Bytes(), nil
}

// NewEncodedLeaf creates a new encoded leaf node and writes it to the writer
func NewEncodedLeaf(partialKey []byte, value codec.EncodedValue, writer io.Writer) error {
	// Write encoded header
	if value.IsHashed() {
		err := codec.EncodeHeader(partialKey, codec.LeafWithHashedValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for leaf with hashed value: %w", err)
		}
	} else {
		err := codec.EncodeHeader(partialKey, codec.LeafNode, writer)
		if err != nil {
			return fmt.Errorf("encoding header for leaf node value: %w", err)
		}
	}

	// Write partial key
	keyLE := nibbles.NibblesToKeyLE(partialKey)
	_, err := writer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	// Write encoded value
	err = value.Write(writer)
	if err != nil {
		return fmt.Errorf("writing leaf value: %w", err)
	}
	return nil
}

// NewEncodedBranch creates a new encoded branch node and writes it to the writer
func NewEncodedBranch(
	partialKey []byte,
	children [codec.ChildrenCapacity]ChildReference,
	value codec.EncodedValue,
	writer io.Writer,
) error {
	// Write encoded header
	if value == nil {
		err := codec.EncodeHeader(partialKey, codec.BranchWithoutValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for branch without value: %w", err)
		}
	} else if value.IsHashed() {
		err := codec.EncodeHeader(partialKey, codec.BranchWithHashedValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for branch with hashed value: %w", err)
		}
	} else {
		err := codec.EncodeHeader(partialKey, codec.BranchWithValue, writer)
		if err != nil {
			return fmt.Errorf("encoding header for branch with value: %w", err)
		}
	}

	// Write partial key
	keyLE := nibbles.NibblesToKeyLE(partialKey)
	_, err := writer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	// Write bitmap
	var bitmap uint16
	for i := range children {
		if children[i] == nil {
			continue
		}
		bitmap |= 1 << uint(i)
	}
	childrenBitmap := common.Uint16ToBytes(bitmap)
	_, err = writer.Write(childrenBitmap)
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
