// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// ImportNotificationAction describes which block import notification stream should be notified.
type ImportNotificationAction uint

const (
	// RecentBlockImportNotificationAction notifies only when the node has synced to the tip or there is a re-org.
	RecentBlockImportNotificationAction ImportNotificationAction = iota
	// EveryBlockImportNotificationAction notifies for every single block no matter what the sync state is.
	EveryBlockImportNotificationAction
	// BothBlockImportNotificationAction means both [RecentBlockImportNotificationAction] and
	// [EveryBlockImportNotificationAction] should be fired.
	BothBlockImportNotificationAction
	// NoneBlockImportNotificationAction means no block import notification should be fired.
	NoneBlockImportNotificationAction
)

// StorageChanges contains a [statemachine.StorageCollection] and [statemachine.ChildStorageCollection]
type StorageChanges struct {
	backend.StorageCollection
	backend.ChildStorageCollection
}

// ImportSummary contains information about the block that just got imported,
// including storage changes, reorged blocks, etc.
type ImportSummary[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] struct {
	Hash           H                     // Block hash of the imported block.
	Origin         consensus.BlockOrigin // Import origin.
	Header         Header                // Header of the imported block.
	IsNewBest      bool                  // Is this block a new best block.
	StorageChanges *StorageChanges       // Optional storage changes.
	// TreeRoute from old best to new best.
	// If nil, there was no re-org while importing.
	TreeRoute                *blockchain.TreeRoute[H, N]
	ImportNotificationAction ImportNotificationAction // Which notify action to take for this import.
}

// FinalizeSummary contains information about the block that just got finalized, including tree heads that became
// stale at the moment of finalization.
type FinalizeSummary[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] struct {
	// Last finalized block header.
	Header Header
	// Blocks that were finalized.
	// The last entry is the one that has been explicitly finalized.
	Finalized []H
	// Heads that became stale during this finalization operation.
	StaleHeads []H
}

// ClientImportOperation is an import operation wrapper.
type ClientImportOperation[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	Op              BlockImportOperation[N, H, Hasher, Header, E] // DB Operation.
	NotifyImported  *ImportSummary[H, N, Header]                  // Summary of imported block.
	NotifyFinalized *FinalizeSummary[H, N, Header]                // Summary of finalized block.
}

// NewBlockState is the state of a new block.
type NewBlockState uint8

const (
	// NewBlockStateNormal is a normal block.
	NewBlockStateNormal NewBlockState = iota
	// NewBlockStateBest is a new best block.
	NewBlockStateBest
	// NewBlockStateFinal is a newly finalized block.
	NewBlockStateFinal
)

// IsBest returns true if equal to [NewBlockStateBest]
func (nbs NewBlockState) IsBest() bool {
	return nbs == NewBlockStateBest
}

// IsFinal returns true if equal to [NewBlockStateFinal]
func (nbs NewBlockState) IsFinal() bool {
	return nbs == NewBlockStateFinal
}

// BlockImportOperation keeps hold of the inserted block state and data.
type BlockImportOperation[
	N runtime.Number,
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	// State returns the pending state.
	// Returns nil for backends with locally-unavailable state data.
	State() (backend.Backend[H, Hasher], error)

	// SetBlockData will set block data to the transaction.
	SetBlockData(
		header Header,
		body []E,
		indexedBody [][]byte,
		justifications runtime.Justifications,
		state NewBlockState,
	) error

	// UpdateDBStorage will inject storage data into the database.
	UpdateDBStorage(update backend.BackendTransaction[H, Hasher]) error

	// SetGenesisState will set genesis state. If commit is false the state is saved in memory, but is not written
	// to the database.
	SetGenesisState(storage storage.Storage, commit bool, stateVersion storage.StateVersion) (H, error)

	// ResetStorage will inject storage data into the database replacing any existing data.
	ResetStorage(storage storage.Storage, stateVersion storage.StateVersion) (H, error)

	// UpdateStorage will set storage changes.
	UpdateStorage(update backend.StorageCollection, childUpdate backend.ChildStorageCollection) error

	// UpdateOffchainStorage will write offchain storage changes to the database.
	UpdateOffchainStorage(offchainUpdate overlayedchanges.OffchainChangesCollection) error

	// InsertAux will insert auxiliary keys.
	// Values that are nil respresent the keys should be deleted.
	InsertAux(ops AuxDataOperations) error

	// MarkFinalized marks a block as finalized.
	MarkFinalized(hash H, justification *runtime.Justification) error

	// MarkHead marks a block as the new head. If the block import changes the head and MarkHead is called with
	// a different block hash, MarkHead will override the changed head as a result of the block import.
	MarkHead(hash H) error

	// UpdateTransactionIndex adds a transaction index operation.
	UpdateTransactionIndex(index []overlayedchanges.IndexOperation) error
}

// LockImportRun is the interface for performing operations on the backend.
type LockImportRun[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	/// LockImportRun locks the import lock, and run operations inside.
	LockImportRun(
		f func(*ClientImportOperation[H, Hasher, N, Header, E]) error,
	) error
}

// KeyValue is used in [AuxStore.InsertAux].  Key and Value should not be nil.
type KeyValue struct {
	Key   []byte
	Value []byte
}

// AuxStore provides access to an auxiliary database.
//
// This is a simple global database not aware of forks. Can be used for storing auxiliary
// information like total block weight/difficulty for fork resolution purposes as a common use
// case.
type AuxStore interface {
	// Insert auxiliary data into key-value store.
	//
	// Deletions occur after insertions.
	InsertAux(insert []KeyValue, delete [][]byte) error

	// Query auxiliary data from key-value store.
	GetAux(key []byte) ([]byte, error)
}

// Backend is the client backend.
//
// Manages the data layer.
//
// # State Pruning
//
// While an object from StateAt is alive, the state
// should not be pruned. The backend should internally reference-count
// its state objects.
//
// The same applies for live BlockImportOperation instances: while an import operation building on a
// parent P is alive, the state for P should not be pruned.
//
// # Block Pruning
//
// Users can pin blocks in memory by calling PinBlock. When
// a block would be pruned, its value is kept in an in-memory cache
// until it is unpinned via UnpinBlock.
//
// While a block is pinned, its state is also preserved.
//
// The backend should internally reference count the number of pin / unpin calls.
type Backend[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	AuxStore // Insert auxiliary data into key-value store.

	// BeginOperation begins a new block insertion transaction with given parent block id.
	// When constructing the genesis, this is called with all-zero hash.
	BeginOperation() (BlockImportOperation[N, H, Hasher, Header, E], error)

	// BeginStateOperation notes an operation to contain state transition.
	BeginStateOperation(operation BlockImportOperation[N, H, Hasher, Header, E], block H) error

	// CommitOperation will commit block insertion.
	CommitOperation(transaction BlockImportOperation[N, H, Hasher, Header, E]) error

	// FinalizeBlock will finalize block with given hash.
	//
	// This should only be called if the parent of the given block has been finalized.
	FinalizeBlock(hash H, justification *runtime.Justification) error

	// AppendJustification appends justification to the block with the given hash.
	//
	// This should only be called for blocks that are already finalized.
	AppendJustification(hash H, justification runtime.Justification) error

	// Blockchain returns reference to blockchain backend.
	Blockchain() blockchain.Backend[H, N, Header, E]

	// OffchainStorage returns a pointer to offchain storage.
	OffchainStorage() offchain.OffchainStorage

	// PinBlock pins the block to keep body, justification and state available after pruning.
	// Number of pins are reference counted. Users need to make sure to perform
	// one call to UnpinBlock per call to PinBlock.
	PinBlock(hash H) error

	// Unpin the block to allow pruning.
	UnpinBlock(hash H)

	// HaveStateAt returns true if state for given block is available.
	HaveStateAt(hash H, number N) bool

	// StateAt returns state backend with post-state of given block.
	StateAt(hash H) (backend.Backend[H, Hasher], error)

	// Revert attempts to revert the chain by n blocks. If revertFinalized is set it will attempt to
	// revert past any finalized block. This is unsafe and can potentially leave the node in an
	// inconsistent state. All blocks higher than the best block are also reverted and not counting
	// towards n.
	//
	// Returns the number of blocks that were successfully reverted and the list of finalized
	// blocks that has been reverted.
	Revert(n N, revertFinalized bool) (N, map[H]any, error)

	// RemoveLeafBlock discards non-best, unfinalized leaf block.
	RemoveLeafBlock(hash H) error

	// GetImportLock returns access to the import lock for this backend.
	//
	// NOTE: Backend isn't expected to acquire the lock by itself ever. Rather
	// the using components should acquire and hold the lock whenever they do
	// something that the import of a block would interfere with, e.g. importing
	// a new block or calculating the best head.
	GetImportLock() *sync.RWMutex

	// RequiresFullSync tells whether the backend requires full-sync mode.
	RequiresFullSync() bool

	// UsageInfo returns current usage statistics.
	// TODO: implement UsageInfo if we require it
	// UsageInfo() *UsageInfo
}
