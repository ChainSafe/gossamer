package types

import parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"

type CandidateEnvironment struct {
	SessionIndex      parachainTypes.SessionIndex
	Session           parachainTypes.SessionInfo
	ControlledIndices map[parachainTypes.ValidatorIndex]struct{}
}

func NewCandidateEnvironment(sessionIndex parachainTypes.SessionIndex,
	session parachainTypes.SessionInfo,
	controlledIndices map[parachainTypes.ValidatorIndex]struct{},
) CandidateEnvironment {
	return CandidateEnvironment{
		SessionIndex:      sessionIndex,
		Session:           session,
		ControlledIndices: controlledIndices,
	}
}
