package blockchain

import (
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

// / Blockchain database header backend. Does not perform any validation.
// pub trait HeaderBackend<Block: BlockT>: Send + Sync {
type HeaderBackend[Hash runtime.Hash, N runtime.Number] interface {
	/// Get block header. Returns `None` if block is not found.
	// fn header(&self, hash: Block::Hash) -> Result<Option<Block::Header>>;
	Header(hash Hash) (*runtime.Header[N, Hash], error)

	/// Get blockchain info.
	// fn info(&self) -> Info<Block>;
	Info() Info[Hash, N]

	/// Get block status.
	// fn status(&self, hash: Block::Hash) -> Result<BlockStatus>;
	Status(hash Hash) (BlockStatus, error)

	/// Get block number by hash. Returns `None` if the header is not in the chain.
	// fn number(
	// 	&self,
	// 	hash: Block::Hash,
	// ) -> Result<Option<<<Block as BlockT>::Header as HeaderT>::Number>>;
	Number(hash Hash) (*N, error)

	/// Get block hash by number. Returns `None` if the header is not in the chain.
	// fn hash(&self, number: NumberFor<Block>) -> Result<Option<Block::Hash>>;
	Hash(number N) (*Hash, error)

	/// Convert an arbitrary block ID into a block hash.
	// fn block_hash_from_id(&self, id: &BlockId<Block>) -> Result<Option<Block::Hash>> {
	// 	match *id {
	// 		BlockId::Hash(h) => Ok(Some(h)),
	// 		BlockId::Number(n) => self.hash(n),
	// 	}
	// }
	BlockHashFromID(id generic.BlockID) (*Hash, error)

	/// Convert an arbitrary block ID into a block hash.
	// fn block_number_from_id(&self, id: &BlockId<Block>) -> Result<Option<NumberFor<Block>>> {
	// 	match *id {
	// 		BlockId::Hash(h) => self.number(h),
	// 		BlockId::Number(n) => Ok(Some(n)),
	// 	}
	// }
	BlockNumberFromID(id generic.BlockID) (*N, error)

	/// Get block header. Returns `UnknownBlock` error if block is not found.
	// fn expect_header(&self, hash: Block::Hash) -> Result<Block::Header> {
	// 	self.header(hash)?
	// 		.ok_or_else(|| Error::UnknownBlock(format!("Expect header: {}", hash)))
	// }
	ExpectHeader(hash Hash) (runtime.Header[N, Hash], error)

	/// Convert an arbitrary block ID into a block number. Returns `UnknownBlock` error if block is
	/// not found.
	// fn expect_block_number_from_id(&self, id: &BlockId<Block>) -> Result<NumberFor<Block>> {
	// 	self.block_number_from_id(id).and_then(|n| {
	// 		n.ok_or_else(|| Error::UnknownBlock(format!("Expect block number from id: {}", id)))
	// 	})
	// }
	ExpectBlockNumberFromID(id generic.BlockID) (N, error)

	/// Convert an arbitrary block ID into a block hash. Returns `UnknownBlock` error if block is
	/// not found.
	// fn expect_block_hash_from_id(&self, id: &BlockId<Block>) -> Result<Block::Hash> {
	// 	self.block_hash_from_id(id).and_then(|h| {
	// 		h.ok_or_else(|| Error::UnknownBlock(format!("Expect block hash from id: {}", id)))
	// 	})
	// }
	ExpectBlockHashFromID(id generic.BlockID) (Hash, error)
}

// / Blockchain database backend. Does not perform any validation.
// pub trait Backend<Block: BlockT>:
type Backend[Hash runtime.Hash, N runtime.Number] interface {
	//	HeaderBackend<Block> + HeaderMetadata<Block, Error = Error>
	HeaderBackend[Hash, N]
	HeaderMetaData[Hash, N]

	// /// Get block body. Returns `None` if block is not found.
	// fn body(&self, hash: Block::Hash) -> Result<Option<Vec<<Block as BlockT>::Extrinsic>>>;
	// /// Get block justifications. Returns `None` if no justification exists.
	// fn justifications(&self, hash: Block::Hash) -> Result<Option<Justifications>>;
	Justifications(hash Hash) (*runtime.Justifications, error)
	// /// Get last finalized block hash.
	// fn last_finalized(&self) -> Result<Block::Hash>;

	// /// Returns hashes of all blocks that are leaves of the block tree.
	// /// in other words, that have no children, are chain heads.
	// /// Results must be ordered best (longest, highest) chain first.
	// fn leaves(&self) -> Result<Vec<Block::Hash>>;

	// /// Returns displaced leaves after the given block would be finalized.
	// ///
	// /// The returned leaves do not contain the leaves from the same height as `block_number`.
	// fn displaced_leaves_after_finalizing(
	// 	&self,
	// 	block_number: NumberFor<Block>,
	// ) -> Result<Vec<Block::Hash>>;

	// /// Return hashes of all blocks that are children of the block with `parent_hash`.
	// fn children(&self, parent_hash: Block::Hash) -> Result<Vec<Block::Hash>>;

	// /// Get the most recent block hash of the longest chain that contains
	// /// a block with the given `base_hash`.
	// ///
	// /// The search space is always limited to blocks which are in the finalized
	// /// chain or descendents of it.
	// ///
	// /// Returns `Ok(None)` if `base_hash` is not found in search space.
	// // TODO: document time complexity of this, see [#1444](https://github.com/paritytech/substrate/issues/1444)
	// fn longest_containing(
	// 	&self,
	// 	base_hash: Block::Hash,
	// 	import_lock: &RwLock<()>,
	// ) -> Result<Option<Block::Hash>> {
	// 	let Some(base_header) = self.header(base_hash)? else {
	// 		return Ok(None)
	// 	};

	// 	let leaves = {
	// 		// ensure no blocks are imported during this code block.
	// 		// an import could trigger a reorg which could change the canonical chain.
	// 		// we depend on the canonical chain staying the same during this code block.
	// 		let _import_guard = import_lock.read();
	// 		let info = self.info();
	// 		if info.finalized_number > *base_header.number() {
	// 			// `base_header` is on a dead fork.
	// 			return Ok(None)
	// 		}
	// 		self.leaves()?
	// 	};

	// 	// for each chain. longest chain first. shortest last
	// 	for leaf_hash in leaves {
	// 		let mut current_hash = leaf_hash;
	// 		// go backwards through the chain (via parent links)
	// 		loop {
	// 			if current_hash == base_hash {
	// 				return Ok(Some(leaf_hash))
	// 			}

	// 			let current_header = self
	// 				.header(current_hash)?
	// 				.ok_or_else(|| Error::MissingHeader(current_hash.to_string()))?;

	// 			// stop search in this chain once we go below the target's block number
	// 			if current_header.number() < base_header.number() {
	// 				break
	// 			}

	// 			current_hash = *current_header.parent_hash();
	// 		}
	// 	}

	// 	// header may be on a dead fork -- the only leaves that are considered are
	// 	// those which can still be finalized.
	// 	//
	// 	// FIXME #1558 only issue this warning when not on a dead fork
	// 	warn!(
	// 		"Block {:?} exists in chain but not found when following all leaves backwards",
	// 		base_hash,
	// 	);

	// 	Ok(None)
	// }

	// /// Get single indexed transaction by content hash. Note that this will only fetch transactions
	// /// that are indexed by the runtime with `storage_index_transaction`.
	// fn indexed_transaction(&self, hash: Block::Hash) -> Result<Option<Vec<u8>>>;

	// /// Check if indexed transaction exists.
	// fn has_indexed_transaction(&self, hash: Block::Hash) -> Result<bool> {
	// 	Ok(self.indexed_transaction(hash)?.is_some())
	// }

	// fn block_indexed_body(&self, hash: Block::Hash) -> Result<Option<Vec<Vec<u8>>>>;
}

// / Blockchain info
// pub struct Info<Block: BlockT> {
type Info[H, N any] struct {
	/// Best block hash.
	// pub best_hash: Block::Hash,
	BestHash H
	/// Best block number.
	// pub best_number: <<Block as BlockT>::Header as HeaderT>::Number,
	BestNumber N
	/// Genesis block hash.
	// pub genesis_hash: Block::Hash,
	GenesisHash H
	/// The head of the finalized chain.
	// pub finalized_hash: Block::Hash,
	FinalizedHash H
	/// Last finalized block number.
	// pub finalized_number: <<Block as BlockT>::Header as HeaderT>::Number,
	FinalizedNumber N
	/// Last finalized state.
	// pub finalized_state: Option<(Block::Hash, <<Block as BlockT>::Header as HeaderT>::Number)>,
	FinalizedState *struct {
		Hash   H
		Number N
	}
	/// Number of concurrent leave forks.
	// pub number_leaves: usize,
	NumberLeaves uint
	/// Missing blocks after warp sync. (start, end).
	// pub block_gap: Option<(NumberFor<Block>, NumberFor<Block>)>,
	BlockGap *[2]N
}

// / Block status.
type BlockStatus uint

const (
	/// Already in the blockchain.
	BlockStatusInChain BlockStatus = iota
	/// Not in the queue or the blockchain.
	BlockStatusUnknown
)
