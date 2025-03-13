// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/log"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// / A global communication input stream for commits and catch up messages. Not
// / exposed publicly, used internally to simplify types in the communication
// / layer.
// type communicationIn<Block> = voter::communicationIn<
//
//	<Block as BlockT>::Hash,
//	NumberFor<Block>,
//	AuthoritySignature,
//	AuthorityId,
//
// >;
type communicationIn[H runtime.Hash, N runtime.Number] grandpa.CommunicationIn[H, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]

// / Global communication sink for commits with the hash type not being derived
// / from the block, useful for forcing the hash to some type (e.g. `H256`) when
// / the compiler can't do the inference.
// type CommunicationOutH<Block, H> =
//
//	voter::communicationOut<H, NumberFor<Block>, AuthoritySignature, AuthorityId>;
type communicationOut[H runtime.Hash, N runtime.Number] grandpa.CommunicationOut[H, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]

// newAuthoritySet A new authority set along with the canonical block it changed at.
type newAuthoritySet[H, N any] struct {
	CanonNumber N
	CanonHash   H
	SetID       pgrandpa.SetID
	Authorities pgrandpa.AuthorityList
}

// type SharedVoterState[AuthorityID comparable] struct {
// 	inner grandpa.VoterState[AuthorityID]
// 	sync.Mutex
// }

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

// 		*shared_voter_state = Some(voter_state);
// 		Some(())
// 	}

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

// / A trait that includes all the client functionalities grandpa requires.
// / Ideally this would be a trait alias, we're not there yet.
// / tracking issue <https://github.com/rust-lang/rust/issues/41517>
// pub trait ClientForGrandpa<Block, BE>:
//
//	LockImportRun<Block, BE>
//	+ Finalizer<Block, BE>
//	+ AuxStore
//	+ HeaderMetadata<Block, Error = sp_blockchain::Error>
//	+ HeaderBackend<Block>
//	+ BlockchainEvents<Block>
//	+ ProvideRuntimeApi<Block>
//	+ ExecutorProvider<Block>
//	+ BlockImport<Block, Transaction = TransactionFor<BE, Block>, Error = sp_consensus::Error>
//	+ StorageProvider<Block, BE>
//
// where
//
//	BE: Backend<Block>,
//	Block: BlockT,
//
// // {}
// type ClientForGrandpa[
// 	H runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[H],
// 	Header runtime.Header[N, H],
// 	E runtime.Extrinsic,
// ] interface {
// 	api.LockImportRun[H, N, Hasher, Header, E]
// 	api.Finalizer[H, N, Hasher, Header, E]
// 	api.AuxStore
// 	blockchain.HeaderMetadata[H, N]
// 	blockchain.HeaderBackend[H, N, Header]
// 	api.BlockchainEvents[H, N, Header]
// 	papi.ProvideRuntimeAPI[H, N, Hasher, E]
// 	api.ExecutorProvider[H, N]
// 	consensus.BlockImport[H, N]
// 	api.StorageProvider[H, N, Hasher]
// }

type Config struct {
	/// The expected duration for a message to be gossiped across the network.
	GossipDuration time.Duration
	/// Justification generation period (in blocks). GRANDPA will try to generate justifications
	/// at least every justification_period blocks. There are some other events which might cause
	/// justification generation.
	JustificationGenerationPeriod uint32
	/// Whether the GRANDPA observer protocol is live on the network and thereby
	/// a full-node not running as a validator is running the GRANDPA observer
	/// protocol (we will only issue catch-up requests to authorities when the
	/// observer protocol is enabled).
	ObserverEnabled bool
	/// The role of the local node (i.e. authority, full-node or light).
	LocalRole role.Role
	/// Some local identifier of the voter.
	Name *string
	/// The keystore that manages the keys of this node.
	KeyStore keystore.KeyStore // can be nil for optionality
	/// TelemetryHandle instance.
	// TODO: telemetry
	// Telemetry *telemetry.TelemetryHandle
	/// Chain specific GRANDPA protocol name. See [`crate::protocol_standard_name`].
	ProtocolName network.ProtocolName
}

func (c Config) name() string {
	if c.Name == nil {
		return "<unknown>"
	}
	return *c.Name
}

// type voterCommands[H comparable, N constraints.Unsigned] interface {
// 	voterCommandPause | voterCommandChangeAuthorities[H, N]
// }
// type voterCommandPause string

// func (voterCommandPause) isVoterCommand() {}

// type voterCommandChangeAuthorities[H comparable, N constraints.Unsigned] newAuthoritySet[H, N]

// func (voterCommandChangeAuthorities[H, N]) isVoterCommand() {}

// // / Commands issued to the voter.
// type voterCommand[H runtime.Hash, N runtime.Number] interface {
// 	isVoterCommand()
// }

// type communicationInVoterCommandError[H runtime.Hash, N runtime.Number] struct {
// 	communicationIn[H, N]
// 	VoterCommand voterCommand[H, N]
// 	Error        error
// }
// type communicationOutVoterCommandError[H runtime.Hash, N runtime.Number] struct {
// 	communicationOut[H, N]
// 	VoterCommand voterCommand[H, N]
// 	Error        error
// }

// func globalCommunication[
// 	H runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[H],
// 	Header runtime.Header[N, H],
// 	E runtime.Extrinsic,
// ](
// 	setID pgrandpa.SetID,
// 	voters grandpa.VoterSet[pgrandpa.AuthorityID],
// 	client ClientForGrandpa[H, N, Hasher, Header, E],
// 	network networkBridge[H, N, Hasher],
// 	keystore keystore.KeyStore, // can be nil
// 	metrics any,
// ) (chan communicationInVoterCommandError[H, N], chan communicationOutVoterCommandError[H, N]) {
// 	var isVoter bool
// 	authID := localAuthorityID(voters, keystore)
// 	if authID != nil {
// 		isVoter = true
// 	}

// 	// verification stream
// 	// network.
// 	_ = isVoter
// 	panic("unimpl")
// }

// // / Future that powers the voter.
// type voterWork[
// 	H runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[H],
// 	Header runtime.Header[N, H],
// 	E runtime.Extrinsic,
// ] struct {
// 	// use string for AuthorityID and AuthoritySignature
// 	voter            *grandpa.Voter[H, N, string, string]
// 	sharedVoterState SharedVoterState[string]
// 	env              environment[H, N, Hasher, Header, E]
// 	voterCommandsRx  <-chan voterCommand[H, N]
// 	network          networkBridge[H, N, Hasher]
// 	telemetry        *telemetry.TelemetryHandle
// 	metrics          *metrics
// }

// func newVoterWork[
// 	H runtime.Hash,
// 	N runtime.Number,
// 	Hasher runtime.Hasher[H],
// 	Header runtime.Header[N, H],
// 	E runtime.Extrinsic,
// ](
// 	client ClientForGrandpa[H, N, Hasher, Header, E],
// 	config Config,
// 	network networkBridge[H, N, Hasher],
// 	selectChain consensus.SelectChain[H, N],
// 	votingRule VotingRule[H, N, Header],
// 	persistentData persistentData[H, N],
// 	voterCommandsRx <-chan voterCommand[H, N],
// 	prometheusRegistry prometheus.Registry,
// 	sharedVoterState SharedVoterState[string],
// 	justificationSender GrandpaJustificationSender[H, N],
// 	telemetry *telemetry.TelemetryHandle,
// ) voterWork[H, N, Hasher, Header, E] {
// 	// TODO: register to prometheus registry

// 	voters := persistentData.authoritySet.CurrentAuthorities()
// 	env := environment[H, N, Hasher, Header, E]{
// 		Client:              client,
// 		SelectChain:         selectChain,
// 		VotingRule:          votingRule,
// 		Voters:              voters,
// 		Config:              config,
// 		Network:             network,
// 		SetID:               pgrandpa.SetID(persistentData.authoritySet.inner.SetID),
// 		AuthoritySet:        persistentData.authoritySet,
// 		VoterSetState:       persistentData.setState,
// 		Metrics:             nil, // TOOD: use metrics
// 		JustificationSender: &justificationSender,
// 		Telemetry:           telemetry,
// 	}

// 	work := voterWork[H, N, Hasher, Header, E]{
// 		// `voter` is set to a temporary value and replaced below when
// 		// calling `rebuild_voter`.
// 		voter:            nil,
// 		sharedVoterState: sharedVoterState,
// 		env:              env,
// 		voterCommandsRx:  voterCommandsRx,
// 		network:          network,
// 		telemetry:        telemetry,
// 		metrics:          nil,
// 	}
// 	work.rebuildVoter()
// 	return work
// }

// // / Rebuilds the `self.voter` field using the current authority set
// // / state. This method should be called when we know that the authority set
// // / has changed (e.g. as signalled by a voter command).
// func (vw *voterWork[H, N, Hasher, Header, E]) rebuildVoter() {
// 	// debug!(
// 	// 	target: LOG_TARGET,
// 	// 	"{}: Starting new voter with set ID {}",
// 	// 	self.env.config.name(),
// 	// 	self.env.set_id
// 	// );
// 	logger.Debugf("%s: Starting new voter with set ID %v", vw.env.Config.name(), vw.env.SetID)

// 	maybeAuthorityID := localAuthorityID(vw.env.Voters, vw.env.Config.KeyStore)
// 	var authorityID string
// 	if maybeAuthorityID != nil {
// 		authorityID = string(maybeAuthorityID.Bytes())
// 	} else {
// 		authorityID = "<unknown>"
// 	}

// 	// telemetry!(
// 	// 	self.telemetry;
// 	// 	CONSENSUS_DEBUG;
// 	// 	"afg.starting_new_voter";
// 	// 	"name" => ?self.env.config.name(),
// 	// 	"set_id" => ?self.env.set_id,
// 	// 	"authority_id" => authority_id,
// 	// );

// 	chainInfo := vw.env.Client.Info()

// 	// let authorities = self.env.voters.iter().map(|(id, _)| id.to_string()).collect::<Vec<_>>();

// 	// let authorities = serde_json::to_string(&authorities).expect(
// 	// 	"authorities is always at least an empty vector; elements are always of type string; qed.",
// 	// );

// 	// telemetry!(
// 	// 	self.telemetry;
// 	// 	CONSENSUS_INFO;
// 	// 	"afg.authority_set";
// 	// 	"number" => ?chain_info.finalized_number,
// 	// 	"hash" => ?chain_info.finalized_hash,
// 	// 	"authority_id" => authority_id,
// 	// 	"authority_set_id" => ?self.env.set_id,
// 	// 	"authorities" => authorities,
// 	// );
// 	_ = authorityID
// 	_ = chainInfo

// 	vw.env.VoterSetState.innerMtx.RLock()
// 	defer vw.env.VoterSetState.innerMtx.RUnlock()

// 	switch vw.env.VoterSetState.inner.(type) {
// 	case voterSetStateLive[H, N]:

// 	case voterSetStatePaused[H, N]:
// 	default:
// 		panic("unreachable")
// 	}

// }

// // / Checks if this node has any available keys in the keystore for any authority id in the given
// // / voter set.  Returns the authority id for which keys are available, or `None` if no keys are
// // / available.
// func localAuthorityID(voters grandpa.VoterSet[pgrandpa.AuthorityID], keystore keystore.KeyStore) *pgrandpa.AuthorityID {
// 	if keystore == nil {
// 		return nil
// 	}
// 	for _, idVoterInfo := range voters.Voters() {
// 		p := idVoterInfo.ID
// 		publicKeys := []struct {
// 			Key []byte
// 			crypto.KeyTypeID
// 		}{
// 			{
// 				Key:       []byte(p),
// 				KeyTypeID: crypto.GRANDPA,
// 			},
// 		}
// 		if keystore.HasKeys(publicKeys) {
// 			authID, err := pgrandpa.NewAuthorityID([]byte(p))
// 			if err != nil {
// 				return nil
// 			}
// 			return &authID
// 		}
// 	}
// 	return nil
// }
