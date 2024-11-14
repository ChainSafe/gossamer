// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"iter"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
)

// A struct containing arguments for iterating over the storage.
type IterArgs struct {
	// The prefix of the keys over which to iterate.
	Prefix []byte

	// The prefix from which to start the iteration from.
	//
	// This is inclusive and the iteration will include the key which is specified here.
	StartAt []byte

	// If this is true then the iteration will *not* include
	// the key specified in StartAt, if there is such a key.
	StartAtExclusive bool

	// The info of the child trie over which to iterate over.
	ChildInfo storage.ChildInfo

	// Whether to stop iteration when a missing trie node is reached.
	//
	// When a missing trie node is reached the iterator will:
	//   - return an error if this is set to false (default)
	//   - return nil if this is set to true
	StopOnIncompleteDatabase bool
}

// An interface for a raw storage iterator.
type StorageIterator[Hash runtime.Hash, Hasher runtime.Hasher[Hash]] interface {
	// Fetches the next key from the storage.
	NextKey(backend *TrieBackend[Hash, Hasher]) (StorageKey, error)

	// Fetches the next key and value from the storage.
	NextKeyValue(backend *TrieBackend[Hash, Hasher]) (*StorageKeyValue, error)

	// Returns whether the end of iteration was reached without an error.
	Complete() bool
}

// An iterator over storage keys and values.
type PairsIter[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	backend *TrieBackend[H, Hasher]
	rawIter StorageIterator[H, Hasher]
}

func (pi *PairsIter[H, Hasher]) Next() (*StorageKeyValue, error) {
	return pi.rawIter.NextKeyValue(pi.backend)
}

func (pi *PairsIter[H, Hasher]) All() iter.Seq2[StorageKeyValue, error] {
	return func(yield func(StorageKeyValue, error) bool) {
		for {
			item, err := pi.Next()
			if err != nil {
				return
			}
			if item == nil {
				return
			}
			if !yield(*item, err) {
				return
			}
		}
	}
}

// An iterator over storage keys.
type KeysIter[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	backend *TrieBackend[H, Hasher]
	rawIter StorageIterator[H, Hasher]
}

func (ki *KeysIter[H, Hasher]) Next() (StorageKey, error) {
	return ki.rawIter.NextKey(ki.backend)
}

func (ki *KeysIter[H, Hasher]) All() iter.Seq2[StorageKey, error] {
	return func(yield func(StorageKey, error) bool) {
		for {
			item, err := ki.Next()
			if err != nil {
				return
			}
			if item == nil {
				return
			}
			if !yield(item, err) {
				return
			}
		}
	}
}

// The transaction type used by [Backend].
//
// This transaction contains all the changes that need to be applied to the backend to create the
// state for a new block.
type BackendTransaction[Hash runtime.Hash, Hasher runtime.Hasher[Hash]] struct {
	*trie.PrefixedMemoryDB[Hash, Hasher]
}

func NewBackendTransaction[Hash runtime.Hash, Hasher runtime.Hasher[Hash]]() BackendTransaction[Hash, Hasher] {
	return BackendTransaction[Hash, Hasher]{trie.NewPrefixedMemoryDB[Hash, Hasher]()}
}

// Reexport of [trie.KeyValue]
type Delta = trie.KeyValue

type ChildDelta struct {
	storage.ChildInfo
	Deltas []Delta
}

// A state backend is used to read state data and can have changes committed
// to it.
//
// The clone operation (if implemented) should be cheap.
type Backend[Hash runtime.Hash, H runtime.Hasher[Hash]] interface {
	// Get keyed storage or nil if there is nothing associated.
	Storage(key []byte) (StorageValue, error)

	// Get keyed storage value hash or nil if there is nothing associated.
	StorageHash(key []byte) (*Hash, error)

	// Get the merkle value or nil if there is nothing associated.
	ClosestMerkleValue(key []byte) (triedb.MerkleValue[Hash], error)

	// Get the child merkle value or nil if there is nothing associated.
	ChildClosestMerkleValue(childInfo storage.ChildInfo, key []byte) (triedb.MerkleValue[Hash], error)

	// Get keyed child storage or nil if there is nothing associated.
	ChildStorage(childInfo storage.ChildInfo, key []byte) (StorageValue, error)

	// Get child keyed storage value hash or nil if there is nothing associated.
	ChildStorageHash(childInfo storage.ChildInfo, key []byte) (*Hash, error)

	// true if a key exists in storage.
	ExistsStorage(key []byte) (bool, error)

	// true if a key exists in child storage.
	ExistsChildStorage(childInfo storage.ChildInfo, key []byte) (bool, error)

	// Return the next key in storage in lexicographic order or nil if there is no value.
	NextStorageKey(key []byte) (StorageKey, error)

	// Return the next key in child storage in lexicographic order or nil if there is no value.
	NextChildStorageKey(childInfo storage.ChildInfo, key []byte) (StorageKey, error)

	// Calculate the storage root, with given delta over what is already stored in
	// the backend, and produce a "transaction" that can be used to commit.
	// Does not include child storage updates.
	StorageRoot(delta []Delta, stateVersion storage.StateVersion) (Hash, BackendTransaction[Hash, H])

	// Calculate the child storage root, with given delta over what is already stored in
	// the backend, and produce a "transaction" that can be used to commit. The second argument
	// is true if child storage root equals default storage root.
	ChildStorageRoot(
		childInfo storage.ChildInfo, delta []Delta, stateVersion storage.StateVersion,
	) (Hash, bool, BackendTransaction[Hash, H])

	// Returns a lifetimeless raw storage iterator.
	RawIter(args IterArgs) (StorageIterator[Hash, H], error)

	// Get an iterator over key/value pairs.
	Pairs(args IterArgs) (PairsIter[Hash, H], error)

	// Get an iterator over keys.
	Keys(args IterArgs) (KeysIter[Hash, H], error)

	// Calculate the storage root, with given delta over what is already stored
	// in the backend, and produce a "transaction" that can be used to commit.
	// Does include child storage updates.
	FullStorageRoot(
		delta []Delta, childDeltas []ChildDelta, stateVersion storage.StateVersion,
	) (Hash, BackendTransaction[Hash, H])
}
