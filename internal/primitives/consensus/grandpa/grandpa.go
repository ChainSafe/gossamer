// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/log"
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

// / Proof of voter misbehavior on a given set id. Misbehavior/equivocation in
// / GRANDPA happens when a voter votes on the same round (either at prevote or
// / precommit stage) for different blocks. Proving is achieved by collecting the
// / signed messages of conflicting votes.
// #[derive(Clone, Debug, Decode, Encode, PartialEq, Eq, TypeInfo)]
//
//	pub struct EquivocationProof<H, N> {
//		set_id: SetId,
//		equivocation: Equivocation<H, N>,
//	}
type EquiovcationProof[H runtime.Hash, N runtime.Number] struct {
	SetID        SetID
	Equivocation Equivocation
}

//	impl<H, N> EquivocationProof<H, N> {
//		/// Create a new `EquivocationProof` for the given set id and using the
//		/// given equivocation as proof.
//		pub fn new(set_id: SetId, equivocation: Equivocation<H, N>) -> Self {
//			EquivocationProof { set_id, equivocation }
//		}
func NewEquivocationProof[H runtime.Hash, N runtime.Number](
	setID SetID,
	equivocation Equivocation,
) EquiovcationProof[H, N] {
	return EquiovcationProof[H, N]{
		SetID:        setID,
		Equivocation: equivocation,
	}
}

// 	/// Returns the set id at which the equivocation occurred.
// 	pub fn set_id(&self) -> SetId {
// 		self.set_id
// 	}

// 	/// Returns the round number at which the equivocation occurred.
// 	pub fn round(&self) -> RoundNumber {
// 		match self.equivocation {
// 			Equivocation::Prevote(ref equivocation) => equivocation.round_number,
// 			Equivocation::Precommit(ref equivocation) => equivocation.round_number,
// 		}
// 	}

// 	/// Returns the authority id of the equivocator.
// 	pub fn offender(&self) -> &AuthorityId {
// 		self.equivocation.offender()
// 	}
// }

// / Wrapper object for GRANDPA equivocation proofs, useful for unifying prevote
// / and precommit equivocations under a common type.
// #[derive(Clone, Debug, Decode, Encode, PartialEq, Eq, TypeInfo)]
//
//	pub enum Equivocation<H, N> {
//		/// Proof of equivocation at prevote stage.
//		Prevote(
//			finality_grandpa::Equivocation<
//				AuthorityId,
//				finality_grandpa::Prevote<H, N>,
//				AuthoritySignature,
//			>,
//		),
//		/// Proof of equivocation at precommit stage.
//		Precommit(
//			finality_grandpa::Equivocation<
//				AuthorityId,
//				finality_grandpa::Precommit<H, N>,
//				AuthoritySignature,
//			>,
//		),
//	}
type Equivocation interface {
	Round() RoundNumber
	Offender() AuthorityID
}

// / Proof of equivocation at prevote stage.
type EquivocationPrevote[H runtime.Hash, N runtime.Number] grandpa.Equivocation[AuthorityID, grandpa.Prevote[H, N], AuthoritySignature]

func (ep EquivocationPrevote[H, N]) Round() RoundNumber {
	return RoundNumber(ep.RoundNumber)
}
func (ep EquivocationPrevote[H, N]) Offender() AuthorityID {
	return ep.Identity
}

// / Proof of equivocation at precommit stage.
type EquivocationPrecommit[H runtime.Hash, N runtime.Number] grandpa.Equivocation[AuthorityID, grandpa.Precommit[H, N], AuthoritySignature]

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

// LocalizedPayload will encode round message localised to a given round and set id.
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

// Localizes the message to the given set and round and signs the payload.
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

// / An opaque type used to represent the key ownership proof at the runtime API
// / boundary. The inner value is an encoded representation of the actual key
// / ownership proof which will be parameterized when defining the runtime. At
// / the runtime API boundary this type is unknown and as such we keep this
// / opaque representation, implementors of the runtime API will have to make
// / sure that all usages of `OpaqueKeyOwnershipProof` refer to the same type.
// pub type OpaqueKeyOwnershipProof = OpaqueValue;
type OpaqueKeyOwnershipProof = runtime.OpaqueValue

// / APIs for integrating the GRANDPA finality gadget into runtimes.
// / This should be implemented on the runtime side.
// /
// / This is primarily used for negotiating authority-set changes for the
// / gadget. GRANDPA uses a signaling model of changing authority sets:
// / changes should be signaled with a delay of N blocks, and then automatically
// / applied in the runtime after those N blocks have passed.
// /
// / The consensus protocol will coordinate the handoff externally.
// #[api_version(3)]
// pub trait GrandpaApi {
type GrandpaAPI[H runtime.Hash, N runtime.Number] interface {
	// 	/// Get the current GRANDPA authorities and weights. This should not change except
	// 	/// for when changes are scheduled and the corresponding delay has passed.
	// 	///
	// 	/// When called at block B, it will return the set of authorities that should be
	// 	/// used to finalize descendants of this block (B+1, B+2, ...). The block B itself
	// 	/// is finalized by the authorities from block B-1.
	// 	fn grandpa_authorities() -> AuthorityList;

	// 	/// Submits an unsigned extrinsic to report an equivocation. The caller
	// 	/// must provide the equivocation proof and a key ownership proof
	// 	/// (should be obtained using `generate_key_ownership_proof`). The
	// 	/// extrinsic will be unsigned and should only be accepted for local
	// 	/// authorship (not to be broadcast to the network). This method returns
	// 	/// `None` when creation of the extrinsic fails, e.g. if equivocation
	// 	/// reporting is disabled for the given runtime (i.e. this method is
	// 	/// hardcoded to return `None`). Only useful in an offchain context.
	// 	fn submit_report_equivocation_unsigned_extrinsic(
	// 		equivocation_proof: EquivocationProof<Block::Hash, NumberFor<Block>>,
	// 		key_owner_proof: OpaqueKeyOwnershipProof,
	// 	) -> Option<()>;
	SubmitReportEquivocationUnsignedExtrinsic(
		hash H,
		equivocationProof EquiovcationProof[H, N],
		keyOwnerProof OpaqueKeyOwnershipProof,
	) error

	// 	/// Generates a proof of key ownership for the given authority in the
	// 	/// given set. An example usage of this module is coupled with the
	// 	/// session historical module to prove that a given authority key is
	// 	/// tied to a given staking identity during a specific session. Proofs
	// 	/// of key ownership are necessary for submitting equivocation reports.
	// 	/// NOTE: even though the API takes a `set_id` as parameter the current
	// 	/// implementations ignore this parameter and instead rely on this
	// 	/// method being called at the correct block height, i.e. any point at
	// 	/// which the given set id is live on-chain. Future implementations will
	// 	/// instead use indexed data through an offchain worker, not requiring
	// 	/// older states to be available.
	// 	fn generate_key_ownership_proof(
	// 		set_id: SetId,
	// 		authority_id: AuthorityId,
	// 	) -> Option<OpaqueKeyOwnershipProof>;
	GenerateKeyOwnershipProof(
		hash H,
		setID SetID,
		authorityID AuthorityID,
	) *OpaqueKeyOwnershipProof

	// /// Get current GRANDPA authority set id.
	// fn current_set_id() -> SetId;
}
