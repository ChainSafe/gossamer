// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"slices"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	triedb "github.com/ChainSafe/gossamer/pkg/trie/triedb"
)

// Reexport from [memorydb.MemoryDB] where supplied [memorydb.KeyFunction] is [memorydb.PrefixedKey] for prefixing
// keys internally (avoiding key conflict for non random keys).
type PrefixedMemoryDB[Hash runtime.Hash, Hasher hashdb.Hasher[Hash]] struct {
	memorydb.MemoryDB[Hash, Hasher, string, memorydb.PrefixedKey[Hash]]
}

// Constructor for [PrefixedMemoryDB]
func NewPrefixedMemoryDB[Hash runtime.Hash, Hasher hashdb.Hasher[Hash]]() *PrefixedMemoryDB[Hash, Hasher] {
	return &PrefixedMemoryDB[Hash, Hasher]{
		memorydb.NewMemoryDB[Hash, Hasher, string, memorydb.PrefixedKey[Hash]]([]byte{0}),
	}
}

// Reexport from [memorydb.MemoryDB] where supplied [memorydb.KeyFunction] is [memorydb.HashKey] which is a noop
// operation on the supplied prefix, and only uses the hash.
type MemoryDB[Hash runtime.Hash, Hasher runtime.Hasher[Hash]] struct {
	memorydb.MemoryDB[Hash, Hasher, Hash, memorydb.HashKey[Hash]]
}

// Constructor for [MemoryDB].
func NewMemoryDB[Hash runtime.Hash, Hasher runtime.Hasher[Hash]]() *MemoryDB[Hash, Hasher] {
	return &MemoryDB[Hash, Hasher]{
		MemoryDB: memorydb.NewMemoryDB[Hash, Hasher, Hash, memorydb.HashKey[Hash]]([]byte{0}),
	}
}

// KeyValue is a byte slice for key and value, where the value can be optional (nil).
type KeyValue struct {
	Key   []byte
	Value []byte
}

// Determine a trie root given a hash DB and delta values.
func DeltaTrieRoot[H runtime.Hash, Hasher runtime.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	delta []KeyValue,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (H, error) {
	trieDB := triedb.NewTrieDB(root, db, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)

	slices.SortStableFunc(delta, func(a KeyValue, b KeyValue) int {
		if string(a.Key) < string(b.Key) {
			return -1
		} else if string(a.Key) == string(b.Key) {
			return 0
		} else {
			return 1
		}
	})

	for i, kv := range delta {
		_ = i
		if kv.Value != nil {
			err := trieDB.Set(kv.Key, kv.Value)
			if err != nil {
				return *(new(H)), err
			}
		} else {
			err := trieDB.Delete(kv.Key)
			if err != nil {
				return *(new(H)), err
			}
		}
	}

	hash, err := trieDB.Hash()
	return hash, err
}

// Read a value from the trie.
func ReadTrieValue[H runtime.Hash, Hasher runtime.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) ([]byte, error) {
	trieDB := triedb.NewTrieDB(root, db, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)
	b, err := triedb.GetWith(trieDB, key, func(data []byte) []byte { return data })
	if err != nil {
		return nil, err
	}
	if b != nil {
		return *b, nil
	}
	return nil, nil
}

// Read a value from the trie with given [triedb.Query].
func ReadTrieValueWith[H runtime.Hash, Hasher runtime.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
	query triedb.Query[[]byte],
) ([]byte, error) {
	trieDB := triedb.NewTrieDB(root, db, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)
	b, err := triedb.GetWith(trieDB, key, query)
	if err != nil {
		return nil, err
	}
	if b != nil {
		return *b, nil
	}
	return nil, nil
}

// Read the [triedb.MerkleValue] of the node that is the closest descendant for
// the provided key.
func ReadTrieFirstDescendantValue[H runtime.Hash, Hasher runtime.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (triedb.MerkleValue[H], error) {
	trieDB := triedb.NewTrieDB(root, db, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)

	return trieDB.LookupFirstDescendant(key)
}

// EmptyTrieRoot returns the empty trie root.
func EmptyTrieRoot[H runtime.Hash, Hasher runtime.Hasher[H]]() H {
	hasher := *new(Hasher)
	root := hasher.Hash([]byte{0})
	return root
}

// EmptyChildTrieRoot returns the empty child trie root.
func EmptyChildTrieRoot[H runtime.Hash, Hasher runtime.Hasher[H]]() H {
	return EmptyTrieRoot[H, Hasher]()
}

// ChildDeltaTrieRoot determines a child trie root given a hash DB and delta values.
func ChildDeltaTrieRoot[H runtime.Hash, Hasher runtime.Hasher[H]](
	keyspace []byte,
	db hashdb.HashDB[H],
	root H,
	delta []KeyValue,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (H, error) {
	ksdb := NewKeyspacedDB(db, keyspace)
	return DeltaTrieRoot[H, Hasher](ksdb, root, delta, recorder, cache, stateVersion)
}

// Read a value from the child trie.
func ReadChildTrieValue[H runtime.Hash, Hasher runtime.Hasher[H]](
	keyspace []byte,
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) ([]byte, error) {
	ksdb := NewKeyspacedDB(db, keyspace)
	trieDB := triedb.NewTrieDB(
		root, ksdb, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)
	val, err := triedb.GetWith(trieDB, key, func(data []byte) []byte { return data })
	if err != nil {
		return nil, err
	}
	if val != nil {
		return *val, nil
	}
	return nil, nil
}

// Read a hash from the child trie.
func ReadChildTrieHash[H runtime.Hash, Hasher runtime.Hasher[H]](
	keyspace []byte,
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (*H, error) {
	ksdb := NewKeyspacedDB(db, keyspace)
	trieDB := triedb.NewTrieDB(
		root, ksdb, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)
	return trieDB.GetHash(key)
}

// Read the [triedb.MerkleValue] of the node that is the closest descendant for
// the provided child key.
func ReadChildTrieFirstDescendantValue[H runtime.Hash, Hasher runtime.Hasher[H]](
	keyspace []byte,
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (triedb.MerkleValue[H], error) {
	ksdb := NewKeyspacedDB(db, keyspace)
	trieDB := triedb.NewTrieDB(
		root, ksdb, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	trieDB.SetVersion(stateVersion)
	return trieDB.LookupFirstDescendant(key)
}

// KeyspacedDB is a [hashdb.HashDB] implementation that appends a keyspace (unique id bytes) in addition to the
// prefix of every key value.
type KeyspacedDB[Hash comparable] struct {
	db       hashdb.HashDB[Hash]
	keySpace []byte
}

// Constructor for [KeyspacedDB]
func NewKeyspacedDB[Hash comparable](db hashdb.HashDB[Hash], ks []byte) *KeyspacedDB[Hash] {
	return &KeyspacedDB[Hash]{
		db:       db,
		keySpace: ks,
	}
}

// Utility function used to merge some byte data (keyspace) and prefix data
// before calling key value database primitives.
func keyspaceAsPrefix(ks []byte, prefix hashdb.Prefix) hashdb.Prefix {
	result := ks
	result = append(result, prefix.Key...)
	return hashdb.Prefix{
		Key:    result,
		Padded: prefix.Padded,
	}
}

func (tbe *KeyspacedDB[H]) Get(key H, prefix hashdb.Prefix) []byte {
	derivedPrefix := keyspaceAsPrefix(tbe.keySpace, prefix)
	return tbe.db.Get(key, derivedPrefix)
}

func (tbe *KeyspacedDB[H]) Contains(key H, prefix hashdb.Prefix) bool {
	derivedPrefix := keyspaceAsPrefix(tbe.keySpace, prefix)
	return tbe.db.Contains(key, derivedPrefix)
}

func (tbe *KeyspacedDB[H]) Insert(prefix hashdb.Prefix, value []byte) H {
	derivedPrefix := keyspaceAsPrefix(tbe.keySpace, prefix)
	h := tbe.db.Insert(derivedPrefix, value)
	return h
}

func (tbe *KeyspacedDB[H]) Emplace(key H, prefix hashdb.Prefix, value []byte) {
	derivedPrefix := keyspaceAsPrefix(tbe.keySpace, prefix)
	tbe.db.Emplace(key, derivedPrefix, value)
}

func (tbe *KeyspacedDB[H]) Remove(key H, prefix hashdb.Prefix) {
	derivedPrefix := keyspaceAsPrefix(tbe.keySpace, prefix)
	tbe.db.Remove(key, derivedPrefix)
}
