package backing

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

type CandidateBacking struct {
	SubSystemToOverseer <-chan any
	OverseerToSubSystem <-chan any
}

func New(overseerChan <-chan any) *CandidateBacking {
	return &CandidateBacking{
		SubSystemToOverseer: overseerChan,
	}
}

func (cb *CandidateBacking) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) error {
	// TODO: handle_validated_candidate_command
	// There is one more case where we handle results of candidate validation.
	// My feeling is that instead of doing it here, we would be able to do that along with processing
	// other backing related overseer message.
	// This would become more clear after we complete processMessages function. It would give us clarity
	// if we need background_validation_rx or background_validation_tx, as done in rust.
	cb.processMessages()
	return nil
}

func (cb *CandidateBacking) processMessages() {
	for msg := range cb.OverseerToSubSystem {
		// process these received messages by referenceing https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/core/backing/src/lib.rs#L741
		switch msg.(type) {
		case ActiveLeavesUpdate:
			// TODO: Implement this
		case GetBackedCandidates:
			// TODO: Implement this
		case CanSecond:
			// TODO: Implement this
		case Second:
			// TODO: Implement this
		case Statement:
			// TODO: Implement this
		default:
			logger.Error("unknown message type")
		}
	}
}

func (cb *CandidateBacking) handleActiveLeavesUpdate() {
	// TODO: Implement this
	// https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/core/backing/src/lib.rs#L347
}

// Messages from overseer

type ActiveLeavesUpdate struct {
	// TODO: Complete this struct
	// https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/subsystem-types/src/lib.rs#L153
}

// GetBackedCandidates is a message received from overseer that requests a set of backable
// candidates that could be backed in a child of the given relay-parent.
type GetBackedCandidates []struct {
	CandidateHash        parachaintypes.CandidateHash
	CandidateRelayParent common.Hash
}

// TODO: Complete this struct
// https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/subsystem-types/src/messages.rs#L88
type CanSecond struct {
}

// Second is a message received from overseer. Candidate Backing subsystem should second the given
// candidate in the context of the given relay parent. This candidate must be validated.
type Second struct {
	RelayParent      common.Hash
	CandidateReceipt parachaintypes.CandidateReceipt
	// TODO: Add PersistedValidationData
	PoV parachaintypes.PoV
}

// TODO: Complete this struct.
// Note a validator's statement about a particular candidate. Disagreements about validity must be escalated
// to a broader check by the Disputes Subsystem, though that escalation is deferred until the approval voting
// stage to guarantee availability. Agreements are simply tallied until a quorum is reached.
type Statement struct {
	RelayParent common.Hash
	// SignedFullStatement SignedFullStatement
}
