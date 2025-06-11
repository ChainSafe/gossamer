package common

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/api"
	chain "github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// / Implement Longest Chain Select implementation
// / where 'longest' is defined as the highest number of blocks
//
//	pub struct LongestChain<B, Block> {
//		backend: Arc<B>,
//		_phantom: PhantomData<Block>,
//	}
type LongestChain[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	backend api.Backend[H, N, Hasher, Header, E]
}

// impl<B, Block> Clone for LongestChain<B, Block> {
// 	fn clone(&self) -> Self {
// 		let backend = self.backend.clone();
// 		LongestChain { backend, _phantom: Default::default() }
// 	}
// }

// impl<B, Block> LongestChain<B, Block>
// where
//
//	B: backend::Backend<Block>,
//	Block: BlockT,
//
//	{
//		/// Instantiate a new LongestChain for Backend B
//		pub fn new(backend: Arc<B>) -> Self {
//			LongestChain { backend, _phantom: Default::default() }
//		}
func NewLongestChain[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	backend api.Backend[H, N, Hasher, Header, E],
) LongestChain[H, N, Hasher, Header, E] {
	return LongestChain[H, N, Hasher, Header, E]{backend: backend}
}

//	fn best_hash(&self) -> sp_blockchain::Result<<Block as BlockT>::Hash> {
//		let info = self.backend.blockchain().info();
//		let import_lock = self.backend.get_import_lock();
//		let best_hash = self
//			.backend
//			.blockchain()
//			.longest_containing(info.best_hash, import_lock)?
//			.unwrap_or(info.best_hash);
//		Ok(best_hash)
//	}
func (lc LongestChain[H, N, Hasher, Header, E]) bestHash() (H, error) {
	info := lc.backend.Blockchain().Info()

	var best H
	importLock := lc.backend.GetImportLock()
	bestHash, err := lc.backend.Blockchain().LongestContaining(info.BestHash, importLock)
	if err != nil {
		return best, err
	}
	if bestHash == nil {
		return info.BestHash, nil
	}
	return *bestHash, nil
}

//	fn best_header(&self) -> sp_blockchain::Result<<Block as BlockT>::Header> {
//		let best_hash = self.best_hash()?;
//		Ok(self
//			.backend
//			.blockchain()
//			.header(best_hash)?
//			.expect("given block hash was fetched from block in db; qed"))
//	}
func (lc LongestChain[H, N, Hasher, Header, E]) bestHeader() (Header, error) {
	var h Header
	bestHash, err := lc.bestHash()
	if err != nil {
		return h, err
	}
	header, err := lc.backend.Blockchain().Header(bestHash)
	if err != nil {
		return h, err
	}
	if header == nil {
		panic("given block hash was fetched from block in db; qed")
	}
	return *header, nil
}

// / Returns the highest descendant of the given block that is a valid
// / candidate to be finalized.
// /
// / In this context, being a valid target means being an ancestor of
// / the best chain according to the `best_header` method.
// /
// / If `maybe_max_number` is `Some(max_block_number)` the search is
// / limited to block `number <= max_block_number`. In other words
// / as if there were no blocks greater than `max_block_number`.
//
//	fn finality_target(
//		&self,
//		base_hash: Block::Hash,
//		maybe_max_number: Option<NumberFor<Block>>,
//	) -> sp_blockchain::Result<Block::Hash> {
func (lc LongestChain[H, N, Hasher, Header, E]) finalityTarget(
	baseHash H,
	maybeMaxNumber *N,
) (h H, err error) {
	// 		use sp_blockchain::Error::{Application, MissingHeader};
	// 		let blockchain = self.backend.blockchain();
	blockchain := lc.backend.Blockchain()

	// 		let mut current_head = self.best_header()?;
	// 		let mut best_hash = current_head.hash();
	currentHead, err := lc.bestHeader()
	if err != nil {
		return h, err
	}
	bestHash := currentHead.Hash()

	// 		let base_header = blockchain
	// 			.header(base_hash)?
	// 			.ok_or_else(|| MissingHeader(base_hash.to_string()))?;
	// 		let base_number = *base_header.number();
	baseHeader, err := blockchain.Header(baseHash)
	if err != nil {
		return h, err
	}
	if baseHeader == nil {
		return h, fmt.Errorf("%w: %s", chain.ErrMissingHeader, baseHash)
	}
	baseNumber := (*baseHeader).Number()

	// 		if let Some(max_number) = maybe_max_number {
	if maybeMaxNumber != nil {
		// 			if max_number < base_number {
		// 				let msg = format!(
		// 					"Requested a finality target using max number {} below the base number {}",
		// 					max_number, base_number
		// 				);
		// 				return Err(Application(msg.into()))
		// 			}
		maxNumber := *maybeMaxNumber
		if maxNumber < baseNumber {
			msg := fmt.Sprintf(
				"Requested a finality target using max number %d below the base number %d",
				maxNumber, baseNumber,
			)
			return h, fmt.Errorf("%w: %s", chain.ErrApplication, msg)
		}

		// 			while current_head.number() > &max_number {
		// 				best_hash = *current_head.parent_hash();
		// 				current_head = blockchain
		// 					.header(best_hash)?
		// 					.ok_or_else(|| MissingHeader(format!("{best_hash:?}")))?;
		// 			}
		for currentHead.Number() > maxNumber {
			bestHash = currentHead.ParentHash()
			current, err := blockchain.Header(bestHash)
			if err != nil {
				return h, err
			}
			if current == nil {
				return h, fmt.Errorf("%w: %s", chain.ErrMissingHeader, bestHash)
			}
			currentHead = *current
		}
	}

	// 		while current_head.hash() != base_hash {
	// 			if *current_head.number() < base_number {
	// 				let msg = format!(
	// 					"Requested a finality target using a base {:?} not in the best chain {:?}",
	// 					base_hash, best_hash,
	// 				);
	// 				return Err(Application(msg.into()))
	// 			}
	// 			let current_hash = *current_head.parent_hash();
	// 			current_head = blockchain
	// 				.header(current_hash)?
	// 				.ok_or_else(|| MissingHeader(format!("{best_hash:?}")))?;
	// 		}
	for currentHead.Hash() != baseHash {
		if currentHead.Number() < baseNumber {
			msg := fmt.Sprintf(
				"Requested a finality target using a base %s not in the best chain %s",
				baseHash, bestHash,
			)
			return h, fmt.Errorf("%w: %s", chain.ErrApplication, msg)
		}
		currentHash := currentHead.ParentHash()
		current, err := blockchain.Header(currentHash)
		if err != nil {
			return h, fmt.Errorf("%w: %s", chain.ErrMissingHeader, bestHash)
		}
		currentHead = *current
	}

	// 		Ok(best_hash)
	return bestHash, nil
}

//	fn leaves(&self) -> Result<Vec<<Block as BlockT>::Hash>, sp_blockchain::Error> {
//		self.backend.blockchain().leaves()
//	}
func (lc LongestChain[H, N, Hasher, Header, E]) leaves() ([]H, error) {
	leaves, err := lc.backend.Blockchain().Leaves()
	if err != nil {
		return nil, err
	}
	return leaves, nil
}

// }

// #[async_trait::async_trait]
// impl<B, Block> SelectChain<Block> for LongestChain<B, Block>
// where
//
//	B: backend::Backend<Block>,
//	Block: BlockT,
//
//	{
//		async fn leaves(&self) -> Result<Vec<<Block as BlockT>::Hash>, ConsensusError> {
//			LongestChain::leaves(self).map_err(|e| ConsensusError::ChainLookup(e.to_string()))
//		}
func (lc LongestChain[H, N, Hasher, Header, E]) Leaves() <-chan common.LeavesError[H] {
	leavesChan := make(chan common.LeavesError[H])
	go func() {
		defer close(leavesChan)
		leaves, err := lc.leaves()
		if err != nil {
			leavesChan <- common.LeavesError[H]{Error: err}
			return
		}
		leavesChan <- common.LeavesError[H]{Leaves: leaves}
	}()
	return leavesChan
}

//	async fn best_chain(&self) -> Result<<Block as BlockT>::Header, ConsensusError> {
//		LongestChain::best_header(self).map_err(|e| ConsensusError::ChainLookup(e.to_string()))
//	}
func (lc LongestChain[H, N, Hasher, Header, E]) BestChain() <-chan common.HeaderError[H, N, Header] {
	bestChainChan := make(chan common.HeaderError[H, N, Header])
	go func() {
		defer close(bestChainChan)
		header, err := lc.bestHeader()
		if err != nil {
			bestChainChan <- common.HeaderError[H, N, Header]{Error: err}
			return
		}
		bestChainChan <- common.HeaderError[H, N, Header]{Header: header}
	}()
	return bestChainChan
}

//		async fn finality_target(
//			&self,
//			base_hash: Block::Hash,
//			maybe_max_number: Option<NumberFor<Block>>,
//		) -> Result<Block::Hash, ConsensusError> {
//			LongestChain::finality_target(self, base_hash, maybe_max_number)
//				.map_err(|e| ConsensusError::ChainLookup(e.to_string()))
//		}
//	}
func (lc LongestChain[H, N, Hasher, Header, E]) FinalityTarget(
	baseHash H,
	maybeMaxNumber *N,
) <-chan common.HashError[H] {
	finalityTargetChan := make(chan common.HashError[H])
	go func() {
		defer close(finalityTargetChan)
		hash, err := lc.finalityTarget(baseHash, maybeMaxNumber)
		if err != nil {
			finalityTargetChan <- common.HashError[H]{Error: err}
			return
		}
		finalityTargetChan <- common.HashError[H]{Hash: hash}
	}()
	return finalityTargetChan
}
