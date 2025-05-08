package client

import (
	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

type executorI interface {
	core.CodeExecutor
	executor.RuntimeVersionOf
}

// / Call executor that executes methods locally, querying all required
// / data from local backend.
// pub struct LocalCallExecutor<Block: BlockT, B, E> {
type LocalCallExecutor[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// backend: Arc<B>,
	backend api.Backend[H, N, Hasher, Header, E]
	// executor: E,
	executor executorI

	// code_provider: CodeProvider<Block, B, E>,
	codeProvider CodeProvider[H, N, Hasher, Header, E]
	// execution_extensions: Arc<ExecutionExtensions<Block>>,
}

// impl<Block: BlockT, B, E> LocalCallExecutor<Block, B, E>
// where
// 	E: CodeExecutor + RuntimeVersionOf + Clone + 'static,
// 	B: backend::Backend<Block>,
// {
// 	/// Creates new instance of local call executor.
// 	pub fn new(
// 		backend: Arc<B>,
// 		executor: E,
// 		client_config: ClientConfig<Block>,
// 		execution_extensions: ExecutionExtensions<Block>,
// 	) -> sp_blockchain::Result<Self> {
// 		let code_provider = CodeProvider::new(&client_config, executor.clone(), backend.clone())?;

// 		Ok(LocalCallExecutor {
// 			backend,
// 			executor,
// 			code_provider,
// 			execution_extensions: Arc::new(execution_extensions),
// 		})
// 	}
// }

// impl<Block: BlockT, B, E> Clone for LocalCallExecutor<Block, B, E>
// where
// 	E: Clone,
// {
// 	fn clone(&self) -> Self {
// 		LocalCallExecutor {
// 			backend: self.backend.clone(),
// 			executor: self.executor.clone(),
// 			code_provider: self.code_provider.clone(),
// 			execution_extensions: self.execution_extensions.clone(),
// 		}
// 	}
// }

// impl<B, E, Block> CallExecutor<Block> for LocalCallExecutor<Block, B, E>
// where
// 	B: backend::Backend<Block>,
// 	E: CodeExecutor + RuntimeVersionOf + Clone + 'static,
// 	Block: BlockT,
// {
// 	type Error = E::Error;

// 	type Backend = B;

// 	fn execution_extensions(&self) -> &ExecutionExtensions<Block> {
// 		&self.execution_extensions
// 	}

// 	fn call(
// 		&self,
// 		at_hash: Block::Hash,
// 		method: &str,
// 		call_data: &[u8],
// 		context: CallContext,
// 	) -> sp_blockchain::Result<Vec<u8>> {
// 		let mut changes = OverlayedChanges::default();
// 		let at_number =
// 			self.backend.blockchain().expect_block_number_from_id(&BlockId::Hash(at_hash))?;
// 		let state = self.backend.state_at(at_hash)?;

// 		let state_runtime_code = sp_state_machine::backend::BackendRuntimeCode::new(&state);
// 		let runtime_code =
// 			state_runtime_code.runtime_code().map_err(sp_blockchain::Error::RuntimeCode)?;

// 		let runtime_code = self.code_provider.maybe_override_code(runtime_code, &state, at_hash)?.0;

// 		let mut extensions = self.execution_extensions.extensions(at_hash, at_number);

// 		let mut sm = StateMachine::new(
// 			&state,
// 			&mut changes,
// 			&self.executor,
// 			method,
// 			call_data,
// 			&mut extensions,
// 			&runtime_code,
// 			context,
// 		)
// 		.set_parent_hash(at_hash);

// 		sm.execute().map_err(Into::into)
// 	}

// 	fn contextual_call(
// 		&self,
// 		at_hash: Block::Hash,
// 		method: &str,
// 		call_data: &[u8],
// 		changes: &RefCell<OverlayedChanges<HashingFor<Block>>>,
// 		recorder: &Option<ProofRecorder<Block>>,
// 		call_context: CallContext,
// 		extensions: &RefCell<Extensions>,
// 	) -> Result<Vec<u8>, sp_blockchain::Error> {
// 		let state = self.backend.state_at(at_hash)?;

// 		let changes = &mut *changes.borrow_mut();

// 		// It is important to extract the runtime code here before we create the proof
// 		// recorder to not record it. We also need to fetch the runtime code from `state` to
// 		// make sure we use the caching layers.
// 		let state_runtime_code = sp_state_machine::backend::BackendRuntimeCode::new(&state);

// 		let runtime_code =
// 			state_runtime_code.runtime_code().map_err(sp_blockchain::Error::RuntimeCode)?;
// 		let runtime_code = self.code_provider.maybe_override_code(runtime_code, &state, at_hash)?.0;
// 		let mut extensions = extensions.borrow_mut();

// 		match recorder {
// 			Some(recorder) => {
// 				let trie_state = state.as_trie_backend();

// 				let backend = sp_state_machine::TrieBackendBuilder::wrap(&trie_state)
// 					.with_recorder(recorder.clone())
// 					.build();

// 				let mut state_machine = StateMachine::new(
// 					&backend,
// 					changes,
// 					&self.executor,
// 					method,
// 					call_data,
// 					&mut extensions,
// 					&runtime_code,
// 					call_context,
// 				)
// 				.set_parent_hash(at_hash);
// 				state_machine.execute()
// 			},
// 			None => {
// 				let mut state_machine = StateMachine::new(
// 					&state,
// 					changes,
// 					&self.executor,
// 					method,
// 					call_data,
// 					&mut extensions,
// 					&runtime_code,
// 					call_context,
// 				)
// 				.set_parent_hash(at_hash);
// 				state_machine.execute()
// 			},
// 		}
// 		.map_err(Into::into)
// 	}

// 	fn runtime_version(&self, at_hash: Block::Hash) -> sp_blockchain::Result<RuntimeVersion> {
// 		let state = self.backend.state_at(at_hash)?;
// 		let state_runtime_code = sp_state_machine::backend::BackendRuntimeCode::new(&state);

// 		let runtime_code =
// 			state_runtime_code.runtime_code().map_err(sp_blockchain::Error::RuntimeCode)?;
// 		self.code_provider
// 			.maybe_override_code(runtime_code, &state, at_hash)
// 			.map(|(_, v)| v)
// 	}

// 	fn prove_execution(
// 		&self,
// 		at_hash: Block::Hash,
// 		method: &str,
// 		call_data: &[u8],
// 	) -> sp_blockchain::Result<(Vec<u8>, StorageProof)> {
// 		let at_number =
// 			self.backend.blockchain().expect_block_number_from_id(&BlockId::Hash(at_hash))?;
// 		let state = self.backend.state_at(at_hash)?;

// 		let trie_backend = state.as_trie_backend();

// 		let state_runtime_code = sp_state_machine::backend::BackendRuntimeCode::new(trie_backend);
// 		let runtime_code =
// 			state_runtime_code.runtime_code().map_err(sp_blockchain::Error::RuntimeCode)?;
// 		let runtime_code = self.code_provider.maybe_override_code(runtime_code, &state, at_hash)?.0;

// 		sp_state_machine::prove_execution_on_trie_backend(
// 			trie_backend,
// 			&mut Default::default(),
// 			&self.executor,
// 			method,
// 			call_data,
// 			&runtime_code,
// 			&mut self.execution_extensions.extensions(at_hash, at_number),
// 		)
// 		.map_err(Into::into)
// 	}
// }

// impl<B, E, Block> RuntimeVersionOf for LocalCallExecutor<Block, B, E>
// where
// 	E: RuntimeVersionOf,
// 	Block: BlockT,
// {
// 	fn runtime_version(
// 		&self,
// 		ext: &mut dyn sp_externalities::Externalities,
// 		runtime_code: &sp_core::traits::RuntimeCode,
// 	) -> Result<sp_version::RuntimeVersion, sc_executor::error::Error> {
// 		RuntimeVersionOf::runtime_version(&self.executor, ext, runtime_code)
// 	}
// }

// impl<Block, B, E> sp_version::GetRuntimeVersionAt<Block> for LocalCallExecutor<Block, B, E>
// where
// 	B: backend::Backend<Block>,
// 	E: CodeExecutor + RuntimeVersionOf + Clone + 'static,
// 	Block: BlockT,
// {
// 	fn runtime_version(&self, at: Block::Hash) -> Result<sp_version::RuntimeVersion, String> {
// 		CallExecutor::runtime_version(self, at).map_err(|e| e.to_string())
// 	}
// }

// impl<Block, B, E> sp_version::GetNativeVersion for LocalCallExecutor<Block, B, E>
// where
// 	B: backend::Backend<Block>,
// 	E: CodeExecutor + sp_version::GetNativeVersion + Clone + 'static,
// 	Block: BlockT,
// {
// 	fn native_version(&self) -> &sp_version::NativeVersion {
// 		self.executor.native_version()
// 	}
// }
