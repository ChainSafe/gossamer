// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"encoding/binary"

	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
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
	overlay OverlayedChanges[H, Hasher]
	// The storage backend to read from.
	backend B
	// Pseudo-unique id used for tracing.
	Id uint16
}

func (e *Ext[H, Hasher, B]) SetOffchainStorage(key []byte, value []byte) {
	e.overlay.SetOffchainStorage(key, value)
}

func (e *Ext[H, Hasher, B]) Storage(key []byte) []byte {
	defer guard()

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

func (e *Ext[H, Hasher, B]) StorageHash(key []byte) []byte {
	defer guard()

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

func (e *Ext[H, Hasher, B]) ChildStorage(childInfo storage.ChildInfo, key []byte) []byte {
	defer guard()

	result, has := e.overlay.ChildStorage(childInfo, key)
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

func (e *Ext[H, Hasher, B]) ChildStorageHash(childInfo storage.ChildInfo, key []byte) []byte {
	defer guard()

	var hash H
	result, has := e.overlay.ChildStorage(childInfo, key)

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
		return scale.MustMarshal(result)
	}

	return nil
}

func (e *Ext[H, Hasher, B]) ExistsStorage(key []byte) bool {
	defer guard()

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

func (e *Ext[H, Hasher, B]) ExistsChildStorage(childInfo storage.ChildInfo, key []byte) bool {
	defer guard()

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

func (e Ext[H, Hasher, B]) NextStorageKey(key []byte) []byte {
	panic("TODO unimplemented")
}

func (e Ext[H, Hasher, B]) NextChildStorageKey(childInfo storage.ChildInfo, key []byte) []byte {
	panic("TODO unimplemented")
}

func (e Ext[H, Hasher, B]) PlaceStorage(key []byte, value []byte) {
	defer guard()

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

	defer guard()

	e.overlay.SetChildStorage(childInfo, key, value)
}

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

	defer guard()

	overlay := e.overlay.ClearChildStorage(childInfo)
	cursor, backend, loops := e.limitRemoveFromBackend(&childInfo, nil, maybeLimit, maybeCursor)
	return externalities.MultiRemovalResults{
		Cursor:  cursor,
		Backend: backend,
		Unique:  overlay + backend,
		Loops:   loops,
	}
}

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

	defer guard()

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

	defer guard()

	overlay := e.overlay.ClearChildPrefix(childInfo, prefix)
	cursor, backend, loops := e.limitRemoveFromBackend(&childInfo, prefix, limit, cursor)

	return externalities.MultiRemovalResults{
		Cursor:  cursor,
		Backend: backend,
		Unique:  overlay + backend,
		Loops:   loops,
	}
}

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

	defer guard()

	e.overlay.AppendStorage(key, value, func() StorageValue {
		def, err := e.backend.Storage(key)
		if err != nil {
			panic(ExtNotAllowedToFail)
		}

		return def
	})
}

func (e Ext[H, Hasher, B]) StorageRoot(stateVersion storage.StateVersion) []byte {
	defer guard()

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

func (e Ext[H, Hasher, B]) ChildStorageRoot(
	childInfo storage.ChildInfo,
	stateVersion storage.StateVersion,
) []byte {
	defer guard()

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

func (e Ext[H, Hasher, B]) StorageStartTransaction() {
	e.overlay.StartTransaction()
}

func (e Ext[H, Hasher, B]) StorageRollbackTransaction() error {
	_ = e.overlay.RollbackTransaction()
	return nil
}

func (e Ext[H, Hasher, B]) limitRemoveFromBackend(
	childInfo *storage.ChildInfo,
	prefix []byte,
	maybeLimit *uint32,
	startAt []byte,
) ([]byte, uint32, uint32) {
	panic("TODO: uninmplemented")
}

func leID(id uint16) []byte {
	IDLe := make([]byte, 2)
	binary.LittleEndian.PutUint16(IDLe, id)
	return IDLe
}

func guard() func() {
	return statemachine.NewGuard(statemachine.Abort).Done
}
