package test

import (
	"github.com/ChainSafe/gossamer/internal/client"
	"github.com/ChainSafe/gossamer/internal/client/network/common/sync"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/test-utils/runtime"
	testruntimeclient "github.com/ChainSafe/gossamer/internal/test-utils/runtime/client"
)

// pub type PeersFullClient = Client<
//
//	substrate_test_runtime_client::Backend,
//	substrate_test_runtime_client::ExecutorDispatch,
//	Block,
//	substrate_test_runtime_client::runtime::RuntimeApi,
//
// >;
type PeersFullClient = client.Client[
	runtime.Hash, runtime.Hasher, runtime.BlockNumber, runtime.Extrinsic, runtime.Header,
]

// #[derive(Clone)]
// pub struct PeersClient {
type PeersClient struct {
	// client: Arc<PeersFullClient>,
	client PeersFullClient
	// backend: Arc<substrate_test_runtime_client::Backend>,
	backend testruntimeclient.Backend
}

// pub struct Peer<D, BlockImport> {
type Peer[D any, BlockImport any] struct {
	// pub data: D,
	Data D
	// client: PeersClient,
	client PeersClient
	/// We keep a copy of the verifier so that we can invoke it for locally-generated blocks,
	/// instead of going through the import queue.
	// verifier: VerifierAdapter<Block>,
	// /// We keep a copy of the block_import so that we can invoke it for locally-generated blocks,
	// /// instead of going through the import queue.
	// block_import: BlockImportAdapter<BlockImport>,
	// select_chain: Option<LongestChain<substrate_test_runtime_client::Backend, Block>>,
	// backend: Option<Arc<substrate_test_runtime_client::Backend>>,
	// network: NetworkWorker<Block, <Block as BlockT>::Hash>,
	// sync_service: Arc<SyncingService<Block>>,
	// imported_blocks_stream: Pin<Box<dyn Stream<Item = BlockImportNotification<Block>> + Send>>,
	// finality_notification_stream: Pin<Box<dyn Stream<Item = FinalityNotification<Block>> + Send>>,
	// listen_addr: Multiaddr,
	// notification_services: HashMap<ProtocolName, Box<dyn NotificationService>>,
}

// / Configuration for a full peer.
// #[derive(Default)]
// pub struct FullPeerConfig {
type FullPeerConfig struct {
	/// Pruning window size.
	///
	/// NOTE: only finalized blocks are subject for removal!
	BlocksPruning *uint32
	// /// Block announce validator.
	// pub block_announce_validator: Option<Box<dyn BlockAnnounceValidator<Block> + Send + Sync>>,
	// /// List of notification protocols that the network must support.
	// pub notifications_protocols: Vec<ProtocolName>,
	// /// List of request-response protocols that the network must support.
	// pub request_response_protocols: Vec<RequestResponseConfig>,
	// /// The indices of the peers the peer should be connected to.
	// ///
	// /// If `None`, it will be connected to all other peers.
	// pub connect_to_peers: Option<Vec<usize>>,
	// /// Whether the full peer should have the authority role.
	// pub is_authority: bool,
	/// Syncing mode
	SyncMode sync.SyncMode
	/// Extra genesis storage.
	ExtraStorage *storage.Storage
	/// Enable transaction indexing.
	StorageChain bool
	// /// Optional target block header to sync to
	// pub target_header: Option<<Block as BlockT>::Header>,
	/// Force genesis even in case of warp & light state sync.
	ForceGenesis bool
}
