// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"iter"
	"maps"

	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var NoExtrinsicIndex uint32 = 0xffffffff

// OffchainChangesCollection is slice of storage values.
type OffchainChangesCollection []struct {
	PrefixKey struct {
		Prefix []byte
		Key    []byte
	}
	ValueOperation offchain.OffchainOverlayedChange
}

// IndexOperations is interface constraint of [IndexOperation].
type IndexOperations interface {
	IndexOperationInsert | IndexOperationRenew
}

// IndexOperation is a transaction index operation.
type IndexOperation interface {
	isIndexOperation()
}

// IndexOperationInsert is an insert transaction into index.
type IndexOperationInsert struct {
	// Extrinsic index in the current block.
	Extrinsic uint32
	// Data content hash.
	Hash []byte
	// Indexed data size.
	Size uint32
}

// IndexOperationRenew renews existing transaction storage.
type IndexOperationRenew struct {
	// Extrinsic index in the current block.
	Extrinsic uint32
	// Referenced index hash.
	Hash []byte
}

func (IndexOperationInsert) isIndexOperation() {}
func (IndexOperationRenew) isIndexOperation()  {}

type childStorageValue struct {
	OverlayedChangeSet
	ChildInfo
}

type OffchainOverlayedChange interface {
	isOffchainOverlayedChange()
}

type (
	OffchainOverlayedChangeRemove   struct{}
	OffchainOverlayedChangeSetValue []byte
)

func (OffchainOverlayedChangeRemove) isOffchainOverlayedChange()   {}
func (OffchainOverlayedChangeSetValue) isOffchainOverlayedChange() {}

type OffchainOverlayedChanges struct {
	OverlayedMap[string, OffchainOverlayedChange]
}

func NewOffchainOverlayedChanges() OffchainOverlayedChanges {
	return OffchainOverlayedChanges{
		NewOverlayedMap[string, OffchainOverlayedChange](),
	}
}

func (oc *OffchainOverlayedChanges) Clone() OffchainOverlayedChanges {
	return OffchainOverlayedChanges{
		oc.OverlayedMap.Clone(),
	}
}

// Remove a key and its associated value from the offchain database.
func (oc *OffchainOverlayedChanges) Set(prefix []byte, key []byte, value []byte) {
	prefixedKey := string(append(prefix, key...))
	oc.SetOffchain(prefixedKey, OffchainOverlayedChangeSetValue(value), nil)
}

// Remove a key and its associated value from the offchain database.
func (oc *OffchainOverlayedChanges) Remove(prefix []byte, key []byte) {
	prefixedKey := string(append(prefix, key...))
	oc.SetOffchain(prefixedKey, OffchainOverlayedChangeRemove{}, nil)
}

// Storage transactions are calculated as part of the `storage_root`.
// These transactions can be reused for importing the block into the
// storage. So, we cache them to not require a recomputation of those transactions.
type StorageTransactionCache[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// Contains the changes for the main and the child storages as one transaction.
	transaction backend.BackendTransaction[H, Hasher]
	// The storage root after applying the transaction.
	transactionStorageRoot H
}

// The set of changes that are overlaid onto the backend.
// It allows changes to be modified using nestable transactions.
type OverlayedChanges[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// Top level storage changes.
	top OverlayedChangeSet
	// Child storage changes. The map key is the child storage key without the common prefix.
	children map[string]childStorageValue
	// Offchain related changes.
	offchain OffchainOverlayedChanges
	// Transaction index changes,
	transactionIndexOps []IndexOperation
	// True if extrinsics stats must be collected.
	collectExtrinsics bool
	// Collect statistic on this execution.
	stats *StateMachineStats
	// Caches the "storage transaction" that is created while calling `storage_root`.
	// This transaction can be applied to the backend to persist the state changes.
	storageTransactionCache *StorageTransactionCache[H, Hasher]
}

func NewOverlayedChanges[H runtime.Hash, Hasher runtime.Hasher[H]]() *OverlayedChanges[H, Hasher] {
	return &OverlayedChanges[H, Hasher]{
		top:                     NewOverlayedChangeSet(),
		children:                make(map[string]childStorageValue),
		offchain:                NewOffchainOverlayedChanges(),
		transactionIndexOps:     make([]IndexOperation, 0),
		collectExtrinsics:       false,
		stats:                   NewStateMachineStats(),
		storageTransactionCache: nil,
	}
}

func (oc *OverlayedChanges[H, Hasher]) Clone() *OverlayedChanges[H, Hasher] {
	return &OverlayedChanges[H, Hasher]{
		top:                     oc.top.Clone(),
		children:                maps.Clone(oc.children),
		offchain:                oc.offchain.Clone(),
		transactionIndexOps:     oc.transactionIndexOps,
		collectExtrinsics:       oc.collectExtrinsics,
		stats:                   oc.stats.Clone(),
		storageTransactionCache: oc.storageTransactionCache,
	}
}

// Whether no changes are contained in the top nor in any of the child changes.
func (oc *OverlayedChanges[H, Hasher]) IsEmpty() bool {
	return oc.top.IsEmpty() && len(oc.children) == 0
}

// Ask to collect/not to collect extrinsics indices where key(s) has been changed.
func (oc *OverlayedChanges[H, Hasher]) SetCollectExtrinsic(collectExtrinsic bool) {
	oc.collectExtrinsics = collectExtrinsic
}

// Returns (nil, false) if the key is unknown (i.e. and the query should be referred
// to the backend); (nil, true) if the key has been deleted. or a (value, true) for a key whose
// value has been set.
func (oc *OverlayedChanges[H, Hasher]) Storage(key string) ([]byte, bool) {
	entry := oc.top.Get(key)
	if entry == nil {
		return nil, false
	}

	value := entry.StorageValue()
	if value == nil {
		oc.stats.TallyReadModified(0)
		return nil, true
	}

	oc.stats.TallyReadModified(uint64(len(value)))
	return value, true
}

// Should be called when there are changes that require to reset the
func (oc *OverlayedChanges[H, Hasher]) markDirty() {
	oc.storageTransactionCache = nil
}

// Returns (nil, false) if the key is unknown (i.e. and the query should be referred
// to the backend); (nil, true) if the key has been deleted. or a (value, true) for a key whose
// value has been set.
func (oc *OverlayedChanges[H, Hasher]) ChildStorage(childInfo ChildInfo, key *string) ([]byte, bool) {
	entry, has := oc.children[string(childInfo.StorageKey())]

	if !has {
		return nil, false
	}

	value := entry.OverlayedChangeSet.Get(*key).StorageValue()
	if value == nil {
		oc.stats.TallyReadModified(0)
		return nil, true
	}

	oc.stats.TallyReadModified(uint64(len(value)))
	return value, true
}

func (oc *OverlayedChanges[H, Hasher]) SetStorage(key StorageKey, value StorageValue) {
	oc.markDirty()

	var sizeWrite uint64
	if value == nil {
		sizeWrite = 0
	} else {
		sizeWrite = uint64(len(value))
	}

	oc.stats.TallyWriteOverlay(sizeWrite)
	extrinsicIndex := oc.extrinsicIndex()
	oc.top.Set(key, value, extrinsicIndex)
}

// Returns current extrinsic index to use in changes trie construction.
// nil is returned if it is not set or changes trie config is not set.
// Persistent value (from the backend) can be ignored because runtime must
// set this index before first and unset after last extrinsic is executed.
// Changes that are made outside of extrinsics, are marked with
// `NO_EXTRINSIC_INDEX` index.
func (oc *OverlayedChanges[H, Hasher]) extrinsicIndex() *uint32 {
	if !oc.collectExtrinsics {
		return nil
	}

	val, has := oc.Storage(string(keys.ExtrinsicIndexKey))
	if !has {
		return nil
	}

	var result uint32
	err := scale.Unmarshal(val, &result)
	if err != nil {
		return &NoExtrinsicIndex
	}

	return &result
}

// Start a new nested transaction.
//
// This allows to either commit or roll back all changes that where made while this
// transaction was open. Any transaction must be closed by either `rollback_transaction` or
// `commit_transaction` before this overlay can be converted into storage changes.
//
// Changes made without any open transaction are committed immediately.
func (oc *OverlayedChanges[H, Hasher]) StartTransaction() {
	oc.top.StartTransaction()
	for _, changeset := range oc.children {
		changeset.StartTransaction()
	}
	oc.offchain.OverlayedMap.StartTransaction()
}

// Commit the last transaction started by `start_transaction`.
//
// Any changes made during that transaction are committed. Returns an error if there
// is no open transaction that can be committed.
func (oc *OverlayedChanges[H, Hasher]) CommitTransaction() error {
	if err := oc.top.CommitTransaction(); err != nil {
		return err
	}

	for _, changeset := range oc.children {
		if err := changeset.CommitTransaction(); err != nil {
			panic("Top and children changesets are started in lockstep; qed")
		}
	}

	if err := oc.offchain.OverlayedMap.CommitTransactionOffchain(); err != nil {
		panic("Top and offchain changesets are started in lockstep; qed")
	}

	return nil
}

// Rollback the last transaction started by `start_transaction`.
//
// Any changes made during that transaction are discarded. Returns an error if
// there is no open transaction that can be rolled back.
func (oc *OverlayedChanges[H, Hasher]) RollbackTransaction() error {
	oc.markDirty()

	if err := oc.top.RollbackTransaction(); err != nil {
		return err
	}

	maps.DeleteFunc(oc.children, func(key string, changeset childStorageValue) bool {
		if err := changeset.RollbackTransaction(); err != nil {
			panic("Top and children changesets are started in lockstep; qed")
		}

		return changeset.IsEmpty()
	})

	if err := oc.offchain.RollbackTransactionOffchain(); err != nil {
		panic("Top and offchain changesets are started in lockstep; qed")
	}

	return nil
}

// Consume all changes (top + children) and return them.
//
// After calling this function no more changes are contained in this changeset.
//
// Panics:
// Panics if `transaction_depth() > 0`
func (oc *OverlayedChanges[H, Hasher]) offchainDrainCommited() iter.Seq2[string, OffchainOverlayedChange] {
	return oc.offchain.DrainCommited()
}

func (oc *OverlayedChanges[H, Hasher]) SetOffchainStorage(key []byte, value []byte) {
	if value == nil {
		oc.offchain.Remove(offchain.StoragePrefix, key)
	} else {
		oc.offchain.Set(offchain.StoragePrefix, key, value)
	}
}
