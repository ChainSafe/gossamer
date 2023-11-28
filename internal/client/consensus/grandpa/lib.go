// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa/communication"
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/telemetry"
	"github.com/ChainSafe/gossamer/internal/keystore"
	"github.com/ChainSafe/gossamer/internal/log"
	papi "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// GrandpaEngineID is the hard-coded grandpa ID
var GrandpaEngineID = ConsensusEngineID{'F', 'R', 'N', 'K'}

type AuthorityID interface {
	constraints.Ordered
	Verify(msg []byte, sig []byte) (bool, error)
}

type AuthoritySignature any

// Authority represents a grandpa authority
type Authority[ID AuthorityID] struct {
	Key    ID
	Weight uint64
}

type AuthorityList[ID AuthorityID] []Authority[ID]

// newAuthoritySet A new authority set along with the canonical block it changed at.
type newAuthoritySet[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	CanonNumber N
	CanonHash   H
	SetId       N
	Authorities []Authority[ID]
}

type messageData[H comparable, N constraints.Unsigned] struct {
	Round   uint64
	SetID   uint64
	Message grandpa.Message[H, N]
}

// Check a message signature by encoding the message as a localised payload and
// verifying the provided signature using the expected authority id.
// The encoding necessary to verify the signature will be done using the given
// buffer, the original content of the buffer will be cleared.
func checkMessageSignature[H comparable, N constraints.Unsigned, ID AuthorityID](
	message grandpa.Message[H, N],
	id ID,
	signature any,
	round uint64,
	setID uint64) (bool, error) {

	sig, ok := signature.([]byte)

	// Verify takes []byte, but string is a valid signature type,
	// so if signature is not already type []byte, check if it is a string
	sigString, okString := signature.(string)
	if !okString && !ok {
		sig = []byte(sigString)
	}

	m := messageData[H, N]{
		round,
		setID,
		message,
	}

	enc, err := scale.Marshal(m)
	if err != nil {
		return false, err
	}
	valid, err := id.Verify(enc, sig[:])
	if err != nil {
		return false, err
	}
	return valid, nil
}

type SharedVoterState[AuthorityID comparable] struct {
	inner grandpa.VoterState[AuthorityID]
	sync.Mutex
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
// {}
type ClientForGrandpa[R any, N runtime.Number, H statemachine.HasherOut] interface {
	api.LockImportRun[R, N, H]
	api.Finalizer[N, H]
	api.AuxStore
	blockchain.HeaderMetaData[H, N]
	blockchain.HeaderBackend[H, N]
	api.BlockchainEvents[H, N]
	papi.ProvideRuntimeAPI
	api.ExecutorProvider
	consensus.BlockImport[H, N]
	api.StorageProvider[H, N]
}

// / Commands issued to the voter.
type voterCommand any
type voterCommands[H comparable, N constraints.Unsigned, ID AuthorityID] interface {
	voterCommandPause | voterCommandChangeAuthorities[H, N, ID]
}
type voterCommandPause string
type voterCommandChangeAuthorities[H comparable, N constraints.Unsigned, ID AuthorityID] newAuthoritySet[H, N, ID]

type environment struct{}

type Config struct {
	/// The expected duration for a message to be gossiped across the network.
	GossipDuration time.Duration
	/// Justification generation period (in blocks). GRANDPA will try to generate justifications
	/// at least every justification_period blocks. There are some other events which might cause
	/// justification generation.
	JustificationPeriod uint32
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
	KeyStore *keystore.KeyStore
	/// TelemetryHandle instance.
	Telemetry *telemetry.TelemetryHandle
	/// Chain specific GRANDPA protocol name. See [`crate::protocol_standard_name`].
	ProtocolName network.ProtocolName
}

// / Future that powers the voter.
type voterWork[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] struct {
	voter            *grandpa.Voter[Hash, Number, Signature, ID]
	sharedVoterState SharedVoterState[ID]
	env              environment
	voterCommandsRx  any
	network          any
	telemetry        any
	metrics          any
}

func newVoterWork[Hash constraints.Ordered, Number runtime.Number, Signature comparable, ID AuthorityID, R any](
	client ClientForGrandpa[R, Number, Hash],
	config Config,
	network communication.NetworkBridge[Hash, Number],
	selectChain consensus.SelectChain[Hash, Number],
	votingRule VotingRule[Hash, Number],
	persistendData persistentData[Hash, Number, ID, Signature],
	voterCommandsRx <-chan voterCommand,
	prometheusRegistry prometheus.Registry,
	sharedVoterState SharedVoterState[ID],
	JustificationSender GrandpaJustificationSender[Hash, Number, Signature, ID],
	telemetry *telemetry.TelemetryHandle,
) *voterWork[Hash, Number, Signature, ID] {
	// grandpa.NewVoter[]()
	return nil
}
