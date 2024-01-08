// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import "github.com/ChainSafe/gossamer/pkg/trie/hashdb"

type MemoryDBValue[T any] struct {
	value T
	rc    int32
}

type MemoryDB[Hash hashdb.HasherOut, Hasher hashdb.Hasher[Hash], KF KeyFunction[Hash, Hasher], T any] struct {
	data           map[Hash]MemoryDBValue[T]
	hashedNullNode Hash
	nullNodeData   T
	keyFunction    KF
}

func newFromNullNode[Hash hashdb.HasherOut, Hasher hashdb.Hasher[Hash], KF KeyFunction[Hash, Hasher], T any](
	nullKey []byte,
	nullNodeData T,
	hasher Hasher,
	keyFunction KF,
) *MemoryDB[Hash, Hasher, KF, T] {
	return &MemoryDB[Hash, Hasher, KF, T]{
		data:           make(map[Hash]MemoryDBValue[T]),
		hashedNullNode: hasher.Hash(nullKey),
		nullNodeData:   nullNodeData,
		keyFunction:    keyFunction,
	}
}

func NewMemoryDB[Hash hashdb.HasherOut, Hasher hashdb.Hasher[Hash], KF KeyFunction[Hash, Hasher], T any](
	data []byte,
	hasher Hasher,
	keyFunction KF,
) *MemoryDB[Hash, Hasher, KF, []byte] {
	return newFromNullNode[Hash](data, data, hasher, keyFunction)
}

type KeyFunction[Hash hashdb.HasherOut, H hashdb.Hasher[Hash]] interface {
	Key(hash Hash, prefix hashdb.Prefix) Hash
}
