package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
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

// / The monotonic identifier of a GRANDPA set of authorities.
// pub type SetId = u64;
type SetID uint64

// / The round indicator.
// pub type RoundNumber = u64;
type RoundNumber uint64

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
