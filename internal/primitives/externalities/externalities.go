// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package externalities

import (
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/tidwall/btree"
)

// Id from any type
// e.g. reflect.TypeOf(v).String()
type TypeId string

type Extension interface {
	TypeId() TypeId
}

type Extensions struct {
	extensions btree.Map[string, Extension]
}

func NewExtensions() Extensions {
	return Extensions{
		extensions: btree.Map[string, Extension]{},
	}
}

// Results concerning an operation to remove many keys.
type MultiRemovalResults struct {
	// A continuation cursor which, if `Some` must be provided to the subsequent removal call.
	// If `None` then all removals are complete and no further calls are needed.
	Cursor []byte
	// The number of items removed from the backend database.
	Backend uint32
	// The number of unique keys removed, taking into account both the backend and the overlay.
	Unique uint32
	// The number of iterations (each requiring a storage seek/read) which were done.
	Loops uint32
}

// Something that provides access to the [Extensions] store.
// This is a super trait of the [Externalities].
type ExtensionStore interface {
	// Tries to find a registered extension by the given `typeId` and returns it
	ExtensionById(typeId TypeId) any

	// Register extension `extension` with specified `typeId`.
	RegisterExtensionWithTypeId(typeId TypeId, extension Extension)

	// Deregister extension with specified 'typeId' and drop it.
	DeregisterExtensionByTypeId(typeId TypeId) error
}

// Externalities provides access to the storage and to other registered extensions.
type Externalities interface {
	// SetOffchainStorage writes a key value pair to the offchain storage database.
	SetOffchainStorage(key []byte, value []byte)
	// Storage reads runtime storage.
	Storage(key []byte) []byte
	// StorageHash gets storage value hash.
	StorageHash(key []byte) []byte
	// ChildStorageHash gets child storage value hash.
	ChildStorageHash(childInfo storage.ChildInfo, key []byte) []byte
	// ChildStorage reads child runtime storage.
	ChildStorage(childInfo storage.ChildInfo, key []byte) []byte
	// SetStorage sets storage entry `key` of current contract being called (effective immediately).
	SetStorage(key []byte, value []byte)
	// SetChildStorage sets child storage entry `key` of current contract being called (effective immediately).
	SetChildStorage(childInfo storage.ChildInfo, key []byte, value []byte)
	// ClearStorage clears a storage entry (`key`) of current contract being called (effective immediately).
	ClearStorage(key []byte)
	// ClearChildStorage clears a child storage entry (`key`) of current contract being called (effective immediately).
	ClearChildStorage(childInfo storage.ChildInfo, key []byte)
	// ExistsStorage checks if a storage entry exists.
	ExistsStorage(key []byte) bool
	// ExistsChildStorage checks if a child storage entry exists.
	ExistsChildStorage(childInfo storage.ChildInfo, key []byte) bool
	// NextStorageKey returns the key immediately following the given key, if it exists.
	NextStorageKey(key []byte) []byte
	// NextChildStorageKey returns the key immediately following the given key, if it exists, in child storage.
	NextChildStorageKey(childInfo storage.ChildInfo, key []byte) []byte

	// Clear an entire child storage.
	//
	// Deletes all keys from the overlay and up to `maybeLimit` keys from the backend. No
	// limit is applied if `maybeLimit` is `nil`. Returns the cursor for the next call as `Some`
	// if the child trie deletion operation is incomplete. In this case, it should be passed into
	// the next call to avoid unaccounted iterations on the backend. Returns also the the number
	// of keys that were removed from the backend, the number of unique keys removed in total
	// (including from the overlay) and the number of backend iterations done.
	//
	// As long as `maybeCursor` is passed from the result of the previous call, then the number of
	// iterations done will only ever be one more than the number of keys removed.
	//
	// # Note
	//
	// An implementation is free to delete more keys than the specified limit as long as
	// it is able to do that in constant time.
	KillChildStorage(childInfo storage.ChildInfo, maybeLimit *uint32, maybeCursor []byte) MultiRemovalResults

	// ClearPrefix clears storage entries which keys are start with the given prefix.
	// `maybeLimit`, `maybeCursor` and result works as for `KillChildStorage`.
	ClearPrefix(prefix []byte, limit *uint32, cursor []byte) MultiRemovalResults

	// ClearChildPrefix clears child storage entries which keys are start with the given prefix.
	// `maybeLimit`, `maybeCursor` and result works as for `KillChildStorage`.
	ClearChildPrefix(childInfo storage.ChildInfo, prefix []byte, limit *uint32, cursor []byte) MultiRemovalResults

	// PlaceStorage set or clear a storage entry (`key`) of current contract being called (effective
	// immediately).
	PlaceStorage(key []byte, value []byte)

	// PlaceChildStorage sets or clears a child storage entry.
	PlaceChildStorage(childInfo storage.ChildInfo, key []byte, value []byte)

	// StorageRoot gets the trie root of the current storage map.
	// This will also update all child storage keys in the top-level storage map.
	// The returned hash is defined by the `Block` and is SCALE encoded.
	StorageRoot(stateVersion storage.StateVersion) []byte

	// ChildStorageRoot gets the trie root of a child storage map.
	//
	// This will also update the value of the child storage keys in the top-level storage map.
	//
	// If the storage root equals the default hash as defined by the trie, the key in the top-level
	// storage map will be removed.
	ChildStorageRoot(childInfo storage.ChildInfo, stateVersion storage.StateVersion) []byte

	// StorageAppend appends storage item.
	// This assumes specific format of the storage item. Also there is no way to undo this
	// operation.
	StorageAppend(key []byte, value []byte)

	// StorageStartTransaction starts a new nested transaction.
	//
	// This allows to either commit or roll back all changes made after this call to the
	// top changes or the default child changes. For every transaction there can be a
	// matching call to either `StorageRollbackTransaction` or `StorageCommitTransaction`.
	// Any transactions that are still open after returning from runtime are committed
	// automatically.
	//
	// Changes made without any open transaction are committed immediately.
	StorageStartTransaction()

	// StorageRollbackTransaction rollback the last transaction started by `StorageStartTransaction`.
	//
	// Any changes made during that storage transaction are discarded. Returns an error when
	// no transaction is open that can be closed.
	StorageRollbackTransaction() error

	// StorageCommitTransaction commits the current transaction.
	//
	// This will apply all changes made after the last call to `StorageStartTransaction`
	// and clear the transaction state.
	StorageCommitTransaction() error

	// StorageIndexTransaction indexes specified transaction slice and stores it.
	StorageIndexTransaction(index uint32, hash []byte, size uint32)

	// StorageRenewTransactionIndex renews existing piece of transaction storage.
	StorageRenewTransactionIndex(index uint32, hash []byte)
}
