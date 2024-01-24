package blockbuilder

import (
	clientapi "github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

type BlockBuilderProvider[H runtime.Hash, N runtime.Number] interface {
	/// Create a new block, built on top of `parent`.
	///
	/// When proof recording is enabled, all accessed trie nodes are saved.
	/// These recorded trie nodes can be used by a third party to proof the
	/// output of this block builder without having access to the full storage.
	// fn new_block_at<R: Into<RecordProof>>(
	// 	&self,
	// 	parent: Block::Hash,
	// 	inherent_digests: Digest,
	// 	record_proof: R,
	// ) -> sp_blockchain::Result<BlockBuilder<Block, RA, B>>;

	/// Create a new block, built on the head of the chain.
	// fn new_block(
	// 	&self,
	// 	inherent_digests: Digest,
	// ) -> sp_blockchain::Result<BlockBuilder<Block, RA, B>>;
	NewBlock(inherentDigests runtime.Digest) (BlockBuilder[H, N], error)
}

// / Utility for building new (valid) blocks from a stream of extrinsics.
// pub struct BlockBuilder<'a, Block: BlockT, A: ProvideRuntimeApi<Block>, B> {
type BlockBuilder[H runtime.Hash, N runtime.Number] struct {
	// extrinsics: Vec<Block::Extrinsic>,
	extrinsics []runtime.Extrinsic
	// api: ApiRef<'a, A::Api>,
	api api.APIExt
	// version: u32,
	version uint32
	// parent_hash: Block::Hash,
	parentHash H
	// backend: &'a B,
	backend clientapi.Backend[H, N]
	/// The estimated size of the block header.
	// estimated_header_size: usize,
	estiamtedHeaderSize uint
}
