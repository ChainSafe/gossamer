// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import (
	"maps"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"golang.org/x/exp/constraints"
)

type dataRC struct {
	Data []byte
	RC   int32
}

type Hash interface {
	constraints.Ordered
	Bytes() []byte
}

type Value interface {
	~[]byte
}

// Reference-counted memory-based [hashdb.HashDB] implementation.
type MemoryDB[H Hash, Hasher hashdb.Hasher[H], Key constraints.Ordered, KF KeyFunction[H, Key]] struct {
	data           map[Key]dataRC
	hashedNullNode H
	nullNodeData   []byte
}

func NewMemoryDB[H Hash, Hasher hashdb.Hasher[H], Key constraints.Ordered, KF KeyFunction[H, Key]](
	data []byte,
) MemoryDB[H, Hasher, Key, KF] {
	return newMemoryDBFromNullNode[H, Hasher, Key, KF](data, data)
}

func newMemoryDBFromNullNode[H Hash, Hasher hashdb.Hasher[H], Key constraints.Ordered, KF KeyFunction[H, Key], T Value](
	nullKey []byte,
	nullNodeData T,
) MemoryDB[H, Hasher, Key, KF] {
	return MemoryDB[H, Hasher, Key, KF]{
		data:           make(map[Key]dataRC),
		hashedNullNode: (*new(Hasher)).Hash(nullKey),
		nullNodeData:   nullNodeData,
	}
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Clone() MemoryDB[H, Hasher, Key, KF] {
	return MemoryDB[H, Hasher, Key, KF]{
		data:           maps.Clone(mdb.data),
		hashedNullNode: mdb.hashedNullNode,
		nullNodeData:   mdb.nullNodeData,
	}
}

// Purge all zero-referenced data from the database.
func (mdb *MemoryDB[H, Hasher, Key, KF]) Purge() {
	for k, val := range mdb.data {
		if val.RC == 0 {
			delete(mdb.data, k)
		}
	}
}

// Return the internal key-value Map, clearing the current state.
func (mdb *MemoryDB[H, Hasher, Key, KF]) Drain() map[Key]dataRC {
	data := mdb.data
	mdb.data = make(map[Key]dataRC)
	return data
}

// Grab the raw information associated with a key. Returns None if the key
// doesn't exist.
//
// Even when Some is returned, the data is only guaranteed to be useful
// when the refs > 0.
func (mdb *MemoryDB[H, Hasher, Key, KF]) raw(key H, prefix hashdb.Prefix) *dataRC {
	if key == mdb.hashedNullNode {
		return &dataRC{mdb.nullNodeData, 1}
	}
	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		return &data
	}
	return nil
}

// Consolidate all the entries of other into self.
func (mdb *MemoryDB[H, Hasher, Key, KF]) Consolidate(other *MemoryDB[H, Hasher, Key, KF]) {
	for key, value := range other.Drain() {
		entry, ok := mdb.data[key]
		if ok {
			if entry.RC < 0 {
				entry.Data = value.Data
			}

			entry.RC += value.RC
			mdb.data[key] = entry
		} else {
			mdb.data[key] = dataRC{
				Data: value.Data,
				RC:   value.RC,
			}
		}
	}
}

// Remove an element and delete it from storage if reference count reaches zero.
// If the value was purged, return the old value.
func (mdb *MemoryDB[H, Hasher, Key, KF]) removeAndPurge(key H, prefix hashdb.Prefix) []byte {
	if key == mdb.hashedNullNode {
		return nil
	}
	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		if data.RC == 1 {
			delete(mdb.data, kfKey)
			return data.Data
		}
		data.RC -= 1
		mdb.data[kfKey] = data
		return nil
	}
	mdb.data[kfKey] = dataRC{RC: -1}
	return nil
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Get(key H, prefix hashdb.Prefix) []byte {
	if key == mdb.hashedNullNode {
		return mdb.nullNodeData
	}

	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		if data.RC > 0 {
			return data.Data
		}
	}
	return nil
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Contains(key H, prefix hashdb.Prefix) bool {
	if key == mdb.hashedNullNode {
		return true
	}

	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		if data.RC > 0 {
			return true
		}
	}
	return false
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Emplace(key H, prefix hashdb.Prefix, value []byte) {
	if string(mdb.nullNodeData) == string(value) {
		return
	}

	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		if data.RC <= 0 {
			data.Data = value
		}
		data.RC += 1
		mdb.data[kfKey] = data
	} else {
		mdb.data[kfKey] = dataRC{value, 1}
	}
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Insert(prefix hashdb.Prefix, value []byte) H {
	if string(mdb.nullNodeData) == string(value) {
		return mdb.hashedNullNode
	}

	key := (*new(Hasher)).Hash(value)
	mdb.Emplace(key, prefix, value)
	return key
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Remove(key H, prefix hashdb.Prefix) {
	if key == mdb.hashedNullNode {
		return
	}

	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		data.RC -= 1
		mdb.data[kfKey] = data
	} else {
		mdb.data[kfKey] = dataRC{RC: -1}
	}
}

func (mdb *MemoryDB[H, Hasher, Key, KF]) Keys() map[Key]int32 {
	keyCounts := make(map[Key]int32)
	for key, drc := range mdb.data {
		if drc.RC != 0 {
			keyCounts[key] = drc.RC
		}
	}
	return keyCounts
}

type KeyFunction[Hash constraints.Ordered, Key any] interface {
	Key(hash Hash, prefix hashdb.Prefix) Key
}

// Key function that only uses the hash
type HashKey[H Hash] struct{}

func (HashKey[Hash]) Key(hash Hash, prefix hashdb.Prefix) Hash {
	return hash
}

// Key function that concatenates prefix and hash.
type PrefixedKey[H Hash] struct{}

func (PrefixedKey[H]) Key(key H, prefix hashdb.Prefix) string {
	return string(NewPrefixedKey(key, prefix))
}

// Derive a database key from hash value of the node (key) and the node prefix.
func NewPrefixedKey[H Hash](key H, prefix hashdb.Prefix) []byte {
	prefixedKey := prefix.Key
	if prefix.Padded != nil {
		prefixedKey = append(prefixedKey, *prefix.Padded)
	}
	prefixedKey = append(prefixedKey, key.Bytes()...)
	return prefixedKey
}
