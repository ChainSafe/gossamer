// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"bytes"
	"iter"
	"maps"

	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	backend "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Re-exported types from backend package for simplicity.
type StorageKey = backend.StorageKey
type StorageValue = backend.StorageValue
type StorageKeyValue = backend.StorageKeyValue
type StorageCollection = backend.StorageCollection
type ChildStorageCollection = backend.ChildStorageCollection

var NoExtrinsicIndex uint32 = 0xffffffff

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
	overlayedChangeSet
	storage.ChildInfo
}

// Storage transactions are calculated as part of the `storage_root`.
// These transactions can be reused for importing the block into the
// storage. So, we cache them to not require a recomputation of those transactions.
type storageTransactionCache[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// Contains the changes for the main and the child storages as one transaction.
	transaction statemachine.BackendTransaction[H, Hasher]
	// The storage root after applying the transaction.
	transactionStorageRoot H
}

// The set of changes that are overlaid onto the
// It allows changes to be modified using nestable transactions.
type OverlayedChanges[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// Top level storage changes.
	top overlayedChangeSet
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
	storageTransactionCache *storageTransactionCache[H, Hasher]
}

func NewOverlayedChanges[H runtime.Hash, Hasher runtime.Hasher[H]]() *OverlayedChanges[H, Hasher] {
	return &OverlayedChanges[H, Hasher]{
		top:                     newOverlayedChangeSet(),
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
	entry, has := oc.top.Get(key)
	if !has {
		return nil, false
	}

	value := entry.Value()
	if value == nil {
		oc.stats.TallyReadModified(0)
		return nil, true
	}

	oc.stats.TallyReadModified(uint64(len(value)))
	return value, true
}

// Should be called when there are changes that require to reset the `storageTransactionCache`.
func (oc *OverlayedChanges[H, Hasher]) markDirty() {
	oc.storageTransactionCache = nil
}

// Returns (nil, false) if the key is unknown (i.e. and the query should be referred
// to the backend); (nil, true) if the key has been deleted. or a (value, true) for a key whose
// value has been set.
func (oc *OverlayedChanges[H, Hasher]) ChildStorage(childInfo storage.ChildInfo, key *string) ([]byte, bool) {
	childEntry, has := oc.children[string(childInfo.StorageKey())]

	if !has {
		return nil, false
	}

	entry, has := childEntry.overlayedChangeSet.Get(*key)
	if !has {
		oc.stats.TallyReadModified(0)
		return nil, true
	}

	value := entry.Value()
	oc.stats.TallyReadModified(uint64(len(value)))
	return value, true
}

// Set a new value for the specified key.
// Can be rolled back or committed when called inside a transaction.
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
	oc.top.set(key, value, extrinsicIndex)
}

// Append a element to storage, init with existing value if first write.
func (oc *OverlayedChanges[H, Hasher]) AppendStorage(
	key StorageKey,
	element StorageValue,
	init func() StorageValue,
) {
	extrinsicIndex := oc.extrinsicIndex()
	sizeWrite := uint64(len(element))
	oc.stats.TallyWriteOverlay(sizeWrite)
	oc.top.appendStorage(key, element, init, extrinsicIndex)
}

// Set a new value for the specified key and child.
// `nil` can be used to delete a value specified by the given key.
// Can be rolled back or committed when called inside a transaction.
func (oc *OverlayedChanges[H, Hasher]) SetChildStorage(
	childInfo storage.ChildInfo,
	key StorageKey,
	value StorageValue,
) {
	oc.markDirty()

	extrinsicIndex := oc.extrinsicIndex()

	sizeWrite := uint64(0)
	if value != nil {
		sizeWrite = uint64(len(value))
	}

	oc.stats.TallyWriteOverlay(sizeWrite)

	storageKey := childInfo.StorageKey()

	entry, has := oc.children[string(storageKey)]
	if !has {
		entry = childStorageValue{
			overlayedChangeSet: oc.top.SpawnChild(),
			ChildInfo:          childInfo,
		}
	}

	changeset := &entry.overlayedChangeSet
	info := entry.ChildInfo

	updatable := info.TryUpdate(childInfo)
	if !updatable {
		panic("ChildInfo mismatch, not updatable")
	}
	changeset.set(key, value, extrinsicIndex)
	oc.children[string(storageKey)] = entry
}

// Clear child storage of given storage key.
// Can be rolled back or committed when called inside a transaction.
func (oc *OverlayedChanges[H, Hasher]) ClearChildStorage(childInfo storage.ChildInfo) {
	oc.markDirty()

	extrinsicIndex := oc.extrinsicIndex()
	storageKey := childInfo.StorageKey()
	entry, has := oc.children[string(storageKey)]
	if !has {
		entry = childStorageValue{
			overlayedChangeSet: oc.top.SpawnChild(),
			ChildInfo:          childInfo,
		}
		oc.children[string(storageKey)] = entry
	}

	changeset := entry.overlayedChangeSet
	info := entry.ChildInfo

	updatable := info.TryUpdate(childInfo)
	if !updatable {
		panic("ChildInfo mismatch, not updatable")
	}

	changeset.clearWhere(func(key []byte, value *overlayedValue) bool {
		return true
	}, extrinsicIndex)
}

// Removes all key-value pairs which keys share the given prefix.
// Can be rolled back or committed when called inside a transaction.
func (oc *OverlayedChanges[H, Hasher]) ClearPrefix(prefix []byte) {
	oc.markDirty()

	extrinsicIndex := oc.extrinsicIndex()
	oc.top.clearWhere(func(key []byte, value *overlayedValue) bool {
		return bytes.HasPrefix(key, prefix)
	}, extrinsicIndex)
}

// Removes all key-value pairs which keys share the given prefix.
// Can be rolled back or committed when called inside a transaction
func (oc *OverlayedChanges[H, Hasher]) ClearChildPrefix(childInfo storage.ChildInfo, prefix []byte) {
	oc.markDirty()

	extrinsicIndex := oc.extrinsicIndex()
	storageKey := childInfo.StorageKey()
	entry, has := oc.children[string(storageKey)]
	if !has {
		entry = childStorageValue{
			overlayedChangeSet: oc.top.SpawnChild(),
			ChildInfo:          childInfo,
		}
		oc.children[string(storageKey)] = entry
	}

	changeset := entry.overlayedChangeSet
	info := entry.ChildInfo

	updatable := info.TryUpdate(childInfo)
	if !updatable {
		panic("ChildInfo mismatch, not updatable")
	}

	changeset.clearWhere(func(key []byte, value *overlayedValue) bool {
		return bytes.HasPrefix(key, prefix)
	}, extrinsicIndex)
}

// Returns the current nesting depth of the transaction stack.
// A value of zero means that no transaction is open and changes are committed on write.
func (oc *OverlayedChanges[H, Hasher]) TransactionDepth() uint {
	return oc.top.TransactionDepth()
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

// Rollback the last transaction started by `start_transaction`.
//
// Any changes made during that transaction are discarded. Returns an error if
// there is no open transaction that can be rolled back.
func (oc *OverlayedChanges[H, Hasher]) RollbackTransaction() error {
	oc.markDirty()

	if err := oc.top.rollbackTransaction(); err != nil {
		return err
	}

	maps.DeleteFunc(oc.children, func(key string, changeset childStorageValue) bool {
		if err := changeset.rollbackTransaction(); err != nil {
			panic("Top and children changesets are started in lockstep; qed")
		}

		return changeset.IsEmpty()
	})

	if err := oc.offchain.RollbackTransactionOffchain(); err != nil {
		panic("Top and offchain changesets are started in lockstep; qed")
	}

	return nil
}

// Commit the last transaction started by `start_transaction`.
//
// Any changes made during that transaction are committed. Returns an error if there
// is no open transaction that can be committed.
func (oc *OverlayedChanges[H, Hasher]) CommitTransaction() error {
	if err := oc.top.commitTransaction(); err != nil {
		return err
	}

	for _, changeset := range oc.children {
		if err := changeset.commitTransaction(); err != nil {
			panic("Top and children changesets are started in lockstep; qed")
		}
	}

	if err := oc.offchain.OverlayedMap.CommitTransactionOffchain(); err != nil {
		panic("Top and offchain changesets are started in lockstep; qed")
	}

	return nil
}

// Call this before transferring control to the runtime.
// This protects all existing transactions from being removed by the runtime.
// Calling this while already inside the runtime will return an error.
func (oc *OverlayedChanges[H, Hasher]) EnterRuntime() error {
	if err := oc.top.enterRuntime(); err != nil {
		return err
	}

	for _, changeset := range oc.children {
		if err := changeset.enterRuntime(); err != nil {
			panic("Top and children changesets are entering runtime in lockstep; qed")
		}
	}

	if err := oc.offchain.enterRuntime(); err != nil {
		panic("Top and offchain changesets are started in lockstep; qed")
	}

	return nil
}

// Call this when control returns from the runtime.
// This rollbacks all dangling transaction left open by the runtime.
// Calling this while outside the runtime will return an error.
func (oc *OverlayedChanges[H, Hasher]) ExitRuntime() error {
	if err := oc.top.exitRuntime(); err != nil {
		return err
	}

	for _, changeset := range oc.children {
		if err := changeset.exitRuntime(); err != nil {
			panic("Top and children changesets are entering runtime in lockstep; qed")
		}
	}

	if err := oc.offchain.exitRuntimeoffchain(); err != nil {
		panic("Top and offchain changesets are entering runtime in lockstep; qed")
	}

	return nil
}

// Consume all changes (top + children) and return them.
// After calling this function no more changes are contained in this changeset.
//
// Panics:
// Panics if `transaction_depth() > 0`
func (oc *OverlayedChanges[H, Hasher]) offchainDrainCommited() iter.Seq2[StorageKey, OffchainOverlayedChange] {
	return oc.offchain.DrainCommited()
}

// / Get an iterator over all child changes as seen by the current transaction.
func (oc *OverlayedChanges[H, Hasher]) Children() iter.Seq2[iter.Seq2[StorageKey, *OverlayedStorageEntry],
	storage.ChildInfo,
] {
	return func(yield func(iter.Seq2[StorageKey, *OverlayedStorageEntry], storage.ChildInfo) bool) {
		for _, child := range oc.children {
			if !yield(child.overlayedChangeSet.Changes(), child.ChildInfo) {
				return
			}
		}
	}
}

// Get an iterator over all top changes as been by the current transaction.
func (oc *OverlayedChanges[H, Hasher]) Changes() iter.Seq2[StorageKey, *OverlayedStorageEntry] {
	return oc.top.Changes()
}

// Get an optional iterator over all child changes stored under the supplied key.
func (oc *OverlayedChanges[H, Hasher]) ChildChanges(key StorageKey) (
	iter.Seq2[StorageKey, *OverlayedStorageEntry],
	storage.ChildInfo,
) {
	childChanges, has := oc.children[string(key)]
	if !has {
		return nil, nil
	}

	return childChanges.overlayedChangeSet.Changes(), childChanges.ChildInfo
}

// Inserts storage entry responsible for current extrinsic index.
func (oc *OverlayedChanges[H, Hasher]) SetExtrinsicIndex(index uint32) {
	oc.top.set(keys.ExtrinsicIndexKey, StorageValue(scale.MustMarshal(index)), nil)
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
		return &NoExtrinsicIndex
	}

	var result uint32
	err := scale.Unmarshal(val, &result)
	if err != nil {
		return &NoExtrinsicIndex
	}

	return &result
}

// Generate the storage root using `backend` and all changes
// as seen by the current transaction.
//
// Returns the storage root and whether it was already cached.
func (oc *OverlayedChanges[H, Hasher]) StorageRoot(
	b backend.Backend[H, Hasher],
	stateVersion storage.StateVersion,
) (H, bool) {
	if oc.storageTransactionCache != nil {
		return oc.storageTransactionCache.transactionStorageRoot, true
	}

	delta := make([]backend.Delta, 0)
	for key, value := range oc.top.Changes() {
		delta = append(delta, backend.Delta{Key: []byte(key), Value: value.Value()})
	}

	childDeltas := make([]backend.ChildDelta, 0)
	for _, child := range oc.children {
		deltas := make([]backend.Delta, 0)
		for key, value := range child.Changes() {
			deltas = append(deltas, backend.Delta{Key: []byte(key), Value: value.Value()})
		}

		childDeltas = append(childDeltas, backend.ChildDelta{
			ChildInfo: child.ChildInfo,
			Deltas:    deltas,
		})
	}

	root, tx := b.FullStorageRoot(delta, childDeltas, stateVersion)
	oc.storageTransactionCache = &storageTransactionCache[H, Hasher]{
		transaction:            tx,
		transactionStorageRoot: root,
	}

	return root, false
}

func (oc *OverlayedChanges[H, Hasher]) ChildStorageRoot(
	childInfo storage.ChildInfo,
	b backend.Backend[H, Hasher],
	stateVersion storage.StateVersion,
) (H, bool, error) {
	storageKey := StorageKey(childInfo.StorageKey())
	prefixedStorageKey := childInfo.PrefixedStorageKey()

	var root H

	if oc.storageTransactionCache != nil {
		value, has := oc.Storage(string(prefixedStorageKey))
		if !has {
			backendValue, err := b.Storage(prefixedStorageKey)
			if err != nil {
				return trie.EmptyChildTrieRoot[H, Hasher](), false, err
			}

			if backendValue == nil || scale.Unmarshal(backendValue, &value) != nil {
				root = trie.EmptyChildTrieRoot[H, Hasher]()
			}
		} else {
			hasher := *new(Hasher)
			root = hasher.NewHash(value)
		}

		return root, true, nil
	}

	var calculatedRoot H
	var isEmpty bool

	changes, info := oc.ChildChanges(storageKey)
	if changes == nil || info == nil {
		root = trie.EmptyChildTrieRoot[H, Hasher]()
	} else {
		delta := make([]backend.Delta, 0)
		for k, v := range changes {
			delta = append(delta, backend.Delta{Key: []byte(k), Value: v.Value()})
		}
		calculatedRoot, isEmpty, _ = b.ChildStorageRoot(info, delta, stateVersion)
	}

	if calculatedRoot != trie.EmptyChildTrieRoot[H, Hasher]() {
		if isEmpty {
			oc.SetStorage(StorageKey(prefixedStorageKey), nil)
		} else {
			oc.SetStorage(StorageKey(prefixedStorageKey), scale.MustMarshal(calculatedRoot))
		}
		oc.markDirty()
		root = calculatedRoot
	} else {
		// empty overlay
		backendValue, err := b.Storage(prefixedStorageKey)
		if err != nil {
			return trie.EmptyChildTrieRoot[H, Hasher](), false, err
		}

		err = scale.Unmarshal(backendValue, &root)
		if err != nil {
			root = trie.EmptyChildTrieRoot[H, Hasher]()
		}
	}

	return root, false, nil
}

func (oc *OverlayedChanges[H, Hasher]) IterAfter(key StorageKey) iter.Seq2[StorageKey,
	*OverlayedStorageEntry] {
	return oc.top.changesAfter(key)
}

func (oc *OverlayedChanges[H, Hasher]) ChildIterAfter(
	storageKey StorageKey,
	key StorageKey,
) iter.Seq2[StorageKey, *overlayedValue] {
	entry, has := oc.children[string(storageKey)]
	if !has {
		return nil
	}

	return entry.overlayedChangeSet.changesAfter(key)
}

func (oc *OverlayedChanges[H, Hasher]) SetOffchainStorage(key []byte, value []byte) {
	if value == nil {
		oc.offchain.Remove(offchain.StoragePrefix, key)
	} else {
		oc.offchain.Set(offchain.StoragePrefix, key, value)
	}
}

func (oc *OverlayedChanges[H, Hasher]) AddTransactionIndex(op IndexOperation) {
	oc.transactionIndexOps = append(oc.transactionIndexOps, op)
}
