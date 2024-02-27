package api

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
)

// / A type that records all accessed trie nodes and generates a proof out of it.
// #[cfg(feature = "std")]
// pub type ProofRecorder<B> = sp_trie::recorder::Recorder<HashFor<B>>;
type ProofRecorder[H runtime.Hash] recorder.Recorder[H]

// pub type StorageChanges<SBackend, Block> = sp_state_machine::StorageChanges<
//
//	<SBackend as StateBackend<HashFor<Block>>>::Transaction,
//	HashFor<Block>,
//
// >;
type StorageChanges[H runtime.Hash, Hasher runtime.Hasher[H]] statemachine.StorageChanges[H, Hasher]

// / Something that can be constructed to a runtime api.
// #[cfg(feature = "std")]
// pub trait ConstructRuntimeApi<Block: BlockT, C: CallApiAt<Block>> {
type ConstructRuntimeAPI[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] interface {
	// 	/// The actual runtime api that will be constructed.
	// 	type RuntimeApi: ApiExt<Block>;

	// /// Construct an instance of the runtime api.
	// fn construct_runtime_api(call: &C) -> ApiRef<Self::RuntimeApi>;
	ConstructRuntimeAPI(call CallAPIAt[H, N]) APIExt[H, N, Hasher]
}

// / The `Core` runtime api that every Substrate runtime needs to implement.
// #[core_trait]
// #[api_version(4)]
// pub trait Core {
type Core[H runtime.Hash, N runtime.Number] interface {
	// NOTE: add `at` param to all methods so we can fetch the correct runtime
	// /// Returns the version of the runtime.
	// fn version() -> RuntimeVersion;
	// /// Execute the given block.
	// fn execute_block(block: Block);
	ExecuteBlock(at H, block runtime.Block[N, H]) error

	// /// Initialize a block with the given header.
	// #[renamed("initialise_block", 2)]
	// fn initialize_block(header: &<Block as BlockT>::Header);
	InitializeBlock(at H, header runtime.Header[N, H]) error
}

// / Extends the runtime api implementation with some common functionality.
// pub trait ApiExt<Block: BlockT> {
type APIExt[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] interface {
	Core[H, N]
	// 	/// The state backend that is used to store the block states.
	// 	type StateBackend: StateBackend<HashFor<Block>>;

	// 	/// Execute the given closure inside a new transaction.
	// 	///
	// 	/// Depending on the outcome of the closure, the transaction is committed or rolled-back.
	// 	///
	// 	/// The internal result of the closure is returned afterwards.
	// 	fn execute_in_transaction<F: FnOnce(&Self) -> TransactionOutcome<R>, R>(&self, call: F) -> R
	// 	where
	// 		Self: Sized;

	// 	/// Checks if the given api is implemented and versions match.
	// 	fn has_api<A: RuntimeApiInfo + ?Sized>(&self, at_hash: Block::Hash) -> Result<bool, ApiError>
	// 	where
	// 		Self: Sized;

	// 	/// Check if the given api is implemented and the version passes a predicate.
	// 	fn has_api_with<A: RuntimeApiInfo + ?Sized, P: Fn(u32) -> bool>(
	// 		&self,
	// 		at_hash: Block::Hash,
	// 		pred: P,
	// 	) -> Result<bool, ApiError>
	// 	where
	// 		Self: Sized;

	// 	/// Returns the version of the given api.
	// 	fn api_version<A: RuntimeApiInfo + ?Sized>(
	// 		&self,
	// 		at_hash: Block::Hash,
	// 	) -> Result<Option<u32>, ApiError>
	// 	where
	// 		Self: Sized;
	APIVersion(at H) (*uint32, error)

	// 	/// Start recording all accessed trie nodes for generating proofs.
	// 	fn record_proof(&mut self);
	RecordProof()

	// 	/// Extract the recorded proof.
	// 	///
	// 	/// This stops the proof recording.
	// 	///
	// 	/// If `record_proof` was not called before, this will return `None`.
	// 	fn extract_proof(&mut self) -> Option<StorageProof>;

	// 	/// Returns the current active proof recorder.
	// 	fn proof_recorder(&self) -> Option<ProofRecorder<Block>>;

	// /// Convert the api object into the storage changes that were done while executing runtime
	// /// api functions.
	// ///
	// /// After executing this function, all collected changes are reset.
	// fn into_storage_changes(
	//
	//	&self,
	//	backend: &Self::StateBackend,
	//	parent_hash: Block::Hash,
	//
	// ) -> Result<StorageChanges<Self::StateBackend, Block>, String>
	// where
	//
	//	Self: Sized;
	IntoStorageChanges(backend statemachine.Backend[H, Hasher], parentHash H) (StorageChanges[H, Hasher], error)
}

// /// Parameters for [`CallApiAt::call_api_at`].
// #[cfg(feature = "std")]
// pub struct CallApiAtParams<'a, Block: BlockT, Backend: StateBackend<HashFor<Block>>> {
type CallAPIAtParams[H runtime.Hash, N runtime.Number] struct {
	// /// The block id that determines the state that should be setup when calling the function.
	// pub at: Block::Hash,
	At H
	// /// The name of the function that should be called.
	// pub function: &'static str,
	Function string
	// /// The encoded arguments of the function.
	// pub arguments: Vec<u8>,
	Arguments []byte
	// /// The overlayed changes that are on top of the state.
	// pub overlayed_changes: &'a RefCell<OverlayedChanges>,
	OverlayedChanges statemachine.OverlayedChanges
	// /// The call context of this call.
	// pub call_context: CallContext,
	CallContext core.CallContext
	// /// The optional proof recorder for recording storage accesses.
	// pub recorder: &'a Option<ProofRecorder<Block>>,
	Recorder *ProofRecorder[H]
	// /// The extensions that should be used for this call.
	// pub extensions: &'a RefCell<Extensions>,
}

// /// Something that can call into the an api at a given block.
// #[cfg(feature = "std")]
// pub trait CallApiAt<Block: BlockT> {
type CallAPIAt[H runtime.Hash, N runtime.Number] interface {
	// 	/// The state backend that is used to store the block states.
	// 	type StateBackend: StateBackend<HashFor<Block>> + AsTrieBackend<HashFor<Block>>;

	// 	/// Calls the given api function with the given encoded arguments at the given block and returns
	// 	/// the encoded result.
	// 	fn call_api_at(
	// 		&self,
	// 		params: CallApiAtParams<Block, Self::StateBackend>,
	// 	) -> Result<Vec<u8>, ApiError>;
	CallAPIAt(params CallAPIAtParams[H, N]) ([]byte, error)

	// 	/// Returns the runtime version at the given block.
	// 	fn runtime_version_at(&self, at_hash: Block::Hash) -> Result<RuntimeVersion, ApiError>;

	// /// Get the state `at` the given block.
	// fn state_at(&self, at: Block::Hash) -> Result<Self::StateBackend, ApiError>;
}

// /// Something that provides a runtime api.
// pub trait ProvideRuntimeApi<Block: BlockT> {
type ProvideRuntimeAPI[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] interface {
	// 	/// The concrete type that provides the api.
	// 	type Api: ApiExt<Block>;

	// /// Returns the runtime api.
	// /// The returned instance will keep track of modifications to the storage. Any successful
	// /// call to an api function, will `commit` its changes to an internal buffer. Otherwise,
	// /// the modifications will be `discarded`. The modifications will not be applied to the
	// /// storage, even on a `commit`.
	// fn runtime_api(&self) -> ApiRef<Self::Api>;
	RuntimeAPI() APIExt[H, N, Hasher]
}
