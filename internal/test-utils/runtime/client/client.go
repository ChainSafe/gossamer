package client

import (
	service_client "github.com/ChainSafe/gossamer/internal/client"
	"github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/db"
	pruntime "github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/test-utils/client"
	"github.com/ChainSafe/gossamer/internal/test-utils/runtime"
)

// / Test client database backend.
// pub type Backend = substrate_test_client::Backend<substrate_test_runtime::Block>;
type Backend = db.Backend[runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic, runtime.Header]

// / Parameters of test-client builder with test-runtime.
// #[derive(Default)]
//
//	pub struct GenesisParameters {
//		heap_pages_override: Option<u64>,
//		extra_storage: Storage,
//		wasm_code: Option<Vec<u8>>,
//	}
type GenesisParameters struct {
	heapPagesOverride *uint64
	extraStorage      storage.Storage
	wasmCode          []byte
}

// impl GenesisParameters {
// 	/// Set the wasm code that should be used at genesis.
// 	pub fn set_wasm_code(&mut self, code: Vec<u8>) {
// 		self.wasm_code = Some(code);
// 	}

//		/// Access extra genesis storage.
//		pub fn extra_storage(&mut self) -> &mut Storage {
//			&mut self.extra_storage
//		}
//	}
func (g *GenesisParameters) ExtraStorage() *storage.Storage {
	return &g.extraStorage
}

//	impl GenesisInit for GenesisParameters {
//		fn genesis_storage(&self) -> Storage {
//			GenesisStorageBuilder::default()
//				.with_heap_pages(self.heap_pages_override)
//				.with_wasm_code(&self.wasm_code)
//				.with_extra_storage(self.extra_storage.clone())
//				.build()
//		}
//	}
func (g GenesisParameters) GenesisStorage() storage.Storage {
	panic("unimpl")
}

type TestClientBuilder = client.TestClientBuilder[
	runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic, runtime.Header, *GenesisParameters,
]

func NewTestClientBuilderWithDefaultBackend() TestClientBuilder {
	return client.NewTestClientBuilderWithDefaultBackend[
		runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic, runtime.Header, *GenesisParameters,
	]()
}

// / Create new `TestClientBuilder` with default backend and pruning window size
//
//	pub fn with_pruning_window(blocks_pruning: u32) -> Self {
//		let backend = Arc::new(Backend::new_test(blocks_pruning, 0));
//		Self::with_backend(backend)
//	}
func NewTestClientBuilderWithPruningWindow(blocksPruning uint32) TestClientBuilder {
	return client.NewTestClientBuilderWithPruningWindow[
		runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic, runtime.Header, *GenesisParameters,
	](blocksPruning)
}

// / Create new `TestClientBuilder` with default backend and storage chain mode
//
//		pub fn with_tx_storage(blocks_pruning: u32) -> Self {
//			let backend =
//				Arc::new(Backend::new_test_with_tx_storage(BlocksPruning::Some(blocks_pruning), 0));
//			Self::with_backend(backend)
//		}
//	}
func NewTestClientBuilderWithTxStorage(blocksPruning uint32) TestClientBuilder {
	return client.NewTestClientBuilderWithTxStorage[
		runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic, runtime.Header, *GenesisParameters,
	](blocksPruning)
}

// / Test client type with `WasmExecutor` and generic Backend.
// pub type Client<B> = client::Client<
//
//	B,
//	client::LocalCallExecutor<substrate_test_runtime::Block, B, WasmExecutor>,
//	substrate_test_runtime::Block,
//	substrate_test_runtime::RuntimeApi,
//
// >;
type Client struct {
	service_client.Client[
		runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic,
		runtime.Header,
	]
}

// / A `test-runtime` extensions to `TestClientBuilder`.
// pub trait TestClientBuilderExt<B>: Sized {
type TestClientBuilderExt[
	H pruntime.Hash,
	N pruntime.Number,
	Hasher pruntime.Hasher[H],
	Header pruntime.Header[N, H],
	E pruntime.Extrinsic,
] interface {
	// 	/// Returns a mutable reference to the genesis parameters.
	// 	fn genesis_init_mut(&mut self) -> &mut GenesisParameters;

	// 	/// Override the default value for Wasm heap pages.
	// 	fn set_heap_pages(mut self, heap_pages: u64) -> Self {
	// 		self.genesis_init_mut().heap_pages_override = Some(heap_pages);
	// 		self
	// 	}

	// 	/// Add an extra value into the genesis storage.
	// 	///
	// 	/// # Panics
	// 	///
	// 	/// Panics if the key is empty.
	// 	fn add_extra_child_storage<K: Into<Vec<u8>>, V: Into<Vec<u8>>>(
	// 		mut self,
	// 		child_info: &ChildInfo,
	// 		key: K,
	// 		value: V,
	// 	) -> Self {
	// 		let storage_key = child_info.storage_key().to_vec();
	// 		let key = key.into();
	// 		assert!(!storage_key.is_empty());
	// 		assert!(!key.is_empty());
	// 		self.genesis_init_mut()
	// 			.extra_storage
	// 			.children_default
	// 			.entry(storage_key)
	// 			.or_insert_with(|| StorageChild {
	// 				data: Default::default(),
	// 				child_info: child_info.clone(),
	// 			})
	// 			.data
	// 			.insert(key, value.into());
	// 		self
	// 	}

	// 	/// Add an extra child value into the genesis storage.
	// 	///
	// 	/// # Panics
	// 	///
	// 	/// Panics if the key is empty.
	// 	fn add_extra_storage<K: Into<Vec<u8>>, V: Into<Vec<u8>>>(mut self, key: K, value: V) -> Self {
	// 		let key = key.into();
	// 		assert!(!key.is_empty());
	// 		self.genesis_init_mut().extra_storage.top.insert(key, value.into());
	// 		self
	// 	}

	// 	/// Build the test client.
	// 	fn build(self) -> Client<B> {
	// 		self.build_with_longest_chain().0
	// 	}

	/// Build the test client and longest chain selector.
	// 	fn build_with_longest_chain(
	// 		self,
	// 	) -> (Client<B>, sc_consensus::LongestChain<B, substrate_test_runtime::Block>);
	BuildWithLongestChain() (Client, common.LongestChain[H, N, Hasher, Header, E])
	// /// Build the test client and the backend.
	// fn build_with_backend(self) -> (Client<B>, Arc<B>);
}

// impl<B> TestClientBuilderExt<B>
// 	for TestClientBuilder<client::LocalCallExecutor<substrate_test_runtime::Block, B, WasmExecutor>, B>
// where
// 	B: sc_client_api::backend::Backend<substrate_test_runtime::Block> + 'static,
// {
// 	fn genesis_init_mut(&mut self) -> &mut GenesisParameters {
// 		Self::genesis_init_mut(self)
// 	}

// fn build_with_longest_chain(
//
//	self,
//
//	) -> (Client<B>, sc_consensus::LongestChain<B, substrate_test_runtime::Block>) {
//		self.build_with_native_executor(None)
//	}
func (c *Client) BuildWithLongestChain() (Client, common.LongestChain[runtime.Hash, runtime.BlockNumber, runtime.Hasher, runtime.Header, runtime.Extrinsic]) {
	// 	self.build_with_native_executor(None)
	// return c.Client.BuildWithNativeExecutor(nil)
	panic("unimpl")
}

// 	fn build_with_backend(self) -> (Client<B>, Arc<B>) {
// 		let backend = self.backend();
// 		(self.build_with_native_executor(None).0, backend)
// 	}
// }
