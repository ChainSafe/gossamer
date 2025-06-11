package client

import (
	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// / Provider for fetching `:code` of a block.
// /
// / As a node can run with code overrides or substitutes, this will ensure that these are taken into
// / account before returning the actual `code` for a block.
// pub struct CodeProvider<Block: BlockT, Backend, Executor> {
type CodeProvider[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// backend: Arc<Backend>,
	backend api.Backend[H, N, Hasher, Header, E]
	// executor: Arc<Executor>,
	executor executor.RuntimeVersionOf
	// wasm_override: Arc<Option<WasmOverride>>,
	// wasm_substitutes: WasmSubstitutes<Block, Executor, Backend>,
}

// impl<Block: BlockT, Backend, Executor: Clone> Clone for CodeProvider<Block, Backend, Executor> {
// 	fn clone(&self) -> Self {
// 		Self {
// 			backend: self.backend.clone(),
// 			executor: self.executor.clone(),
// 			wasm_override: self.wasm_override.clone(),
// 			wasm_substitutes: self.wasm_substitutes.clone(),
// 		}
// 	}
// }

// impl<Block, Backend, Executor> CodeProvider<Block, Backend, Executor>
// where
// 	Block: BlockT,
// 	Backend: backend::Backend<Block>,
// 	Executor: RuntimeVersionOf,
// {
// 	/// Create a new instance.
// 	pub fn new(
// 		client_config: &ClientConfig<Block>,
// 		executor: Executor,
// 		backend: Arc<Backend>,
// 	) -> sp_blockchain::Result<Self> {
// 		let wasm_override = client_config
// 			.wasm_runtime_overrides
// 			.as_ref()
// 			.map(|p| WasmOverride::new(p.clone(), &executor))
// 			.transpose()?;

// 		let executor = Arc::new(executor);

// 		let wasm_substitutes = WasmSubstitutes::new(
// 			client_config.wasm_runtime_substitutes.clone(),
// 			executor.clone(),
// 			backend.clone(),
// 		)?;

// 		Ok(Self { backend, executor, wasm_override: Arc::new(wasm_override), wasm_substitutes })
// 	}

// 	/// Returns the `:code` for the given `block`.
// 	///
// 	/// This takes into account potential overrides/substitutes.
// 	pub fn code_at_ignoring_overrides(&self, block: Block::Hash) -> sp_blockchain::Result<Vec<u8>> {
// 		let state = self.backend.state_at(block)?;

// 		let state_runtime_code = sp_state_machine::backend::BackendRuntimeCode::new(&state);
// 		let runtime_code =
// 			state_runtime_code.runtime_code().map_err(sp_blockchain::Error::RuntimeCode)?;

// 		self.maybe_override_code_internal(runtime_code, &state, block, true)
// 			.and_then(|r| {
// 				r.0.fetch_runtime_code().map(Into::into).ok_or_else(|| {
// 					sp_blockchain::Error::Backend("Could not find `:code` in backend.".into())
// 				})
// 			})
// 	}

// 	/// Maybe override the given `onchain_code`.
// 	///
// 	/// This takes into account potential overrides/substitutes.
// 	pub fn maybe_override_code<'a>(
// 		&'a self,
// 		onchain_code: RuntimeCode<'a>,
// 		state: &Backend::State,
// 		hash: Block::Hash,
// 	) -> sp_blockchain::Result<(RuntimeCode<'a>, RuntimeVersion)> {
// 		self.maybe_override_code_internal(onchain_code, state, hash, false)
// 	}

// 	/// Maybe override the given `onchain_code`.
// 	///
// 	/// This takes into account potential overrides(depending on `ignore_overrides`)/substitutes.
// 	fn maybe_override_code_internal<'a>(
// 		&'a self,
// 		onchain_code: RuntimeCode<'a>,
// 		state: &Backend::State,
// 		hash: Block::Hash,
// 		ignore_overrides: bool,
// 	) -> sp_blockchain::Result<(RuntimeCode<'a>, RuntimeVersion)> {
// 		let on_chain_version = self.on_chain_runtime_version(&onchain_code, state)?;
// 		let code_and_version = if let Some(d) = self.wasm_override.as_ref().as_ref().and_then(|o| {
// 			if ignore_overrides {
// 				return None
// 			}

// 			o.get(
// 				&on_chain_version.spec_version,
// 				onchain_code.heap_pages,
// 				&on_chain_version.spec_name,
// 			)
// 		}) {
// 			tracing::debug!(target: "code-provider::overrides", block = ?hash, "using WASM override");
// 			d
// 		} else if let Some(s) =
// 			self.wasm_substitutes
// 				.get(on_chain_version.spec_version, onchain_code.heap_pages, hash)
// 		{
// 			tracing::debug!(target: "code-provider::substitutes", block = ?hash, "Using WASM substitute");
// 			s
// 		} else {
// 			tracing::debug!(
// 				target: "code-provider",
// 				block = ?hash,
// 				"Neither WASM override nor substitute available, using onchain code",
// 			);
// 			(onchain_code, on_chain_version)
// 		};

// 		Ok(code_and_version)
// 	}

// 	/// Returns the on chain runtime version.
// 	fn on_chain_runtime_version(
// 		&self,
// 		code: &RuntimeCode,
// 		state: &Backend::State,
// 	) -> sp_blockchain::Result<RuntimeVersion> {
// 		let mut overlay = OverlayedChanges::default();

// 		let mut ext = Ext::new(&mut overlay, state, None);

// 		self.executor
// 			.runtime_version(&mut ext, code)
// 			.map_err(|e| sp_blockchain::Error::VersionInvalid(e.to_string()))
// 	}
// }
