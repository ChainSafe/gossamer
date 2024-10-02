package messages

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Backed is a statement distribution message.
// it represents a message indicating that a candidate has received sufficient
// validity votes from the backing group. If backed as a result of a local statement,
// it must be preceded by a `Share` message for that statement to ensure awareness of
// full candidates before the `Backed` notification, even in groups of size 1.
type Backed parachaintypes.CandidateHash

// Share is a statement distribution message.
// It is a signed statement in the context of
// given relay-parent hash and it should be distributed to other validators.
type Share struct {
	RelayParent                common.Hash
	SignedFullStatementWithPVD parachaintypes.SignedFullStatementWithPVD
}
