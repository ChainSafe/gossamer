// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashdb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"

type HasherOut interface {
	comparable
	ToBytes() []byte
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
	Get(key Hash, prefix nibble.Prefix) *T
	Contains(key Hash, prefix nibble.Prefix) bool
	Insert(prefix nibble.Prefix, value []byte) Hash
	Emplace(key Hash, prefix nibble.Prefix, value T)
	remove(key Hash, prefix nibble.Prefix)
}
