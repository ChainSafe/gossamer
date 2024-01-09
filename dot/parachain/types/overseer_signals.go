package parachaintypes

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
)

var ErrUnknownOverseerMessage = errors.New("unknown overseer message type")

// ActivatedLeaf is a parachain head which we care to work on.
type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
}

// ActiveLeavesUpdateSignal changes in the set of active leaves:  the parachain heads which we care to work on.
//
// note: activated field indicates deltas, not complete sets.
type ActiveLeavesUpdateSignal struct {
	Activated *ActivatedLeaf
	// Relay chain block hashes no longer of interest.
	Deactivated []common.Hash
}

// BlockFinalized signal is used to inform subsystems of a finalized block.
type BlockFinalizedSignal struct {
	Hash        common.Hash
	BlockNumber uint32
}
