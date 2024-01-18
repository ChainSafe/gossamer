package node

import (
	"errors"

	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
)

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandleOwned[H hashdb.HashOut] interface {
	Type() string
	AsChildReference(codec NodeCodec[H]) ChildReference[H]
}
type (
	NodeHandleOwnedHash[H hashdb.HashOut] struct {
		Hash H
	}
	NodeHandleOwnedInline[H hashdb.HashOut] struct {
		Node NodeOwned[H]
	}
)

func (h NodeHandleOwnedHash[H]) Type() string { return "Hash" }
func (h NodeHandleOwnedHash[H]) AsChildReference(codec NodeCodec[H]) ChildReference[H] {
	return ChildReferenceHash[H]{hash: h.Hash}
}
func (h NodeHandleOwnedInline[H]) Type() string { return "Inline" }
func (h NodeHandleOwnedInline[H]) AsChildReference(codec NodeCodec[H]) ChildReference[H] {
	encoded := EncodeNodeOwned(h.Node, codec)
	if len(encoded) > codec.Hasher().Length() {
		panic("Invalid inline node handle")
	}
	return ChildReferenceInline[H]{hash: codec.Hasher().FromBytes(encoded), length: uint(len(encoded))}
}

// NodeHandle is a reference to a trie node which may be stored within another trie node.
type NodeHandle interface {
	Type() string
}
type (
	Hash struct {
		Data []byte
	}
	Inline struct {
		Data []byte
	}
)

func (h Hash) Type() string   { return "Hash" }
func (h Inline) Type() string { return "Inline" }

func DecodeHash[H hashdb.HashOut](hasher hashdb.Hasher[H], data []byte) (H, error) {
	if len(data) != hasher.Length() {
		return hasher.FromBytes([]byte{}), errors.New("decoding hash")
	}
	return hasher.FromBytes(data), nil
}
