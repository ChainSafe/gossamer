// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "primitives/consensus/grandpa"))

// AuthorityID is the identity of a Grandpa authority.
type AuthorityID = app.Public

// NewAuthorityID is constructor for AuthorityID
func NewAuthorityID(data []byte) (AuthorityID, error) {
	return app.NewPublic(data)
}

// AuthoritySignature is the signature for a Grandpa authority.
type AuthoritySignature = app.Signature

// GrandpaEngineID is the ConsensusEngineID of GRANDPA.
var GrandpaEngineID = runtime.ConsensusEngineID{'F', 'R', 'N', 'K'}

// AuthorityWeight is the weight of an authority.
type AuthorityWeight uint64

// AuthorityIndex is the index of an authority.
type AuthorityIndex uint64

// SetID is the monotonic identifier of a GRANDPA set of authorities.
type SetID uint64

// RoundNumber is the round indicator.
type RoundNumber uint64

// AuthorityIDWeight is struct containing AuthorityID and AuthorityWeight
type AuthorityIDWeight struct {
	AuthorityID AuthorityID // define attribute so SCALE doesn't call embedded AuthorityID.MarshalSCALE only
	AuthorityWeight
}

// AuthorityList is a list of Grandpa authorities with associated weights.
type AuthorityList []AuthorityIDWeight

// A GRANDPA message for a substrate chain.
type Message[H, N any] grandpa.Message[H, N]

// SignedMessage is a signed message.
type SignedMessage[H, N any] struct {
	grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID]
}

// A primary propose message for this chain's block type.
type PrimaryPropose[H, N any] grandpa.PrimaryPropose[H, N]

// A prevote message for this chain's block type.
type Prevote[H, N any] grandpa.Prevote[H, N]

// A precommit message for this chain's block type.
type Precommit[H, N any] grandpa.Precommit[H, N]

// A catch up message for this chain's block type.
type CatchUp[H, N any] grandpa.CatchUp[H, N, AuthoritySignature, AuthorityID]

// Commit is a commit message for this chain's block type.
type Commit[H, N any] grandpa.Commit[H, N, AuthoritySignature, AuthorityID]

// A compact commit message for this chain's block type.
type CompactCommit[H, N any] grandpa.CompactCommit[H, N, AuthoritySignature, AuthorityID]

// ScheduledChange is a scheduled authority change.
type ScheduledChange[N runtime.Number] struct {
	NextAuthorities AuthorityList
	Delay           N
}

// GrandpaJustification is A GRANDPA justification for block finality, it includes a commit message and an ancestry
// proof including all headers routing all precommit target blocks to the commit target block. Due to the current
// voting strategy the precommit targets should be the same as the commit target, since honest voters don't vote past
// authority set change blocks.
//
// This is meant to be stored in the db and passed around the network to other nodes, and are used by syncing nodes to
// prove authority set handoffs.
type GrandpaJustification[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	Round          uint64
	Commit         Commit[H, N]
	VoteAncestries []Header
}

// / An consensus log item for GRANDPA.
// #[derive(Decode, Encode, PartialEq, Eq, Clone, RuntimeDebug)]
// #[cfg_attr(feature = "serde", derive(Serialize))]
// pub enum ConsensusLog<N: Codec> {
type ConsensusLog interface {
	isConsensusLog()
}

// / Schedule an authority set change.
// /
// / The earliest digest of this type in a single block will be respected,
// / provided that there is no `ForcedChange` digest. If there is, then the
// / `ForcedChange` will take precedence.
// /
// / No change should be scheduled if one is already and the delay has not
// / passed completely.
// /
// / This should be a pure function: i.e. as long as the runtime can interpret
// / the digest type it should return the same result regardless of the current
// / state.
// #[codec(index = 1)]
// ScheduledChange(ScheduledChange<N>),
type ConensusLogScheduledChange[N runtime.Number] ScheduledChange[N]

// / Force an authority set change.
// /
// / Forced changes are applied after a delay of _imported_ blocks,
// / while pending changes are applied after a delay of _finalized_ blocks.
// /
// / The earliest digest of this type in a single block will be respected,
// / with others ignored.
// /
// / No change should be scheduled if one is already and the delay has not
// / passed completely.
// /
// / This should be a pure function: i.e. as long as the runtime can interpret
// / the digest type it should return the same result regardless of the current
// / state.
//
//	#[codec(index = 2)]
//	ForcedChange(N, ScheduledChange<N>),
type ConsensusLogForcedChange[N runtime.Number] struct {
	Median N
	ScheduledChange[N]
}

// / Note that the authority with given index is disabled until the next change.
//
//	#[codec(index = 3)]
//	OnDisabled(AuthorityIndex),
type ConsensusLogOnDisabled AuthorityIndex

// / A signal to pause the current authority set after the given delay.
// / After finalizing the block at _delay_ the authorities should stop voting.
// #[codec(index = 4)]
// Pause(N),
type ConsensusLogPause[N runtime.Number] struct {
	Delay N
}

// / A signal to resume the current authority set after the given delay.
// / After authoring the block at _delay_ the authorities should resume voting.
//
//	#[codec(index = 5)]
//	Resume(N),
type ConsensusLogResume[N runtime.Number] struct {
	Delay N
}

func (ConensusLogScheduledChange[N]) isConsensusLog() {}
func (ConsensusLogForcedChange[N]) isConsensusLog()   {}
func (ConsensusLogOnDisabled) isConsensusLog()        {}
func (ConsensusLogPause[N]) isConsensusLog()          {}
func (ConsensusLogResume[N]) isConsensusLog()         {}

type ConsensusLogVDT[N runtime.Number] struct {
	inner ConsensusLog
}

func (mvdt *ConsensusLogVDT[N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ConensusLogScheduledChange[N]:
		mvdt.inner = value
		return
	case ConsensusLogForcedChange[N]:
		mvdt.inner = value
		return
	case ConsensusLogOnDisabled:
		mvdt.inner = value
		return
	case ConsensusLogPause[N]:
		mvdt.inner = value
		return
	case ConsensusLogResume[N]:
		mvdt.inner = value
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ConsensusLogVDT[N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ConensusLogScheduledChange[N]:
		return 1, mvdt.inner, nil
	case ConsensusLogForcedChange[N]:
		return 2, mvdt.inner, nil
	case ConsensusLogOnDisabled:
		return 3, mvdt.inner, nil
	case ConsensusLogPause[N]:
		return 4, mvdt.inner, nil
	case ConsensusLogResume[N]:
		return 5, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ConsensusLogVDT[N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ConsensusLogVDT[N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return ConensusLogScheduledChange[N]{}, nil
	case 2:
		return ConsensusLogForcedChange[N]{}, nil
	case 3:
		return ConsensusLogOnDisabled(0), nil
	case 4:
		return ConsensusLogPause[N]{}, nil
	case 5:
		return ConsensusLogResume[N]{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// EquiovcationProof is proof of voter misbehavior on a given set id. Misbehavior/equivocation in GRANDPA happens when
// a voter votes on the same round (either at prevote or precommit stage) for different blocks. Proving is achieved by
// collecting the signed messages of conflicting votes.
type EquivocationProof[H runtime.Hash, N runtime.Number] struct {
	SetID        SetID
	Equivocation Equivocation
}

// NewEquivocationProof will create a new [EquivocationProof] for the given set id and using the given equivocation as
// proof.
func NewEquivocationProof[H runtime.Hash, N runtime.Number](
	setID SetID,
	equivocation Equivocation,
) EquivocationProof[H, N] {
	return EquivocationProof[H, N]{
		SetID:        setID,
		Equivocation: equivocation,
	}
}

// Equivocation is interface for GRANDPA equivocation proofs, useful for unifying prevote and precommit equivocations
// under a common type.
type Equivocation interface {
	Round() RoundNumber
	Offender() AuthorityID
}

// EquivocationPrevote is proof of equivocation at prevote stage.
type EquivocationPrevote[H runtime.Hash, N runtime.Number] grandpa.Equivocation[
	AuthorityID, grandpa.Prevote[H, N], AuthoritySignature]

func (ep EquivocationPrevote[H, N]) Round() RoundNumber {
	return RoundNumber(ep.RoundNumber)
}
func (ep EquivocationPrevote[H, N]) Offender() AuthorityID {
	return ep.Identity
}

// EquivocationPrecommit is proof of equivocation at precommit stage.
type EquivocationPrecommit[H runtime.Hash, N runtime.Number] grandpa.Equivocation[
	AuthorityID, grandpa.Precommit[H, N], AuthoritySignature]

func (ep EquivocationPrecommit[H, N]) Round() RoundNumber {
	return RoundNumber(ep.RoundNumber)
}
func (ep EquivocationPrecommit[H, N]) Offender() AuthorityID {
	return ep.Identity
}

// CheckMessageSignature will check a message signature by encoding the message as a localised payload and verifying
// the provided signature using the expected authority id.
func CheckMessageSignature[H comparable, N constraints.Unsigned](
	message grandpa.Message[H, N],
	id AuthorityID,
	signature AuthoritySignature,
	round RoundNumber,
	setID SetID) bool {

	buf := NewLocalizedPayload(round, setID, message)
	valid := id.Verify(signature, buf)

	if !valid {
		logger.Debugf("Bad signature on message from %v", id)
	}
	return valid
}

// NewLocalizedPayload will encode round message localised to a given round and set id.
func NewLocalizedPayload[H comparable, N constraints.Unsigned](
	round RoundNumber,
	setID SetID,
	message grandpa.Message[H, N],
) []byte {
	return scale.MustMarshal(struct {
		Message grandpa.MessageVDT[H, N]
		RoundNumber
		SetID
	}{grandpa.NewMessageVDT(message), round, setID})
}

// SignMessage localizes the message to the given set and round and signs the payload.
func SignMessage[H comparable, N constraints.Unsigned](
	keystore keystore.KeyStore,
	message grandpa.Message[H, N],
	public AuthorityID,
	round RoundNumber,
	setID SetID,
) *grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID] {
	encoded := NewLocalizedPayload(round, setID, message)
	signature, err := keystore.Ed25519Sign(crypto.GRANDPA, public[:], encoded)
	if err != nil {
		return nil
	}
	return &grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID]{
		Message:   message,
		Signature: *signature,
		ID:        public,
	}
}

// OpaqueKeyOwnershipProof is an opaque type used to represent the key ownership proof at the runtime API boundary.
// The inner value is an encoded representation of the actual key ownership proof which will be parameterized when
// defining the runtime. At the runtime API boundary this type is unknown and as such we keep this opaque
// representation, implementors of the runtime API will have to make sure that all usages of OpaqueKeyOwnershipProof
// refer to the same type.
type OpaqueKeyOwnershipProof = runtime.OpaqueValue

// APIs for integrating the GRANDPA finality gadget into runtimes. This should be implemented on the runtime side.
//
// This is primarily used for negotiating authority-set changes for the gadget. GRANDPA uses a signalling model of
// changing authority sets: changes should be signalled with a delay of N blocks, and then automatically applied in the
// runtime after those N blocks have passed.
//
// The consensus protocol will coordinate the handoff externally.
type GrandpaAPI[H runtime.Hash, N runtime.Number] interface {
	api.Core[H]
	// Get the current GRANDPA authorities and weights. This should not change except for when changes are scheduled
	// and the corresponding delay has passed.
	//
	// When called at block B, it will return the set of authorities that should be used to finalize descendants of
	// this block (B+1, B+2, ...). The block B itself is finalized by the authorities from block B-1.
	GrandpaAuthorities(hash H) (AuthorityList, error)

	// Submits an unsigned extrinsic to report an equivocation. The caller must provide the equivocation proof and a
	// key ownership proof (should be obtained using GenerateKeyOwnershipProof). The extrinsic will be unsigned
	// and should only be accepted for local authorship (not to be broadcast to the network). This method returns
	// nil when creation of the extrinsic fails, e.g. if equivocation reporting is disabled for the given runtime
	// (i.e. this method is hardcoded to return nil). Only useful in an offchain context.
	SubmitReportEquivocationUnsignedExtrinsic(
		hash H,
		equivocationProof EquivocationProof[H, N],
		keyOwnerProof OpaqueKeyOwnershipProof,
	) error

	// Generates a proof of key ownership for the given authority in the given set. An example usage of this module
	// is coupled with the session historical module to prove that a given authority key is tied to a given staking
	// identity during a specific session. Proofs of key ownership are necessary for submitting equivocation reports.
	//
	// NOTE: even though the API takes a setID as parameter the current implementations ignore this parameter and
	// instead rely on this method being called at the correct block height, i.e. any point at which the given set id
	// is live on-chain. Future implementations will instead use indexed data through an offchain worker, not requiring
	// older states to be available.
	GenerateKeyOwnershipProof(
		hash H,
		setID SetID,
		authorityID AuthorityID,
	) *OpaqueKeyOwnershipProof

	/// Get current GRANDPA authority set id.
	// fn current_set_id() -> SetId;
	CurrentSetID(hash H) (SetID, error)
}
