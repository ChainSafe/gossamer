package grandpa

import "github.com/ChainSafe/gossamer/dot/peerset"

// Costs/benefits that don't require calculation
var (
	pastRejection = peerset.ReputationChange{
		Value:  peerset.Reputation(-50),
		Reason: "Grandpa: Past message",
	}

	invalidViewChange = peerset.ReputationChange{
		Value:  peerset.Reputation(-500),
		Reason: "Grandpa: Invalid view change",
	}
)

// Ones that do implement the cost function

type BadCommitMessage struct {
	signaturesChecked   int
	blocksLoaded        int
	equivocationsCaught int
}

func (BadCommitMessage) cost() peerset.ReputationChange {
	// TODO implement
	return peerset.ReputationChange{}
}
