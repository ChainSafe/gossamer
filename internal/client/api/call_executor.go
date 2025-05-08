package api

// / Method call executor.
// pub trait CallExecutor<B: BlockT>: RuntimeVersionOf {
type CallExecutor interface {
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
	// 		changes: &RefCell<OverlayedChanges<HashingFor<B>>>,
	// 		proof_recorder: &Option<ProofRecorder<B>>,
	// 		call_context: CallContext,
	// 		extensions: &RefCell<Extensions>,
	// 	) -> sp_blockchain::Result<Vec<u8>>;

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
