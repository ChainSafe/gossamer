package client

import (
	"fmt"
	"log"
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/api"
	blockbuilder "github.com/ChainSafe/gossamer/internal/client/block-builder"
	"github.com/ChainSafe/gossamer/internal/client/chain-spec/genesis"
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	papi "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
)

// / Substrate Client
// pub struct Client<B, E, Block, RA>
// where
//
//	Block: BlockT,
//
// {
type Client[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], RA papi.ConstructRuntimeAPI[H, N, T], T statemachine.Transaction] struct {
	// backend: Arc<B>,
	backend api.Backend[H, N, T]
	// executor: E,
	executor api.CallExecutor[H, N]
	// storage_notifications: StorageNotifications<Block>,
	// import_notification_sinks: NotificationSinks<BlockImportNotification<Block>>,
	// every_import_notification_sinks: NotificationSinks<BlockImportNotification<Block>>,
	// finality_notification_sinks: NotificationSinks<FinalityNotification<Block>>,
	finalityNotificationSinks      []chan api.FinalityNotification[H, N]
	finalityNotificationSinksMutex sync.Mutex
	// // Collects auxiliary operations to be performed atomically together with
	// // block import operations.
	// import_actions: Mutex<Vec<OnImportAction<Block>>>,
	importActions      []api.OnImportAction[H, N]
	importActionsMutex sync.Mutex
	// // Collects auxiliary operations to be performed atomically together with
	// // block finalization operations.
	// finality_actions: Mutex<Vec<OnFinalityAction<Block>>>,
	finalityActions      []api.OnFinalityAction[H, N]
	finalityActionsMutex sync.Mutex
	// // Holds the block hash currently being imported. TODO: replace this with block queue.
	// importing_block: RwLock<Option<Block::Hash>>,
	importingBlock        *H
	importingBlockRWMutex sync.RWMutex
	// block_rules: BlockRules<Block>,
	// config: ClientConfig<Block>,
	// telemetry: Option<TelemetryHandle>,
	// unpin_worker_sender: TracingUnboundedSender<Block::Hash>,
	// _phantom: PhantomData<RA>,
}

func NewClient[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], RA papi.ConstructRuntimeAPI[H, N, T], T statemachine.Transaction](
	// backend: Arc<B>,
	backend api.Backend[H, N, T],
	// 	executor: E,
	executor api.CallExecutor[H, N],
	// 	spawn_handle: Box<dyn SpawnNamed>,
	// genesis_block_builder: G,
	genesisBlockBuilder genesis.BuildGenesisBlock[H, N, api.BlockImportOperation[N, H, T]],
	// 	fork_blocks: ForkBlocks<Block>,
	// 	bad_blocks: BadBlocks<Block>,
	// 	prometheus_registry: Option<Registry>,
	// 	telemetry: Option<TelemetryHandle>,
	// 	config: ClientConfig<Block>,
) (Client[H, N, Hasher, RA, T], error) {
	info := backend.Blockchain().Info()
	if info.FinalizedState == nil {
		genesisBlock, op, err := genesisBlockBuilder.BuildGenesisBlock()
		if err != nil {
			return Client[H, N, Hasher, RA, T]{}, err
		}
		log.Printf("🔨 Initializing Genesis block/state (state: %v, header-hash: %v)",
			genesisBlock.Header().StateRoot(), genesisBlock.Header().Hash())
		// Genesis may be written after some blocks have been imported and finalized.
		// So we only finalize it when the database is empty.
		var blockState api.NewBlockState
		if info.BestHash == *new(H) {
			blockState = api.NewBlockStateFinal
		} else {
			blockState = api.NewBlockStateNormal
		}
		header, body := genesisBlock.Deconstruct()
		err = op.SetBlockData(header, body, nil, nil, blockState)
		if err != nil {
			return Client[H, N, Hasher, RA, T]{}, err
		}
		err = backend.CommitOperation(op)
		if err != nil {
			return Client[H, N, Hasher, RA, T]{}, err
		}
	}

	// let (unpin_worker_sender, mut rx) =
	// 	tracing_unbounded::<Block::Hash>("unpin-worker-channel", 10_000);
	// let task_backend = Arc::downgrade(&backend);
	// spawn_handle.spawn(
	// 	"unpin-worker",
	// 	None,
	// 	async move {
	// 		while let Some(message) = rx.next().await {
	// 			if let Some(backend) = task_backend.upgrade() {
	// 				backend.unpin_block(message);
	// 			} else {
	// 				log::debug!("Terminating unpin-worker, backend reference was dropped.");
	// 				return
	// 			}
	// 		}
	// 		log::debug!("Terminating unpin-worker, stream terminated.")
	// 	}
	// 	.boxed(),
	// );

	// Ok(Client {
	// 	backend,
	// 	executor,
	// 	storage_notifications: StorageNotifications::new(prometheus_registry),
	// 	import_notification_sinks: Default::default(),
	// 	every_import_notification_sinks: Default::default(),
	// 	finality_notification_sinks: Default::default(),
	// 	import_actions: Default::default(),
	// 	finality_actions: Default::default(),
	// 	importing_block: Default::default(),
	// 	block_rules: BlockRules::new(fork_blocks, bad_blocks),
	// 	config,
	// 	telemetry,
	// 	unpin_worker_sender,
	// 	_phantom: Default::default(),
	// })
	return Client[H, N, Hasher, RA, T]{
		backend: backend,
	}, nil
}

// impl<B, E, Block, RA> LockImportRun<Block, B> for Client<B, E, Block, RA>
// where
//
//	B: backend::Backend<Block>,
//	E: CallExecutor<Block>,
//	Block: BlockT,
//
//	{
//		fn lock_import_and_run<R, Err, F>(&self, f: F) -> Result<R, Err>
//		where
//			F: FnOnce(&mut ClientImportOperation<Block, B>) -> Result<R, Err>,
//			Err: From<sp_blockchain::Error>,
//		{
func (c *Client[H, N, Hasher, RA, T]) LockImportAndRun(f func(*api.ClientImportOperation[N, H, T]) error) error {
	var inner = func() error {
		mtx := c.backend.GetImportLock()
		mtx.Lock()
		defer mtx.Unlock()

		blockImportOp, err := c.backend.BeginOperation()
		if err != nil {
			return err
		}

		clientImportOp := api.ClientImportOperation[N, H, T]{
			Op:               blockImportOp,
			NotifyImported:   nil,
			NotifiyFinalized: nil,
		}

		err = f(&clientImportOp)
		if err != nil {
			return err
		}

		var finalityNotification *api.FinalityNotification[H, N]
		if clientImportOp.NotifiyFinalized != nil {
			summary := *clientImportOp.NotifiyFinalized
			notification := api.NewFinalityNotificationFromSummary[H, N](summary)
			finalityNotification = &notification
		}

		var importNotification *api.BlockImportNotification[H, N]
		var storageChanges *struct {
			statemachine.StorageCollection
			statemachine.ChildStorageCollection
		}
		var importNotificationAction api.ImportNotificationAction

		if clientImportOp.NotifyImported != nil {
			summary := clientImportOp.NotifyImported
			importNotificationAction = summary.ImportNotificationAction
			storageChanges = summary.StorageChanges
			summary.StorageChanges = nil
			blockImportNotification := api.NewBlockImportNotificationFromSummary(*summary)
			importNotification = &blockImportNotification

		} else {
			importNotification = nil
			storageChanges = nil
			importNotificationAction = api.ImportNotificationActionNone
		}

		if finalityNotification != nil {
			c.finalityActionsMutex.Lock()
			for _, action := range c.finalityActions {
				err := clientImportOp.Op.InsertAux(action(*finalityNotification))
				if err != nil {
					c.finalityActionsMutex.Unlock()
					return err
				}
			}
			c.finalityActionsMutex.Unlock()
		}
		if importNotification != nil {
			c.importActionsMutex.Lock()
			for _, action := range c.importActions {
				err := clientImportOp.Op.InsertAux(action(*importNotification))
				if err != nil {
					c.importActionsMutex.Unlock()
					return err
				}
			}
			c.importActionsMutex.Unlock()
		}

		err = c.backend.CommitOperation(clientImportOp.Op)
		if err != nil {
			return err
		}

		// We need to pin the block in the backend once
		// // for each notification. Once all notifications are
		// // dropped, the block will be unpinned automatically.
		// if let Some(ref notification) = finality_notification {
		// 	if let Err(err) = self.backend.pin_block(notification.hash) {
		// 		error!(
		// 			"Unable to pin block for finality notification. hash: {}, Error: {}",
		// 			notification.hash, err
		// 		);
		// 	};
		// }

		// if let Some(ref notification) = import_notification {
		// 	if let Err(err) = self.backend.pin_block(notification.hash) {
		// 		error!(
		// 			"Unable to pin block for import notification. hash: {}, Error: {}",
		// 			notification.hash, err
		// 		);
		// 	};
		// }

		err = c.notifyFinalized(finalityNotification)
		if err != nil {
			return err
		}
		err = c.notifyImported(importNotification, importNotificationAction, storageChanges)
		if err != nil {
			return err
		}

		return nil
	}

	err := inner()
	c.importingBlockRWMutex.Lock()
	defer c.importingBlockRWMutex.Unlock()
	c.importingBlock = nil
	return err
}

// / Returns true if state for given block is available.
//
//	fn have_state_at(&self, hash: Block::Hash, _number: NumberFor<Block>) -> bool {
//		self.state_at(hash).is_ok()
//	}
func (c *Client[H, N, Hasher, RA, T]) HaveStateAt(hash H, number N) bool {
	_, err := c.StateAt(hash)
	if err != nil {
		return false
	}
	return true
}

// / Get a reference to the state at a given block.
//
//	pub fn state_at(&self, hash: Block::Hash) -> sp_blockchain::Result<B::State> {
//		self.backend.state_at(hash)
//	}
func (c *Client[H, N, Hasher, RA, T]) StateAt(hash H) (statemachine.Backend[H, T], error) {
	return c.backend.StateAt(hash)
}

// / Get blockchain info.
func (c *Client[H, N, Hasher, RA, T]) ChainInfo() blockchain.Info[H, N] {
	return c.backend.Blockchain().Info()
}

/// Get block status.
// pub fn block_status(&self, hash: Block::Hash) -> sp_blockchain::Result<BlockStatus> {
// 	// this can probably be implemented more efficiently
// 	if self
// 		.importing_block
// 		.read()
// 		.as_ref()
// 		.map_or(false, |importing| &hash == importing)
// 	{
// 		return Ok(BlockStatus::Queued)
// 	}

//		let hash_and_number = self.backend.blockchain().number(hash)?.map(|n| (hash, n));
//		match hash_and_number {
//			Some((hash, number)) =>
//				if self.backend.have_state_at(hash, number) {
//					Ok(BlockStatus::InChainWithState)
//				} else {
//					Ok(BlockStatus::InChainPruned)
//				},
//			None => Ok(BlockStatus::Unknown),
//		}
//	}
func (c *Client[H, N, Hasher, RA, T]) BlockStatus(hash H) (consensus.BlockStatus, error) {
	c.importingBlockRWMutex.RLock()
	defer c.importingBlockRWMutex.RUnlock()
	if c.importingBlock != nil {
		if hash == *c.importingBlock {
			return consensus.BlockStatusQueued, nil
		}
	}
	number, err := c.backend.Blockchain().Number(hash)
	if err != nil {
		return 0, err
	}
	if number == nil {
		return consensus.BlockStatusUnknown, nil
	}
	if c.backend.HaveStateAt(hash, *number) {
		return consensus.BlockStatusInChainWithState, nil
	} else {
		return consensus.BlockStatusInChainPruned, nil
	}
}

func (c *Client[H, N, Hasher, RA, T]) NewBlock(inherentDigests runtime.Digest) (blockbuilder.BlockBuilder[H, N, T], error) {
	info := c.ChainInfo()
	return blockbuilder.NewBlockBuilder[H, N, Hasher](
		c, info.BestHash, info.BestNumber, false, inherentDigests, c.backend,
	)
}

// impl<B, E, Block, RA> ProvideRuntimeApi<Block> for Client<B, E, Block, RA>
// where
// 	B: backend::Backend<Block>,
// 	E: CallExecutor<Block, Backend = B> + Send + Sync,
// 	Block: BlockT,
// 	RA: ConstructRuntimeApi<Block, Self> + Send + Sync,
// {
// 	type Api = <RA as ConstructRuntimeApi<Block, Self>>::RuntimeApi;

//		fn runtime_api(&self) -> ApiRef<Self::Api> {
//			RA::construct_runtime_api(self)
//		}
//	}
func (c *Client[H, N, Hasher, RA, T]) RuntimeAPI() papi.APIExt[H, N, T] {
	// RA::construct_runtime_api(self)
	ra := *(new(RA))
	return ra.ConstructRuntimeAPI(c)
}

// impl<B, E, Block, RA> CallApiAt<Block> for Client<B, E, Block, RA>
// where
// 	B: backend::Backend<Block>,
// 	E: CallExecutor<Block, Backend = B> + Send + Sync,
// 	Block: BlockT,
// 	RA: Send + Sync,
// {
// 	type StateBackend = B::State;

// 	fn call_api_at(
// 		&self,
// 		params: CallApiAtParams<Block, B::State>,
// 	) -> Result<Vec<u8>, sp_api::ApiError> {
// 		self.executor
// 			.contextual_call(
// 				params.at,
// 				params.function,
// 				&params.arguments,
// 				params.overlayed_changes,
// 				Some(params.storage_transaction_cache),
// 				params.recorder,
// 				params.context,
// 			)
// 			.map_err(Into::into)
// 	}

// 	fn runtime_version_at(&self, hash: Block::Hash) -> Result<RuntimeVersion, sp_api::ApiError> {
// 		CallExecutor::runtime_version(&self.executor, hash).map_err(Into::into)
// 	}

//		fn state_at(&self, at: Block::Hash) -> Result<Self::StateBackend, sp_api::ApiError> {
//			self.state_at(at).map_err(Into::into)
//		}
//	}
func (c *Client[H, N, Hasher, RA, T]) CallAPIAt(params papi.CallAPIAtParams[H, N]) ([]byte, error) {
	return c.executor.ContextualCall(
		params.At,
		params.Function,
		params.Arguments,
		params.OverlayedChanges,
		&params.StorageTransactionCache,
		params.Recorder,
	)
}

//	enum PrepareStorageChangesResult<B: backend::Backend<Block>, Block: BlockT> {
//		Discard(ImportResult),
//		Import(Option<sc_consensus::StorageChanges<Block, backend::TransactionFor<B, Block>>>),
//	}
type prepareStorageChangesResult any
type prepareStorageChangesResults interface {
	prepareStorageChangesResultDiscard | prepareStorageChangesResultImport
}
type prepareStorageChangesResultDiscard consensus.ImportResult
type prepareStorageChangesResultImport *consensus.StorageChanges

// / Prepares the storage changes for a block.
// ///
// /// It checks if the state should be enacted and if the `import_block` maybe already provides
// /// the required storage changes. If the state should be enacted and the storage changes are not
// /// provided, the block is re-executed to get the storage changes.
// fn prepare_block_storage_changes(
//
//	&self,
//	import_block: &mut BlockImportParams<Block, backend::TransactionFor<B, Block>>,
//
// ) -> sp_blockchain::Result<PrepareStorageChangesResult<B, Block>>
// where
//
//	Self: ProvideRuntimeApi<Block>,
//	<Self as ProvideRuntimeApi<Block>>::Api:
//		CoreApi<Block> + ApiExt<Block, StateBackend = B::State>,
//
// {
func (c *Client[H, N, Hasher, RA, T]) prepareBlockStorageChanges(importBlock consensus.BlockImportParams[H, N]) (prepareStorageChangesResult, error) {
	parentHash := importBlock.Header.ParentHash()
	stateAction := importBlock.StateAction
	importBlock.StateAction = consensus.StateActionSkip{}

	fmt.Println(parentHash, stateAction)
	blockStatus, err := c.BlockStatus(parentHash)
	if err != nil {
		return nil, err
	}

	var enactState bool
	var storageChanges *consensus.StorageChanges

	changes, isStateActionApplyChanges := stateAction.(consensus.StateActionApplyChanges)
	_, isStateActionSkip := stateAction.(consensus.StateActionSkip)

	switch {
	case blockStatus == consensus.BlockStatusKnownBad:
		return prepareStorageChangesResultDiscard(consensus.ImportResultKnownBad{}), nil
	case blockStatus == consensus.BlockStatusInChainPruned && isStateActionApplyChanges:
		return prepareStorageChangesResultDiscard(consensus.ImportResultMissingState{}), nil
	case isStateActionApplyChanges:
		enactState = true
		changes := consensus.StorageChanges(changes)
		storageChanges = &changes
	case blockStatus == consensus.BlockStatusUnknown:
		return prepareStorageChangesResultDiscard(consensus.ImportResultUnknownParent{}), nil
	case isStateActionSkip:
		enactState = false
		storageChanges = nil
	case blockStatus == consensus.BlockStatusInChainPruned:
		switch stateAction.(type) {
		case consensus.StateActionExecute:
			return prepareStorageChangesResultDiscard(consensus.ImportResultMissingState{}), nil
		case consensus.StateActionExecuteIfPossible:
			enactState = false
			storageChanges = nil
		default:
			panic("huh?")
		}
	default:
		switch stateAction.(type) {
		case consensus.StateActionExecute:
			enactState = true
			storageChanges = nil
		case consensus.StateActionExecuteIfPossible:
			enactState = true
			storageChanges = nil
		default:
			panic("huh?")
		}
	}

	var returnedStorageChanges *consensus.StorageChanges
	switch {
	case enactState && storageChanges != nil:
		returnedStorageChanges = storageChanges
	case enactState && storageChanges == nil && importBlock.Body != nil:
		runtimeAPI := c.RuntimeAPI()
		// executionContext := importBlock.Origin
		block := generic.NewBlock[N, H, Hasher](importBlock.Header, *importBlock.Body)
		err := runtimeAPI.ExecuteBlock(parentHash, block)
		if err != nil {
			return nil, err
		}

		state, err := c.backend.StateAt(parentHash)
		if err != nil {
			return nil, err
		}
		genStorageChanges, err := runtimeAPI.IntoStorageChanges(state, parentHash)
		if err != nil {
			return nil, err
		}
		if importBlock.Header.StateRoot() != genStorageChanges.TransactionStorageRoot {
			// return Err(Error::InvalidStateRoot)
			return nil, fmt.Errorf("invalid state root")
		}
		sc := consensus.StorageChanges(consensus.StorageChangesChanges[T, H](genStorageChanges))
		returnedStorageChanges = &sc
	case enactState && storageChanges == nil && importBlock.Body == nil:
		returnedStorageChanges = nil
	case !enactState:
		returnedStorageChanges = nil
	default:
		panic("wtf?")
	}

	return prepareStorageChangesResultImport(returnedStorageChanges), nil
}

func (c *Client[H, N, Hasher, RA, T]) notifyFinalized(notification *api.FinalityNotification[H, N]) error {
	c.finalityNotificationSinksMutex.Lock()
	defer c.finalityNotificationSinksMutex.Unlock()

	if notification != nil {
		// // Cleanup any closed finality notification sinks
		// // since we won't be running the loop below which
		// // would also remove any closed sinks.
		// sinks.retain(|sink| !sink.is_closed());
		// return Ok(())
	}
	// telemetry!(
	// 	self.telemetry;
	// 	SUBSTRATE_INFO;
	// 	"notify.finalized";
	// 	"height" => format!("{}", notification.header.number()),
	// 	"best" => ?notification.hash,
	// );
	for _, ch := range c.finalityNotificationSinks {
		ch <- *notification
	}
	return nil
}

func (c *Client[H, N, Hasher, RA, T]) notifyImported(
	notification *api.BlockImportNotification[H, N],
	importNotificationAction api.ImportNotificationAction,
	storageChanges *struct {
		statemachine.StorageCollection
		statemachine.ChildStorageCollection
	},
) error {
	return nil
}

/// NOTE: only use this implementation when you are sure there are NO consensus-level BlockImport
/// objects. Otherwise, importing blocks directly into the client would be bypassing
/// important verification work.
// #[async_trait::async_trait]
// impl<B, E, Block, RA> sc_consensus::BlockImport<Block> for &Client<B, E, Block, RA>
// where
// 	B: backend::Backend<Block>,
// 	E: CallExecutor<Block> + Send + Sync,
// 	Block: BlockT,
// 	Client<B, E, Block, RA>: ProvideRuntimeApi<Block>,
// 	<Client<B, E, Block, RA> as ProvideRuntimeApi<Block>>::Api:
// 		CoreApi<Block> + ApiExt<Block, StateBackend = B::State>,
// 	RA: Sync + Send,
// 	backend::TransactionFor<B, Block>: Send + 'static,
// {
// 	type Error = ConsensusError;
// 	type Transaction = backend::TransactionFor<B, Block>;

// /// Import a checked and validated block. If a justification is provided in
// /// `BlockImportParams` then `finalized` *must* be true.
// ///
// /// NOTE: only use this implementation when there are NO consensus-level BlockImport
// /// objects. Otherwise, importing blocks directly into the client would be bypassing
// /// important verification work.
// ///
// /// If you are not sure that there are no BlockImport objects provided by the consensus
// /// algorithm, don't use this function.
// async fn import_block(
//
//	&mut self,
//	mut import_block: BlockImportParams<Block, backend::TransactionFor<B, Block>>,
//
// ) -> Result<ImportResult, Self::Error> {
func (c *Client[H, N, Hasher, RA, T]) ImportBlock(importBlock consensus.BlockImportParams[H, N]) chan<- struct {
	consensus.ImportResult
	Error error
} {
	ch := make(chan<- struct {
		consensus.ImportResult
		Error error
	})
	type item struct {
		consensus.ImportResult
		Error error
	}
	go func() {

		// 	let span = tracing::span!(tracing::Level::DEBUG, "import_block");
		// 	let _enter = span.enter();

		// 		let storage_changes =
		// 			match self.prepare_block_storage_changes(&mut import_block).map_err(|e| {
		// 				warn!("Block prepare storage changes error: {}", e);
		// 				ConsensusError::ClientImport(e.to_string())
		// 			})? {
		// 				PrepareStorageChangesResult::Discard(res) => return Ok(res),
		// 				PrepareStorageChangesResult::Import(storage_changes) => storage_changes,
		// 			};
		res, err := c.prepareBlockStorageChanges(importBlock)
		if err != nil {
			ch <- item{
				Error: err,
			}
			return
		}
		var storageChanges *consensus.StorageChanges
		switch res := res.(type) {
		case prepareStorageChangesResultDiscard:
			ch <- item{
				ImportResult: res,
			}
			return
		case prepareStorageChangesResultImport:
			storageChanges = res
		}
		//		self.lock_import_and_run(|operation| {
		//			self.apply_block(operation, import_block, storage_changes)
		//		})
		//		.map_err(|e| {
		//			warn!("Block import error: {}", e);
		//			ConsensusError::ClientImport(e.to_string())
		//		})
		//	}
		c.Lock
	}()

	return ch
}

// 	/// Check block preconditions.
// 	async fn check_block(
// 		&mut self,
// 		block: BlockCheckParams<Block>,
// 	) -> Result<ImportResult, Self::Error> {
// 		let BlockCheckParams {
// 			hash,
// 			number,
// 			parent_hash,
// 			allow_missing_state,
// 			import_existing,
// 			allow_missing_parent,
// 		} = block;

// 		// Check the block against white and black lists if any are defined
// 		// (i.e. fork blocks and bad blocks respectively)
// 		match self.block_rules.lookup(number, &hash) {
// 			BlockLookupResult::KnownBad => {
// 				trace!("Rejecting known bad block: #{} {:?}", number, hash);
// 				return Ok(ImportResult::KnownBad)
// 			},
// 			BlockLookupResult::Expected(expected_hash) => {
// 				trace!(
// 					"Rejecting block from known invalid fork. Got {:?}, expected: {:?} at height {}",
// 					hash,
// 					expected_hash,
// 					number
// 				);
// 				return Ok(ImportResult::KnownBad)
// 			},
// 			BlockLookupResult::NotSpecial => {},
// 		}

// 		// Own status must be checked first. If the block and ancestry is pruned
// 		// this function must return `AlreadyInChain` rather than `MissingState`
// 		match self
// 			.block_status(hash)
// 			.map_err(|e| ConsensusError::ClientImport(e.to_string()))?
// 		{
// 			BlockStatus::InChainWithState | BlockStatus::Queued =>
// 				return Ok(ImportResult::AlreadyInChain),
// 			BlockStatus::InChainPruned if !import_existing =>
// 				return Ok(ImportResult::AlreadyInChain),
// 			BlockStatus::InChainPruned => {},
// 			BlockStatus::Unknown => {},
// 			BlockStatus::KnownBad => return Ok(ImportResult::KnownBad),
// 		}

// 		match self
// 			.block_status(parent_hash)
// 			.map_err(|e| ConsensusError::ClientImport(e.to_string()))?
// 		{
// 			BlockStatus::InChainWithState | BlockStatus::Queued => {},
// 			BlockStatus::Unknown if allow_missing_parent => {},
// 			BlockStatus::Unknown => return Ok(ImportResult::UnknownParent),
// 			BlockStatus::InChainPruned if allow_missing_state => {},
// 			BlockStatus::InChainPruned => return Ok(ImportResult::MissingState),
// 			BlockStatus::KnownBad => return Ok(ImportResult::KnownBad),
// 		}

// 		Ok(ImportResult::imported(false))
// 	}
// }
