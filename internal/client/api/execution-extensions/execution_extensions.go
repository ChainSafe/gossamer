package executionextensions

// / A producer of execution extensions for offchain calls.
// /
// / This crate aggregates extensions available for the offchain calls
// / and is responsible for producing a correct `Extensions` object.
// / for each call, based on required `Capabilities`.
// pub struct ExecutionExtensions<Block: BlockT> {
type ExecutionExtensions struct {
	// strategies: ExecutionStrategies,
	// keystore: Option<KeystorePtr>,
	// offchain_db: Option<Box<dyn DbExternalitiesFactory>>,
	// // FIXME: these three are only RwLock because of https://github.com/paritytech/substrate/issues/4587
	// //        remove when fixed.
	// // To break retain cycle between `Client` and `TransactionPool` we require this
	// // extension to be a `Weak` reference.
	// // That's also the reason why it's being registered lazily instead of
	// // during initialization.
	// transaction_pool: RwLock<Option<Weak<dyn OffchainSubmitTransaction<Block>>>>,
	// extensions_factory: RwLock<Box<dyn ExtensionsFactory<Block>>>,
	// statement_store: RwLock<Option<Weak<dyn sp_statement_store::StatementStore>>>,
	// read_runtime_version: Arc<dyn ReadRuntimeVersion>,
}
