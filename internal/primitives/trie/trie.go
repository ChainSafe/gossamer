// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"math"
	"slices"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	"github.com/ChainSafe/gossamer/internal/primitives/kv"
	trieroot "github.com/ChainSafe/gossamer/internal/primitives/trie/trie-root"
	triedb "github.com/ChainSafe/gossamer/pkg/trie/triedb"
)

type Layout[H hashdb.Hash] interface {
	MaxInlineValue() int
	TrieRoot(input []kv.KeyValue) H
}

type (
	// substrate trie layout
	LayoutV0[Hasher hashdb.Hasher[H], H hashdb.Hash] struct{}
	// substrate trie layout, with external value nodes.
	LayoutV1[Hasher hashdb.Hasher[H], H hashdb.Hash] struct{}
)

func (l LayoutV0[Hasher, H]) MaxInlineValue() int {
	return math.MaxInt
}
func (l LayoutV1[Hasher, H]) MaxInlineValue() int {
	return 32
}

func (l LayoutV0[Hasher, H]) TrieRoot(input []kv.KeyValue) H {
	return trieroot.TrieRoot[Hasher, H](input, nil, NewTrieStream[Hasher, H]())
}

func (l LayoutV1[Hasher, H]) TrieRoot(input []kv.KeyValue) H {
	max := uint32(32)
	return trieroot.TrieRoot[Hasher, H](input, &max, NewTrieStream[Hasher, H]())
}

// PrefixedMemoryDB is reexport from [memorydb.MemoryDB] where supplied [memorydb.KeyFunction] is [memorydb.PrefixedKey]
// for prefixing keys internally (avoiding key conflict for non random keys).
type PrefixedMemoryDB[Hash hashdb.Hash, Hasher hashdb.Hasher[Hash]] struct {
	memorydb.MemoryDB[Hash, Hasher, string, memorydb.PrefixedKey[Hash]]
}

// NewPrefixedMemoryDB is constructor for [PrefixedMemoryDB]
func NewPrefixedMemoryDB[Hash hashdb.Hash, Hasher hashdb.Hasher[Hash]]() *PrefixedMemoryDB[Hash, Hasher] {
	return &PrefixedMemoryDB[Hash, Hasher]{
		memorydb.NewMemoryDB[Hash, Hasher, string, memorydb.PrefixedKey[Hash]]([]byte{0}),
	}
}

// MemoryDB is reexport from [memorydb.MemoryDB] where supplied [memorydb.KeyFunction] is [memorydb.HashKey] which is
// a noop operation on the supplied prefix, and only uses the hash.
type MemoryDB[Hash hashdb.Hash, Hasher hashdb.Hasher[Hash]] struct {
	memorydb.MemoryDB[Hash, Hasher, Hash, memorydb.HashKey[Hash]]
}

// NewMemoryDB is constructor for [MemoryDB].
func NewMemoryDB[Hash hashdb.Hash, Hasher hashdb.Hasher[Hash]]() *MemoryDB[Hash, Hasher] {
	return &MemoryDB[Hash, Hasher]{
		MemoryDB: memorydb.NewMemoryDB[Hash, Hasher, Hash, memorydb.HashKey[Hash]]([]byte{0}),
	}
}

// KeyValue is a byte slice for key and value, where the value can be optional (nil).
type KeyValue = kv.KeyValue

// DeltaTrieRoot determines a trie root given a hash DB and delta values.
func DeltaTrieRoot[H hashdb.Hash, Hasher hashdb.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	delta []KeyValue,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (H, error) {
	trieDB := triedb.NewTrieDB(
		root,
		db,
		stateVersion,
		triedb.WithCache[H, Hasher](cache),
		triedb.WithRecorder[H, Hasher](recorder),
	)

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

// ReadTrieValue reads a value from the trie.
func ReadTrieValue[H hashdb.Hash, Hasher hashdb.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) ([]byte, error) {
	trieDB := triedb.NewTrieDB(
		root, db, stateVersion,
		triedb.WithCache[H, Hasher](cache),
		triedb.WithRecorder[H, Hasher](recorder),
	)
	b, err := triedb.GetWith(trieDB, key, func(data []byte) []byte { return data })
	if err != nil {
		return nil, err
	}
	if b != nil {
		return *b, nil
	}
	return nil, nil
}

// ReadTrieValueWith reads a value from the trie with given [triedb.Query].
func ReadTrieValueWith[H hashdb.Hash, Hasher hashdb.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
	query triedb.Query[[]byte],
) ([]byte, error) {
	trieDB := triedb.NewTrieDB(
		root, db, stateVersion,
		triedb.WithCache[H, Hasher](cache),
		triedb.WithRecorder[H, Hasher](recorder),
	)
	b, err := triedb.GetWith(trieDB, key, query)
	if err != nil {
		return nil, err
	}
	if b != nil {
		return *b, nil
	}
	return nil, nil
}

// ReadTrieFirstDescendantValue reads the [triedb.MerkleValue] of the node that is the closest descendant for
// the provided key.
func ReadTrieFirstDescendantValue[H hashdb.Hash, Hasher hashdb.Hasher[H]](
	db hashdb.HashDB[H],
	root H,
	key []byte,
	recorder triedb.TrieRecorder,
	cache triedb.TrieCache[H],
	stateVersion triedb.TrieLayout,
) (triedb.MerkleValue[H], error) {
	trieDB := triedb.NewTrieDB(
		root, db, stateVersion,
		triedb.WithCache[H, Hasher](cache),
		triedb.WithRecorder[H, Hasher](recorder),
	)

	return trieDB.LookupFirstDescendant(key)
}

// EmptyTrieRoot returns the empty trie root.
func EmptyTrieRoot[H hashdb.Hash, Hasher hashdb.Hasher[H]]() H {
	hasher := *new(Hasher)
	root := hasher.Hash([]byte{0})
	return root
}

// EmptyChildTrieRoot returns the empty child trie root.
func EmptyChildTrieRoot[H hashdb.Hash, Hasher hashdb.Hasher[H]]() H {
	return EmptyTrieRoot[H, Hasher]()
}

// ChildDeltaTrieRoot determines a child trie root given a hash DB and delta values.
func ChildDeltaTrieRoot[H hashdb.Hash, Hasher hashdb.Hasher[H]](
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

// ReadChildTrieValue reads a value from the child trie.
func ReadChildTrieValue[H hashdb.Hash, Hasher hashdb.Hasher[H]](
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
		root, ksdb, stateVersion, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	val, err := triedb.GetWith(trieDB, key, func(data []byte) []byte { return data })
	if err != nil {
		return nil, err
	}
	if val != nil {
		return *val, nil
	}
	return nil, nil
}

// ReadChildTrieHash reads a hash from the child trie.
func ReadChildTrieHash[H hashdb.Hash, Hasher hashdb.Hasher[H]](
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
		root, ksdb, stateVersion, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	return trieDB.GetHash(key)
}

// ReadChildTrieFirstDescendantValue reads the [triedb.MerkleValue] of the node that is the closest descendant for
// the provided child key.
func ReadChildTrieFirstDescendantValue[H hashdb.Hash, Hasher hashdb.Hasher[H]](
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
		root, ksdb, stateVersion, triedb.WithCache[H, Hasher](cache), triedb.WithRecorder[H, Hasher](recorder))
	return trieDB.LookupFirstDescendant(key)
}

// KeyspacedDB is a [hashdb.HashDB] implementation that appends a keyspace (unique id bytes) in addition to the
// prefix of every key value.
type KeyspacedDB[Hash comparable] struct {
	db       hashdb.HashDB[Hash]
	keySpace []byte
}

// NewKeyspacedDB is constructor for [KeyspacedDB]
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

const (
	firstPrefix              uint8 = 0b_00 << 6
	leafPrefixMask           uint8 = 0b_01 << 6
	branchWithoutMask        uint8 = 0b_10 << 6
	branchWithMask           uint8 = 0b_11 << 6
	emptyTrie                uint8 = firstPrefix | (0b_00 << 4)
	altHashingLeafPrefixMask uint8 = firstPrefix | (0b_1 << 5)
	altHashingBranchWithMask uint8 = firstPrefix | (0b_01 << 4)
	escapeCompactHeader      uint8 = emptyTrie | 0b_00_01
)
