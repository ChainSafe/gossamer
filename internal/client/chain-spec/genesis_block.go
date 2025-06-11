// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package chainspec

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

var ErrMissingRuntime = errors.New("runtime missing from initial storage, could not read state version")

func ResolveStateVersionFromWasm[Hasher runtime.Hasher[H], H runtime.Hash](
	storage storage.Storage,
	executor executor.RuntimeVersionOf,
) (storage.StateVersion, error) {
	panic("unimpl")
}

// / Create a genesis block, given the initial storage.
// pub fn construct_genesis_block<Block: BlockT>(
//
//	state_root: Block::Hash,
//	state_version: StateVersion,
//
// ) -> Block {
func ConstructGenesisBlock[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	stateRoot H,
	stateVersion storage.StateVersion,
) runtime.Block[H, N, E, Header] {

	// 	let extrinsics_root = <<<Block as BlockT>::Header as HeaderT>::Hashing as HashT>::trie_root(
	// 		Vec::new(),
	// 		state_version,
	// 	);
	extrinsicsRoot := (*(new(Hasher))).TrieRoot(
		make([]runtime.KeyValue, 0),
		stateVersion,
	)

	// Block::new(
	//
	//	<<Block as BlockT>::Header as HeaderT>::new(
	//		Zero::zero(),
	//		extrinsics_root,
	//		state_root,
	//		Default::default(),
	//		Default::default(),
	//	),
	//	Default::default(),
	//
	// )

	var parentHash H
	var digest runtime.Digest
	var extrinsics []E
	header := generic.NewHeader[N, H, Hasher](0, extrinsicsRoot, stateRoot, parentHash, digest)
	block := generic.NewBlock[Hasher, E, N, H, Header](
		any(header).(Header),
		extrinsics,
	)
	return block
}

// / Trait for building the genesis block.
// pub trait BuildGenesisBlock<Block: BlockT> {
type BuildGenesisBlock[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	// 	/// The import operation used to import the genesis block into the backend.
	// 	type BlockImportOperation;

	// /// Returns the built genesis block along with the block import operation
	// /// after setting the genesis storage.
	// fn build_genesis_block(self) -> sp_blockchain::Result<(Block, Self::BlockImportOperation)>;
	BuildGenesisBlock(runtime.Block[H, N, E, Header], api.BlockImportOperation[N, H, Hasher, Header, E])
}

// /// Default genesis block builder in Substrate.
// pub struct GenesisBlockBuilder<Block: BlockT, B, E> {
type GenesisBlockBuilder[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// genesis_storage: Storage,
	gensisStorage storage.Storage
	// commit_genesis_state: bool,
	commitGenesisState bool
	// backend: Arc<B>,
	backend api.Backend[H, N, Hasher, Header, E]
	// executor: E,
	executor executor.RuntimeVersionOf
	// _phantom: PhantomData<Block>,
}

//	impl<Block: BlockT, B: Backend<Block>, E: RuntimeVersionOf> GenesisBlockBuilder<Block, B, E> {
//		/// Constructs a new instance of [`GenesisBlockBuilder`].
//		pub fn new(
//			build_genesis_storage: &dyn BuildStorage,
//			commit_genesis_state: bool,
//			backend: Arc<B>,
//			executor: E,
//		) -> sp_blockchain::Result<Self> {
//			let genesis_storage =
//				build_genesis_storage.build_storage().map_err(sp_blockchain::Error::Storage)?;
//			Self::new_with_storage(genesis_storage, commit_genesis_state, backend, executor)
//		}
func NewGenesisBlockBuilder[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	buildGenesisStorage runtime.BuildStorage,
	commitGenesisState bool,
	backend api.Backend[H, N, Hasher, Header, E],
	executor executor.RuntimeVersionOf,
) (*GenesisBlockBuilder[H, N, Hasher, Header, E], error) {
	genesisStorage, err := buildGenesisStorage.BuildStorage()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", blockchain.ErrStorage, err)
	}
	return NewGenesisBlockBuilderWithStorage(
		genesisStorage,
		commitGenesisState,
		backend,
		executor,
	), nil
}

//		/// Constructs a new instance of [`GenesisBlockBuilder`] using provided storage.
//		pub fn new_with_storage(
//			genesis_storage: Storage,
//			commit_genesis_state: bool,
//			backend: Arc<B>,
//			executor: E,
//		) -> sp_blockchain::Result<Self> {
//			Ok(Self {
//				genesis_storage,
//				commit_genesis_state,
//				backend,
//				executor,
//				_phantom: PhantomData::<Block>,
//			})
//		}
//	}
func NewGenesisBlockBuilderWithStorage[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	genesisStorage storage.Storage,
	commitGenesisState bool,
	backend api.Backend[H, N, Hasher, Header, E],
	executor executor.RuntimeVersionOf,
) *GenesisBlockBuilder[H, N, Hasher, Header, E] {
	return &GenesisBlockBuilder[H, N, Hasher, Header, E]{
		gensisStorage:      genesisStorage,
		commitGenesisState: commitGenesisState,
		backend:            backend,
		executor:           executor,
	}
}

// impl<Block: BlockT, B: Backend<Block>, E: RuntimeVersionOf> BuildGenesisBlock<Block>
// 	for GenesisBlockBuilder<Block, B, E>
// {
// 	type BlockImportOperation = <B as Backend<Blgenesisock>>::BlockImportOperation;

// fn build_genesis_block(self) -> sp_blockchain::Result<(Block, Self::BlockImportOperation)> {
func (g *GenesisBlockBuilder[H, N, Hasher, Header, E]) BuildGenesisBlock() (runtime.Block[H, N, E, Header], api.BlockImportOperation[N, H, Hasher, Header, E], error) {
	// 		let Self { genesis_storage, commit_genesis_state, backend, executor, _phantom } = self;

	// 		let genesis_state_version =
	// 			resolve_state_version_from_wasm::<_, HashingFor<Block>>(&genesis_storage, &executor)?;
	// 		let mut op = backend.begin_operation()?;
	// 		let state_root =
	// 			op.set_genesis_state(genesis_storage, commit_genesis_state, genesis_state_version)?;
	// 		let genesis_block = construct_genesis_block::<Block>(state_root, genesis_state_version);

	genesisStateVersion, err := ResolveStateVersionFromWasm[Hasher, H](g.gensisStorage, g.executor)
	if err != nil {
		return nil, nil, err
	}
	op, err := g.backend.BeginOperation()
	stateRoot, err := op.SetGenesisState(g.gensisStorage, g.commitGenesisState, genesisStateVersion)
	if err != nil {
		return nil, nil, err
	}
	genesisBlock := ConstructGenesisBlock[H, N, Hasher, Header, E](stateRoot, genesisStateVersion)
	// 		Ok((genesis_block, op))
	return genesisBlock, op, nil
}

// }
