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
	rc    int
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
		hasher:         hasher,
	}
}

// Raw returns the raw value for the given key
func (self *MemoryDB[H]) Raw(key H, prefix nibble.Prefix) *MemoryDBValue {
	if key.ComparableKey() == self.hashedNullNode.ComparableKey() {
		return &MemoryDBValue{
			value: self.nullNodeData,
			rc:    1,
		}
	}

	key = self.keyFunction(key, prefix, self.hasher)
	value, ok := self.data[key.ComparableKey()]
	if ok {
		return &value
	}

	return nil
}

// Get returns the value for the given key
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

// Contains returns true if the key exists in the database
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

// Insert inserts a value into the database and returns the key
func (self *MemoryDB[H]) Insert(prefix nibble.Prefix, value []byte) H {
	if bytes.Equal(value, self.nullNodeData) {
		return self.hashedNullNode
	}

	key := self.keyFunction(self.hasher.Hash(value), prefix, self.hasher)
	self.Emplace(key, prefix, value)
	return key
}

// Emplace inserts a value into the database an updates its reference count
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

// Remove removes reduces the reference count for that key by 1 or set -1 if the value does not exists
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
		self.data[key.ComparableKey()] = MemoryDBValue{
			value: nil,
			rc:    -1,
		}
	}
}

// RemoveAndPurge removes an element and delete it from storage if reference count reaches 0.
// If the value was purged, return the old value.
func (self *MemoryDB[H]) RemoveAndPurge(key H, prefix nibble.Prefix) *[]byte {
	if key.ComparableKey() == self.hashedNullNode.ComparableKey() {
		return nil
	}

	key = self.keyFunction(key, prefix, self.hasher)

	entry, ok := self.data[key.ComparableKey()]
	if ok {
		if entry.rc == 1 {
			delete(self.data, key.ComparableKey())
			return &entry.value
		}
		entry.rc--
		self.data[key.ComparableKey()] = entry
		return nil
	}

	self.data[key.ComparableKey()] = MemoryDBValue{
		value: nil,
		rc:    -1,
	}

	return nil
}

// Purge purges all zero-referenced data from the database
func (self *MemoryDB[H]) Purge() {
	for key, value := range self.data {
		if value.rc == 0 {
			delete(self.data, key)
		}
	}
}

// Drain returns the internal key-value map, clearing the current state
func (self *MemoryDB[H]) Drain() map[string]MemoryDBValue {
	data := self.data
	self.data = make(map[string]MemoryDBValue)
	return data
}

// Consolidate all the entries of `other` into `self`
func (self *MemoryDB[H]) Consolidate(other *MemoryDB[H]) {
	for key, dbvalue := range other.Drain() {
		entry, ok := self.data[key]
		if ok {
			if entry.rc < 0 {
				entry.value = dbvalue.value
			}
			entry.rc += dbvalue.rc
			self.data[key] = entry
		} else {
			self.data[key] = MemoryDBValue{
				value: dbvalue.value,
				rc:    dbvalue.rc,
			}
		}
	}
}

// NewMemoryDB creates a new memoryDB with a null node data
func NewMemoryDB[Hash hashdb.HashOut](
	hasher hashdb.Hasher[Hash],
	keyFunction KeyFunction[Hash],
) *MemoryDB[Hash] {
	data := []byte{0x0}
	return newFromNullNode(data, data, hasher, keyFunction)
}

// NewMemoryDBWithRoot creates a new memoryDB with a null node data and returns the DB and the root
func NewMemoryDBWithRoot[Hash hashdb.HashOut](
	hasher hashdb.Hasher[Hash],
	keyFunction KeyFunction[Hash],
) (*MemoryDB[Hash], Hash) {
	db := NewMemoryDB(hasher, keyFunction)
	root := db.hashedNullNode
	return db, root
}
