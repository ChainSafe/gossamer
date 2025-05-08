// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/version"
)

type RuntimeVersionOf interface {
	RuntimeVersion(
		externalities externalities.Externalities,
		runtimeCode core.RuntimeCode,
	) (version.RuntimeVersion, error)
}

// / An abstraction over Wasm code executor. Supports selecting execution backend and
// / manages runtime cache.
// pub struct WasmExecutor<H = sp_io::SubstrateHostFunctions> {
type WasmExecutor struct {
	// /// Method used to execute fallback Wasm code.
	// method: WasmExecutionMethod,
	// /// The heap allocation strategy for onchain Wasm calls.
	// default_onchain_heap_alloc_strategy: HeapAllocStrategy,
	// /// The heap allocation strategy for offchain Wasm calls.
	// default_offchain_heap_alloc_strategy: HeapAllocStrategy,
	// /// Ignore onchain heap pages value.
	// ignore_onchain_heap_pages: bool,
	// /// WASM runtime cache.
	// cache: Arc<RuntimeCache>,
	// /// The path to a directory which the executor can leverage for a file cache, e.g. put there
	// /// compiled artifacts.
	// cache_path: Option<PathBuf>,
	// /// Ignore missing function imports.
	// allow_missing_host_functions: bool,
	// phantom: PhantomData<H>,
}

// impl<H> Clone for WasmExecutor<H> {
// 	fn clone(&self) -> Self {
// 		Self {
// 			method: self.method,
// 			default_onchain_heap_alloc_strategy: self.default_onchain_heap_alloc_strategy,
// 			default_offchain_heap_alloc_strategy: self.default_offchain_heap_alloc_strategy,
// 			ignore_onchain_heap_pages: self.ignore_onchain_heap_pages,
// 			cache: self.cache.clone(),
// 			cache_path: self.cache_path.clone(),
// 			allow_missing_host_functions: self.allow_missing_host_functions,
// 			phantom: self.phantom,
// 		}
// 	}
// }

// impl Default for WasmExecutor<sp_io::SubstrateHostFunctions> {
// 	fn default() -> Self {
// 		WasmExecutorBuilder::new().build()
// 	}
// }

// impl<H> WasmExecutor<H> {
// 	/// Create new instance.
// 	///
// 	/// # Parameters
// 	///
// 	/// `method` - Method used to execute Wasm code.
// 	///
// 	/// `default_heap_pages` - Number of 64KB pages to allocate for Wasm execution. Internally this
// 	/// will be mapped as [`HeapAllocStrategy::Static`] where `default_heap_pages` represent the
// 	/// static number of heap pages to allocate. Defaults to `DEFAULT_HEAP_ALLOC_STRATEGY` if `None`
// 	/// is provided.
// 	///
// 	/// `max_runtime_instances` - The number of runtime instances to keep in memory ready for reuse.
// 	///
// 	/// `cache_path` - A path to a directory where the executor can place its files for purposes of
// 	///   caching. This may be important in cases when there are many different modules with the
// 	///   compiled execution method is used.
// 	///
// 	/// `runtime_cache_size` - The capacity of runtime cache.
// 	#[deprecated(note = "use `Self::builder` method instead of it")]
// 	pub fn new(
// 		method: WasmExecutionMethod,
// 		default_heap_pages: Option<u64>,
// 		max_runtime_instances: usize,
// 		cache_path: Option<PathBuf>,
// 		runtime_cache_size: u8,
// 	) -> Self {
// 		WasmExecutor {
// 			method,
// 			default_onchain_heap_alloc_strategy: unwrap_heap_pages(
// 				default_heap_pages.map(|h| HeapAllocStrategy::Static { extra_pages: h as _ }),
// 			),
// 			default_offchain_heap_alloc_strategy: unwrap_heap_pages(
// 				default_heap_pages.map(|h| HeapAllocStrategy::Static { extra_pages: h as _ }),
// 			),
// 			ignore_onchain_heap_pages: false,
// 			cache: Arc::new(RuntimeCache::new(
// 				max_runtime_instances,
// 				cache_path.clone(),
// 				runtime_cache_size,
// 			)),
// 			cache_path,
// 			allow_missing_host_functions: false,
// 			phantom: PhantomData,
// 		}
// 	}

// 	/// Instantiate a builder for creating an instance of `Self`.
// 	pub fn builder() -> WasmExecutorBuilder<H> {
// 		WasmExecutorBuilder::new()
// 	}

// 	/// Ignore missing function imports if set true.
// 	#[deprecated(note = "use `Self::builder` method instead of it")]
// 	pub fn allow_missing_host_functions(&mut self, allow_missing_host_functions: bool) {
// 		self.allow_missing_host_functions = allow_missing_host_functions
// 	}
// }

// impl<H> WasmExecutor<H>
// where
// 	H: HostFunctions,
// {
// 	/// Execute the given closure `f` with the latest runtime (based on `runtime_code`).
// 	///
// 	/// The closure `f` is expected to return `Err(_)` when there happened a `panic!` in native code
// 	/// while executing the runtime in Wasm. If a `panic!` occurred, the runtime is invalidated to
// 	/// prevent any poisoned state. Native runtime execution does not need to report back
// 	/// any `panic!`.
// 	///
// 	/// # Safety
// 	///
// 	/// `runtime` and `ext` are given as `AssertUnwindSafe` to the closure. As described above, the
// 	/// runtime is invalidated on any `panic!` to prevent a poisoned state. `ext` is already
// 	/// implicitly handled as unwind safe, as we store it in a global variable while executing the
// 	/// native runtime.
// 	pub fn with_instance<R, F>(
// 		&self,
// 		runtime_code: &RuntimeCode,
// 		ext: &mut dyn Externalities,
// 		heap_alloc_strategy: HeapAllocStrategy,
// 		f: F,
// 	) -> Result<R>
// 	where
// 		F: FnOnce(
// 			AssertUnwindSafe<&dyn WasmModule>,
// 			AssertUnwindSafe<&mut dyn WasmInstance>,
// 			Option<&RuntimeVersion>,
// 			AssertUnwindSafe<&mut dyn Externalities>,
// 		) -> Result<Result<R>>,
// 	{
// 		match self.cache.with_instance::<H, _, _>(
// 			runtime_code,
// 			ext,
// 			self.method,
// 			heap_alloc_strategy,
// 			self.allow_missing_host_functions,
// 			|module, instance, version, ext| {
// 				let module = AssertUnwindSafe(module);
// 				let instance = AssertUnwindSafe(instance);
// 				let ext = AssertUnwindSafe(ext);
// 				f(module, instance, version, ext)
// 			},
// 		)? {
// 			Ok(r) => r,
// 			Err(e) => Err(e),
// 		}
// 	}

// 	/// Perform a call into the given runtime.
// 	///
// 	/// The runtime is passed as a [`RuntimeBlob`]. The runtime will be instantiated with the
// 	/// parameters this `WasmExecutor` was initialized with.
// 	///
// 	/// In case of problems with during creation of the runtime or instantiation, a `Err` is
// 	/// returned. that describes the message.
// 	#[doc(hidden)] // We use this function for tests across multiple crates.
// 	pub fn uncached_call(
// 		&self,
// 		runtime_blob: RuntimeBlob,
// 		ext: &mut dyn Externalities,
// 		allow_missing_host_functions: bool,
// 		export_name: &str,
// 		call_data: &[u8],
// 	) -> std::result::Result<Vec<u8>, Error> {
// 		self.uncached_call_impl(
// 			runtime_blob,
// 			ext,
// 			allow_missing_host_functions,
// 			export_name,
// 			call_data,
// 			&mut None,
// 		)
// 	}

// 	/// Same as `uncached_call`, except it also returns allocation statistics.
// 	#[doc(hidden)] // We use this function in tests.
// 	pub fn uncached_call_with_allocation_stats(
// 		&self,
// 		runtime_blob: RuntimeBlob,
// 		ext: &mut dyn Externalities,
// 		allow_missing_host_functions: bool,
// 		export_name: &str,
// 		call_data: &[u8],
// 	) -> (std::result::Result<Vec<u8>, Error>, Option<AllocationStats>) {
// 		let mut allocation_stats = None;
// 		let result = self.uncached_call_impl(
// 			runtime_blob,
// 			ext,
// 			allow_missing_host_functions,
// 			export_name,
// 			call_data,
// 			&mut allocation_stats,
// 		);
// 		(result, allocation_stats)
// 	}

// 	fn uncached_call_impl(
// 		&self,
// 		runtime_blob: RuntimeBlob,
// 		ext: &mut dyn Externalities,
// 		allow_missing_host_functions: bool,
// 		export_name: &str,
// 		call_data: &[u8],
// 		allocation_stats_out: &mut Option<AllocationStats>,
// 	) -> std::result::Result<Vec<u8>, Error> {
// 		let module = crate::wasm_runtime::create_wasm_runtime_with_code::<H>(
// 			self.method,
// 			self.default_onchain_heap_alloc_strategy,
// 			runtime_blob,
// 			allow_missing_host_functions,
// 			self.cache_path.as_deref(),
// 		)
// 		.map_err(|e| format!("Failed to create module: {}", e))?;

// 		let instance =
// 			module.new_instance().map_err(|e| format!("Failed to create instance: {}", e))?;

// 		let mut instance = AssertUnwindSafe(instance);
// 		let mut ext = AssertUnwindSafe(ext);
// 		let mut allocation_stats_out = AssertUnwindSafe(allocation_stats_out);

// 		with_externalities_safe(&mut **ext, move || {
// 			let (result, allocation_stats) =
// 				instance.call_with_allocation_stats(export_name.into(), call_data);
// 			**allocation_stats_out = allocation_stats;
// 			result
// 		})
// 		.and_then(|r| r)
// 	}
// }

// impl<H> sp_core::traits::ReadRuntimeVersion for WasmExecutor<H>
// where
// 	H: HostFunctions,
// {
// 	fn read_runtime_version(
// 		&self,
// 		wasm_code: &[u8],
// 		ext: &mut dyn Externalities,
// 	) -> std::result::Result<Vec<u8>, String> {
// 		let runtime_blob = RuntimeBlob::uncompress_if_needed(wasm_code)
// 			.map_err(|e| format!("Failed to create runtime blob: {:?}", e))?;

// 		if let Some(version) = crate::wasm_runtime::read_embedded_version(&runtime_blob)
// 			.map_err(|e| format!("Failed to read the static section: {:?}", e))
// 			.map(|v| v.map(|v| v.encode()))?
// 		{
// 			return Ok(version)
// 		}

// 		// If the blob didn't have embedded runtime version section, we fallback to the legacy
// 		// way of fetching the version: i.e. instantiating the given instance and calling
// 		// `Core_version` on it.

// 		self.uncached_call(
// 			runtime_blob,
// 			ext,
// 			// If a runtime upgrade introduces new host functions that are not provided by
// 			// the node, we should not fail at instantiation. Otherwise nodes that are
// 			// updated could run this successfully and it could lead to a storage root
// 			// mismatch when importing this block.
// 			true,
// 			"Core_version",
// 			&[],
// 		)
// 		.map_err(|e| e.to_string())
// 	}
// }

// impl<H> CodeExecutor for WasmExecutor<H>
// where
// 	H: HostFunctions,
// {
// 	type Error = Error;

// 	fn call(
// 		&self,
// 		ext: &mut dyn Externalities,
// 		runtime_code: &RuntimeCode,
// 		method: &str,
// 		data: &[u8],
// 		context: CallContext,
// 	) -> (Result<Vec<u8>>, bool) {
// 		tracing::trace!(
// 			target: "executor",
// 			%method,
// 			"Executing function",
// 		);

// 		let on_chain_heap_alloc_strategy = if self.ignore_onchain_heap_pages {
// 			self.default_onchain_heap_alloc_strategy
// 		} else {
// 			runtime_code
// 				.heap_pages
// 				.map(|h| HeapAllocStrategy::Static { extra_pages: h as _ })
// 				.unwrap_or_else(|| self.default_onchain_heap_alloc_strategy)
// 		};

// 		let heap_alloc_strategy = match context {
// 			CallContext::Offchain => self.default_offchain_heap_alloc_strategy,
// 			CallContext::Onchain => on_chain_heap_alloc_strategy,
// 		};

// 		let result = self.with_instance(
// 			runtime_code,
// 			ext,
// 			heap_alloc_strategy,
// 			|_, mut instance, _on_chain_version, mut ext| {
// 				with_externalities_safe(&mut **ext, move || instance.call_export(method, data))
// 			},
// 		);

// 		(result, false)
// 	}
// }

// impl<H> RuntimeVersionOf for WasmExecutor<H>
// where
// 	H: HostFunctions,
// {
// 	fn runtime_version(
// 		&self,
// 		ext: &mut dyn Externalities,
// 		runtime_code: &RuntimeCode,
// 	) -> Result<RuntimeVersion> {
// 		let on_chain_heap_pages = if self.ignore_onchain_heap_pages {
// 			self.default_onchain_heap_alloc_strategy
// 		} else {
// 			runtime_code
// 				.heap_pages
// 				.map(|h| HeapAllocStrategy::Static { extra_pages: h as _ })
// 				.unwrap_or_else(|| self.default_onchain_heap_alloc_strategy)
// 		};

//			self.with_instance(
//				runtime_code,
//				ext,
//				on_chain_heap_pages,
//				|_module, _instance, version, _ext| {
//					Ok(version.cloned().ok_or_else(|| Error::ApiError("Unknown version".into())))
//				},
//			)
//		}
//	}
func (e WasmExecutor) RuntimeVersion(
	externalities externalities.Externalities,
	runtimeCode core.RuntimeCode,
) (version.RuntimeVersion, error) {
	panic("unimpl")
}
