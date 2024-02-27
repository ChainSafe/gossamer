package api

import (
	executionextensions "github.com/ChainSafe/gossamer/internal/client/api/execution-extensions"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
)

// / Executor Provider
// pub trait ExecutorProvider<Block: BlockT> {
type ExecutorProvider[H runtime.Hash, N runtime.Number] interface {
	// 	/// executor instance
	// 	type Executor: CallExecutor<Block>;

	// 	/// Get call executor reference.
	// 	fn executor(&self) -> &Self::Executor;
	Executor() CallExecutor[H, N]

	// /// Get a reference to the execution extensions.
	// fn execution_extensions(&self) -> &ExecutionExtensions<Block>;
	ExecutionExtensions() executionextensions.ExecutionExtensions
}

// /// Method call executor.
// pub trait CallExecutor<B: BlockT>: RuntimeVersionOf {
type CallExecutor[H runtime.Hash, N runtime.Number] interface {
	executor.RuntimeVersionOf
	// 	/// Externalities error type.
	// 	type Error: sp_state_machine::Error;

	// 	/// The backend used by the node.
	// 	type Backend: crate::backend::Backend<B>;

	// 	/// Returns the [`ExecutionExtensions`].
	// 	fn execution_extensions(&self) -> &ExecutionExtensions<B>;

	// 	/// Execute a call to a contract on top of state in a block of given hash.
	// 	///
	// 	/// No changes are made.
	// 	fn call(
	// 		&self,
	// 		at_hash: B::Hash,
	// 		method: &str,
	// 		call_data: &[u8],
	// 		strategy: ExecutionStrategy,
	// 		context: CallContext,
	// 	) -> Result<Vec<u8>, sp_blockchain::Error>;

	// 	/// Execute a contextual call on top of state in a block of a given hash.
	// 	///
	// 	/// No changes are made.
	// 	/// Before executing the method, passed header is installed as the current header
	// 	/// of the execution context.
	// 	fn contextual_call(
	// 		&self,
	// 		at_hash: B::Hash,
	// 		method: &str,
	// 		call_data: &[u8],
	// 		changes: &RefCell<OverlayedChanges>,
	// 		storage_transaction_cache: Option<
	// 			&RefCell<
	// 				StorageTransactionCache<B, <Self::Backend as crate::backend::Backend<B>>::State>,
	// 			>,
	// 		>,
	// 		proof_recorder: &Option<ProofRecorder<B>>,
	// 		context: ExecutionContext,
	// 	) -> sp_blockchain::Result<Vec<u8>>;
	ContextualCall(
		atHash H,
		method string,
		callData []byte,
		changes statemachine.OverlayedChanges,
		proofRecorder *api.ProofRecorder[H],
		callContext core.CallContext,
		// extensions: &RefCell<Extensions>,
	) ([]byte, error)

	// 	/// Extract RuntimeVersion of given block
	// 	///
	// 	/// No changes are made.
	// 	fn runtime_version(&self, at_hash: B::Hash) -> Result<RuntimeVersion, sp_blockchain::Error>;

	// /// Prove the execution of the given `method`.
	// ///
	// /// No changes are made.
	// fn prove_execution(
	//
	//	&self,
	//	at_hash: B::Hash,
	//	method: &str,
	//	call_data: &[u8],
	//
	// ) -> Result<(Vec<u8>, StorageProof), sp_blockchain::Error>;
}
