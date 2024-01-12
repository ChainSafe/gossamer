// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashdb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"

type HashOut interface {
	Bytes() []byte
	ComparableKey() string
}

type Hasher[Hash HashOut] interface {
	Length() int
	Hash(value []byte) Hash
	FromBytes(value []byte) Hash
}

type HashDB[Hash HashOut, T any] interface {
	Get(key Hash, prefix nibble.Prefix) *T
	Contains(key Hash, prefix nibble.Prefix) bool
	Insert(prefix nibble.Prefix, value []byte) Hash
	Emplace(key Hash, prefix nibble.Prefix, value T)
	Remove(key Hash, prefix nibble.Prefix)
}
