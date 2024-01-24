package client

import (
	"log"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/chain-spec/genesis"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// / Substrate Client
// pub struct Client<B, E, Block, RA>
// where
//
//	Block: BlockT,
//
// {
type Client[H runtime.Hash, N runtime.Number] struct {
	// backend: Arc<B>,
	backend api.Backend[H, N]
	// executor: E,
	// storage_notifications: StorageNotifications<Block>,
	// import_notification_sinks: NotificationSinks<BlockImportNotification<Block>>,
	// every_import_notification_sinks: NotificationSinks<BlockImportNotification<Block>>,
	// finality_notification_sinks: NotificationSinks<FinalityNotification<Block>>,
	// // Collects auxiliary operations to be performed atomically together with
	// // block import operations.
	// import_actions: Mutex<Vec<OnImportAction<Block>>>,
	// // Collects auxiliary operations to be performed atomically together with
	// // block finalization operations.
	// finality_actions: Mutex<Vec<OnFinalityAction<Block>>>,
	// // Holds the block hash currently being imported. TODO: replace this with block queue.
	// importing_block: RwLock<Option<Block::Hash>>,
	// block_rules: BlockRules<Block>,
	// config: ClientConfig<Block>,
	// telemetry: Option<TelemetryHandle>,
	// unpin_worker_sender: TracingUnboundedSender<Block::Hash>,
	// _phantom: PhantomData<RA>,
}

func NewClient[H runtime.Hash, N runtime.Number](
	// backend: Arc<B>,
	backend api.Backend[H, N],
	// 	executor: E,
	// 	spawn_handle: Box<dyn SpawnNamed>,
	// genesis_block_builder: G,
	genesisBlockBuilder genesis.BuildGenesisBlock[H, N, api.BlockImportOperation[N, H]],
	// 	fork_blocks: ForkBlocks<Block>,
	// 	bad_blocks: BadBlocks<Block>,
	// 	prometheus_registry: Option<Registry>,
	// 	telemetry: Option<TelemetryHandle>,
	// 	config: ClientConfig<Block>,
) (Client[H, N], error) {
	info := backend.Blockchain().Info()
	if info.FinalizedState == nil {
		genesisBlock, op, err := genesisBlockBuilder.BuildGenesisBlock()
		if err != nil {
			return Client[H, N]{}, err
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
			return Client[H, N]{}, err
		}
		err = backend.CommitOperation(op)
		if err != nil {
			return Client[H, N]{}, err
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
	return Client[H, N]{
		backend: backend,
	}, nil
}
