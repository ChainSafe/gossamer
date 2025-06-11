package grandpa_test

import (
	"math"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa"
	common_sync "github.com/ChainSafe/gossamer/internal/client/network/common/sync"
	"github.com/ChainSafe/gossamer/internal/client/network/test"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/test-utils/runtime"
	"github.com/ChainSafe/gossamer/internal/test-utils/runtime/client"
)

// type TestLinkHalf = LinkHalf<Block, PeersFullClient, LongestChain<substrate_test_runtime_client::Backend, Block>>;
type TestLinkHalf = grandpa.LinkHalf[runtime.Hash, runtime.BlockNumber, runtime.Hasher, runtime.Header, runtime.Extrinsic]

// type PeerData = Mutex<Option<TestLinkHalf>>;
type PeerData struct {
	*TestLinkHalf
	sync.Mutex
}
type GrandpaPeer = test.Peer[*PeerData, grandpa.GrandpaBlockImport[runtime.Hash, runtime.BlockNumber, runtime.Hasher, runtime.Header, runtime.Extrinsic]]

// #[derive(Default)]
//
//	struct GrandpaTestNet {
//		peers: Vec<GrandpaPeer>,
//		test_config: TestApi,
//	}
type GrandpaTestNet struct {
	peers      []GrandpaPeer
	testConfig TestAPI
}

// impl GrandpaTestNet {
// 	fn new(test_config: TestApi, n_authority: usize, n_full: usize) -> Self {
// 		let mut net =
// 			GrandpaTestNet { peers: Vec::with_capacity(n_authority + n_full), test_config };

// 		for _ in 0..n_authority {
// 			net.add_authority_peer();
// 		}

// 		for _ in 0..n_full {
// 			net.add_full_peer();
// 		}

//			net
//		}
//	}
func NewGrandpaTestNet(testConfig TestAPI, nAuthority, nFull int) *GrandpaTestNet {
	net := &GrandpaTestNet{
		peers:      make([]GrandpaPeer, 0, nAuthority+nFull),
		testConfig: testConfig,
	}

	for i := 0; i < nAuthority; i++ {
		net.addAuthorityPeer()
	}

	for i := 0; i < nFull; i++ {
		net.addFullPeer()
	}

	return net
}

//	impl GrandpaTestNet {
//		fn add_authority_peer(&mut self) {
func (gtn *GrandpaTestNet) addAuthorityPeer() {
	// 		self.add_full_peer_with_config(FullPeerConfig {
	// 			notifications_protocols: vec![grandpa_protocol_name::NAME.into()],
	// 			is_authority: true,
	// 			..Default::default()
	// 		})
	// 	}
	panic("unimplemented")
}

// impl TestNetFactory for GrandpaTestNet {
// 	type Verifier = PassThroughVerifier;
// 	type PeerData = PeerData;
// 	type BlockImport = GrandpaBlockImport;

//	fn add_full_peer(&mut self) {
//		self.add_full_peer_with_config(FullPeerConfig {
//			notifications_protocols: vec![grandpa_protocol_name::NAME.into()],
//			is_authority: false,
//			..Default::default()
//		})
//	}
func (gtn *GrandpaTestNet) addFullPeer() {
	// 		self.add_full_peer_with_config(FullPeerConfig {
	// 			notifications_protocols: vec![grandpa_protocol_name::NAME.into()],
	// 			is_authority: false,
	// 			..Default::default()
	// 		})
	// 	}
	panic("unimplemented")
}

// 	fn make_verifier(&self, _client: PeersClient, _: &PeerData) -> Self::Verifier {
// 		PassThroughVerifier::new(false) // use non-instant finality.
// 	}

// fn make_block_import(
//
//	&self,
//	client: PeersClient,
//
//	) -> (BlockImportAdapter<Self::BlockImport>, Option<BoxJustificationImport<Block>>, PeerData) {
//		let (client, backend) = (client.as_client(), client.as_backend());
//		let (import, link) = block_import(
//			client.clone(),
//			JUSTIFICATION_IMPORT_PERIOD,
//			&self.test_config,
//			LongestChain::new(backend.clone()),
//			None,
//		)
//		.expect("Could not create block import for fresh peer.");
//		let justification_import = Box::new(import.clone());
//		(BlockImportAdapter::new(import), Some(justification_import), Mutex::new(Some(link)))
//	}
func (gtn *GrandpaTestNet) makeBlockImport(client test.PeersClient) {
	panic("unimpl")
}

// 	fn peer(&mut self, i: usize) -> &mut GrandpaPeer {
// 		&mut self.peers[i]
// 	}

// 	fn peers(&self) -> &Vec<GrandpaPeer> {
// 		&self.peers
// 	}

// 	fn peers_mut(&mut self) -> &mut Vec<GrandpaPeer> {
// 		&mut self.peers
// 	}

// 	fn mut_peers<F: FnOnce(&mut Vec<GrandpaPeer>)>(&mut self, closure: F) {
// 		closure(&mut self.peers);
// 	}
// }

// / Add a full peer.
// fn add_full_peer_with_config(&mut self, config: FullPeerConfig) {
func (gtn *GrandpaTestNet) addFullPeerWithConfig(config test.FullPeerConfig) {
	// 	let mut test_client_builder = match (config.blocks_pruning, config.storage_chain) {
	// 		(Some(blocks_pruning), true) => TestClientBuilder::with_tx_storage(blocks_pruning),
	// 		(None, true) => TestClientBuilder::with_tx_storage(u32::MAX),
	// 		(Some(blocks_pruning), false) => TestClientBuilder::with_pruning_window(blocks_pruning),
	// 		(None, false) => TestClientBuilder::with_default_backend(),
	// 	};
	var testClientBuilder client.TestClientBuilder
	switch {
	case config.BlocksPruning != nil && config.StorageChain:
		testClientBuilder = client.NewTestClientBuilderWithTxStorage(*config.BlocksPruning)
	case config.BlocksPruning == nil && config.StorageChain:
		testClientBuilder = client.NewTestClientBuilderWithTxStorage(math.MaxUint32)
	case config.BlocksPruning != nil && !config.StorageChain:
		testClientBuilder = client.NewTestClientBuilderWithPruningWindow(*config.BlocksPruning)
	case config.BlocksPruning == nil && !config.StorageChain:
		testClientBuilder = client.NewTestClientBuilderWithDefaultBackend()
	default:
		panic("unreachable")
	}
	// 	if let Some(storage) = config.extra_storage {
	// 		let genesis_extra_storage = test_client_builder.genesis_init_mut().extra_storage();
	// 		*genesis_extra_storage = storage;
	// 	}
	if config.ExtraStorage != nil {
		genesisExtraStorage := testClientBuilder.GenesisInitMut().ExtraStorage()
		*genesisExtraStorage = *config.ExtraStorage
	}

	// 	if !config.force_genesis &&
	// 		matches!(config.sync_mode, SyncMode::LightState { .. } | SyncMode::Warp)
	// 	{
	// 		test_client_builder = test_client_builder.set_no_genesis();
	// 	}
	if !config.ForceGenesis {
		switch config.SyncMode.(type) {
		case common_sync.SyncModeLightState, common_sync.SyncModeWarp:
			testClientBuilder.SetNoGenesis()
		}
	}
	// 	let backend = test_client_builder.backend();
	// 	let (c, longest_chain) = test_client_builder.build_with_longest_chain();
	// 	let client = Arc::new(c);
	backend := testClientBuilder.Backend()
	c, longestChain := testClientBuilder.BuildWithLongestChain()

	_, _, _ = backend, c, longestChain

	// 	let (block_import, justification_import, data) = self
	// 		.make_block_import(PeersClient { client: client.clone(), backend: backend.clone() });

	// 	let verifier = self
	// 		.make_verifier(PeersClient { client: client.clone(), backend: backend.clone() }, &data);
	// 	let verifier = VerifierAdapter::new(verifier);

	// 	let import_queue = Box::new(BasicQueue::new(
	// 		verifier.clone(),
	// 		Box::new(block_import.clone()),
	// 		justification_import,
	// 		&sp_core::testing::TaskExecutor::new(),
	// 		None,
	// 	));

	// 	let listen_addr = build_multiaddr![Memory(rand::random::<u64>())];

	// 	let mut network_config =
	// 		NetworkConfiguration::new("test-node", "test-client", Default::default(), None);
	// 	network_config.sync_mode = config.sync_mode;
	// 	network_config.transport = TransportConfig::MemoryOnly;
	// 	network_config.listen_addresses = vec![listen_addr.clone()];
	// 	network_config.allow_non_globals_in_dht = true;

	// 	let (notif_configs, notif_handles): (Vec<_>, Vec<_>) = config
	// 		.notifications_protocols
	// 		.into_iter()
	// 		.map(|p| {
	// 			let (config, handle) = NonDefaultSetConfig::new(
	// 				p.clone(),
	// 				Vec::new(),
	// 				1024 * 1024,
	// 				None,
	// 				Default::default(),
	// 			);

	// 			(config, (p, handle))
	// 		})
	// 		.unzip();

	// 	if let Some(connect_to) = config.connect_to_peers {
	// 		let addrs = connect_to
	// 			.iter()
	// 			.map(|v| {
	// 				let peer_id = self.peer(*v).network_service().local_peer_id();
	// 				let multiaddr = self.peer(*v).listen_addr.clone();
	// 				MultiaddrWithPeerId { peer_id, multiaddr }
	// 			})
	// 			.collect();
	// 		network_config.default_peers_set.reserved_nodes = addrs;
	// 		network_config.default_peers_set.non_reserved_mode = NonReservedPeerMode::Deny;
	// 	}
	// 	let mut full_net_config = FullNetworkConfiguration::new(&network_config, None);

	// 	let protocol_id = ProtocolId::from("test-protocol-name");

	// 	let fork_id = Some(String::from("test-fork-id"));

	// 	let chain_sync_network_provider = NetworkServiceProvider::new();
	// 	let chain_sync_network_handle = chain_sync_network_provider.handle();
	// 	let mut block_relay_params = BlockRequestHandler::new::<NetworkWorker<_, _>>(
	// 		chain_sync_network_handle.clone(),
	// 		&protocol_id,
	// 		None,
	// 		client.clone(),
	// 		50,
	// 	);
	// 	self.spawn_task(Box::pin(async move {
	// 		block_relay_params.server.run().await;
	// 	}));

	// 	let state_request_protocol_config = {
	// 		let (handler, protocol_config) = StateRequestHandler::new::<NetworkWorker<_, _>>(
	// 			&protocol_id,
	// 			None,
	// 			client.clone(),
	// 			50,
	// 		);
	// 		self.spawn_task(handler.run().boxed());
	// 		protocol_config
	// 	};

	// 	let light_client_request_protocol_config =
	// 		{
	// 			let (handler, protocol_config) = LightClientRequestHandler::new::<
	// 				NetworkWorker<_, _>,
	// 			>(&protocol_id, None, client.clone());
	// 			self.spawn_task(handler.run().boxed());
	// 			protocol_config
	// 		};

	// 	let warp_sync = Arc::new(TestWarpSyncProvider(client.clone()));

	// 	let warp_sync_config = match config.target_header {
	// 		Some(target_header) => WarpSyncConfig::WithTarget(target_header),
	// 		_ => WarpSyncConfig::WithProvider(warp_sync.clone()),
	// 	};

	// 	let warp_protocol_config = {
	// 		let (handler, protocol_config) =
	// 			warp_request_handler::RequestHandler::new::<_, NetworkWorker<_, _>>(
	// 				protocol_id.clone(),
	// 				client
	// 					.block_hash(0u32.into())
	// 					.ok()
	// 					.flatten()
	// 					.expect("Genesis block exists; qed"),
	// 				None,
	// 				warp_sync.clone(),
	// 			);
	// 		self.spawn_task(handler.run().boxed());
	// 		protocol_config
	// 	};

	// 	let peer_store = PeerStore::new(
	// 		network_config
	// 			.boot_nodes
	// 			.iter()
	// 			.map(|bootnode| bootnode.peer_id.into())
	// 			.collect(),
	// 		None,
	// 	);
	// 	let peer_store_handle = Arc::new(peer_store.handle());
	// 	self.spawn_task(peer_store.run().boxed());

	// 	let block_announce_validator = config
	// 		.block_announce_validator
	// 		.unwrap_or_else(|| Box::new(DefaultBlockAnnounceValidator));
	// 	let metrics = <NetworkWorker<_, _> as sc_network::NetworkBackend<
	// 		Block,
	// 		<Block as BlockT>::Hash,
	// 	>>::register_notification_metrics(None);

	// 	let syncing_config = PolkadotSyncingStrategyConfig {
	// 		mode: network_config.sync_mode,
	// 		max_parallel_downloads: network_config.max_parallel_downloads,
	// 		max_blocks_per_request: network_config.max_blocks_per_request,
	// 		metrics_registry: None,
	// 		state_request_protocol_name: state_request_protocol_config.name.clone(),
	// 		block_downloader: block_relay_params.downloader,
	// 	};
	// 	// Initialize syncing strategy.
	// 	let syncing_strategy = Box::new(
	// 		PolkadotSyncingStrategy::new(
	// 			syncing_config,
	// 			client.clone(),
	// 			Some(warp_sync_config),
	// 			Some(warp_protocol_config.name.clone()),
	// 		)
	// 		.unwrap(),
	// 	);

	// 	let (engine, sync_service, block_announce_config) =
	// 		sc_network_sync::engine::SyncingEngine::new(
	// 			Roles::from(if config.is_authority { &Role::Authority } else { &Role::Full }),
	// 			client.clone(),
	// 			None,
	// 			metrics,
	// 			&full_net_config,
	// 			protocol_id.clone(),
	// 			fork_id.as_deref(),
	// 			block_announce_validator,
	// 			syncing_strategy,
	// 			chain_sync_network_handle,
	// 			import_queue.service(),
	// 			peer_store_handle.clone(),
	// 		)
	// 		.unwrap();
	// 	let sync_service = Arc::new(sync_service.clone());

	// 	for config in config.request_response_protocols {
	// 		full_net_config.add_request_response_protocol(config);
	// 	}
	// 	for config in [
	// 		block_relay_params.request_response_config,
	// 		state_request_protocol_config,
	// 		light_client_request_protocol_config,
	// 		warp_protocol_config,
	// 	] {
	// 		full_net_config.add_request_response_protocol(config);
	// 	}

	// 	for config in notif_configs {
	// 		full_net_config.add_notification_protocol(config);
	// 	}

	// 	let genesis_hash =
	// 		client.hash(Zero::zero()).ok().flatten().expect("Genesis block exists; qed");
	// 	let network = NetworkWorker::new(sc_network::config::Params {
	// 		role: if config.is_authority { Role::Authority } else { Role::Full },
	// 		executor: Box::new(|f| {
	// 			tokio::spawn(f);
	// 		}),
	// 		network_config: full_net_config,
	// 		genesis_hash,
	// 		protocol_id,
	// 		fork_id,
	// 		metrics_registry: None,
	// 		block_announce_config,
	// 		bitswap_config: None,
	// 		notification_metrics: NotificationMetrics::new(None),
	// 	})
	// 	.unwrap();

	// 	trace!(target: "test_network", "Peer identifier: {}", network.service().local_peer_id());

	// 	let service = network.service().clone();
	// 	tokio::spawn(async move {
	// 		chain_sync_network_provider.run(service).await;
	// 	});

	// 	tokio::spawn({
	// 		let sync_service = sync_service.clone();

	// 		async move {
	// 			import_queue.run(sync_service.as_ref()).await;
	// 		}
	// 	});

	// 	tokio::spawn(async move {
	// 		engine.run().await;
	// 	});

	// 	self.mut_peers(move |peers| {
	// 		for peer in peers.iter_mut() {
	// 			peer.network.add_known_address(
	// 				network.service().local_peer_id().into(),
	// 				listen_addr.clone().into(),
	// 			);
	// 		}

	// 		let imported_blocks_stream = Box::pin(client.import_notification_stream().fuse());
	// 		let finality_notification_stream =
	// 			Box::pin(client.finality_notification_stream().fuse());

	//		peers.push(Peer {
	//			data,
	//			client: PeersClient { client: client.clone(), backend: backend.clone() },
	//			select_chain: Some(longest_chain),
	//			backend: Some(backend),
	//			imported_blocks_stream,
	//			finality_notification_stream,
	//			notification_services: HashMap::from_iter(notif_handles.into_iter()),
	//			block_import,
	//			verifier,
	//			network,
	//			sync_service,
	//			listen_addr,
	//		});
	//	});
}

// #[derive(Default, Clone)]
// pub(crate) struct TestApi {
type TestAPI struct {
	// genesis_authorities: AuthorityList,
	genesisAuthorities primitives.AuthorityList
}

//	impl TestApi {
//		pub fn new(genesis_authorities: AuthorityList) -> Self {
//			TestApi { genesis_authorities }
//		}
//	}
func NewTestAPI(genesisAuthorities primitives.AuthorityList) TestAPI {
	return TestAPI{
		genesisAuthorities: genesisAuthorities,
	}
}

//	fn make_ids(keys: &[Ed25519Keyring]) -> AuthorityList {
//		keys.iter().map(|&key| key.public().into()).map(|id| (id, 1)).collect()
//	}
func makeIDs(keys []ed25519.Keyring) primitives.AuthorityList {
	ids := make(primitives.AuthorityList, len(keys))
	for i, key := range keys {
		ids[i] = primitives.AuthorityIDWeight{
			AuthorityID:     key.Public(),
			AuthorityWeight: 1,
		}
	}
	return ids
}
func TestVoter(t *testing.T) {
	// async fn finalize_3_voters_no_observers() {
	t.Run("finalize_3_voters_no_observers", func(t *testing.T) {
		// 	sp_tracing::try_init_simple();
		// 	let peers = &[Ed25519Keyring::Alice, Ed25519Keyring::Bob, Ed25519Keyring::Charlie];
		peers := []ed25519.Keyring{ed25519.Alice, ed25519.Bob, ed25519.Charlie}
		// 	let voters = make_ids(peers);
		voters := makeIDs(peers)
		_ = voters

		// 	let mut net = GrandpaTestNet::new(TestApi::new(voters), 3, 0);
		// 	tokio::spawn(initialize_grandpa(&mut net, peers));
		// 	net.peer(0).push_blocks(20, false);
		// 	net.run_until_sync().await;
		// 	let hashof20 = net.peer(0).client().info().best_hash;

		// 	for i in 0..3 {
		// 		assert_eq!(net.peer(i).client().info().best_number, 20, "Peer #{} failed to sync", i);
		// 		assert_eq!(net.peer(i).client().info().best_hash, hashof20, "Peer #{} failed to sync", i);
		// 	}

		// 	let net = Arc::new(Mutex::new(net));
		// 	run_to_completion(20, net.clone(), peers).await;

		// 	// all peers should have stored the justification for the best finalized block #20
		// 	for peer_id in 0..3 {
		// 		let client = net.lock().peers[peer_id].client().as_client();
		// 		let justification =
		// 			crate::aux_schema::best_justification::<_, Block>(&*client).unwrap().unwrap();

		//			assert_eq!(justification.justification.commit.target_number, 20);
		//		}
	})
}
