// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// / State of a new block.
type NewBlockState uint

const (
	/// Normal block.
	NewBlockStateNormal NewBlockState = iota
	/// New best block.
	NewBlockStateBest
	/// Newly finalized block (implicitly best).
	NewBlockStateFinal
)

func (nbs NewBlockState) IsBest() bool {
	return nbs == NewBlockStateBest
}

func (nbs NewBlockState) IsFinal() bool {
	return nbs == NewBlockStateFinal
}

// / Block insertion operation.
// /
// / Keeps hold if the inserted block state and data.
type BlockImportOperation[
	N runtime.Number,
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	/// Returns pending state.
	///
	/// Returns nil for backends with locally-unavailable state data.
	State() (statemachine.Backend[H, Hasher], error)

	/// Append block data to the transaction.
	SetBlockData(
		header Header,
		body []E,
		indexedBody [][]byte,
		justifications runtime.Justifications,
		state NewBlockState,
	) error

	/// Inject storage data into the database.
	UpdateDBStorage(update statemachine.BackendTransaction[H, Hasher]) error

	/// Set genesis state. If commit is false the state is saved in memory, but is not written
	/// to the database.
	SetGenesisState(storage storage.Storage, commit bool, stateVersion storage.StateVersion) (H, error)

	/// Inject storage data into the database replacing any existing data.
	ResetStorage(storage storage.Storage, stateVersion storage.StateVersion) (H, error)

	/// Set storage changes.
	UpdateStorage(update statemachine.StorageCollection, childUpdate statemachine.ChildStorageCollection) error

	/// Write offchain storage changes to the database.
	UpdateOffchainStorage(offchainUpdate statemachine.OffchainChangesCollection) error

	/// Insert auxiliary keys.
	///
	/// Values that are nil respresent the keys should be deleted.
	InsertAux(ops AuxDataOperations) error

	/// Mark a block as finalized.
	MarkFinalized(hash H, justification *runtime.Justification) error

	/// Mark a block as new head. If both block import and set head are specified, set head
	/// overrides block import's best block rule.
	MarkHead(hash H) error

	/// Add a transaction index operation.
	UpdateTransactionIndex(index []statemachine.IndexOperation) error
}

type KeyValue struct {
	Key   []byte
	Value []byte
}

// / Provides access to an auxiliary database.
// /
// / This is a simple global database not aware of forks. Can be used for storing auxiliary
// / information like total block weight/difficulty for fork resolution purposes as a common use
// / case.
type AuxStore interface {
	/// Insert auxiliary data into key-value store.
	///
	/// Deletions occur after insertions.
	InsertAux(insert []KeyValue, delete [][]byte) error

	/// Query auxiliary data from key-value store.
	GetAux(key []byte) ([]byte, error)
}

// / Client backend.
// /
// / Manages the data layer.
// /
// / # State Pruning
// /
// / While an object from StateAt is alive, the state
// / should not be pruned. The backend should internally reference-count
// / its state objects.
// /
// / The same applies for live BlockImportOperation instances: while an import operation building on a
// / parent P is alive, the state for P should not be pruned.
// /
// / # Block Pruning
// /
// / Users can pin blocks in memory by calling PinBlock. When
// / a block would be pruned, its value is kept in an in-memory cache
// / until it is unpinned via UnpinBlock.
// /
// / While a block is pinned, its state is also preserved.
// /
// / The backend should internally reference count the number of pin / unpin calls.
type Backend[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	/// Insert auxiliary data into key-value store.
	AuxStore

	/// Begin a new block insertion transaction with given parent block id.
	///
	/// When constructing the genesis, this is called with all-zero hash.
	BeginOperation() (BlockImportOperation[N, H, Hasher, Header, E], error)

	/// Note an operation to contain state transition.
	BeginStateOperation(operation BlockImportOperation[N, H, Hasher, Header, E], block H) error

	/// Commit block insertion.
	CommitOperation(transaction BlockImportOperation[N, H, Hasher, Header, E]) error

	/// Finalize block with given `hash`.
	///
	/// This should only be called if the parent of the given block has been finalized.
	FinalizeBlock(hash H, justification *runtime.Justification) error

	/// Append justification to the block with the given hash.
	///
	/// This should only be called for blocks that are already finalized.
	AppendJustification(hash H, justification runtime.Justification) error

	/// Returns reference to blockchain backend.
	Blockchain() blockchain.Backend[H, N, Header]

	/// Returns a pointer to offchain storage.
	OffchainStorage() offchain.OffchainStorage

	/// Pin the block to keep body, justification and state available after pruning.
	/// Number of pins are reference counted. Users need to make sure to perform
	/// one call to UnpinBlock per call to PinBlock.
	PinBlock(hash H) error

	/// Unpin the block to allow pruning.
	UnpinBlock(hash H)

	/// Returns true if state for given block is available.
	HaveStateAt(hash H, number N) bool

	/// Returns state backend with post-state of given block.
	StateAt(hash H) (statemachine.Backend[H, Hasher], error)

	/// Attempts to revert the chain by n blocks. If revertFinalized is set it will attempt to
	/// revert past any finalized block. This is unsafe and can potentially leave the node in an
	/// inconsistent state. All blocks higher than the best block are also reverted and not counting
	/// towards n.
	///
	/// Returns the number of blocks that were successfully reverted and the list of finalized
	/// blocks that has been reverted.
	Revert(n N, revertFinalized bool) (N, map[H]any, error)

	/// Discard non-best, unfinalized leaf block.
	RemoveLeafBlock(hash H) error

	/// Gain access to the import lock around this backend.
	///
	/// NOTE: Backend isn't expected to acquire the lock by itself ever. Rather
	/// the using components should acquire and hold the lock whenever they do
	/// something that the import of a block would interfere with, e.g. importing
	/// a new block or calculating the best head.
	GetImportLock() *sync.RWMutex

	/// Tells whether the backend requires full-sync mode.
	RequiresFullSync() bool

	/// Returns current usage statistics.
	// TODO: implement UsageInfo if we require it
	// UsageInfo() *UsageInfo
}
