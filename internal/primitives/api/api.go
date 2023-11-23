package api

// / Extends the runtime api implementation with some common functionality.
// pub trait ApiExt<Block: BlockT> {
type APIExt interface {
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

	// 	/// Start recording all accessed trie nodes for generating proofs.
	// 	fn record_proof(&mut self);

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
}

// /// Something that provides a runtime api.
// pub trait ProvideRuntimeApi<Block: BlockT> {
type ProvideRuntimeAPI interface {
	// 	/// The concrete type that provides the api.
	// 	type Api: ApiExt<Block>;

	// /// Returns the runtime api.
	// /// The returned instance will keep track of modifications to the storage. Any successful
	// /// call to an api function, will `commit` its changes to an internal buffer. Otherwise,
	// /// the modifications will be `discarded`. The modifications will not be applied to the
	// /// storage, even on a `commit`.
	// fn runtime_api(&self) -> ApiRef<Self::Api>;
	RuntimeAPI() APIExt
}
