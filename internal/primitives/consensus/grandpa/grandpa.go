package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// / Identity of a Grandpa authority.
// pub type AuthorityId = app::Public;
type AuthorityID = app.Public

// / Signature for a Grandpa authority.
// pub type AuthoritySignature = app::Signature;
type AuthoritySignature = app.Signature

// / The `ConsensusEngineId` of GRANDPA.
// pub const GRANDPA_ENGINE_ID: ConsensusEngineId = *b"FRNK";
var GrandpaEngineID = runtime.ConsensusEngineID{'F', 'R', 'N', 'K'}

// / The weight of an authority.
type AuthorityWeight uint64

// / The index of an authority.
type AuthorityIndex uint64

// / The monotonic identifier of a GRANDPA set of authorities.
// pub type SetId = u64;
type SetID uint64

// / The round indicator.
// pub type RoundNumber = u64;
type RoundNumber uint64

// / A list of Grandpa authorities with associated weights.
// pub type AuthorityList = Vec<(AuthorityId, AuthorityWeight)>;
type AuthorityList []struct {
	AuthorityID
	AuthorityWeight
}

// / A commit message for this chain's block type.
type Commit[H, N any] grandpa.Commit[H, N, AuthoritySignature, AuthorityID]

// / A GRANDPA justification for block finality, it includes a commit message and
// / an ancestry proof including all headers routing all precommit target blocks
// / to the commit target block. Due to the current voting strategy the precommit
// / targets should be the same as the commit target, since honest voters don't
// / vote past authority set change blocks.
// /
// / This is meant to be stored in the db and passed around the network to other
// / nodes, and are used by syncing nodes to prove authority set handoffs.
// #[derive(Clone, Encode, Decode, PartialEq, Eq, TypeInfo)]
// #[cfg_attr(feature = "std", derive(Debug))]
// pub struct GrandpaJustification<Header: HeaderT> {
type GrandpaJustification[H runtime.Hash, N runtime.Number] struct {
	// 	pub round: u64,
	Round uint64
	// 	pub commit: Commit<Header>,
	Commit Commit[H, N]
	// 	pub votes_ancestries: Vec<Header>,
	VoteAncestries []runtime.Header[N, H]
}

// / Check a message signature by encoding the message as a localized payload and
// / verifying the provided signature using the expected authority id.
func CheckMessageSignature[H comparable, N constraints.Unsigned](
	message grandpa.Message[H, N],
	id AuthorityID,
	signature AuthoritySignature,
	round RoundNumber,
	setID SetID) bool {

	buf := LocalizedPayload(round, setID, message)
	valid := id.Verify(buf, signature)

	if !valid {
		// if !valid {
		// 	let log_target = if cfg!(feature = "std") { CLIENT_LOG_TARGET } else { RUNTIME_LOG_TARGET };

		// 	log::debug!(target: log_target, "Bad signature on message from {:?}", id);
		// }
		logger.Debugf("Bad signature on message from %v", id)
	}
	return valid
}

// / Encode round message localized to a given round and set id using the given
// / buffer.
func LocalizedPayload(round RoundNumber, setID SetID, message any) []byte {
	return scale.MustMarshal(struct {
		Message any
		RoundNumber
		SetID
	}{message, round, setID})
}
