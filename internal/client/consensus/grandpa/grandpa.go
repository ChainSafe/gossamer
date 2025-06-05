// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	client_common "github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/log"
	papi "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// A global communication input stream for commits and catch up messages. Not exposed publicly, used internally to
// simplify types in the communication layer.
type communicationIn[H runtime.Hash, N runtime.Number] grandpa.CommunicationIn[
	H, N, primitives.AuthoritySignature, primitives.AuthorityID]

// Global communication sink for commits with the hash type not being derived from the block, useful for forcing the
// hash to some type (e.g. `H256`) when the compiler can't do the inference.
type communicationOut[H runtime.Hash, N runtime.Number] grandpa.CommunicationOut[ //nolint: unused
	H, N, primitives.AuthoritySignature, primitives.AuthorityID]

// / Shared voter state for querying.
//
//	pub struct SharedVoterState {
//		inner: Arc<RwLock<Option<Box<dyn voter::VoterState<AuthorityId> + Sync + Send>>>>,
//	}
type SharedVoterState[AuthorityID comparable] struct {
	inner grandpa.VoterState[AuthorityID]
	sync.RWMutex
}

// impl SharedVoterState {
// 	/// Create a new empty `SharedVoterState` instance.
// 	pub fn empty() -> Self {
// 		Self { inner: Arc::new(RwLock::new(None)) }
// 	}

// 	fn reset(
// 		&self,
// 		voter_state: Box<dyn voter::VoterState<AuthorityId> + Sync + Send>,
// 	) -> Option<()> {
// 		let mut shared_voter_state = self.inner.try_write_for(Duration::from_secs(1))?;

//		*shared_voter_state = Some(voter_state);
//		Some(())
//	}
func (svs *SharedVoterState[AuthorityID]) reset(voterState grandpa.VoterState[AuthorityID]) error {
	svs.Lock()
	defer svs.Unlock()
	svs.inner = voterState
	return nil
}

// 	/// Get the inner `VoterState` instance.
// 	pub fn voter_state(&self) -> Option<report::VoterState<AuthorityId>> {
// 		self.inner.read().as_ref().map(|vs| vs.get())
// 	}
// }

// impl Clone for SharedVoterState {
// 	fn clone(&self) -> Self {
// 		SharedVoterState { inner: self.inner.clone() }
// 	}
// }

type Config struct {
	// The expected duration for a message to be gossiped across the network.
	GossipDuration time.Duration
	// Justification generation period (in blocks). GRANDPA will try to generate justifications at least every
	// justification_period blocks. There are some other events which might cause justification generation.
	JustificationGenerationPeriod uint32
	// Whether the GRANDPA observer protocol is live on the network and thereby a full-node not running as a validator
	// is running the GRANDPA observer protocol (we will only issue catch-up requests to authorities when the observer
	// protocol is enabled).
	ObserverEnabled bool
	// The role of the local node (i.e. authority, full-node or light).
	LocalRole role.Role
	// Some local identifier of the voter.
	Name *string
	// The keystore that manages the keys of this node.
	KeyStore keystore.KeyStore // can be nil for optionality
	// Chain specific GRANDPA protocol name.
	ProtocolName network.ProtocolName
	// TODO: telemetry
}

func (c Config) name() string {
	if c.Name == nil {
		return "<unknown>"
	}
	return *c.Name
}

// Errors that can occur while voting in GRANDPA.
var (
	// ErrClient means we could not complete a round on disk.
	ErrClient = errors.New("could not complete a round on disk")

	// ErrSafety means an invariant has been violated (e.g. not finalizing pending change blocks in-order)
	ErrSafety = errors.New("safety invariant has been violated")

	// ErrRuntimeApi means a runtime api request failed.
	ErrRuntimeApi = errors.New("runtime API request failed")
)

// Something which can determine if a block is known.
type BlockStatus[H runtime.Hash, N runtime.Number] interface {
	// Return a number or nil depending on whether the block is definitely known and has been imported. If an
	// unexpected error occurs, return that.
	Number(hash H) (*N, error)
}

// ClientForGrandpa is an interface that includes all the client functionalities grandpa requires.
type ClientForGrandpa[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	api.LockImportRun[H, N, Hasher, Header, E]
	api.Finalizer[H, N, Hasher, Header, E]
	api.AuxStore
	blockchain.HeaderMetadata[H, N]
	blockchain.HeaderBackend[H, N, Header]
	api.BlockchainEvents[H, N, Header]
	papi.ProvideRuntimeAPI[primitives.GrandpaAPI[H, N]]
	// api.ExecutorProvider
	client_common.BlockImport[H, N, E, Header]
	api.StorageProvider[H, N, Hasher]
}

// Something that one can ask to do a block sync request.
type BlockSyncRequester[H runtime.Hash, N runtime.Number] interface {
	// Notifies the sync service to try and sync the given block from the given peers.
	//
	// If the given vector of peers is empty then the underlying implementation should make a best effort to fetch
	// the block from any peers it is connected to (NOTE: this assumption will change in the future substrate #3629).
	SetSyncForkRequest(peers []peerid.PeerID, hash H, number N)
}

// A new authority set along with the canonical block it changed at.
type newAuthoritySet[H, N any] struct {
	CanonNumber N
	CanonHash   H
	SetID       primitives.SetID
	Authorities primitives.AuthorityList
}

// Commands issued to the voter.
type voterCommand interface {
	Error() string
	isVoterCommand()
}

// Pause the voter for given reason.
type voterCommandPause string //nolint: unused

func (vcp voterCommandPause) Error() string { //nolint: unused
	return fmt.Sprintf("Pausing voter: %s", string(vcp))
}
func (vcp voterCommandPause) isVoterCommand() {}

// New authorities.
type voterCommandChangeAuthorities[H, N any] newAuthoritySet[H, N]

func (vcca voterCommandChangeAuthorities[H, N]) Error() string {
	return "Changing authorities"
}
func (voterCommandChangeAuthorities[H, N]) isVoterCommand() {}

// / Link between the block importer and the background voter.
// pub struct LinkHalf<Block: BlockT, C, SC> {
type LinkHalf[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// client: Arc<C>,
	client ClientForGrandpa[H, N, Hasher, Header, E]
	// select_chain: SC,
	selectChain common.SelectChain[H, N, Header]
	// persistent_data: PersistentData<Block>,
	persistentData persistentData[H, N]
	// voter_commands_rx: TracingUnboundedReceiver<VoterCommand<Block::Hash, NumberFor<Block>>>,
	voterCommandsRx chan voterCommand
	// justification_sender: GrandpaJustificationSender<Block>,
	justificationSender GrandpaJustificationSender[H, N, Header]
	// justification_stream: GrandpaJustificationStream<Block>,
	justificationStream GrandpaJustificationStream[H, N, Header]
	// telemetry: Option<TelemetryHandle>,
}

// impl<Block: BlockT, C, SC> LinkHalf<Block, C, SC> {
// 	/// Get the shared authority set.
// 	pub fn shared_authority_set(&self) -> &SharedAuthoritySet<Block::Hash, NumberFor<Block>> {
// 		&self.persistent_data.authority_set
// 	}

// 	/// Get the receiving end of justification notifications.
// 	pub fn justification_stream(&self) -> GrandpaJustificationStream<Block> {
// 		self.justification_stream.clone()
// 	}
// }

// / Provider for the Grandpa authority set configured on the genesis block.
// pub trait GenesisAuthoritySetProvider<Block: BlockT> {
type GenesisAuthoritySetProvider interface {
	/// Get the authority set at the genesis block.
	// fn get(&self) -> Result<AuthorityList, ClientError>;
	Get() (primitives.AuthorityList, error)
}

// // / Make block importer and link half necessary to tie the background voter
// // / to it.
// // /
// // / The `justification_import_period` sets the minimum period on which
// // / justifications will be imported.  When importing a block, if it includes a
// // / justification it will only be processed if it fits within this period,
// // / otherwise it will be ignored (and won't be validated). This is to avoid
// // / slowing down sync by a peer serving us unnecessary justifications which
// // / aren't trivial to validate.
// // pub fn block_import<BE, Block: BlockT, Client, SC>(
// //
// //	client: Arc<Client>,
// //	justification_import_period: u32,
// //	genesis_authorities_provider: &dyn GenesisAuthoritySetProvider<Block>,
// //	select_chain: SC,
// //	telemetry: Option<TelemetryHandle>,
// //
// // ) -> Result<(GrandpaBlockImport<BE, Block, Client, SC>, LinkHalf<Block, Client, SC>), ClientError>
// // where
// //
// //	SC: SelectChain<Block>,
// //	BE: Backend<Block> + 'static,
// //	Client: ClientForGrandpa<Block, BE> + 'static,
// //
// //	{
// //		block_import_with_authority_set_hard_forks(
// //			client,
// //			justification_import_period,
// //			genesis_authorities_provider,
// //			select_chain,
// //			Default::default(),
// //			telemetry,
// //		)
// //	}
// func BlockImport[
// 	H runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[H],
// 	Header runtime.Header[N, H],
// 	E runtime.Extrinsic,
// ](
// 	client ClientForGrandpa[H, N, Hasher, Header, E],
// 	justificationImportPeriod uint32,
// 	genesisAuthoritySetProvider GenesisAuthoritySetProvider,
// 	selectChain common.SelectChain[H, N, Header],
// 	// TODO: telemetry
// ) (GrandpaBlockImport[H, N, Hasher, Header, E], LinkHalf[H, N, Hasher, Header, E], error) {
// 	return blockImportWithAuthoritySetHardForks(
// 		client,
// 		justificationImportPeriod,
// 		genesisAuthoritySetProvider,
// 		selectChain,
// 		nil,
// 	)
// }

// / A descriptor for an authority set hard fork. These are authority set changes
// / that are not signalled by the runtime and instead are defined off-chain
// / (hence the hard fork).
// pub struct AuthoritySetHardFork<Block: BlockT> {
type AuthoritySetHardFork[H, N any] struct {
	// /// The new authority set id.
	// pub set_id: SetId,
	SetID SetID
	// /// The block hash and number at which the hard fork should be applied.
	// pub block: (Block::Hash, NumberFor<Block>),
	Block HashNumber[H, N]
	// /// The authorities in the new set.
	// pub authorities: AuthorityList,
	Authorities primitives.AuthorityList
	// /// The latest block number that was finalized before this authority set
	// /// hard fork. When defined, the authority set change will be forced, i.e.
	// /// the node won't wait for the block above to be finalized before enacting
	// /// the change, and the given finalized number will be used as a base for
	// /// voting.
	// pub last_finalized: Option<NumberFor<Block>>,
	LastFinalized *N
}

// / Make block importer and link half necessary to tie the background voter to
// / it. A vector of authority set hard forks can be passed, any authority set
// / change signaled at the given block (either already signalled or in a further
// / block when importing it) will be replaced by a standard change with the
// / given static authorities.
// pub fn block_import_with_authority_set_hard_forks<BE, Block: BlockT, Client, SC>(
//
//	client: Arc<Client>,
//	justification_import_period: u32,
//	genesis_authorities_provider: &dyn GenesisAuthoritySetProvider<Block>,
//	select_chain: SC,
//	authority_set_hard_forks: Vec<AuthoritySetHardFork<Block>>,
//	telemetry: Option<TelemetryHandle>,
//
// ) -> Result<(GrandpaBlockImport<BE, Block, Client, SC>, LinkHalf<Block, Client, SC>), ClientError>
// where
//
//	SC: SelectChain<Block>,
//	BE: Backend<Block> + 'static,
//	Client: ClientForGrandpa<Block, BE> + 'static,
//
// {
// func blockImportWithAuthoritySetHardForks[
// 	H runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[H],
// 	Header runtime.Header[N, H],
// 	E runtime.Extrinsic,
// ](
// 	client ClientForGrandpa[H, N, Hasher, Header, E],
// 	justificationImportPeriod uint32,
// 	genesisAuthoritySetProvider GenesisAuthoritySetProvider,
// 	selectChain common.SelectChain[H, N, Header],
// 	authoritySetHardForks []AuthoritySetHardFork[H, N],
// 	// TODO: telemetry
// ) (GrandpaBlockImport[H, N, Hasher, Header, E], LinkHalf[H, N, Hasher, Header, E], error) {
// 	// 	let chain_info = client.info();
// 	// 	let genesis_hash = chain_info.genesis_hash;
// 	chainInfo := client.Info()
// 	genesisHash := chainInfo.GenesisHash

// 	// 	let persistent_data =
// 	// 		aux_schema::load_persistent(&*client, genesis_hash, <NumberFor<Block>>::zero(), {
// 	// 			let telemetry = telemetry.clone();
// 	// 			move || {
// 	// 				let authorities = genesis_authorities_provider.get()?;
// 	// 				telemetry!(
// 	// 					telemetry;
// 	// 					CONSENSUS_DEBUG;
// 	// 					"afg.loading_authorities";
// 	// 					"authorities_len" => ?authorities.len()
// 	// 				);
// 	// 				Ok(authorities)
// 	// 			}
// 	// 		})?;
// 	persistentData, err := loadPersistent[H, N](client, genesisHash, 0, func() (primitives.AuthorityList, error) {
// 		authorities, err := genesisAuthoritySetProvider.Get()
// 		if err != nil {
// 			return nil, err
// 		}
// 		return authorities, nil
// 	})
// 	if err != nil {
// 		return GrandpaBlockImport[H, N, Hasher, Header, E]{}, LinkHalf[H, N, Hasher, Header, E]{}, err
// 	}

// 	_ = persistentData

// 	// 	let (voter_commands_tx, voter_commands_rx) =
// 	// 		tracing_unbounded("mpsc_grandpa_voter_command", 100_000);

// 	// 	let (justification_sender, justification_stream) = GrandpaJustificationStream::channel();

// 	// 	// create pending change objects with 0 delay for each authority set hard fork.
// 	// 	let authority_set_hard_forks = authority_set_hard_forks
// 	// 		.into_iter()
// 	// 		.map(|fork| {
// 	// 			let delay_kind = if let Some(last_finalized) = fork.last_finalized {
// 	// 				authorities::DelayKind::Best { median_last_finalized: last_finalized }
// 	// 			} else {
// 	// 				authorities::DelayKind::Finalized
// 	// 			};

// 	// 			(
// 	// 				fork.set_id,
// 	// 				authorities::PendingChange {
// 	// 					next_authorities: fork.authorities,
// 	// 					delay: Zero::zero(),
// 	// 					canon_hash: fork.block.0,
// 	// 					canon_height: fork.block.1,
// 	// 					delay_kind,
// 	// 				},
// 	// 			)
// 	// 		})
// 	// 		.collect();

// 	// Ok((
// 	//
// 	//	GrandpaBlockImport::new(
// 	//		client.clone(),
// 	//		justification_import_period,
// 	//		select_chain.clone(),
// 	//		persistent_data.authority_set.clone(),
// 	//		voter_commands_tx,
// 	//		authority_set_hard_forks,
// 	//		justification_sender.clone(),
// 	//		telemetry.clone(),
// 	//	),
// 	//	LinkHalf {
// 	//		client,
// 	//		select_chain,
// 	//		persistent_data,
// 	//		voter_commands_rx,
// 	//		justification_sender,
// 	//		justification_stream,
// 	//		telemetry,
// 	//	},
// 	//
// 	// ))
// 	panic("unimpl")
// }

// fn global_communication<BE, Block: BlockT, C, N, S>(
//
//	set_id: SetId,
//	voters: &Arc<VoterSet<AuthorityId>>,
//	client: Arc<C>,
//	network: &NetworkBridge<Block, N, S>,
//	keystore: Option<&KeystorePtr>,
//	metrics: Option<until_imported::Metrics>,
//
// ) -> (
//
//	impl Stream<
//		Item = Result<
//			CommunicationInH<Block, Block::Hash>,
//			CommandOrError<Block::Hash, NumberFor<Block>>,
//		>,
//	>,
//	impl Sink<
//		CommunicationOutH<Block, Block::Hash>,
//		Error = CommandOrError<Block::Hash, NumberFor<Block>>,
//	>,
//
// )
// where
//
//	BE: Backend<Block> + 'static,
//	C: ClientForGrandpa<Block, BE> + 'static,
//	N: NetworkT<Block>,
//	S: SyncingT<Block>,
//	NumberFor<Block>: BlockNumberOps,
//
// {
func globalCommunication[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	setID primitives.SetID,
	voters grandpa.VoterSet[primitives.AuthorityID],
	client ClientForGrandpa[H, N, Hasher, Header, E],
	network *networkBridge[H, N, Hasher],
	keystore keystore.KeyStore,
	// TODO: metrics
) (chan grandpa.GlobalInItem[H, N, primitives.AuthoritySignature, primitives.AuthorityID], commitsOut[H, N, Hasher]) {
	// 	let is_voter = local_authority_id(voters, keystore).is_some();
	isVoter := localAuthorityID(voters, keystore) != nil

	// verification stream
	// 	let (global_in, global_out) =
	// 		network.global_communication(communication::SetId(set_id), voters.clone(), is_voter);
	in, out := network.globalCommunication(SetID(setID), voters, isVoter)

	// block commit and catch up messages until relevant blocks are imported.
	// 	let global_in = UntilGlobalMessageBlocksImported::new(
	// 		client.import_notification_stream(),
	// 		network.clone(),
	// 		client.clone(),
	// 		global_in,
	// 		"global",
	// 		metrics,
	// 	);
	globalIn := newUntilGlobalMessageBlocksImported(
		client.RegisterImportNotificationStream(),
		network,
		client,
		in,
		"global",
	)

	// 	let global_in = global_in.map_err(CommandOrError::from);
	// 	let global_out = global_out.sink_map_err(CommandOrError::from);
	mappedIn := make(chan grandpa.GlobalInItem[H, N, primitives.AuthoritySignature, primitives.AuthorityID])
	go func() {
		defer close(mappedIn)
		for item := range globalIn.Chan() {
			mappedIn <- grandpa.GlobalInItem[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
				CommunicationIn: item.Blocked,
				Error:           item.Error,
			}
		}
	}()

	// (global_in, global_out)
	return mappedIn, out
}

// / Parameters used to run Grandpa.
// pub struct GrandpaParams<Block: BlockT, C, N, S, SC, VR> {
type GrandpaParams[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	/// Configuration for the GRANDPA service.
	// pub config: Config,
	Config Config
	/// A link to the block import worker.
	// pub link: LinkHalf<Block, C, SC>,
	LinkHalf LinkHalf[H, N, Hasher, Header, E]
	/// The Network instance.
	///
	/// It is assumed that this network will feed us Grandpa notifications. When using the
	/// `sc_network` crate, it is assumed that the Grandpa notifications protocol has been passed
	/// to the configuration of the networking. See [`grandpa_peers_set_config`].
	// pub network: N,
	Network Network
	/// Event stream for syncing-related events.
	// pub sync: S,
	Sync Syncing[H, N]
	/// Handle for interacting with `Notifications`.
	// pub notification_service: Box<dyn NotificationService>,
	NotificationService service.NotificationService
	/// A voting rule used to potentially restrict target votes.
	// pub voting_rule: VR,
	VotingRule VotingRule[H, N, Header]
	/// The prometheus metrics registry.
	// pub prometheus_registry: Option<prometheus_endpoint::Registry>,
	/// The voter state is exposed at an RPC endpoint.
	// pub shared_voter_state: SharedVoterState,
	SharedVoterState *SharedVoterState[primitives.AuthorityID]
	/// TelemetryHandle instance.
	// pub telemetry: Option<TelemetryHandle>,
	/// Offchain transaction pool factory.
	///
	/// This will be used to create an offchain transaction pool instance for sending an
	/// equivocation report from the runtime.
	// pub offchain_tx_pool_factory: OffchainTransactionPoolFactory<Block>,
}

// / Run a GRANDPA voter as a task. Provide configuration and a link to a
// / block import worker that has already been instantiated with `block_import`.
// pub fn run_grandpa_voter<Block: BlockT, BE: 'static, C, N, S, SC, VR>(
//
//	grandpa_params: GrandpaParams<Block, C, N, S, SC, VR>,
//
// ) -> sp_blockchain::Result<impl Future<Output = ()> + Send>
// where
//
//	BE: Backend<Block> + 'static,
//	N: NetworkT<Block> + Sync + 'static,
//	S: SyncingT<Block> + Sync + 'static,
//	SC: SelectChain<Block> + 'static,
//	VR: VotingRule<Block, C> + Clone + 'static,
//	NumberFor<Block>: BlockNumberOps,
//	C: ClientForGrandpa<Block, BE> + 'static,
//	C::Api: GrandpaApi<Block>,
//
// {
func RunGrandpaVoter[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	grandpaParams GrandpaParams[H, N, Hasher, Header, E],
) (done chan struct{}, err error) {
	var (
		config              = grandpaParams.Config
		link                = grandpaParams.LinkHalf
		network             = grandpaParams.Network
		sync                = grandpaParams.Sync
		notificationService = grandpaParams.NotificationService
		votingRule          = grandpaParams.VotingRule
		sharedVoterState    = grandpaParams.SharedVoterState
	)
	// 	let GrandpaParams {
	// 		mut config,
	// 		link,
	// 		network,
	// 		sync,
	// 		notification_service,
	// 		voting_rule,
	// 		prometheus_registry,
	// 		shared_voter_state,
	// 		telemetry,
	// 		offchain_tx_pool_factory,
	// 	} = grandpa_params;

	// 	// NOTE: we have recently removed `run_grandpa_observer` from the public
	// 	// API, I felt it is easier to just ignore this field rather than removing
	// 	// it from the config temporarily. This should be removed after #5013 is
	// 	// fixed and we re-add the observer to the public API.
	// 	config.observer_enabled = false;
	config.ObserverEnabled = false

	// 	let LinkHalf {
	// 		client,
	// 		select_chain,
	// 		persistent_data,
	// 		voter_commands_rx,
	// 		justification_sender,
	// 		justification_stream: _,
	// 		telemetry: _,
	// 	} = link;
	var (
		client              = link.client
		selectChain         = link.selectChain
		persistentData      = link.persistentData
		voterCommandsRx     = link.voterCommandsRx
		justificationSender = link.justificationSender
	)

	// 	let network = NetworkBridge::new(
	// 		network,
	// 		sync,
	// 		notification_service,
	// 		config.clone(),
	// 		persistent_data.set_state.clone(),
	// 		prometheus_registry.as_ref(),
	// 		telemetry.clone(),
	// 	);
	networkBridge := newNetworkBridge[H, N, Hasher](
		network,
		sync,
		notificationService,
		config,
		persistentData.setState,
	)
	// 	let conf = config.clone();
	// 	let telemetry_task =
	// 		if let Some(telemetry_on_connect) = telemetry.as_ref().map(|x| x.on_connect_stream()) {
	// 			let authorities = persistent_data.authority_set.clone();
	// 			let telemetry = telemetry.clone();
	// 			let events = telemetry_on_connect.for_each(move |_| {
	// 				let current_authorities = authorities.current_authorities();
	// 				let set_id = authorities.set_id();
	// 				let maybe_authority_id =
	// 					local_authority_id(&current_authorities, conf.keystore.as_ref());

	// 				let authorities =
	// 					current_authorities.iter().map(|(id, _)| id.to_string()).collect::<Vec<_>>();

	// 				let authorities = serde_json::to_string(&authorities).expect(
	// 					"authorities is always at least an empty vector; \
	// 					 elements are always of type string",
	// 				);

	// 				telemetry!(
	// 					telemetry;
	// 					CONSENSUS_INFO;
	// 					"afg.authority_set";
	// 					"authority_id" => maybe_authority_id.map_or("".into(), |s| s.to_string()),
	// 					"authority_set_id" => ?set_id,
	// 					"authorities" => authorities,
	// 				);

	// 				future::ready(())
	// 			});
	// 			future::Either::Left(events)
	// 		} else {
	// 			future::Either::Right(future::pending())
	// 		};

	// 	let voter_work = VoterWork::new(
	// 		client,
	// 		config,
	// 		network,
	// 		select_chain,
	// 		voting_rule,
	// 		persistent_data,
	// 		voter_commands_rx,
	// 		prometheus_registry,
	// 		shared_voter_state,
	// 		justification_sender,
	// 		telemetry,
	// 		offchain_tx_pool_factory,
	// 	);
	voterWork := newVoterWork[H, N, Hasher, Header, E](
		client,
		config,
		networkBridge,
		selectChain,
		votingRule,
		persistentData,
		voterCommandsRx,
		sharedVoterState,
		justificationSender,
	)

	// 	let voter_work = voter_work.map(|res| match res {
	// 		Ok(()) => error!(
	// 			target: LOG_TARGET,
	// 			"GRANDPA voter future has concluded naturally, this should be unreachable."
	// 		),
	// 		Err(e) => error!(target: LOG_TARGET, "GRANDPA voter error: {}", e),
	// 	});
	done = make(chan struct{})
	go func() {
		defer close(done)
		err := voterWork.run()
		if err != nil {
			logger.Errorf("GRANDPA voter error: %v", err)
			return
		}
		logger.Error("GRANDPA voter future has concluded naturally, this should be unreachable.")
	}()

	// 	// Make sure that `telemetry_task` doesn't accidentally finish and kill grandpa.
	// 	let telemetry_task = telemetry_task.then(|_| future::pending::<()>());

	// Ok(future::select(voter_work, telemetry_task).map(drop))
	return done, nil
}

// / Future that powers the voter.
type voterWork[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// use string for AuthorityID and AuthoritySignature
	voter            *grandpa.Voter[H, N, primitives.AuthoritySignature, primitives.AuthorityID]
	voterErrChan     <-chan error
	sharedVoterState *SharedVoterState[primitives.AuthorityID]
	env              *environment[H, N, Hasher, Header, E]
	voterCommandsRx  <-chan voterCommand
	network          *networkBridge[H, N, Hasher]
	// TODO: telemtry, metrics
}

func newVoterWork[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	client ClientForGrandpa[H, N, Hasher, Header, E],
	config Config,
	network *networkBridge[H, N, Hasher],
	selectChain common.SelectChain[H, N, Header],
	votingRule VotingRule[H, N, Header],
	persistentData persistentData[H, N],
	voterCommandsRx <-chan voterCommand,
	sharedVoterState *SharedVoterState[primitives.AuthorityID],
	justificationSender GrandpaJustificationSender[H, N, Header],
	// TODO: telemetry
) voterWork[H, N, Hasher, Header, E] {
	// TODO: register to prometheus registry

	voters := persistentData.authoritySet.CurrentAuthorities()
	env := environment[H, N, Hasher, Header, E]{
		Client:              client,
		SelectChain:         selectChain,
		VotingRule:          votingRule,
		Voters:              voters,
		Config:              config,
		Network:             network,
		SetID:               SetID(persistentData.authoritySet.SetID()),
		AuthoritySet:        persistentData.authoritySet,
		VoterSetState:       persistentData.setState,
		JustificationSender: &justificationSender,
	}

	work := voterWork[H, N, Hasher, Header, E]{
		// `voter` is set to a temporary value and replaced below when
		// calling `rebuild_voter`.
		voter:            nil,
		sharedVoterState: sharedVoterState,
		env:              &env,
		voterCommandsRx:  voterCommandsRx,
		network:          network,
	}
	work.rebuildVoter()
	return work
}

// / Rebuilds the `self.voter` field using the current authority set
// / state. This method should be called when we know that the authority set
// / has changed (e.g. as signalled by a voter command).
func (vw *voterWork[H, N, Hasher, Header, E]) rebuildVoter() {
	// debug!(
	// 	target: LOG_TARGET,
	// 	"{}: Starting new voter with set ID {}",
	// 	self.env.config.name(),
	// 	self.env.set_id
	// );
	logger.Debugf("%s: Starting new voter with set ID %v", vw.env.Config.name(), vw.env.SetID)

	maybeAuthorityID := localAuthorityID(vw.env.Voters, vw.env.Config.KeyStore)
	var authorityID string
	if maybeAuthorityID != nil {
		authorityID = string(maybeAuthorityID.Bytes())
	} else {
		authorityID = "<unknown>"
	}

	// telemetry!(
	// 	self.telemetry;
	// 	CONSENSUS_DEBUG;
	// 	"afg.starting_new_voter";
	// 	"name" => ?self.env.config.name(),
	// 	"set_id" => ?self.env.set_id,
	// 	"authority_id" => authority_id,
	// );
	// TODO: telemetry afg.starting_new_voter

	chainInfo := vw.env.Client.Info()

	// let authorities = self.env.voters.iter().map(|(id, _)| id.to_string()).collect::<Vec<_>>();

	// let authorities = serde_json::to_string(&authorities).expect(
	// 	"authorities is always at least an empty vector; elements are always of type string; qed.",
	// );

	// telemetry!(
	// 	self.telemetry;
	// 	CONSENSUS_INFO;
	// 	"afg.authority_set";
	// 	"number" => ?chain_info.finalized_number,
	// 	"hash" => ?chain_info.finalized_hash,
	// 	"authority_id" => authority_id,
	// 	"authority_set_id" => ?self.env.set_id,
	// 	"authorities" => authorities,
	// );
	// TODO: telemetry afg.authority_set

	_ = authorityID
	_ = chainInfo

	vw.env.VoterSetState.innerMtx.RLock()
	defer vw.env.VoterSetState.innerMtx.RUnlock()

	switch vss := vw.env.VoterSetState.inner.(type) {
	case voterSetStateLive[H, N]:
		// let last_finalized = (chain_info.finalized_hash, chain_info.finalized_number);
		var lastFinalized = grandpa.HashNumber[H, N]{
			Hash:   chainInfo.FinalizedHash,
			Number: chainInfo.FinalizedNumber,
		}
		_ = lastFinalized
		// let global_comms = global_communication(
		// 	self.env.set_id,
		// 	&self.env.voters,
		// 	self.env.client.clone(),
		// 	&self.env.network,
		// 	self.env.config.keystore.as_ref(),
		// 	self.metrics.as_ref().map(|m| m.until_imported.clone()),
		// );
		globalIn, globalOut := globalCommunication(
			primitives.SetID(vw.env.SetID),
			vw.env.Voters,
			vw.env.Client,
			vw.env.Network,
			vw.env.Config.KeyStore,
		)

		// let last_completed_round = completed_rounds.last();
		lastCompletedRound := vss.completedRounds().last()

		// let voter = voter::Voter::new(
		// 	self.env.clone(),
		// 	(*self.env.voters).clone(),
		// 	global_comms,
		// 	last_completed_round.number,
		// 	last_completed_round.votes.clone(),
		// 	last_completed_round.base,
		// 	last_finalized,
		// );
		votes := make([]grandpa.SignedMessage[H, N, primitives.AuthoritySignature, primitives.AuthorityID], len(lastCompletedRound.Votes))
		for i, vote := range lastCompletedRound.Votes {
			votes[i] = grandpa.SignedMessage[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
				Signature: vote.Signature,
				Message:   vote.Message,
				ID:        vote.ID,
			}
		}
		voter := grandpa.NewVoter[H, N, primitives.AuthoritySignature, primitives.AuthorityID](
			vw.env,
			vw.env.Voters,
			globalIn,
			globalOut.preSend,
			uint64(lastCompletedRound.Number),
			votes,
			lastCompletedRound.Base,
			lastFinalized,
		)

		// // Repoint shared_voter_state so that the RPC endpoint can query the state
		// if self.shared_voter_state.reset(voter.voter_state()).is_none() {
		// 	info!(
		// 		target: LOG_TARGET,
		// 		"Timed out trying to update shared GRANDPA voter state. \
		// 		RPC endpoints may return stale data."
		// 	);
		// }
		vw.sharedVoterState.reset(voter.VoterState())

		// self.voter = Box::pin(voter);
		vw.voter = voter
		errChan := make(chan error)
		go func() {
			err := voter.Start()
			errChan <- err
			close(errChan)
		}()
		vw.voterErrChan = errChan
	case voterSetStatePaused[H, N]:
		// VoterSetState::Paused { .. } => self.voter = Box::pin(future::pending()),
		// TODO: don't do anything? I dunno
	default:
		panic("unreachable")
	}
}

// fn handle_voter_command(
//
//	&mut self,
//	command: VoterCommand<Block::Hash, NumberFor<Block>>,
//
// ) -> Result<(), Error> {
func (vw *voterWork[H, N, Hasher, Header, E]) handleVoterCommand(command voterCommand) error {
	// 	match command {
	switch command := command.(type) {
	// 		VoterCommand::ChangeAuthorities(new) => {
	case voterCommandChangeAuthorities[H, N]:
		new := command
		// 			let voters: Vec<String> =
		// 				new.authorities.iter().map(move |(a, _)| format!("{}", a)).collect();
		// 			telemetry!(
		// 				self.telemetry;
		// 				CONSENSUS_INFO;
		// 				"afg.voter_command_change_authorities";
		// 				"number" => ?new.canon_number,
		// 				"hash" => ?new.canon_hash,
		// 				"voters" => ?voters,
		// 				"set_id" => ?new.set_id,
		// 			);
		// TODO: telemetry

		// 			self.env.update_voter_set_state(|_| {
		// 				// start the new authority set using the block where the
		// 				// set changed (not where the signal happened!) as the base.
		// 				let set_state = VoterSetState::live(
		// 					new.set_id,
		// 					&*self.env.authority_set.inner(),
		// 					(new.canon_hash, new.canon_number),
		// 				);

		// 				aux_schema::write_voter_set_state(&*self.env.client, &set_state)?;
		// 				Ok(Some(set_state))
		// 			})?;
		err := vw.env.updateVoterSetState(func(voterSetState voterSetState[H, N]) (voterSetState[H, N], error) {
			// start the new authority set using the block where the
			// set changed (not where the signal happened!) as the base.
			// vw.env.AuthoritySet.mtx.Lock()
			authoritySet, unlock := vw.env.AuthoritySet.inner.DataMut()
			defer unlock()
			setState := newVoterSetStateLive(
				primitives.SetID(vw.env.SetID),
				*authoritySet,
				grandpa.HashNumber[H, N]{
					Hash:   command.CanonHash,
					Number: command.CanonNumber,
				},
			)
			// vw.env.AuthoritySet.mtx.Unlock()
			err := writeVoterSetState(vw.env.Client, &setState)
			if err != nil {
				return nil, err
			}
			return setState, nil
		})
		if err != nil {
			return err
		}

		// 			let voters = Arc::new(VoterSet::new(new.authorities.into_iter()).expect(
		// 				"new authorities come from pending change; pending change comes from \
		// 				 `AuthoritySet`; `AuthoritySet` validates authorities is non-empty and \
		// 				 weights are non-zero; qed.",
		// 			));
		authorites := make([]grandpa.IDWeight[primitives.AuthorityID], len(new.Authorities))
		for i, authority := range new.Authorities {
			authorites[i] = grandpa.IDWeight[primitives.AuthorityID]{
				ID:     authority.AuthorityID,
				Weight: uint64(authority.AuthorityWeight),
			}
		}
		voters := grandpa.NewVoterSet(authorites)
		if voters == nil {
			panic("new authorities come from pending change; pending change comes from AuthoritySet; AuthoritySet validates authorities is non-empty and weights are non-zero")
		}

		// 			self.env = Arc::new(Environment {
		// 				voters,
		// 				set_id: new.set_id,
		// 				voter_set_state: self.env.voter_set_state.clone(),
		// 				client: self.env.client.clone(),
		// 				select_chain: self.env.select_chain.clone(),
		// 				config: self.env.config.clone(),
		// 				authority_set: self.env.authority_set.clone(),
		// 				network: self.env.network.clone(),
		// 				voting_rule: self.env.voting_rule.clone(),
		// 				metrics: self.env.metrics.clone(),
		// 				justification_sender: self.env.justification_sender.clone(),
		// 				telemetry: self.telemetry.clone(),
		// 				offchain_tx_pool_factory: self.env.offchain_tx_pool_factory.clone(),
		// 				_phantom: PhantomData,
		// 			});
		vw.env = &environment[H, N, Hasher, Header, E]{
			Voters:              *voters,
			SetID:               SetID(new.SetID),
			VoterSetState:       vw.env.VoterSetState,
			Client:              vw.env.Client,
			SelectChain:         vw.env.SelectChain,
			Config:              vw.env.Config,
			AuthoritySet:        vw.env.AuthoritySet,
			Network:             vw.env.Network,
			VotingRule:          vw.env.VotingRule,
			JustificationSender: vw.env.JustificationSender,
		}

		// 			self.rebuild_voter();
		vw.rebuildVoter()
		// 			Ok(())
		return nil
		// 		},
		// 		VoterCommand::Pause(reason) => {
	case voterCommandPause:
		// 			info!(target: LOG_TARGET, "Pausing old validator set: {}", reason);
		logger.Infof("Pausing old validator set: %s", string(command))

		// not racing because old voter is shut down.
		// 			self.env.update_voter_set_state(|voter_set_state| {
		err := vw.env.updateVoterSetState(func(voterSetState voterSetState[H, N]) (voterSetState[H, N], error) {
			// 				let completed_rounds = voter_set_state.completed_rounds();
			// 				let set_state = VoterSetState::Paused { completed_rounds };
			completedRounds := voterSetState.completedRounds()
			setState := voterSetStatePaused[H, N]{CompletedRounds: completedRounds}
			// 				aux_schema::write_voter_set_state(&*self.env.client, &set_state)?;
			err := writeVoterSetState(vw.env.Client, setState)
			if err != nil {
				return nil, err
			}
			// 				Ok(Some(set_state))
			return setState, nil
			// 			})?;
		})
		if err != nil {
			return err
		}

		//			self.rebuild_voter();
		vw.rebuildVoter()
		//			Ok(())
		return nil
		//		},
	default:
		panic("unreachable")
	}
}

// fn poll(mut self: Pin<&mut Self>, cx: &mut Context) -> Poll<Self::Output> {
func (vw *voterWork[H, N, Hasher, Header, E]) poll() error {
	// 	match Future::poll(Pin::new(&mut self.voter), cx) {
	// 		Poll::Pending => {},
	// 		Poll::Ready(Ok(())) => {
	// 			// voters don't conclude naturally
	// 			return Poll::Ready(Err(Error::Safety(
	// 				"consensus-grandpa inner voter has concluded.".into(),
	// 			)))
	// 		},
	// 		Poll::Ready(Err(CommandOrError::Error(e))) => {
	// 			// return inner observer error
	// 			return Poll::Ready(Err(e))
	// 		},
	// 		Poll::Ready(Err(CommandOrError::VoterCommand(command))) => {
	// 			// some command issued internally
	// 			self.handle_voter_command(command)?;
	// 			cx.waker().wake_by_ref();
	// 		},
	// 	}
	select {
	case err := <-vw.voterErrChan:
		if err == nil {
			// voters don't conclude naturally
			return fmt.Errorf("consensus-grandpa inner voter has concluded: %w", ErrSafety)
		}
		vc, isVoterCommand := err.(voterCommand)
		if !isVoterCommand {
			// return inner observer error
			return err
		}
		// some command issued internally
		return vw.handleVoterCommand(vc)
	default:
	}

	// 	match Stream::poll_next(Pin::new(&mut self.voter_commands_rx), cx) {
	// 		Poll::Pending => {},
	// 		Poll::Ready(None) => {
	// 			// the `voter_commands_rx` stream should never conclude since it's never closed.
	// 			return Poll::Ready(Err(Error::Safety("`voter_commands_rx` was closed.".into())))
	// 		},
	// 		Poll::Ready(Some(command)) => {
	// 			// some command issued externally
	// 			self.handle_voter_command(command)?;
	// 			cx.waker().wake_by_ref();
	// 		},
	// 	}
	select {
	case vc, ok := <-vw.voterCommandsRx:
		if !ok {
			// the `voter_commands_rx` stream should never conclude since it's never closed.
			return fmt.Errorf("`%w: voter_commands_rx` was closed", ErrSafety)
		}
		// some command issued externally
		return vw.handleVoterCommand(vc)
	default:
	}
	return nil
}

func (vw *voterWork[H, N, Hasher, Header, E]) run() error {
	for {
		err := vw.poll()
		if err != nil {
			return err
		}
	}
}

// Checks if this node has any available keys in the keystore for any authority id in the givenvoter set.  Returns the
// authority id for which keys are available, or nil if no keys are available.
func localAuthorityID(voters grandpa.VoterSet[primitives.AuthorityID], ks keystore.KeyStore) *primitives.AuthorityID {
	if ks == nil {
		return nil
	}

	for _, voter := range voters.Voters() {
		if ks.HasKeys([]keystore.PublicKey{{
			Key:       voter.ID.Bytes(),
			KeyTypeID: crypto.GRANDPA,
		}}) {
			return &voter.ID
		}
	}
	return nil
}
