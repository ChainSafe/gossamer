package blockchain

import (
	"github.com/ChainSafe/gossamer/primitives/runtime"
	"github.com/ChainSafe/gossamer/primitives/runtime/generic"
)

// / Blockchain database header backend. Does not perform any validation.
// pub trait HeaderBackend<Block: BlockT>: Send + Sync {
type HeaderBackend[H runtime.Hash, N runtime.Number] interface {
	/// Get block header. Returns `None` if block is not found.
	// fn header(&self, hash: Block::Hash) -> Result<Option<Block::Header>>;
	Header(hash H) *runtime.Header[N, H]

	/// Get blockchain info.
	// fn info(&self) -> Info<Block>;
	Info() Info[H, N]

	/// Get block status.
	// fn status(&self, hash: Block::Hash) -> Result<BlockStatus>;
	Status(hash H) (BlockStatus, error)

	/// Get block number by hash. Returns `None` if the header is not in the chain.
	// fn number(
	// 	&self,
	// 	hash: Block::Hash,
	// ) -> Result<Option<<<Block as BlockT>::Header as HeaderT>::Number>>;
	Number(hash H) (N, error)

	/// Get block hash by number. Returns `None` if the header is not in the chain.
	// fn hash(&self, number: NumberFor<Block>) -> Result<Option<Block::Hash>>;
	Hash(number N) (H, error)

	/// Convert an arbitrary block ID into a block hash.
	// fn block_hash_from_id(&self, id: &BlockId<Block>) -> Result<Option<Block::Hash>> {
	// 	match *id {
	// 		BlockId::Hash(h) => Ok(Some(h)),
	// 		BlockId::Number(n) => self.hash(n),
	// 	}
	// }
	BlockHashFromID(id generic.BlockID) (*H, error)

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
	ExpectHeader(hash H) (*runtime.Header[N, H], error)

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
	ExpectBlockHashFromID(id generic.BlockID) (H, error)
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
