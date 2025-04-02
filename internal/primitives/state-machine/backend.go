// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"iter"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
)

// IterArgs is a struct containing arguments for iterating over the storage.
type IterArgs struct {
	// Prefix of the keys over which to iterate.
	Prefix []byte

	// StartAt is the prefix from which to start the iteration from.
	//
	// This is inclusive and the iteration will include the key which is specified here.
	StartAt []byte

	// If StartAtExclusive is true then the iteration will *not* include
	// the key specified in StartAt, if there is such a key.
	StartAtExclusive bool

	// ChildInfo is the info of the child trie over which to iterate over.
	ChildInfo storage.ChildInfo

	// StopOnIncompleteDatabase represents whether to stop iteration when a missing trie node is reached.
	//
	// When a missing trie node is reached the iterator will:
	//   - return an error if this is set to false (default)
	//   - return nil if this is set to true
	StopOnIncompleteDatabase bool
}

// StorageIterator is the interface for a raw storage iterator.
type StorageIterator[Hash runtime.Hash, Hasher runtime.Hasher[Hash]] interface {
	// Fetches the next key from the storage.
	NextKey(backend *TrieBackend[Hash, Hasher]) (overlayedchanges.StorageKey, error)

	// Fetches the next key and value from the storage.
	NextKeyValue(backend *TrieBackend[Hash, Hasher]) (*overlayedchanges.StorageKeyValue, error)

	// Returns whether the end of iteration was reached without an error.
	Complete() bool
}

// PairsIter is an iterator over storage keys and values.
type PairsIter[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	backend *TrieBackend[H, Hasher]
	rawIter StorageIterator[H, Hasher]
}

func (pi *PairsIter[H, Hasher]) Next() (*overlayedchanges.StorageKeyValue, error) {
	return pi.rawIter.NextKeyValue(pi.backend)
}

func (pi *PairsIter[H, Hasher]) All() iter.Seq2[overlayedchanges.StorageKeyValue, error] {
	return func(yield func(overlayedchanges.StorageKeyValue, error) bool) {
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

// KeysIter is an iterator over storage keys.
type KeysIter[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	backend *TrieBackend[H, Hasher]
	rawIter StorageIterator[H, Hasher]
}

func (ki *KeysIter[H, Hasher]) Next() (overlayedchanges.StorageKey, error) {
	return ki.rawIter.NextKey(ki.backend)
}

func (ki *KeysIter[H, Hasher]) All() iter.Seq2[overlayedchanges.StorageKey, error] {
	return func(yield func(overlayedchanges.StorageKey, error) bool) {
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

// BackendTransaction is the transaction type used by [Backend].
//
// This transaction contains all the changes that need to be applied to the backend to create the
// state for a new block.
type BackendTransaction[Hash runtime.Hash, Hasher runtime.Hasher[Hash]] struct {
	*trie.PrefixedMemoryDB[Hash, Hasher]
}

func NewBackendTransaction[Hash runtime.Hash, Hasher runtime.Hasher[Hash]]() BackendTransaction[Hash, Hasher] {
	return BackendTransaction[Hash, Hasher]{trie.NewPrefixedMemoryDB[Hash, Hasher]()}
}

// Delta is reexport of [trie.KeyValue]
type Delta = trie.KeyValue

// ChildDelta is child trie deltas
type ChildDelta struct {
	storage.ChildInfo
	Deltas []Delta
}

// A state Backend is used to read state data and can have changes committed to it.
type Backend[Hash runtime.Hash, H runtime.Hasher[Hash]] interface {
	// Storage gets keyed storage or nil if there is nothing associated.
	Storage(key []byte) (overlayedchanges.StorageValue, error)

	// StorageHash get keyed storage value hash or nil if there is nothing associated.
	StorageHash(key []byte) (*Hash, error)

	// ClosestMerkleValue gets the merkle value or nil if there is nothing associated.
	ClosestMerkleValue(key []byte) (triedb.MerkleValue[Hash], error)

	// ChildClosestMerkleValue gets the child merkle value or nil if there is nothing associated.
	ChildClosestMerkleValue(childInfo storage.ChildInfo, key []byte) (triedb.MerkleValue[Hash], error)

	// ChildStorage gets keyed child storage or nil if there is nothing associated.
	ChildStorage(childInfo storage.ChildInfo, key []byte) (overlayedchanges.StorageValue, error)

	// ChildStorageHash gets child keyed storage value hash or nil if there is nothing associated.
	ChildStorageHash(childInfo storage.ChildInfo, key []byte) (*Hash, error)

	// ExistsStorage returns true if a key exists in storage.
	ExistsStorage(key []byte) (bool, error)

	// ExistsChildStorage returns true if a key exists in child storage.
	ExistsChildStorage(childInfo storage.ChildInfo, key []byte) (bool, error)

	// NextStorageKey returns the next key in storage in lexicographic order or nil if there is no value.
	NextStorageKey(key []byte) (overlayedchanges.StorageKey, error)

	// NextChildStorageKey returns the next key in child storage in lexicographic order or nil if there is no value.
	NextChildStorageKey(childInfo storage.ChildInfo, key []byte) (overlayedchanges.StorageKey, error)

	// StorageRoot calculates the storage root, with given delta over what is already stored in
	// the backend, and produce a "transaction" that can be used to commit.
	// Does not include child storage updates.
	StorageRoot(delta []Delta, stateVersion storage.StateVersion) (Hash, BackendTransaction[Hash, H])

	// ChildStorageRoot calculates the child storage root, with given delta over what is already stored in
	// the backend, and produce a "transaction" that can be used to commit. The second argument
	// is true if child storage root equals default storage root.
	ChildStorageRoot(
		childInfo storage.ChildInfo, delta []Delta, stateVersion storage.StateVersion,
	) (Hash, bool, BackendTransaction[Hash, H])

	// RawIter returns a raw storage iterator.
	RawIter(args IterArgs) (StorageIterator[Hash, H], error)

	// Pairs returns an iterator over key/value pairs.
	Pairs(args IterArgs) (PairsIter[Hash, H], error)

	// Keys returns an iterator over keys.
	Keys(args IterArgs) (KeysIter[Hash, H], error)

	// FullStorageRoot calculates the storage root, with given delta over what is already stored
	// in the backend, and produce a "transaction" that can be used to commit.
	// Does include child storage updates.
	FullStorageRoot(
		delta []Delta, childDeltas []ChildDelta, stateVersion storage.StateVersion,
	) (Hash, BackendTransaction[Hash, H])
}
