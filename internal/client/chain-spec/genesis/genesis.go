package genesis

import "github.com/ChainSafe/gossamer/internal/primitives/runtime"

// / Trait for building the genesis block.
// pub trait BuildGenesisBlock<Block: BlockT> {
type BuildGenesisBlock[H runtime.Hash, N runtime.Number, BlockImportOperation any] interface {
	// /// The import operation used to import the genesis block into the backend.
	// type BlockImportOperation;

	// /// Returns the built genesis block along with the block import operation
	// /// after setting the genesis storage.
	// fn build_genesis_block(self) -> sp_blockchain::Result<(Block, Self::BlockImportOperation)>;
	BuildGenesisBlock() (runtime.Block[N, H], BlockImportOperation, error)
}
