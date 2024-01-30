package blockbuilder

import (
	"fmt"

	clientapi "github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
)

type BlockBuilderProvider[H runtime.Hash, N runtime.Number, T statemachine.Transaction] interface {
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
	NewBlock(inherentDigests runtime.Digest) (BlockBuilder[H, N, T], error)
}

// / Utility for building new (valid) blocks from a stream of extrinsics.
// pub struct BlockBuilder<'a, Block: BlockT, A: ProvideRuntimeApi<Block>, B> {
type BlockBuilder[H runtime.Hash, N runtime.Number, T statemachine.Transaction] struct {
	// extrinsics: Vec<Block::Extrinsic>,
	extrinsics []runtime.Extrinsic
	// api: ApiRef<'a, A::Api>,
	api api.APIExt[H, N, T]
	// version: u32,
	version uint32
	// parent_hash: Block::Hash,
	parentHash H
	// backend: &'a B,
	backend clientapi.Backend[H, N, T]
	/// The estimated size of the block header.
	// estimated_header_size: usize,
	estimatedHeaderSize uint
}

func NewBlockBuilder[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], T statemachine.Transaction](
	api api.ProvideRuntimeAPI[H, N, T],
	parentHash H,
	parentNumber N,
	recordProof bool,
	inherentDigests runtime.Digest,
	backend clientapi.Backend[H, N, T],
) (BlockBuilder[H, N, T], error) {
	var defaultHash H
	header := generic.NewHeader[N, H, Hasher](
		parentNumber+1,
		defaultHash,
		defaultHash,
		parentHash,
		inherentDigests,
	)

	encodedHeader, err := header.MarshalSCALE()
	if err != nil {
		return BlockBuilder[H, N, T]{}, err
	}
	estimatedHeaderSize := uint(len(encodedHeader))

	runtimeAPI := api.RuntimeAPI()

	if recordProof {
		runtimeAPI.RecordProof()
	}

	err = runtimeAPI.InitializeBlock(parentHash, &header)
	if err != nil {
		return BlockBuilder[H, N, T]{}, err
	}

	version, err := runtimeAPI.APIVersion(parentHash)
	if err != nil {
		return BlockBuilder[H, N, T]{}, err
	}
	if version == nil {
		return BlockBuilder[H, N, T]{}, fmt.Errorf("VersionInvalid")
	}

	return BlockBuilder[H, N, T]{
		parentHash:          parentHash,
		api:                 runtimeAPI,
		version:             *version,
		backend:             backend,
		estimatedHeaderSize: estimatedHeaderSize,
	}, nil
}
