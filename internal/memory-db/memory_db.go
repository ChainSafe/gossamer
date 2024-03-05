package memorydb

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"golang.org/x/exp/constraints"
)

type dataRC[T any] struct {
	Data T
	RC   int32
}

type Hash interface {
	constraints.Ordered
	Bytes() []byte
}

type Value interface {
	~[]byte
}

type MemoryDB[H Hash, Hasher hashdb.Hasher[H], Key constraints.Ordered, KF KeyFunction[H, Key], T Value] struct {
	data           map[Key]dataRC[T]
	hashedNullNode H
	nullNodeData   T
}

func NewMemoryDB[H Hash, Hasher hashdb.Hasher[H], Key constraints.Ordered, KF KeyFunction[H, Key], T Value](
	data []byte,
) MemoryDB[H, Hasher, Key, KF, T] {
	return newMemoryDBFromNullNode[H, Hasher, Key, KF, T](data, data)
}

func newMemoryDBFromNullNode[H Hash, Hasher hashdb.Hasher[H], Key constraints.Ordered, KF KeyFunction[H, Key], T Value](
	nullKey []byte,
	nullNodeData T,
) MemoryDB[H, Hasher, Key, KF, T] {
	return MemoryDB[H, Hasher, Key, KF, T]{
		data:           make(map[Key]dataRC[T]),
		hashedNullNode: (*new(Hasher)).Hash(nullKey),
		nullNodeData:   nullNodeData,
	}
}

// / Purge all zero-referenced data from the database.
func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Purge() {
	for k, val := range mdb.data {
		if val.RC == 0 {
			delete(mdb.data, k)
		}
	}
}

// / Return the internal key-value Map, clearing the current state.
func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Drain() map[Key]dataRC[T] {
	data := mdb.data
	mdb.data = make(map[Key]dataRC[T])
	return data
}

// / Grab the raw information associated with a key. Returns None if the key
// / doesn't exist.
// /
// / Even when Some is returned, the data is only guaranteed to be useful
// / when the refs > 0.
func (mdb *MemoryDB[H, Hasher, Key, KF, T]) raw(key H, prefix hashdb.Prefix) *dataRC[T] {
	if key == mdb.hashedNullNode {
		return &dataRC[T]{mdb.nullNodeData, 1}
	}
	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		return &data
	}
	return nil
}

// / Consolidate all the entries of `other` into `self`.
func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Consolidate(other *MemoryDB[H, Hasher, Key, KF, T]) {
	for key, value := range other.Drain() {
		entry, ok := mdb.data[key]
		if ok {
			if entry.RC < 0 {
				entry.Data = value.Data
			}

			entry.RC += value.RC
			mdb.data[key] = entry
		} else {
			mdb.data[key] = dataRC[T]{
				Data: value.Data,
				RC:   value.RC,
			}
		}
	}
}

// / Remove an element and delete it from storage if reference count reaches zero.
// / If the value was purged, return the old value.
func (mdb *MemoryDB[H, Hasher, Key, KF, T]) removeAndPurge(key H, prefix hashdb.Prefix) *T {
	if key == mdb.hashedNullNode {
		return nil
	}
	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		if data.RC == 1 {
			delete(mdb.data, kfKey)
			return &data.Data
		}
		data.RC -= 1
		mdb.data[kfKey] = data
		return nil
	}
	mdb.data[kfKey] = dataRC[T]{RC: -1}
	return nil
}

func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Get(key H, prefix hashdb.Prefix) *T {
	if key == mdb.hashedNullNode {
		return &mdb.nullNodeData
	}

	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		if data.RC > 0 {
			return &data.Data
		}
	}
	return nil
}

func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Contains(key H, prefix hashdb.Prefix) bool {
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

func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Emplace(key H, prefix hashdb.Prefix, value T) {
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
		mdb.data[kfKey] = dataRC[T]{value, 1}
	}
}

func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Insert(prefix hashdb.Prefix, value []byte) H {
	if string(mdb.nullNodeData) == string(value) {
		return mdb.hashedNullNode
	}

	key := (*new(Hasher)).Hash(value)
	mdb.Emplace(key, prefix, T(value))
	return key
}

func (mdb *MemoryDB[H, Hasher, Key, KF, T]) Remove(key H, prefix hashdb.Prefix) {
	if key == mdb.hashedNullNode {
		return
	}

	kfKey := (*new(KF)).Key(key, prefix)
	data, ok := mdb.data[kfKey]
	if ok {
		data.RC -= 1
		mdb.data[kfKey] = data
	} else {
		mdb.data[kfKey] = dataRC[T]{RC: -1}
	}
}

type KeyFunction[Hash constraints.Ordered, Key any] interface {
	Key(hash Hash, prefix hashdb.Prefix) Key
}

// / Key function that only uses the hash
type HashKey[H Hash] struct{}

func (HashKey[Hash]) Key(hash Hash, prefix hashdb.Prefix) Hash {
	return hash
}

// / Key function that concatenates prefix and hash.
type PrefixedKey[H Hash] struct{}

func (PrefixedKey[Hash]) Key(key Hash, prefix hashdb.Prefix) []byte {
	return NewPrefixedKey(key, prefix)
}

func NewPrefixedKey[H Hash](key H, prefix hashdb.Prefix) []byte {
	prefixedKey := prefix.Key
	if prefix.Padded != nil {
		prefixedKey = append(prefixedKey, *prefix.Padded)
	}
	prefixedKey = append(prefixedKey, key.Bytes()...)
	return prefixedKey
}
