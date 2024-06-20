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
	inline struct {
		Data []byte
	}

	valueRef struct {
		hash common.Hash
	}

	newValueRef struct {
		hash common.Hash
		Data []byte
	}
)

func NewEncodedValue(value nodeValue, partial []byte, childF childFunc) (codec.EncodedValue, error) {
	switch v := value.(type) {
	case inline:
		return codec.NewInlineValue(v.Data), nil
	case valueRef:
		return codec.NewHashedValue(v.hash.ToBytes()), nil
	case newValueRef:
		// Store value in db
		childRef, err := childF(NewNodeToEncode{partialKey: partial, value: v.Data}, partial, nil)
		if err != nil {
			return nil, err
		}

		// Check and get new new value hash
		switch cr := childRef.(type) {
		case HashChildReference:
			if cr.hash == common.EmptyHash {
				panic("new external value are always added before encoding a node")
			}

			if v.hash != common.EmptyHash {
				if v.hash != cr.hash {
					panic("hash mismatch")
				}
			}

			v.hash = cr.hash
		default:
			panic("value node can never be inlined")
		}

		return codec.NewHashedValue(v.hash.ToBytes()), nil
	default:
		panic("unreachable")
	}
}

func (inline) getHash() common.Hash { return common.EmptyHash }
func (n inline) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case inline:
		return bytes.Equal(n.Data, otherValue.Data)
	default:
		return false
	}
}
func (vr valueRef) getHash() common.Hash { return vr.hash }
func (vr valueRef) equal(other nodeValue) bool {
	switch otherValue := other.(type) {
	case valueRef:
		return vr.hash == otherValue.hash
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
		return newValueRef{Data: data}
	}

	return inline{Data: data}
}

func NewValueFromEncoded(prefix []byte, encodedValue codec.EncodedValue) nodeValue {
	switch encoded := encodedValue.(type) {
	case codec.InlineValue:
		return inline{Data: encoded.Data}
	case codec.HashedValue:
		prefixedKey := bytes.Join([][]byte{prefix, encoded.Data}, nil)
		return valueRef{hash: common.NewHash(prefixedKey)}
	}

	return nil
}

func inMemoryFetchedValue(value nodeValue, prefix []byte, db db.DBGetter) ([]byte, error) {
	switch v := value.(type) {
	case inline:
		return v.Data, nil
	case newValueRef:
		return v.Data, nil
	case valueRef:
		prefixedKey := bytes.Join([][]byte{prefix, v.hash.ToBytes()}, nil)
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
		children   [codec.ChildrenCapacity]NodeHandle
		value      nodeValue
	}
)

func (Empty) getPartialKey() []byte    { return nil }
func (n Leaf) getPartialKey() []byte   { return n.partialKey }
func (n Branch) getPartialKey() []byte { return n.partialKey }

// Create a new node from the encoded data, decoding this data into a codec.Node
// and mapping that with this node type
func newNodeFromEncoded(nodeHash common.Hash, data []byte, storage NodeStorage) (Node, error) {
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

		child := func(i int) (NodeHandle, error) {
			if encodedChildren[i] != nil {
				newChild, err := newFromEncodedMerkleValue(nodeHash, encodedChildren[i], storage)
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

		return Branch{partialKey: key, children: children, value: NewValueFromEncoded(encoded.PartialKey, value)}, nil
	default:
		panic("unreachable")
	}
}

type NodeToEncode interface {
	isNodeToEncode()
}

type (
	NewNodeToEncode struct {
		partialKey []byte
		value      []byte
	}
	TrieNodeToEncode struct {
		child NodeHandle
	}
)

func (NewNodeToEncode) isNodeToEncode()  {}
func (TrieNodeToEncode) isNodeToEncode() {}

type ChildReference interface {
	getNodeData() []byte
}

type (
	HashChildReference struct {
		hash common.Hash
	}
	InlineChildReference struct {
		encodedNode []byte
	}
)

func (h HashChildReference) getNodeData() []byte {
	return h.hash.ToBytes()
}
func (i InlineChildReference) getNodeData() []byte {
	return i.encodedNode
}

type childFunc = func(node NodeToEncode, partialKey []byte, childIndex *byte) (ChildReference, error)

const firstPrefix = (0x00 << 6)
const emptyTrieBytes = firstPrefix | (0x00 << 4)

// TODO: move this to codec package
func NewEncodedNode(node Node, childF childFunc) (encodedNode []byte, err error) {
	encodingBuffer := bytes.NewBuffer(nil)

	switch n := node.(type) {
	case Empty:
		return []byte{emptyTrieBytes}, nil
	case Leaf:
		pr := n.partialKey
		value, err := NewEncodedValue(n.value, pr, childF)
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
			pr := n.partialKey
			value, err = NewEncodedValue(n.value, pr, childF)
			if err != nil {
				return nil, err
			}
		}

		var children [codec.ChildrenCapacity]ChildReference
		for i, child := range n.children {
			if child == nil {
				continue
			}

			pr := n.partialKey[len(n.partialKey)-1:]
			childIndex := byte(i)
			children[i], err = childF(TrieNodeToEncode{child}, pr, &childIndex)
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
	} else {
		if value.IsHashed() {
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
	}

	// Write partial key
	keyLE := nibbles.NibblesToKeyLE(partialKey)
	_, err := writer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	//Write bitmap
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

	//Write encoded value
	if value != nil {
		err := value.Write(writer)
		if err != nil {
			return fmt.Errorf("writing branch value: %w", err)
		}
	}

	//Write children
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
