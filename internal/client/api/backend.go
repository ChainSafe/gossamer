package api

import (
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	overlayedchanges "github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayed-changes"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// / Describes which block import notification stream should be notified.
type ImportNotificationAction uint

const (
	/// Notify only when the node has synced to the tip or there is a re-org.
	ImportNotificationActionRecentBlock ImportNotificationAction = iota
	/// Notify for every single block no matter what the sync state is.
	EveryBlock
	/// Both block import notifications above should be fired.
	Both
	/// No block import notification should be fired.
	None
)

// / Import operation summary.
// /
// / Contains information about the block that just got imported,
// / including storage changes, reorged blocks, etc.
type ImportSummary[N runtime.Number, H statemachine.HasherOut] struct {
	/// Block hash of the imported block.
	// pub hash: Block::Hash,
	Hash H
	/// Import origin.
	// pub origin: BlockOrigin,
	Origin consensus.BlockOrigin
	/// Header of the imported block.
	// pub header: Block::Header,
	Header runtime.Header[N, H]
	/// Is this block a new best block.
	// pub is_new_best: bool,
	IsNewBest bool
	//		/// Optional storage changes.
	//		pub storage_changes: Option<(StorageCollection, ChildStorageCollection)>,
	StorageChanges *struct {
		statemachine.StorageCollection
		statemachine.ChildStorageCollection
	}
	/// Tree route from old best to new best.
	///
	/// If `None`, there was no re-org while importing.
	// pub tree_route: Option<sp_blockchain::TreeRoute<Block>>,
	TreeRoute blockchain.TreeRoute[H, N]
	/// What notify action to take for this import.
	// pub import_notification_action: ImportNotificationAction,
	ImportNotificationAction ImportNotificationAction
}

// / Finalization operation summary.
// /
// / Contains information about the block that just got finalized,
// / including tree heads that became stale at the moment of finalization.
type FinalizeSummary[N runtime.Number, H statemachine.HasherOut] struct {
	/// Last finalized block header.
	// pub header: Block::Header,
	Header runtime.Header[N, H]
	/// Blocks that were finalized.
	/// The last entry is the one that has been explicitly finalized.
	// pub finalized: Vec<Block::Hash>,
	Finalized []H
	/// Heads that became stale during this finalization operation.
	// pub stale_heads: Vec<Block::Hash>,
	StaleHeads []H
}

// / Import operation wrapper.
type ClientImportOperation[N runtime.Number, H statemachine.HasherOut] struct {
	/// DB Operation.
	// pub op: B::BlockImportOperation,
	Op BlockImportOperation[N, H]
	/// Summary of imported block.
	// pub notify_imported: Option<ImportSummary<Block>>,
	NotifyImported *ImportSummary[N, H]
	/// Summary of finalized block.
	// pub notify_finalized: Option<FinalizeSummary<Block>>,
	NotifiyFinalized *FinalizeSummary[N, H]
}

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

// / Block insertion operation.
// /
// / Keeps hold if the inserted block state and data.
type BlockImportOperation[N runtime.Number, H statemachine.HasherOut] interface {
	/// Returns pending state.
	///
	/// Returns None for backends with locally-unavailable state data.
	// 	fn state(&self) -> sp_blockchain::Result<Option<&Self::State>>;
	State() *statemachine.Backend[H]

	/// Append block data to the transaction.
	// 	fn set_block_data(
	// 		&mut self,
	// 		header: Block::Header,
	// 		body: Option<Vec<Block::Extrinsic>>,
	// 		indexed_body: Option<Vec<Vec<u8>>>,
	// 		justifications: Option<Justifications>,
	// 		state: NewBlockState,
	// 	) -> sp_blockchain::Result<()>;
	SetBlockData(header runtime.Header[N, H], body runtime.Extrinsic, indexedBody *[][]byte, justifications *runtime.Justifications, state NewBlockState) error

	/// Inject storage data into the database.
	// 	fn update_db_storage(
	// 		&mut self,
	// 		update: TransactionForSB<Self::State, Block>,
	// 	) -> sp_blockchain::Result<()>;
	UpdateDBStorage(update statemachine.Transaction) error

	/// Set genesis state. If `commit` is `false` the state is saved in memory, but is not written
	/// to the database.
	// 	fn set_genesis_state(
	// 		&mut self,
	// 		storage: Storage,
	// 		commit: bool,
	// 		state_version: StateVersion,
	// 	) -> sp_blockchain::Result<Block::Hash>;
	SetGenesisState(storage storage.Storage, commit bool, stateVersion storage.StateVersion) (H, error)

	/// Inject storage data into the database replacing any existing data.
	// 	fn reset_storage(
	// 		&mut self,
	// 		storage: Storage,
	// 		state_version: StateVersion,
	// 	) -> sp_blockchain::Result<Block::Hash>;
	ResetStorage(storage storage.Storage, stateVersion storage.StateVersion) (H, error)

	/// Set storage changes.
	// 	fn update_storage(
	// 		&mut self,
	// 		update: StorageCollection,
	// 		child_update: ChildStorageCollection,
	// 	) -> sp_blockchain::Result<()>;
	UpdateStorage(update statemachine.StorageCollection, childUpdate statemachine.ChildStorageCollection) error

	/// Write offchain storage changes to the database.
	// 	fn update_offchain_storage(
	// 		&mut self,
	// 		_offchain_update: OffchainChangesCollection,
	// 	) -> sp_blockchain::Result<()> {
	// 		Ok(())
	// 	}
	UpdateOffchainStorage(offchainUpdate statemachine.OffchainChangesCollection) error

	/// Insert auxiliary keys.
	///
	/// Values are `None` if should be deleted.
	// 	fn insert_aux<I>(&mut self, ops: I) -> sp_blockchain::Result<()>
	// 	where
	// 		I: IntoIterator<Item = (Vec<u8>, Option<Vec<u8>>)>;
	InsertAux(ops []struct {
		Key   []byte
		Value *[]byte
	}) error

	/// Mark a block as finalized.
	// 	fn mark_finalized(
	// 		&mut self,
	// 		hash: Block::Hash,
	// 		justification: Option<Justification>,
	// 	) -> sp_blockchain::Result<()>;
	MarkFinalized(hash H, justification *runtime.Justification) error

	/// Mark a block as new head. If both block import and set head are specified, set head
	/// overrides block import's best block rule.
	// 	fn mark_head(&mut self, hash: Block::Hash) -> sp_blockchain::Result<()>;
	MarkHead(hash H) error

	/// Add a transaction index operation.
	//		fn update_transaction_index(&mut self, index: Vec<IndexOperation>)
	//			-> sp_blockchain::Result<()>;
	//	}
	UpdateTransactionIndex(index []overlayedchanges.IndexOperation) error
}

// / Interface for performing operations on the backend.
//
//	pub trait LockImportRun<Block: BlockT, B: Backend<Block>> {
type LockImportRun[R any, N runtime.Number, H statemachine.HasherOut] interface {
	/// Lock the import lock, and run operations inside.
	// fn lock_import_and_run<R, Err, F>(&self, f: F) -> Result<R, Err>
	// where
	// 	F: FnOnce(&mut ClientImportOperation<Block, B>) -> Result<R, Err>,
	// 	Err: From<sp_blockchain::Error>;
	LockImportAndRun(func(ClientImportOperation[N, H]) (R, error)) (R, error)
}

// / Finalize Facilities
// pub trait Finalizer<Block: BlockT, B: Backend<Block>> {
type Finalizer[N runtime.Number, H statemachine.HasherOut] interface {
	/// Mark all blocks up to given as finalized in operation.
	///
	/// If `justification` is provided it is stored with the given finalized
	/// block (any other finalized blocks are left unjustified).
	///
	/// If the block being finalized is on a different fork from the current
	/// best block the finalized block is set as best, this might be slightly
	/// inaccurate (i.e. outdated). Usages that require determining an accurate
	/// best block should use `SelectChain` instead of the client.
	// fn apply_finality(
	// 	&self,
	// 	operation: &mut ClientImportOperation<Block, B>,
	// 	block: Block::Hash,
	// 	justification: Option<Justification>,
	// 	notify: bool,
	// ) -> sp_blockchain::Result<()>;
	ApplyFinality(
		operation ClientImportOperation[N, H],
		block H,
		justifcation *runtime.Justification,
		notify bool) error

	/// Finalize a block.
	///
	/// This will implicitly finalize all blocks up to it and
	/// fire finality notifications.
	///
	/// If the block being finalized is on a different fork from the current
	/// best block, the finalized block is set as best. This might be slightly
	/// inaccurate (i.e. outdated). Usages that require determining an accurate
	/// best block should use `SelectChain` instead of the client.
	///
	/// Pass a flag to indicate whether finality notifications should be propagated.
	/// This is usually tied to some synchronization state, where we don't send notifications
	/// while performing major synchronization work.
	// fn finalize_block(
	// 	&self,
	// 	block: Block::Hash,
	// 	justification: Option<Justification>,
	// 	notify: bool,
	// ) -> sp_blockchain::Result<()>;
	FinalizeBlock(block H, justification *runtime.Justification, notify bool) error
}

// / Provides access to an auxiliary database.
// /
// / This is a simple global database not aware of forks. Can be used for storing auxiliary
// / information like total block weight/difficulty for fork resolution purposes as a common use
// / case.
// pub trait AuxStore {
type AuxStore interface {
	/// Insert auxiliary data into key-value store.
	///
	/// Deletions occur after insertions.
	// fn insert_aux<
	// 	'a,
	// 	'b: 'a,
	// 	'c: 'a,
	// 	I: IntoIterator<Item = &'a (&'c [u8], &'c [u8])>,
	// 	D: IntoIterator<Item = &'a &'b [u8]>,
	// >(
	// 	&self,
	// 	insert: I,
	// 	delete: D,
	// ) -> sp_blockchain::Result<()>;
	InsertAux(insert []struct {
		Key   []byte
		Value []byte
	}, delete [][]byte) error

	/// Query auxiliary data from key-value store.
	// fn get_aux(&self, key: &[u8]) -> sp_blockchain::Result<Option<Vec<u8>>>;
	GetAux(key []byte) (*[]byte, error)
}

// / An `Iterator` that iterates keys in a given block under a prefix.
// pub struct KeysIter<State, Block>
// where
//
//	State: StateBackend<HashFor<Block>>,
//	Block: BlockT,
//
//	{
//		inner: <State as StateBackend<HashFor<Block>>>::RawIter,
//		state: State,
//	}
type KeysIter[N runtime.Number, H statemachine.HasherOut] struct {
	inner statemachine.StorageIterator[H]
	state statemachine.Backend[H]
}

// / An `Iterator` that iterates keys and values in a given block under a prefix.
// pub struct PairsIter<State, Block>
// where
//
//	State: StateBackend<HashFor<Block>>,
//	Block: BlockT,
//
//	{
//		inner: <State as StateBackend<HashFor<Block>>>::RawIter,
//		state: State,
//	}
type PairsIter[N runtime.Number, H statemachine.HasherOut] struct {
	inner statemachine.StorageIterator[H]
	state statemachine.Backend[H]
}

// / Provides access to storage primitives
// pub trait StorageProvider<Block: BlockT, B: Backend<Block>> {
type StorageProvider[H runtime.Hash, N runtime.Number] interface {
	/// Given a block's `Hash` and a key, return the value under the key in that block.
	// fn storage(
	// 	&self,
	// 	hash: Block::Hash,
	// 	key: &StorageKey,
	// ) -> sp_blockchain::Result<Option<StorageData>>;
	Storage(hash H, key storage.StorageKey) (*storage.StorageData, error)

	/// Given a block's `Hash` and a key, return the value under the hash in that block.
	// fn storage_hash(
	// 	&self,
	// 	hash: Block::Hash,
	// 	key: &StorageKey,
	// ) -> sp_blockchain::Result<Option<Block::Hash>>;
	StorageHash(hash H, key storage.StorageKey) (*H, error)

	/// Given a block's `Hash` and a key prefix, returns a `KeysIter` iterates matching storage
	/// keys in that block.
	// fn storage_keys(
	// 	&self,
	// 	hash: Block::Hash,
	// 	prefix: Option<&StorageKey>,
	// 	start_key: Option<&StorageKey>,
	// ) -> sp_blockchain::Result<KeysIter<B::State, Block>>;
	StorageKeys(hash H, prefix *storage.StorageKey, startKey *storage.StorageKey) (KeysIter[N, H], error)

	/// Given a block's `Hash` and a key prefix, returns an iterator over the storage keys and
	/// values in that block.
	// fn storage_pairs(
	// 	&self,
	// 	hash: <Block as BlockT>::Hash,
	// 	prefix: Option<&StorageKey>,
	// 	start_key: Option<&StorageKey>,
	// ) -> sp_blockchain::Result<PairsIter<B::State, Block>>;
	StoragePairs(hash H, prefix *storage.StorageKey, startKey *storage.StorageKey) (PairsIter[N, H], error)

	/// Given a block's `Hash`, a key and a child storage key, return the value under the key in
	/// that block.
	// fn child_storage(
	// 	&self,
	// 	hash: Block::Hash,
	// 	child_info: &ChildInfo,
	// 	key: &StorageKey,
	// ) -> sp_blockchain::Result<Option<StorageData>>;
	ChildStorage(hash H, childInfo storage.ChildInfo, key storage.StorageKey) (*storage.StorageData, error)

	// /// Given a block's `Hash` and a key `prefix` and a child storage key,
	// /// returns a `KeysIter` that iterates matching storage keys in that block.
	// fn child_storage_keys(
	// 	&self,
	// 	hash: Block::Hash,
	// 	child_info: ChildInfo,
	// 	prefix: Option<&StorageKey>,
	// 	start_key: Option<&StorageKey>,
	// ) -> sp_blockchain::Result<KeysIter<B::State, Block>>;
	ChildStorageKeys(hash H, childInfo storage.ChildInfo, prefix *storage.StorageKey, startKey *storage.StorageKey) (KeysIter[N, H], error)

	// /// Given a block's `Hash`, a key and a child storage key, return the hash under the key in that
	// /// block.
	// fn child_storage_hash(
	// 	&self,
	// 	hash: Block::Hash,
	// 	child_info: &ChildInfo,
	// 	key: &StorageKey,
	// ) -> sp_blockchain::Result<Option<Block::Hash>>;
	ChildStorageHash(hash H, childInfo storage.ChildInfo, key storage.StorageKey) (*H, error)
}
