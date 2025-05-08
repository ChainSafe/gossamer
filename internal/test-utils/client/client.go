package client

import (
	"math"

	"github.com/ChainSafe/gossamer/internal/client"
	"github.com/ChainSafe/gossamer/internal/client/api"
	chainspec "github.com/ChainSafe/gossamer/internal/client/chain-spec"
	"github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/db"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	primitives_api "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/tidwall/btree"
)

// / A genesis storage initialization trait.
// pub trait GenesisInit: Default {
type GenesisInit interface {
	/// Construct genesis storage.
	// fn genesis_storage(&self) -> Storage;
	GenesisStorage() storage.Storage
}

type DefaultGenesisInit struct{}

func (d DefaultGenesisInit) GenesisStorage() storage.Storage {
	return storage.Storage{
		Top:             btree.Map[string, []byte]{},
		ChildrenDefault: make(map[string]storage.StorageChild),
	}
}

// / A builder for creating a test client instance.
// pub struct TestClientBuilder<Block: BlockT, ExecutorDispatch, Backend: 'static, G: GenesisInit> {
type TestClientBuilder[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
	G GenesisInit,
	Executor client.ExecutorT,
	RA primitives_api.ConstructRuntimeApi[primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any]],
] struct {
	// genesis_init: G,
	genesisInit G
	/// The key is an unprefixed storage key, this only contains
	/// default child trie content.
	// child_storage_extension: HashMap<Vec<u8>, StorageChild>,
	childStorageExtension map[string]storage.StorageChild
	// backend: Arc<Backend>,
	backend *db.Backend[H, Hasher, N, E, Header]
	// _executor: std::marker::PhantomData<ExecutorDispatch>,
	// fork_blocks: ForkBlocks<Block>,
	forkBlocks api.ForkBlocks[H, N]
	// bad_blocks: BadBlocks<Block>,
	badBlocks api.BadBlocks[H]
	// enable_offchain_indexing_api: bool,
	enableOffchainIndexingAPI bool
	// enable_import_proof_recording: bool,
	enableImportProofRecording bool
	// no_genesis: bool,
	noGenesis bool
}

// impl<Block: BlockT, ExecutorDispatch, G: GenesisInit> Default
// 	for TestClientBuilder<Block, ExecutorDispatch, Backend<Block>, G>
// {
// 	fn default() -> Self {
// 		Self::with_default_backend()
// 	}
// }

// impl<Block: BlockT, ExecutorDispatch, G: GenesisInit>
//
//	TestClientBuilder<Block, ExecutorDispatch, Backend<Block>, G>
//
//	{
//		/// Create new `TestClientBuilder` with default backend.
//		pub fn with_default_backend() -> Self {
//			let backend = Arc::new(Backend::new_test(std::u32::MAX, std::u64::MAX));
//			Self::with_backend(backend)
//		}
//
// / Create new `TestClientBuilder` with default backend.
func NewTestClientBuilderWithDefaultBackend[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
	G GenesisInit,
	Executor client.ExecutorT,
	RA primitives_api.ConstructRuntimeApi[primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any]],
]() TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA] {
	backend := db.NewTestBackend[H, N, E, Hasher, Header](math.MaxUint32, math.MaxUint64)
	return NewTestClientBuilderWithBackend[H, Hasher, N, E, Header, G, Executor, RA](backend)
}

// / Create new `TestClientBuilder` with default backend and pruning window size
//
//	pub fn with_pruning_window(blocks_pruning: u32) -> Self {
//		let backend = Arc::new(Backend::new_test(blocks_pruning, 0));
//		Self::with_backend(backend)
//	}
func NewTestClientBuilderWithPruningWindow[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
	G GenesisInit,
	Executor client.ExecutorT,
	RA primitives_api.ConstructRuntimeApi[primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any]],
](blocksPruning uint32) TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA] {
	backend := db.NewTestBackend[H, N, E, Hasher, Header](blocksPruning, 0)
	return NewTestClientBuilderWithBackend[H, Hasher, N, E, Header, G, Executor, RA](backend)
}

// / Create new `TestClientBuilder` with default backend and storage chain mode
//
//		pub fn with_tx_storage(blocks_pruning: u32) -> Self {
//			let backend =
//				Arc::new(Backend::new_test_with_tx_storage(BlocksPruning::Some(blocks_pruning), 0));
//			Self::with_backend(backend)
//		}
//	}
func NewTestClientBuilderWithTxStorage[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
	G GenesisInit,
	Executor client.ExecutorT,
	RA primitives_api.ConstructRuntimeApi[primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any]],
](blocksPruning uint32) TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA] {
	backend := db.NewTestBackendWithTxStorage[H, N, E, Hasher, Header](db.BlocksPruningSome(blocksPruning), 0)
	return NewTestClientBuilderWithBackend[H, Hasher, N, E, Header, G, Executor, RA](backend)
}

// impl<Block: BlockT, ExecutorDispatch, Backend, G: GenesisInit>
//
//	TestClientBuilder<Block, ExecutorDispatch, Backend, G>
//
//	{
//		/// Create a new instance of the test client builder.
//		pub fn with_backend(backend: Arc<Backend>) -> Self {
//			TestClientBuilder {
//				backend,
//				child_storage_extension: Default::default(),
//				genesis_init: Default::default(),
//				_executor: Default::default(),
//				fork_blocks: None,
//				bad_blocks: None,
//				enable_offchain_indexing_api: false,
//				no_genesis: false,
//				enable_import_proof_recording: false,
//			}
//		}
func NewTestClientBuilderWithBackend[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
	G GenesisInit,
	Executor client.ExecutorT,
	RA primitives_api.ConstructRuntimeApi[primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any]],
](backend *db.Backend[H, Hasher, N, E, Header]) TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA] {
	return TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA]{
		backend:               backend,
		childStorageExtension: make(map[string]storage.StorageChild),
		// genesisInit:                nil,
		forkBlocks:                 nil,
		badBlocks:                  nil,
		enableOffchainIndexingAPI:  false,
		noGenesis:                  false,
		enableImportProofRecording: false,
	}
}

// /// Alter the genesis storage parameters.
//
//	pub fn genesis_init_mut(&mut self) -> &mut G {
//		&mut self.genesis_init
//	}
func (tcb *TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA]) GenesisInitMut() G {
	return tcb.genesisInit
}

// /// Give access to the underlying backend of these clients
//
//	pub fn backend(&self) -> Arc<Backend> {
//		self.backend.clone()
//	}
func (tcb *TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA]) Backend() *db.Backend[H, Hasher, N, E, Header] {
	return tcb.backend
}

// 	/// Extend child storage
// 	pub fn add_child_storage(
// 		mut self,
// 		child_info: &ChildInfo,
// 		key: impl AsRef<[u8]>,
// 		value: impl AsRef<[u8]>,
// 	) -> Self {
// 		let storage_key = child_info.storage_key();
// 		let entry = self.child_storage_extension.entry(storage_key.to_vec()).or_insert_with(|| {
// 			StorageChild { data: Default::default(), child_info: child_info.clone() }
// 		});
// 		entry.data.insert(key.as_ref().to_vec(), value.as_ref().to_vec());
// 		self
// 	}

// 	/// Sets custom block rules.
// 	pub fn set_block_rules(
// 		mut self,
// 		fork_blocks: ForkBlocks<Block>,
// 		bad_blocks: BadBlocks<Block>,
// 	) -> Self {
// 		self.fork_blocks = fork_blocks;
// 		self.bad_blocks = bad_blocks;
// 		self
// 	}

// 	/// Enable the offchain indexing api.
// 	pub fn enable_offchain_indexing_api(mut self) -> Self {
// 		self.enable_offchain_indexing_api = true;
// 		self
// 	}

// 	/// Enable proof recording on import.
// 	pub fn enable_import_proof_recording(mut self) -> Self {
// 		self.enable_import_proof_recording = true;
// 		self
// 	}

// / Disable writing genesis.
//
//	pub fn set_no_genesis(mut self) -> Self {
//		self.no_genesis = true;
//		self
//	}
func (tcb *TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA]) SetNoGenesis() {
	tcb.noGenesis = true
}

type ExecutorDispatch interface {
	api.CallExecutor
	executor.RuntimeVersionOf
}

// /// Build the test client with the given native executor.
// pub fn build_with_executor<RuntimeApi>(
//
//	self,
//	executor: ExecutorDispatch,
//
// ) -> (
//
//	client::Client<Backend, ExecutorDispatch, Block, RuntimeApi>,
//	sc_consensus::LongestChain<Backend, Block>,
//
// )
// where
//
//	ExecutorDispatch:
//		sc_client_api::CallExecutor<Block> + sc_executor::RuntimeVersionOf + Clone + 'static,
//	Backend: sc_client_api::backend::Backend<Block>,
//	<Backend as sc_client_api::backend::Backend<Block>>::OffchainStorage: 'static,
//
// {
func (tcb *TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA]) BuildWithExecutor(
	executor ExecutorDispatch,
) (client.Client[H, Hasher, N, E, Executor, Header, RA], common.LongestChain[H, N, Hasher, Header, E]) {
	// 		let storage = {
	// 			let mut storage = self.genesis_init.genesis_storage();
	// 			// Add some child storage keys.
	// 			for (key, child_content) in self.child_storage_extension {
	// 				storage.children_default.insert(
	// 					key,
	// 					StorageChild {
	// 						data: child_content.data.into_iter().collect(),
	// 						child_info: child_content.child_info,
	// 					},
	// 				);
	// 			}

	// 			storage
	// 		};
	store := tcb.genesisInit.GenesisStorage()
	// Add some child storage keys.
	for key, childContent := range tcb.childStorageExtension {
		store.ChildrenDefault[key] = storage.StorageChild{
			Data:      childContent.Data,
			ChildInfo: childContent.ChildInfo,
		}
	}

	// 		let client_config = ClientConfig {
	// 			enable_import_proof_recording: self.enable_import_proof_recording,
	// 			offchain_indexing_api: self.enable_offchain_indexing_api,
	// 			no_genesis: self.no_genesis,
	// 			..Default::default()
	// 		};
	clientConfig := client.NewClientConfig[N]()
	clientConfig.EnableImportProofRecording = tcb.enableImportProofRecording
	clientConfig.OffchainIndexingAPI = tcb.enableOffchainIndexingAPI
	clientConfig.NoGenesis = tcb.noGenesis

	// 		let genesis_block_builder = sc_service::GenesisBlockBuilder::new(
	// 			&storage,
	// 			!client_config.no_genesis,
	// 			self.backend.clone(),
	// 			executor.clone(),
	// 		)
	// 		.expect("Creates genesis block builder");
	genesisBlockBuilder := chainspec.NewGenesisBlockBuilderWithStorage[H, N, Hasher, Header, E](
		store,
		!clientConfig.NoGenesis,
		tcb.backend,
		executor,
	)
	_ = genesisBlockBuilder

	// 		let spawn_handle = Box::new(TaskExecutor::new());

	// 		let client = client::Client::new(
	// 			self.backend.clone(),
	// 			executor,
	// 			spawn_handle,
	// 			genesis_block_builder,
	// 			self.fork_blocks,
	// 			self.bad_blocks,
	// 			None,
	// 			None,
	// 			client_config,
	// 		)
	// 		.expect("Creates new client");

	// client := client.New[H, Hasher, N, E, Header](
	// 	tcb.backend,
	// 	executor,
	// 	chainspec.NewTaskExecutor(),
	// 	genesisBlockBuilder,
	// 	tcb.forkBlocks,
	// 	tcb.badBlocks,
	// 	nil,
	// 	nil,
	// 	clientConfig,
	// )

	// 		let longest_chain = sc_consensus::LongestChain::new(self.backend);

	// 		(client, longest_chain)
	panic("Not implemented")
}

// }

// impl<Block: BlockT, H, Backend, G: GenesisInit>
//
//	TestClientBuilder<Block, client::LocalCallExecutor<Block, Backend, WasmExecutor<H>>, Backend, G>
//
//	{
//		/// Build the test client with the given native executor.
//		pub fn build_with_native_executor<RuntimeApi, I>(
//			self,
//			executor: I,
//		) -> (
//			client::Client<
//				Backend,
//				client::LocalCallExecutor<Block, Backend, WasmExecutor<H>>,
//				Block,
//				RuntimeApi,
//			>,
//			sc_consensus::LongestChain<Backend, Block>,
//		)
//		where
//			I: Into<Option<WasmExecutor<H>>>,
//			Backend: sc_client_api::backend::Backend<Block> + 'static,
//			H: sc_executor::HostFunctions,
//		{
func (tcb *TestClientBuilder[H, Hasher, N, E, Header, G, Executor, RA]) BuildWithNativeExecutor(
	exec *executor.WasmExecutor,
) (client.Client[H, Hasher, N, E, Executor, Header, RA], common.LongestChain[H, N, Hasher, Header, E]) {

	// 		let executor = executor.into().unwrap_or_else(|| WasmExecutor::<H>::builder().build());
	// 		let executor = LocalCallExecutor::new(
	// 			self.backend.clone(),
	// 			executor.clone(),
	// 			Default::default(),
	// 			ExecutionExtensions::new(None, Arc::new(executor)),
	// 		)
	// 		.expect("Creates LocalCallExecutor");
	if exec == nil {
		exec = &executor.WasmExecutor{}
	}

	// 		self.build_with_executor(executor)
	return tcb.BuildWithExecutor(exec)
}

// }
