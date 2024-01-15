// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import (
	"bytes"

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
	hasher         hashdb.Hasher[H]
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

func (self *MemoryDB[H]) Get(key H, prefix nibble.Prefix) *[]byte {
	if key.ComparableKey() == self.hashedNullNode.ComparableKey() {
		return &self.nullNodeData
	}

	key = self.keyFunction(key, prefix, self.hasher)
	value, ok := self.data[key.ComparableKey()]
	if ok && value.rc > 0 {
		return &value.value
	}

	return nil
}

func (self *MemoryDB[H]) Contains(key H, prefix nibble.Prefix) bool {
	if key.ComparableKey() == self.hashedNullNode.ComparableKey() {
		return true
	}

	key = self.keyFunction(key, prefix, self.hasher)
	value, ok := self.data[key.ComparableKey()]
	if ok && value.rc > 0 {
		return true
	}

	return false
}

func (self *MemoryDB[H]) Insert(prefix nibble.Prefix, value []byte) H {
	if bytes.Equal(value, self.nullNodeData) {
		return self.hashedNullNode
	}

	key := self.keyFunction(self.hasher.Hash(value), prefix, self.hasher)
	self.Emplace(key, prefix, value)
	return key
}

func (self *MemoryDB[H]) Emplace(key H, prefix nibble.Prefix, value []byte) {
	if bytes.Equal(value, self.nullNodeData) {
		return
	}

	key = self.keyFunction(key, prefix, self.hasher)

	newEntry := MemoryDBValue{
		value: value,
		rc:    1,
	}

	currentEntry, ok := self.data[key.ComparableKey()]
	if ok {
		if currentEntry.rc <= 0 {
			newEntry.value = value
		}
		newEntry.rc = currentEntry.rc + 1
	}

	self.data[key.ComparableKey()] = newEntry
}

func (self *MemoryDB[H]) Remove(key H, prefix nibble.Prefix) {
	if key.ComparableKey() == self.hashedNullNode.ComparableKey() {
		return
	}

	key = self.keyFunction(key, prefix, self.hasher)

	entry, ok := self.data[key.ComparableKey()]
	if ok {
		entry.rc--
		self.data[key.ComparableKey()] = entry
	} else {
		delete(self.data, key.ComparableKey())
	}
}

func NewMemoryDB[Hash hashdb.HashOut](
	hasher hashdb.Hasher[Hash],
	keyFunction KeyFunction[Hash],
) *MemoryDB[Hash] {
	data := []byte{0x0}
	return newFromNullNode(data, data, hasher, keyFunction)
}
