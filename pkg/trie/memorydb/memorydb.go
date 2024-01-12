// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type MemoryDBValue struct {
	value []byte
	rc    int32
}

type MemoryDB[H hashdb.HashOut] struct {
	data           map[string]MemoryDBValue
	hashedNullNode H
	nullNodeData   []byte
	keyFunction    KeyFunction[H]
}

func newFromNullNode[H hashdb.HashOut](
	nullKey []byte,
	nullNodeData []byte,
	hasher hashdb.Hasher[H],
	keyFunction KeyFunction[H],
) *MemoryDB[H] {
	return &MemoryDB[H]{
		data:           make(map[string]MemoryDBValue),
		hashedNullNode: hasher.Hash(nullKey),
		nullNodeData:   nullNodeData,
		keyFunction:    keyFunction,
	}
}

func (db *MemoryDB[H]) Get(key H, prefix nibble.Prefix) *[]byte {
	panic("Implement me")
}

func (db *MemoryDB[H]) Contains(key H, prefix nibble.Prefix) bool {
	panic("Implement me")
}

func (db *MemoryDB[H]) Insert(prefix nibble.Prefix, value []byte) H {
	panic("Implement me")
}

func (db *MemoryDB[H]) Emplace(key H, prefix nibble.Prefix, value []byte) {
	panic("Implement me")
}

func (db *MemoryDB[H]) Remove(key H, prefix nibble.Prefix) {
	panic("Implement me")
}

func NewMemoryDB[Hash hashdb.HashOut](
	hasher hashdb.Hasher[Hash],
	keyFunction KeyFunction[Hash],
) *MemoryDB[Hash] {
	data := []byte{0x0}
	return newFromNullNode(data, data, hasher, keyFunction)
}
