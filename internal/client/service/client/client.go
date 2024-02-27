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
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// / Substrate Client
// pub struct Client<B, E, Block, RA>
// where
//
//	Block: BlockT,
//
// {
type Client[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], RA papi.ConstructRuntimeAPI[H, N, Hasher]] struct {
	// backend: Arc<B>,
	backend api.Backend[H, N, Hasher]
	// executor: E,
	executor api.CallExecutor[H, N]
	// storage_notifications: StorageNotifications<Block>,
	storageNotifications api.StorageNotifications[H]
	// import_notification_sinks: NotificationSinks<BlockImportNotification<Block>>,
	importNotificationSinks      []chan api.BlockImportNotification[H, N]
	importNotificationSinksMutex sync.Mutex
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
	// config ClientConfig
	// telemetry: Option<TelemetryHandle>,
	// unpin_worker_sender: TracingUnboundedSender<Block::Hash>,
	// _phantom: PhantomData<RA>,
}

func NewClient[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], RA papi.ConstructRuntimeAPI[H, N, Hasher]](
	// backend: Arc<B>,
	backend api.Backend[H, N, Hasher],
	// 	executor: E,
	executor api.CallExecutor[H, N],
	// 	spawn_handle: Box<dyn SpawnNamed>,
	// genesis_block_builder: G,
	genesisBlockBuilder genesis.BuildGenesisBlock[H, N, api.BlockImportOperation[N, H, Hasher]],
	// 	fork_blocks: ForkBlocks<Block>,
	// 	bad_blocks: BadBlocks<Block>,
	// 	prometheus_registry: Option<Registry>,
	// 	telemetry: Option<TelemetryHandle>,
	// 	config: ClientConfig<Block>,
) (Client[H, N, Hasher, RA], error) {
	info := backend.Blockchain().Info()
	if info.FinalizedState == nil {
		genesisBlock, op, err := genesisBlockBuilder.BuildGenesisBlock()
		if err != nil {
			return Client[H, N, Hasher, RA]{}, err
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
		err = op.SetBlockData(header, &body, nil, nil, blockState)
		if err != nil {
			return Client[H, N, Hasher, RA]{}, err
		}
		err = backend.CommitOperation(op)
		if err != nil {
			return Client[H, N, Hasher, RA]{}, err
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
	return Client[H, N, Hasher, RA]{
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
func (c *Client[H, N, Hasher, RA]) LockImportAndRun(f func(*api.ClientImportOperation[N, H, Hasher]) error) error {
	var inner = func() error {
		mtx := c.backend.GetImportLock()
		mtx.Lock()
		defer mtx.Unlock()

		blockImportOp, err := c.backend.BeginOperation()
		if err != nil {
			return err
		}

		clientImportOp := api.ClientImportOperation[N, H, Hasher]{
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

// / Used in importing a block, where additional changes are made after the runtime
// / executed.
type prePostHeader[H any] interface {
	Post() H
}

// / they are the same: no post-runtime digest items.
type prePostHeaderSame[H any] struct {
	Hash H
}

func (pphs prePostHeaderSame[H]) Post() H {
	return pphs.Hash
}

// / different headers (pre, post).
type prePostHeaderDifferent[H any] [2]H

func (pphd prePostHeaderDifferent[H]) Post() H {
	return pphd[0]
}

// impl<H> PrePostHeader<H> {
// 	/// get a reference to the "post-header" -- the header as it should be
// 	/// after all changes are applied.
// 	fn post(&self) -> &H {
// 		match *self {
// 			PrePostHeader::Same(ref h) => h,
// 			PrePostHeader::Different(_, ref h) => h,
// 		}
// 	}

// 	/// convert to the "post-header" -- the header as it should be after
// 	/// all changes are applied.
// 	fn into_post(self) -> H {
// 		match self {
// 			PrePostHeader::Same(h) => h,
// 			PrePostHeader::Different(_, h) => h,
// 		}
// 	}
// }

// / Apply a checked and validated block to an operation. If a justification is provided
// / then `finalized` *must* be true.
func (c *Client[H, N, Hasher, RA]) applyBlock(
	// &mut ClientImportOperation<Block, B>,
	operation *api.ClientImportOperation[N, H, Hasher],
	importBlock consensus.BlockImportParams[H, N],
	storageChanges *consensus.StorageChanges,
) (consensus.ImportResult, error) {
	if !(len(importBlock.Intermediates) == 0) {
		return nil, fmt.Errorf("Incomplete block import pipeline.")
	}

	if importBlock.ForkChoice == nil {
		return nil, fmt.Errorf("Incomplete block import pipeline.")
	}
	forkChoice := *importBlock.ForkChoice

	var importHeaders prePostHeader[runtime.Header[N, H]]
	if len(importBlock.PostDigests) == 0 {
		importHeaders = prePostHeaderSame[runtime.Header[N, H]]{importBlock.Header}
	} else {
		postHeader := importBlock.Header
		for _, item := range importBlock.PostDigests {
			digest := postHeader.DigestMut()
			digest.Push(item)
		}
		importHeaders = prePostHeaderDifferent[runtime.Header[N, H]]{importBlock.Header, postHeader}
	}

	hash := importHeaders.Post().Hash()
	// height := uint64(importHeaders.Post().Number())

	c.importingBlockRWMutex.Lock()
	defer c.importingBlockRWMutex.Unlock()
	c.importingBlock = &hash

	var auxDataOperations api.AuxDataOperations
	for _, auxDataOp := range importBlock.Auxiliary {
		auxDataOperations = append(auxDataOperations, api.AuxDataOperation(auxDataOp))
	}

	importResult, err := c.executeAndImportBlock(
		operation,
		importBlock.Origin,
		hash,
		importHeaders,
		importBlock.Justifications,
		importBlock.Body,
		importBlock.IndexedBody,
		storageChanges,
		importBlock.Finalized,
		auxDataOperations,
		forkChoice,
		importBlock.ImportExisting,
	)

	// if let Ok(ImportResult::Imported(ref aux)) = result {
	// 	if aux.is_new_best {
	// 		// don't send telemetry block import events during initial sync for every
	// 		// block to avoid spamming the telemetry server, these events will be randomly
	// 		// sent at a rate of 1/10.
	// 		if origin != BlockOrigin::NetworkInitialSync || rand::thread_rng().gen_bool(0.1) {
	// 			telemetry!(
	// 				self.telemetry;
	// 				SUBSTRATE_INFO;
	// 				"block.import";
	// 				"height" => height,
	// 				"best" => ?hash,
	// 				"origin" => ?origin
	// 			);
	// 		}
	// 	}
	// }

	return importResult, err
}

func (c *Client[H, N, Hasher, RA]) executeAndImportBlock(
	operation *api.ClientImportOperation[N, H, Hasher],
	origin consensus.BlockOrigin,
	hash H,
	importHeaders prePostHeader[runtime.Header[N, H]],
	justifications *runtime.Justifications,
	body *[]runtime.Extrinsic,
	indexedBody *[][]byte,
	storageChanges *consensus.StorageChanges,
	finalized bool,
	aux api.AuxDataOperations,
	forkChoice consensus.ForkChoiceStrategy,
	importExisting bool,
) (consensus.ImportResult, error) {
	parentHash := importHeaders.Post().ParentHash()
	status, err := c.backend.Blockchain().Status(hash)
	if err != nil {
		return nil, err
	}
	parentStatus, err := c.backend.Blockchain().Status(parentHash)
	if err != nil {
		return nil, err
	}
	parentExists := parentStatus == blockchain.BlockStatusInChain
	switch {
	case !importExisting && status == blockchain.BlockStatusInChain:
		return consensus.ImportResultAlreadyInChain{}, nil
	case !importExisting && status == blockchain.BlockStatusUnknown:
	case importExisting && status == blockchain.BlockStatusInChain:
	case importExisting && status == blockchain.BlockStatusUnknown:
	default:
		panic("wtf?")
	}

	info := c.backend.Blockchain().Info()
	var gapBlock bool
	if info.BlockGap != nil {
		start := info.BlockGap[0]
		gapBlock = importHeaders.Post().Number() == start
	}

	if !(justifications != nil && finalized || justifications == nil && gapBlock) {
		panic("wtf?")
	}

	// the block is lower than our last finalized block so it must revert
	// finality, refusing import.
	if status == blockchain.BlockStatusUnknown && importHeaders.Post().Number() <= info.FinalizedNumber && !gapBlock {
		return nil, fmt.Errorf("Potential long-range attack: block not in finalized chain.")
	}

	// this is a fairly arbitrary choice of where to draw the line on making notifications,
	// but the general goal is to only make notifications when we are already fully synced
	// and get a new chain head.
	var makeNotifications bool
	switch origin {
	case consensus.BlockOriginNetworkBroadcast, consensus.BlockOriginOwn, consensus.BlockOriginConsensusBroadcast:
		makeNotifications = true
	case consensus.BlockOriginGenesis, consensus.BlockOriginNetworkInitialSync, consensus.BlockOriginFile:
		makeNotifications = false
	default:
		panic("wtf?")
	}

	var storageChangesOpt *struct {
		statemachine.StorageCollection
		statemachine.ChildStorageCollection
	}
	// var mainStorageChanges *statemachine.StorageCollection
	// var childStorageChanges *statemachine.ChildStorageCollection

	if storageChanges != nil {
		switch changes := (*storageChanges).(type) {
		case consensus.StorageChangesChanges[H, Hasher]:
			err := c.backend.BeginStateOperation(&operation.Op, parentHash)
			if err != nil {
				return nil, err
			}
			mainSC := changes.MainStorageChanges
			childSC := changes.ChildStorageChanges
			// offchainSC := storageChanges.OffchainStorageChanges
			tx := changes.Transaction
			txIndex := changes.TransactionIndexChanges

			// if self.config.offchain_indexing_api {
			// 	operation.op.update_offchain_storage(offchain_sc)?;
			// }

			err = operation.Op.UpdateDBStorage(tx)
			if err != nil {
				return nil, err
			}
			err = operation.Op.UpdateStorage(mainSC, childSC)
			if err != nil {
				return nil, err
			}
			err = operation.Op.UpdateTransactionIndex(txIndex)
			if err != nil {
				return nil, err
			}
			// mainStorageChanges = &mainSC
			// childStorageChanges = &childSC
			storageChangesOpt = &struct {
				statemachine.StorageCollection
				statemachine.ChildStorageCollection
			}{mainSC, childSC}
		case consensus.StorageChangesImport[H]:
			store := storage.Storage{}
			for _, state := range changes.State {
				if len(state.ParentStorageKeys) == 0 && len(state.StateRoot) == 0 {
					for _, kv := range state.KeyValues {
						store.Top.Set(string(kv.Key), kv.Value)
					}
				} else {
					for _, parentStorage := range state.ParentStorageKeys {
						prefixedStorageKey := storage.PrefixedStorageKey(parentStorage)
						var storageKey storage.StorageKey
						switch childType := storage.NewChildTypeFromPrefixedKey(prefixedStorageKey); childType {
						case nil:
							return nil, fmt.Errorf("Invalid child storage key.")
						default:
							storageKey = childType.Key
						}
						entry, ok := store.ChildrenDefault[string(storageKey)]
						if !ok {
							entry = storage.StorageChild{
								ChildInfo: storage.NewDefaultChildInfo(storageKey),
							}
							store.ChildrenDefault[string(storageKey)] = entry
						}
						for _, kv := range state.KeyValues {
							entry.Data.Set(string(kv.Key), kv.Value)
						}
					}
				}
			}
			// This is use by fast sync for runtime version to be resolvable from
			// changes.
			// let state_version =
			// 	resolve_state_version_from_wasm(&storage, &self.executor)?;
			// let state_root = operation.op.reset_storage(storage, state_version)?;
			// if state_root != *import_headers.post().state_root() {
			// 	// State root mismatch when importing state. This should not happen in
			// 	// safe fast sync mode, but may happen in unsafe mode.
			// 	warn!("Error importing state: State root mismatch.");
			// 	return Err(Error::InvalidStateRoot)
			// }

		}
	} else {
		storageChangesOpt = nil
	}

	// Ensure parent chain is finalized to maintain invariant that finality is called
	// sequentially.
	if finalized && parentExists && info.FinalizedHash != parentHash {
		err := c.applyFinalityWithBlockHash(operation, parentHash, nil, info.BestHash, makeNotifications)
		if err != nil {
			return nil, err
		}
	}

	var isNewBest bool
	if !gapBlock {
		if finalized {
			isNewBest = true
		}
		switch forkChoice := forkChoice.(type) {
		case consensus.ForkChainStrategyLongestChain:
			isNewBest = importHeaders.Post().Number() > info.BestNumber
		case consensus.ForkChainStrategyCustom:
			isNewBest = bool(forkChoice)
		}
	}

	var leafState api.NewBlockState
	if finalized {
		leafState = api.NewBlockStateFinal
	} else if isNewBest {
		leafState = api.NewBlockStateBest
	} else {
		leafState = api.NewBlockStateNormal
	}

	var treeRoute *blockchain.TreeRoute[H, N]
	if isNewBest && info.BestHash != parentHash && parentExists {
		routeFromBest, err := blockchain.NewTreeRoute(c.backend.Blockchain(), info.BestHash, parentHash)
		if err != nil {
			return nil, err
		}
		treeRoute = &routeFromBest
	} else {
		treeRoute = nil
	}

	log.Printf("Imported %v, %v, best=%v, origin=%v\n",
		hash,
		importHeaders.Post().Number(),
		isNewBest,
		origin,
	)

	err = operation.Op.SetBlockData(
		importHeaders.Post(),
		body,
		*indexedBody,
		justifications,
		leafState,
	)
	if err != nil {
		return nil, err
	}

	err = operation.Op.InsertAux(aux)
	if err != nil {
		return nil, err
	}

	// let should_notify_every_block = !self.every_import_notification_sinks.lock().is_empty();
	var shouldNotifyEveryBlock bool

	// Notify when we are already synced to the tip of the chain
	// or if this import triggers a re-org
	// let should_notify_recent_block = make_notifications || tree_route.is_some();
	shouldNotifyRecentBlock := makeNotifications || treeRoute != nil

	if shouldNotifyEveryBlock || shouldNotifyRecentBlock {
		header := importHeaders.Post()
		if finalized && shouldNotifyRecentBlock {
			var summary api.FinalizeSummary[N, H]
			summaryOpt := operation.NotifiyFinalized
			operation.NotifiyFinalized = nil
			switch summaryOpt {
			case nil:
				summary = api.FinalizeSummary[N, H]{
					Header:    header,
					Finalized: []H{hash},
				}
			default:
				summaryOpt.Header = header
				summaryOpt.Finalized = append(summary.Finalized, hash)
				summary = *summaryOpt
			}

			if parentExists {
				// Add to the stale list all heads that are branching from parent besides our
				// current `head`.
				leaves, err := c.backend.Blockchain().Leaves()
				if err != nil {
					return nil, err
				}
				for _, leaf := range leaves {
					if leaf == parentHash {
						continue
					}
					routeFromParent, err := blockchain.NewTreeRoute(c.backend.Blockchain(), parentHash, leaf)
					if err != nil {
						return nil, err
					}
					if len(routeFromParent.Retratcted()) == 0 {
						summary.StaleHeads = append(summary.StaleHeads, leaf)
					}
				}
			}
			operation.NotifiyFinalized = &summary
		}

		var importNotificationAction api.ImportNotificationAction
		if shouldNotifyEveryBlock {
			if shouldNotifyRecentBlock {
				importNotificationAction = api.ImportNotificationActionBoth
			} else {
				importNotificationAction = api.ImportNotificationActionEveryBlock
			}
		} else {
			importNotificationAction = api.ImportNotificationActionRecentBlock
		}

		operation.NotifyImported = &api.ImportSummary[N, H]{
			Hash:                     hash,
			Origin:                   origin,
			Header:                   header,
			IsNewBest:                isNewBest,
			StorageChanges:           storageChangesOpt,
			TreeRoute:                treeRoute,
			ImportNotificationAction: importNotificationAction,
		}
	}

	return consensus.ImportResultImported{IsNewBest: isNewBest}, nil
}

// / Returns true if state for given block is available.
//
//	fn have_state_at(&self, hash: Block::Hash, _number: NumberFor<Block>) -> bool {
//		self.state_at(hash).is_ok()
//	}
func (c *Client[H, N, Hasher, RA]) HaveStateAt(hash H, number N) bool {
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
func (c *Client[H, N, Hasher, RA]) StateAt(hash H) (statemachine.Backend[H, Hasher], error) {
	return c.backend.StateAt(hash)
}

// / Get blockchain info.
func (c *Client[H, N, Hasher, RA]) ChainInfo() blockchain.Info[H, N] {
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
func (c *Client[H, N, Hasher, RA]) BlockStatus(hash H) (consensus.BlockStatus, error) {
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

func (c *Client[H, N, Hasher, RA]) NewBlock(inherentDigests runtime.Digest) (blockbuilder.BlockBuilder[H, N, Hasher], error) {
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
func (c *Client[H, N, Hasher, RA]) RuntimeAPI() papi.APIExt[H, N, Hasher] {
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
func (c *Client[H, N, Hasher, RA]) CallAPIAt(params papi.CallAPIAtParams[H, N]) ([]byte, error) {
	return c.executor.ContextualCall(
		params.At,
		params.Function,
		params.Arguments,
		params.OverlayedChanges,
		params.Recorder,
		params.CallContext,
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
func (c *Client[H, N, Hasher, RA]) prepareBlockStorageChanges(importBlock consensus.BlockImportParams[H, N]) (prepareStorageChangesResult, error) {
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
		sc := consensus.StorageChanges(consensus.StorageChangesChanges[H, Hasher](genStorageChanges))
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

func (c *Client[H, N, Hasher, RA]) applyFinalityWithBlockHash(
	operation *api.ClientImportOperation[N, H, Hasher],
	block H,
	justification *runtime.Justification,
	bestBlock H,
	notify bool,
) error {
	// find tree route from last finalized to given block.
	lastFinalized, err := c.backend.Blockchain().LastFinalized()
	if err != nil {
		return err
	}

	if block == lastFinalized {
		// warn!(
		// 	"Possible safety violation: attempted to re-finalize last finalized block {:?} ",
		// 	last_finalized
		// );
		return nil
	}

	routeFromFinalized, err := blockchain.NewTreeRoute[H, N](c.backend.Blockchain(), lastFinalized, block)
	if err != nil {
		return err
	}

	if len(routeFromFinalized.Retratcted()) > 0 {
		retracted := routeFromFinalized.Retratcted()[0]
		// warn!(
		// 	"Safety violation: attempted to revert finalized block {:?} which is not in the \
		// 	same chain as last finalized {:?}",
		// 	retracted, last_finalized
		// );
		log.Printf(
			"Safety violation: attempted to revert finalized block %v which is not in the same chain as last finalized %v\n",
			retracted, lastFinalized,
		)
		return fmt.Errorf("Failed to set the chain head to a block that's too old.")
	}

	// If there is only one leaf, best block is guaranteed to be
	// a descendant of the new finalized block. If not,
	// we need to check.
	leaves, err := c.backend.Blockchain().Leaves()
	if err != nil {
		return err
	}
	if len(leaves) > 1 {
		routeFromBest, err := blockchain.NewTreeRoute[H, N](c.backend.Blockchain(), bestBlock, block)
		if err != nil {
			return err
		}
		// if the block is not a direct ancestor of the current best chain,
		// then some other block is the common ancestor.
		if routeFromBest.CommonBlock().Hash != block {
			// NOTE: we're setting the finalized block as best block, this might
			// be slightly inaccurate since we might have a "better" block
			// further along this chain, but since best chain selection logic is
			// plugable we cannot make a better choice here. usages that need
			// an accurate "best" block need to go through `SelectChain`
			// instead.
			err := operation.Op.MarkHead(block)
			if err != nil {
				return err
			}
		}
	}

	enacted := routeFromFinalized.Enacted()
	if len(enacted) == 0 {
		panic("wtf?")
	}
	for _, finalizeNew := range enacted[:len(enacted)-1] {
		err := operation.Op.MarkFinalized(finalizeNew.Hash, nil)
		if err != nil {
			return err
		}
	}

	if enacted[len(enacted)-1].Hash != block {
		panic("wtf?")
	}
	err = operation.Op.MarkFinalized(block, justification)
	if err != nil {
		return err
	}

	if notify {
		var finalized []H
		for _, block := range routeFromFinalized.Enacted() {
			finalized = append(finalized, block.Hash)
		}

		blockNumber := routeFromFinalized.Last().Number

		staleHeads, err := c.backend.Blockchain().DisplacedLeavesAfterFinalizing(blockNumber)
		if err != nil {
			return err
		}

		header, err := c.backend.Blockchain().Header(block)
		if err != nil {
			return err
		}
		if header == nil {
			panic("Block to finalize expected to be onchain; qed")
		}
		operation.NotifiyFinalized = &api.FinalizeSummary[N, H]{
			Header:     *header,
			Finalized:  finalized,
			StaleHeads: staleHeads,
		}
	}

	return nil
}

func (c *Client[H, N, Hasher, RA]) notifyFinalized(notification *api.FinalityNotification[H, N]) error {
	c.finalityNotificationSinksMutex.Lock()
	defer c.finalityNotificationSinksMutex.Unlock()

	if notification != nil {
		// Cleanup any closed finality notification sinks
		// since we won't be running the loop below which
		// would also remove any closed sinks.
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

func (c *Client[H, N, Hasher, RA]) notifyImported(
	notification *api.BlockImportNotification[H, N],
	importNotificationAction api.ImportNotificationAction,
	storageChanges *struct {
		statemachine.StorageCollection
		statemachine.ChildStorageCollection
	},
) error {
	if notification != nil {
		// Cleanup any closed import notification sinks since we won't
		// be sending any notifications below which would remove any
		// closed sinks. this is necessary since during initial sync we
		// won't send any import notifications which could lead to a
		// temporary leak of closed/discarded notification sinks (e.g.
		// from consensus code).
		// self.import_notification_sinks.lock().retain(|sink| !sink.is_closed());

		// self.every_import_notification_sinks.lock().retain(|sink| !sink.is_closed());

		// return Ok(())
		return nil
	}

	var changes []struct {
		StorageKey  []byte
		StorageData *[]byte
	}
	for _, change := range storageChanges.StorageCollection {
		changes = append(changes, struct {
			StorageKey  []byte
			StorageData *[]byte
		}{change.StorageKey, (*[]byte)(&change.StorageValue)})
	}
	var childChanges []struct {
		StorageKey []byte
		KeyData    []struct {
			StorageKey  []byte
			StorageData *[]byte
		}
	}
	for _, change := range storageChanges.ChildStorageCollection {
		var keyData []struct {
			StorageKey  []byte
			StorageData *[]byte
		}
		for _, data := range change.StorageCollection {
			keyData = append(keyData, struct {
				StorageKey  []byte
				StorageData *[]byte
			}{
				data.StorageKey,
				(*[]byte)(&data.StorageValue),
			})
		}
		childChanges = append(childChanges, struct {
			StorageKey []byte
			KeyData    []struct {
				StorageKey  []byte
				StorageData *[]byte
			}
		}{
			StorageKey: change.StorageKey,
			KeyData:    keyData,
		})
	}

	var triggerStorageChangesNotification = func() {
		if storageChanges != nil {
			c.storageNotifications <- api.StorageNotification[H]{
				Block: notification.Hash,
				Changes: api.StorageChangeSet{
					Changes:      changes,
					ChildChanges: childChanges,
				},
			}
		}
	}

	// TODO: handle every_import_notifications_sinks
	switch importNotificationAction {
	case api.ImportNotificationActionBoth:
		triggerStorageChangesNotification()
		c.importNotificationSinksMutex.Lock()
		for _, ch := range c.importNotificationSinks {
			ch <- *notification
		}
		c.importNotificationSinksMutex.Unlock()

		// self.every_import_notification_sinks
		// .lock()
		// .retain(|sink| sink.unbounded_send(notification.clone()).is_ok());
	case api.ImportNotificationActionRecentBlock:
		triggerStorageChangesNotification()
		c.importNotificationSinksMutex.Lock()
		for _, ch := range c.importNotificationSinks {
			ch <- *notification
		}
		c.importNotificationSinksMutex.Unlock()

		// self.every_import_notification_sinks.lock().retain(|sink| !sink.is_closed());
	case api.ImportNotificationActionEveryBlock:
		// self.every_import_notification_sinks
		// 	.lock()
		// 	.retain(|sink| sink.unbounded_send(notification.clone()).is_ok());

		// self.import_notification_sinks.lock().retain(|sink| !sink.is_closed());
	case api.ImportNotificationActionNone:
		// This branch is unreachable in fact because the block import notification must be
		// Some(_) instead of None (it's already handled at the beginning of this function)
		// at this point.
		// self.import_notification_sinks.lock().retain(|sink| !sink.is_closed());

		// self.every_import_notification_sinks.lock().retain(|sink| !sink.is_closed());
	}

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
func (c *Client[H, N, Hasher, RA]) ImportBlock(importBlock consensus.BlockImportParams[H, N]) chan<- struct {
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
		var importResult consensus.ImportResult
		err = c.LockImportAndRun(func(op *api.ClientImportOperation[N, H, Hasher]) error {
			var err error
			importResult, err = c.applyBlock(op, importBlock, storageChanges)
			return err
		})
		ch <- struct {
			consensus.ImportResult
			Error error
		}{importResult, err}
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
