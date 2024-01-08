// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashdb

type HasherOut interface {
	comparable
	ToBytes() []byte
}

// / A trie node prefix, it is the nibble path from the trie root
// / to the trie node.
// / For a node containing no partial key value it is the full key.
// / For a value node or node containing a partial key, it is the full key minus its node partial
// / nibbles (the node key can be split into prefix and node partial).
// / Therefore it is always the leftmost portion of the node key, so its internal representation
// / is a non expanded byte slice followed by a last padded byte representation.
// / The padded byte is an optional padded value.
type Prefix struct {
	PartialKey []byte
	PaddedByte *byte
}

type Hasher[Hash HasherOut] interface {
	Length() int
	Hash(value []byte) Hash
	FromBytes(value []byte) Hash
}

type PlainDB[K any, V any] interface {
	Get(key K) *V
	Contains(key K) bool
	Emplace(key K, value V)
	Remove(key K)
}

type HashDB[Hash HasherOut, T any] interface {
	Get(key Hash, prefix Prefix) *T
	Contains(key Hash, prefix Prefix) bool
	Insert(prefix Prefix, value []byte) Hash
	Emplace(key Hash, prefix Prefix, value T)
	remove(key Hash, prefix Prefix)
}
