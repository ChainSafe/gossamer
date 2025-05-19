// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/internal/saturating"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const ExtNotAllowedToFail = "Externalities not allowed to fail within runtime"

// storageAppend is a helper struct that appends a StorageValue to a slice.
type storageAppend struct {
	data *StorageValue
}

// newStorageAppend creates a new storageAppend instance.
func newStorageAppend(data *StorageValue) *storageAppend {
	return &storageAppend{data: data}
}

// extractLength extracts the scale encoded length of the StorageValue.
func (sa *storageAppend) extractLength() *uint {
	var length uint
	err := scale.Unmarshal(*sa.data, &length)
	if err != nil {
		return nil
	}
	return &length
}

// replaceLength replaces the length of the StorageValue using scale encoding.
func (sa *storageAppend) replaceLength(oldLength *uint, newLength uint) {
	oldLenEncodedLen := 0
	if oldLength != nil {
		oldLenEncodedLen = compactLen(*oldLength)
	}
	newLenEncoded, _ := scale.Marshal(newLength)
	data := spliceSlice(*sa.data, 0, oldLenEncodedLen, newLenEncoded)
	newStorageValue := StorageValue(data)
	*sa.data = newStorageValue
}

// appendRaw appends a raw byte slice to the current StorageValue.
func (sa *storageAppend) appendRaw(value []byte) {
	*sa.data = append(*sa.data, value...)
}

// spliceSlice is a helper function that replaces a slice of elements with a new slice in the given positions.
func spliceSlice[T any](slice []T, startIdx, endIdx int, replacement []T) []T {
	if startIdx < 0 || endIdx >= len(slice) || startIdx > endIdx {
		panic("invalid range")
	}

	result := make([]T, 0, len(slice)-endIdx+startIdx+len(replacement))
	result = append(result, slice[:startIdx]...)
	result = append(result, replacement...)
	result = append(result, slice[endIdx:]...)

	return result
}

func compactLen(val uint) int {
	switch {
	case val <= 0b0011_1111:
		return 1
	case val <= 0b0011_1111_1111_1111:
		return 2
	case val <= 0b0011_1111_1111_1111_1111_1111_1111_1111:
		return 4
	default:
		return 5
	}
}

// An overlayed extension is either a mutable reference
// or an owned extension.
type OverlayedExtension interface {
	isOverlayedExtension()
}

// Wraps a read-only backend, call executor, and current overlayed changes.
type Ext[H runtime.Hash, Hasher runtime.Hasher[H], B statemachine.Backend[H, Hasher]] struct {
	// The overlayed changes to write to.
	overlay *OverlayedChanges[H, Hasher]
	// The storage backend to read from.
	backend B
	// Pseudo-unique id used for tracing.
	Id uint16
}

// NewExt creates a new Ext instance.
func NewExt[H runtime.Hash, Hasher runtime.Hasher[H], B statemachine.Backend[H, Hasher]](
	overlay *OverlayedChanges[H, Hasher],
	backend B,
) *Ext[H, Hasher, B] {
	return &Ext[H, Hasher, B]{
		overlay: overlay,
		backend: backend,
		Id:      uint16(rand.Intn(65536)), //nolint:gosec
	}
}

// Write a key value pair to the offchain storage database.
func (e *Ext[H, Hasher, B]) SetOffchainStorage(key []byte, value []byte) {
	e.overlay.SetOffchainStorage(key, value)
}

// Read runtime storage.
func (e *Ext[H, Hasher, B]) Storage(key []byte) []byte {
	result, has := e.overlay.Storage(key)

	if !has || result == nil {
		var err error
		result, err = e.backend.Storage(key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}
	}

	logger.Tracef(
		`target = state 
		method = Get
		ext_id = %s
		key = %s
		result = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(key),
		common.BytesToHex(result),
	)

	return result
}

// Get storage value hash.
// This may be optimised for large values.
func (e *Ext[H, Hasher, B]) StorageHash(key []byte) []byte {
	var hash H

	result, has := e.overlay.Storage(key)

	if has && result != nil {
		hasher := *new(Hasher)
		hash = hasher.Hash(result)
	} else {
		result, err := e.backend.StorageHash(key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}

		if result != nil {
			hash = *result
		}
	}

	logger.Tracef(
		`target = state 
		method = Hash
		ext_id = %s
		key = %s
		result = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(key),
		hash.String(),
	)

	if hash.Bytes() != nil {
		return scale.MustMarshal(result)
	}

	return nil
}

// Read child runtime storage.
// Returns an SCALE encoded hash.
func (e *Ext[H, Hasher, B]) ChildStorage(childInfo storage.ChildInfo, key []byte) []byte {

	result, has := e.overlay.ChildStorage(childInfo, key)
	if has && result == nil {
		return nil
	}

	if !has || result == nil {
		var err error
		result, err = e.backend.ChildStorage(childInfo, key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}
	}

	logger.Tracef(
		`target = state 
		method = ChildGet
		ext_id = %s
		child_info = %s
		key = %s
		result = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHash(childInfo.StorageKey()),
		common.BytesToHex(key),
		common.BytesToHex(result),
	)

	return result
}

// Get child storage value hash.
// This may be optimised for large values.
// Returns an SCALE encoded hash.
func (e *Ext[H, Hasher, B]) ChildStorageHash(childInfo storage.ChildInfo, key []byte) []byte {
	var hash H
	result, has := e.overlay.ChildStorage(childInfo, key)

	if has && result == nil {
		return nil
	}

	if has && result != nil {
		hasher := *new(Hasher)
		hash = hasher.Hash(result)
	} else {
		result, err := e.backend.ChildStorageHash(childInfo, key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}

		if result != nil {
			hash = *result
		}
	}

	logger.Tracef(
		`target = state 
		method = ChildHash
		ext_id = %s
		child_info = %s
		key = %s
		result = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHash(childInfo.StorageKey()),
		common.BytesToHex(key),
		hash.String(),
	)

	if hash.Bytes() != nil {
		return scale.MustMarshal(hash)
	}

	return nil
}

// Whether a storage entry exists.
func (e *Ext[H, Hasher, B]) ExistsStorage(key []byte) bool {
	var exists bool

	value, has := e.overlay.Storage(key)
	if has {
		exists = value != nil
	} else {
		var err error
		exists, err = e.backend.ExistsStorage(key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}
	}

	logger.Tracef(
		`target = state 
		method = Exists
		ext_id = %s
		key = %s
		result = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(key),
		exists,
	)

	return exists
}

// Whether a child storage entry exists.
func (e *Ext[H, Hasher, B]) ExistsChildStorage(childInfo storage.ChildInfo, key []byte) bool {
	var exists bool

	value, has := e.overlay.ChildStorage(childInfo, key)
	if has {
		exists = value != nil
	} else {
		var err error
		exists, err = e.backend.ExistsChildStorage(childInfo, key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}
	}

	logger.Tracef(
		`target = state 
		method = ChildExists
		ext_id = %s
		child_info = %s
		key = %s
		result = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHash(childInfo.StorageKey()),
		common.BytesToHex(key),
		exists,
	)

	return exists
}

// Returns the key immediately following the given key, if it exists.
func (e Ext[H, Hasher, B]) NextStorageKey(key []byte) []byte {
	nextBackendKey, err := e.backend.NextStorageKey(key)

	if err != nil {
		panic(ExtNotAllowedToFail)
	}

	overlayedChangesIter := e.overlay.IterAfter(key)
	overlayChanges := common.NewPeekable2(overlayedChangesIter)

	_, _, has := overlayChanges.Peek()

	if !has {
		return nextBackendKey
	}

	if nextBackendKey != nil && has {
		for overlayKey, overlayValue := range overlayedChangesIter {
			cmp := bytes.Compare(nextBackendKey, overlayKey)

			// If nextBackendKey is less than the overlayKey, we found out next key.
			if cmp == -1 {
				return nextBackendKey
			} else if overlayValue.Value() != nil {
				// If there exists a value for the overlayKey in the overlay
				// (aka the key is still valid), it means we have found our next key.
				return overlayKey
			} else if cmp == 0 {
				// If the nextBackendKey and overlayKey are equal, it means that we need
				// to search for the next backend key, because the overlay has overwritten
				// this key.
				nextBackendKey, err = e.backend.NextStorageKey(overlayKey)
				if err != nil {
					panic(ExtNotAllowedToFail)
				}
			}
		}

		return nextBackendKey
	}

	for k, v := range overlayedChangesIter {
		if v.Value() != nil {
			return k
		}
	}

	return nil
}

// Returns the key immediately following the given key, if it exists, in child storage.
func (e Ext[H, Hasher, B]) NextChildStorageKey(childInfo storage.ChildInfo, key []byte) []byte {
	nextBackendKey, err := e.backend.NextChildStorageKey(childInfo, key)

	if err != nil {
		panic(ExtNotAllowedToFail)
	}

	overlayedChangesIter := e.overlay.ChildIterAfter(childInfo.StorageKey(), key)
	overlayChanges := common.NewPeekable2(overlayedChangesIter)

	_, _, has := overlayChanges.Peek()

	if !has {
		return nextBackendKey

	}

	if nextBackendKey != nil && has {
		for overlayKey, overlayValue := range overlayedChangesIter {
			var cmp int
			if nextBackendKey != nil {
				cmp = bytes.Compare(nextBackendKey, overlayKey)
			}

			// If nextBackendKey is less than the overlayKey, we found out next key.
			if cmp == -1 {
				return nextBackendKey
			} else if overlayValue.Value() != nil {
				// If there exists a value for the overlayKey in the overlay
				// (aka the key is still valid), it means we have found our next key.
				return overlayKey
			} else if cmp == 0 {
				// If the nextBackendKey and overlayKey are equal, it means that we need
				// to search for the next backend key, because the overlay has overwritten
				// this key.
				nextBackendKey, err = e.backend.NextChildStorageKey(childInfo, overlayKey)
				if err != nil {
					panic(ExtNotAllowedToFail)
				}
			}
		}

		return nextBackendKey
	}
	for k, v := range overlayedChangesIter {
		if v.Value() != nil {
			return k
		}
	}

	return nil
}

// Set storage entry key of current contract being called (effective immediately).
func (e Ext[H, Hasher, B]) SetStorage(key []byte, value []byte) {
	e.PlaceStorage(key, value)
}

// Set child storage entry key of current contract being called (effective immediately).
func (e Ext[H, Hasher, B]) SetChildStorage(childInfo storage.ChildInfo, key []byte, value []byte) {
	e.PlaceChildStorage(childInfo, key, value)
}

// Clear a storage entry key of current contract being called (effective immediately).
func (e Ext[H, Hasher, B]) ClearStorage(key []byte) {
	e.PlaceStorage(key, nil)
}

// Clear a child storage entry key of current contract being called (effective immediately).
func (e Ext[H, Hasher, B]) ClearChildStorage(childInfo storage.ChildInfo, key []byte) {
	e.PlaceChildStorage(childInfo, key, nil)
}

// Set or clear a storage entry key of current contract being called (effective immediately).
func (e Ext[H, Hasher, B]) PlaceStorage(key []byte, value []byte) {
	if keys.IsChildStorageKey(key) {
		logger.Warnf("refuse to directly set child storage key")
		return
	}

	logger.Tracef(
		`target = state 
		method = Put
		ext_id = %s
		key = %s
		value = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(key),
		common.BytesToHex(value),
	)

	e.overlay.SetStorage(key, value)
}

// Set or clear a child storage entry.
func (e Ext[H, Hasher, B]) PlaceChildStorage(
	childInfo storage.ChildInfo,
	key []byte,
	value []byte,
) {
	logger.Tracef(
		`target = state 
		method = ChildPut
		ext_id = %s
		child_info = %s
		key = %s
		value = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(childInfo.StorageKey()),
		common.BytesToHex(key),
		common.BytesToHex(value),
	)

	e.overlay.SetChildStorage(childInfo, key, value)
}

// Clear an entire child storage.
//
// Deletes all keys from the overlay and up to maybeLimit keys from the backend. No
// limit is applied if maybeLimit is nil. Returns the cursor for the next call
// if the child trie deletion operation is incomplete. In this case, it should be passed into
// the next call to avoid unaccounted iterations on the backend. Returns also the the number
// of keys that were removed from the backend, the number of unique keys removed in total
// (including from the overlay) and the number of backend iterations done.
//
// As long as maybeCursor is passed from the result of the previous call, then the number of
// iterations done will only ever be one more than the number of keys removed.
func (e Ext[H, Hasher, B]) KillChildStorage(
	childInfo storage.ChildInfo,
	maybeLimit *uint32,
	maybeCursor []byte,
) externalities.MultiRemovalResults {
	logger.Tracef(
		`target = state 
		method = ChildKill
		ext_id = %s
		child_info = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(childInfo.StorageKey()),
	)

	overlay := e.overlay.ClearChildStorage(childInfo)
	cursor, backend, loops := e.limitRemoveFromBackend(childInfo, nil, maybeLimit, maybeCursor)
	return externalities.MultiRemovalResults{
		Cursor:  cursor,
		Backend: backend,
		Unique:  overlay + backend,
		Loops:   loops,
	}
}

// Clear storage entries which keys are start with the given prefix.
// limit, cursor and result works as for [Ext.KillChildStorage].
func (e Ext[H, Hasher, B]) ClearPrefix(
	prefix []byte,
	limit *uint32,
	cursor []byte,
) externalities.MultiRemovalResults {
	logger.Tracef(
		`target = state 
		method = ClearPrefix
		ext_id = %s
		prefix = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(prefix),
	)

	if keys.StartsWithChildStorageKey(prefix) {
		logger.Warnf("refuse to directly clear prefix that is part or contains of child storage key")
		return externalities.MultiRemovalResults{}
	}

	overlay := e.overlay.ClearPrefix(prefix)
	cursor, backend, loops := e.limitRemoveFromBackend(nil, prefix, limit, cursor)

	return externalities.MultiRemovalResults{
		Cursor:  cursor,
		Backend: backend,
		Unique:  overlay + backend,
		Loops:   loops,
	}
}

// Clear child storage entries which keys are start with the given prefix.
// limit, cursor and result works as for [Ext.KillChildStorage].
func (e Ext[H, Hasher, B]) ClearChildPrefix(
	childInfo storage.ChildInfo,
	prefix []byte,
	limit *uint32,
	cursor []byte,
) externalities.MultiRemovalResults {
	logger.Tracef(
		`target = state 
		method = ChildClearPrefix
		ext_id = %s
		child_info = %s
		prefix = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(childInfo.StorageKey()),
		common.BytesToHex(prefix),
	)

	overlay := e.overlay.ClearChildPrefix(childInfo, prefix)
	cursor, backend, loops := e.limitRemoveFromBackend(childInfo, prefix, limit, cursor)

	return externalities.MultiRemovalResults{
		Cursor:  cursor,
		Backend: backend,
		Unique:  overlay + backend,
		Loops:   loops,
	}
}

// Append storage item.
// This assumes specific format of the storage item. Also there is no way to undo this operation.
func (e Ext[H, Hasher, B]) StorageAppend(key []byte, value []byte) {
	logger.Tracef(
		`target = state 
		method = Append
		ext_id = %s
		key = %s
		value = %s`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(key),
		common.BytesToHex(value),
	)

	e.overlay.AppendStorage(key, value, func() StorageValue {
		def, err := e.backend.Storage(key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}

		return def
	})
}

// Get the trie root of the current storage map.
// This will also update all child storage keys in the top-level storage map.
// The returned hash is defined by the `Block` and is SCALE encoded.
func (e Ext[H, Hasher, B]) StorageRoot(stateVersion storage.StateVersion) []byte {
	root, cached := e.overlay.StorageRoot(e.backend, stateVersion)

	logger.Tracef(
		`target = state 
		method = StorageRoot
		ext_id = %s
		storage_root = %s
		cached = %v`,
		common.BytesToHex(leID(e.Id)),
		root.String(),
		cached,
	)

	return scale.MustMarshal(root)
}

// Get the trie root of a child storage map.
// This will also update the value of the child storage keys in the top-level storage map.
// If the storage root equals the default hash as defined by the trie, the key in the top-level
// storage map will be removed.
func (e Ext[H, Hasher, B]) ChildStorageRoot(
	childInfo storage.ChildInfo,
	stateVersion storage.StateVersion,
) []byte {
	root, cached, err := e.overlay.ChildStorageRoot(childInfo, e.backend, stateVersion)
	if err != nil {
		panic(ExtNotAllowedToFail)
	}

	logger.Tracef(
		`target = state 
		method = ChildStorageRoot
		ext_id = %s
		child_info = %s
		storage_root = %s
		cached = %v`,
		common.BytesToHex(leID(e.Id)),
		common.BytesToHex(childInfo.StorageKey()),
		root.String(),
		cached,
	)

	return scale.MustMarshal(root)
}

// Index specified transaction slice and store it.
func (e Ext[H, Hasher, B]) StorageIndexTransaction(index uint32, hash []byte, size uint32) {
	logger.Tracef(
		`target = state 
		method = RenewTransactionIndex
		ext_id = %s
		index = %d
		tx_hash = %s`,
		common.BytesToHex(leID(e.Id)),
		index,
		common.BytesToHex(hash),
	)

	e.overlay.AddTransactionIndex(IndexOperationRenew{Extrinsic: index, Hash: hash})
}

// Start a new nested transaction.
//
// This allows to either commit or roll back all changes made after this call to the
// top changes or the default child changes. For every transaction there cam be a
// matching call to either [Ext.StorageRollbackTransaction] or [Ext.StorageCommitTransaction].
// Any transactions that are still open after returning from runtime are committed
// automatically.
//
// Changes made without any open transaction are committed immediately.
func (e Ext[H, Hasher, B]) StorageStartTransaction() {
	e.overlay.StartTransaction()
}

// Rollback the last transaction started by [Ext.StorageStartTransaction].
// Any changes made during that storage transaction are discarded. Returns an error when
// no transaction is open that can be closed.
func (e Ext[H, Hasher, B]) StorageRollbackTransaction() error {
	_ = e.overlay.RollbackTransaction()
	return nil
}

// Commit the last transaction started by [Ext.StorageStartTransaction].
// Any changes made during that storage transaction are committed. Returns an error when
// no transaction is open that can be closed.
func (e Ext[H, Hasher, B]) StorageCommitTransaction() error {
	_ = e.overlay.CommitTransaction()
	return nil
}

func (e Ext[H, Hasher, B]) limitRemoveFromBackend(
	childInfo storage.ChildInfo,
	prefix []byte,
	maybeLimit *uint32,
	startAt []byte,
) ([]byte, uint32, uint32) {
	iter, err := e.backend.Keys(statemachine.IterArgs{
		ChildInfo: childInfo,
		Prefix:    prefix,
		StartAt:   startAt,
	})

	if err != nil {
		logger.Debugf("error while iterating the storage: %w", err)
	}

	deleteCount := uint32(0)
	loopCount := uint32(0)
	var maybeNextKey []byte

	for key, err := range iter.All() {
		if err != nil {
			logger.Debugf("error while iterating the storage: %w", err)
			break
		}

		if maybeLimit != nil && *maybeLimit == loopCount {
			maybeNextKey = key
			break
		}

		var has bool
		var overlay []byte

		if childInfo != nil {
			overlay, has = e.overlay.ChildStorage(childInfo, key)
		} else {
			overlay, has = e.overlay.Storage(key)
		}

		if has {
			// not pending deletion from the backend - delete it.
			if overlay != nil {
				e.overlay.SetChildStorage(childInfo, key, nil)
			} else {
				e.overlay.SetStorage(key, nil)
			}

			deleteCount = saturating.Add(deleteCount, 1)
		}
		loopCount = saturating.Add(loopCount, 1)
	}

	return maybeNextKey, deleteCount, loopCount
}

func leID(id uint16) []byte {
	IDLe := make([]byte, 2)
	binary.LittleEndian.PutUint16(IDLe, id)
	return IDLe
}
