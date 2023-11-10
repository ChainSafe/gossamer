package api

import (
	"github.com/ChainSafe/gossamer/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/primitives/state-machine"
	"github.com/ChainSafe/gossamer/primitives/storage"
)

// / Import operation wrapper.
//
//	pub struct ClientImportOperation<Block: BlockT, B: Backend<Block>> {
//		/// DB Operation.
//		pub op: B::BlockImportOperation,
//		/// Summary of imported block.
//		pub notify_imported: Option<ImportSummary<Block>>,
//		/// Summary of finalized block.
//		pub notify_finalized: Option<FinalizeSummary<Block>>,
//	}
type ClientImportOperation[Block, Backend any] struct {
	// pub Op
	Op BlockImportOperation
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

	/// Set storage changes.
	// 	fn update_storage(
	// 		&mut self,
	// 		update: StorageCollection,
	// 		child_update: ChildStorageCollection,
	// 	) -> sp_blockchain::Result<()>;

	/// Write offchain storage changes to the database.
	// 	fn update_offchain_storage(
	// 		&mut self,
	// 		_offchain_update: OffchainChangesCollection,
	// 	) -> sp_blockchain::Result<()> {
	// 		Ok(())
	// 	}

	/// Insert auxiliary keys.
	///
	/// Values are `None` if should be deleted.
	// 	fn insert_aux<I>(&mut self, ops: I) -> sp_blockchain::Result<()>
	// 	where
	// 		I: IntoIterator<Item = (Vec<u8>, Option<Vec<u8>>)>;

	/// Mark a block as finalized.
	// 	fn mark_finalized(
	// 		&mut self,
	// 		hash: Block::Hash,
	// 		justification: Option<Justification>,
	// 	) -> sp_blockchain::Result<()>;

	/// Mark a block as new head. If both block import and set head are specified, set head
	/// overrides block import's best block rule.
	// 	fn mark_head(&mut self, hash: Block::Hash) -> sp_blockchain::Result<()>;

	/// Add a transaction index operation.
	//		fn update_transaction_index(&mut self, index: Vec<IndexOperation>)
	//			-> sp_blockchain::Result<()>;
	//	}
}

// / Interface for performing operations on the backend.
//
//	pub trait LockImportRun<Block: BlockT, B: Backend<Block>> {
//		/// Lock the import lock, and run operations inside.
//		fn lock_import_and_run<R, Err, F>(&self, f: F) -> Result<R, Err>
//		where
//			F: FnOnce(&mut ClientImportOperation<Block, B>) -> Result<R, Err>,
//			Err: From<sp_blockchain::Error>;
//	}
type LockImportRun[R any] interface {
	LockImportAndRun(func(ClientImportOperation) (R, error)) func() (R, error)
}

// / Client backend.
// /
// / Manages the data layer.
// /
// / # State Pruning
// /
// / While an object from `state_at` is alive, the state
// / should not be pruned. The backend should internally reference-count
// / its state objects.
// /
// / The same applies for live `BlockImportOperation`s: while an import operation building on a
// / parent `P` is alive, the state for `P` should not be pruned.
// /
// / # Block Pruning
// /
// / Users can pin blocks in memory by calling `pin_block`. When
// / a block would be pruned, its value is kept in an in-memory cache
// / until it is unpinned via `unpin_block`.
// /
// / While a block is pinned, its state is also preserved.
// /
// / The backend should internally reference count the number of pin / unpin calls.
type Backend interface {
	// 	/// Associated block insertion operation type.
	// 	type BlockImportOperation: BlockImportOperation<Block, State = Self::State>;
}

// pub trait Backend<Block: BlockT>: AuxStore + Send + Sync {
// 	/// Associated block insertion operation type.
// 	type BlockImportOperation: BlockImportOperation<Block, State = Self::State>;
// 	/// Associated blockchain backend type.
// 	type Blockchain: BlockchainBackend<Block>;
// 	/// Associated state backend type.
// 	type State: StateBackend<HashFor<Block>>
// 		+ Send
// 		+ AsTrieBackend<
// 			HashFor<Block>,
// 			TrieBackendStorage = <Self::State as StateBackend<HashFor<Block>>>::TrieBackendStorage,
// 		>;
// 	/// Offchain workers local storage.
// 	type OffchainStorage: OffchainStorage;

// 	/// Begin a new block insertion transaction with given parent block id.
// 	///
// 	/// When constructing the genesis, this is called with all-zero hash.
// 	fn begin_operation(&self) -> sp_blockchain::Result<Self::BlockImportOperation>;

// 	/// Note an operation to contain state transition.
// 	fn begin_state_operation(
// 		&self,
// 		operation: &mut Self::BlockImportOperation,
// 		block: Block::Hash,
// 	) -> sp_blockchain::Result<()>;

// 	/// Commit block insertion.
// 	fn commit_operation(
// 		&self,
// 		transaction: Self::BlockImportOperation,
// 	) -> sp_blockchain::Result<()>;

// 	/// Finalize block with given `hash`.
// 	///
// 	/// This should only be called if the parent of the given block has been finalized.
// 	fn finalize_block(
// 		&self,
// 		hash: Block::Hash,
// 		justification: Option<Justification>,
// 	) -> sp_blockchain::Result<()>;

// 	/// Append justification to the block with the given `hash`.
// 	///
// 	/// This should only be called for blocks that are already finalized.
// 	fn append_justification(
// 		&self,
// 		hash: Block::Hash,
// 		justification: Justification,
// 	) -> sp_blockchain::Result<()>;

// 	/// Returns reference to blockchain backend.
// 	fn blockchain(&self) -> &Self::Blockchain;

// 	/// Returns current usage statistics.
// 	fn usage_info(&self) -> Option<UsageInfo>;

// 	/// Returns a handle to offchain storage.
// 	fn offchain_storage(&self) -> Option<Self::OffchainStorage>;

// 	/// Pin the block to keep body, justification and state available after pruning.
// 	/// Number of pins are reference counted. Users need to make sure to perform
// 	/// one call to [`Self::unpin_block`] per call to [`Self::pin_block`].
// 	fn pin_block(&self, hash: Block::Hash) -> sp_blockchain::Result<()>;

// 	/// Unpin the block to allow pruning.
// 	fn unpin_block(&self, hash: Block::Hash);

// 	/// Returns true if state for given block is available.
// 	fn have_state_at(&self, hash: Block::Hash, _number: NumberFor<Block>) -> bool {
// 		self.state_at(hash).is_ok()
// 	}

// 	/// Returns state backend with post-state of given block.
// 	fn state_at(&self, hash: Block::Hash) -> sp_blockchain::Result<Self::State>;

// 	/// Attempts to revert the chain by `n` blocks. If `revert_finalized` is set it will attempt to
// 	/// revert past any finalized block, this is unsafe and can potentially leave the node in an
// 	/// inconsistent state. All blocks higher than the best block are also reverted and not counting
// 	/// towards `n`.
// 	///
// 	/// Returns the number of blocks that were successfully reverted and the list of finalized
// 	/// blocks that has been reverted.
// 	fn revert(
// 		&self,
// 		n: NumberFor<Block>,
// 		revert_finalized: bool,
// 	) -> sp_blockchain::Result<(NumberFor<Block>, HashSet<Block::Hash>)>;

// 	/// Discard non-best, unfinalized leaf block.
// 	fn remove_leaf_block(&self, hash: Block::Hash) -> sp_blockchain::Result<()>;

// 	/// Insert auxiliary data into key-value store.
// 	fn insert_aux<
// 		'a,
// 		'b: 'a,
// 		'c: 'a,
// 		I: IntoIterator<Item = &'a (&'c [u8], &'c [u8])>,
// 		D: IntoIterator<Item = &'a &'b [u8]>,
// 	>(
// 		&self,
// 		insert: I,
// 		delete: D,
// 	) -> sp_blockchain::Result<()> {
// 		AuxStore::insert_aux(self, insert, delete)
// 	}
// 	/// Query auxiliary data from key-value store.
// 	fn get_aux(&self, key: &[u8]) -> sp_blockchain::Result<Option<Vec<u8>>> {
// 		AuxStore::get_aux(self, key)
// 	}

// 	/// Gain access to the import lock around this backend.
// 	///
// 	/// _Note_ Backend isn't expected to acquire the lock by itself ever. Rather
// 	/// the using components should acquire and hold the lock whenever they do
// 	/// something that the import of a block would interfere with, e.g. importing
// 	/// a new block or calculating the best head.
// 	fn get_import_lock(&self) -> &RwLock<()>;

// 	/// Tells whether the backend requires full-sync mode.
// 	fn requires_full_sync(&self) -> bool;
// }
