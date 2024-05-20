package triedb

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

type StorageHandle struct{ int }

func (sh StorageHandle) toNodeHandle() NodeHandle {
	return InMemory{idx: sh}
}

type NodeHandle interface {
	isNodeHandle()
}

type (
	InMemory struct {
		idx StorageHandle
	}
	Hash struct {
		hash common.Hash
	}
)

func (InMemory) isNodeHandle() {}
func (Hash) isNodeHandle()     {}

func newFromEncodedMerkleValue(
	parentHash common.Hash,
	encodedNodeHandle codec.MerkleValue,
	storage NodeStorage,
) (NodeHandle, error) {
	switch encoded := encodedNodeHandle.(type) {
	case codec.HashedNode:
		return Hash{hash: common.NewHash(encoded.Data)}, nil
	case codec.InlineNode:
		child, err := newNodeFromEncoded(parentHash, encoded.Data, storage)
		if err != nil {
			return nil, err
		}
		return InMemory{storage.alloc(New{child})}, nil
	default:
		panic("unreachable")
	}
}

type Value interface {
	isValue()
	getHash() common.Hash
}

type (
	Inline struct {
		Data []byte
	}

	ValueRef struct {
		hash common.Hash
	}

	NewValueRef struct {
		hash *common.Hash
		Data []byte
	}
)

func (Inline) isValue()                  {}
func (Inline) getHash() common.Hash      { return common.EmptyHash }
func (ValueRef) isValue()                {}
func (vr ValueRef) getHash() common.Hash { return vr.hash }
func (NewValueRef) isValue()             {}
func (vr NewValueRef) getHash() common.Hash {
	if vr.hash == nil {
		return common.EmptyHash
	}

	return *vr.hash
}

func NewValue(data []byte, threshold int) Value {
	if len(data) >= threshold {
		return NewValueRef{Data: data}
	}

	return Inline{Data: data}
}

func NewFromEncoded(encodedValue codec.NodeValue) Value {
	switch encoded := encodedValue.(type) {
	case codec.InlineValue:
		return Inline{Data: encoded.Data}
	case codec.HashedValue:
		return ValueRef{hash: common.NewHash(encoded.Data)}
	}

	return nil
}

func InMemoryFetchedValue(value Value, prefix []byte, db db.DBGetter, fullKey []byte) ([]byte, error) {
	switch v := value.(type) {
	case Inline:
		return v.Data, nil
	case NewValueRef:
		return v.Data, nil
	case ValueRef:
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

var EmptyNode = []byte{EmptyTrie}
var HashedNullNode = common.MustBlake2bHash(EmptyNode)

type Node interface {
	isNode()
}

type (
	Empty struct{}
	Leaf  struct {
		partialKey []byte
		value      Value
	}
	Branch struct {
		partialKey []byte
		children   [codec.ChildrenCapacity]NodeHandle
		value      Value
	}
)

func (Empty) isNode()  {}
func (Leaf) isNode()   {}
func (Branch) isNode() {}

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
		return Leaf{partialKey: encoded.PartialKey, value: NewFromEncoded(encoded.Value)}, nil
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

		return Branch{partialKey: key, children: children, value: NewFromEncoded(value)}, nil
	default:
		panic("unreachable")
	}
}
