// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/client/consensus"
	"github.com/ChainSafe/gossamer/client/consensus/grandpa/communication"
	"github.com/ChainSafe/gossamer/client/network"
	"github.com/ChainSafe/gossamer/client/network/role"
	"github.com/ChainSafe/gossamer/client/telemetry"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/primitives/runtime"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// Authority represents a grandpa authority
type Authority struct {
	Key    ed25519.PublicKey
	Weight uint64
}

// NewAuthoritySetStruct A new authority set along with the canonical block it changed at.
type NewAuthoritySetStruct[H comparable, N constraints.Unsigned] struct {
	CanonNumber N
	CanonHash   H
	SetId       N
	Authorities []Authority
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

type ClientForGrandpa interface{}

type Backend interface{}

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

func newVoterWork[Hash constraints.Ordered, Number runtime.Number, Signature comparable, ID constraints.Ordered](
	client ClientForGrandpa,
	config Config,
	network communication.NetworkBridge[Hash, Number],
	selectChain consensus.SelectChain,
	votingRule VotingRule,
	persistendData persistentData,
	voterCommandsRX any,
	prometheusRegistry any,
	sharedVoterState SharedVoterState,
	JustificationSender GrandpaJustificationSender,
	telemetry *telemetry.TelemetryHandle,
) *voterWork[Hash, Number, Signature, ID] {
	// grandpa.NewVoter[]()
	return nil
}
