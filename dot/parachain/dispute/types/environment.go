package types

import "github.com/ChainSafe/gossamer/lib/parachain"

type CandidateEnvironment struct {
	SessionIndex      parachain.SessionIndex
	Session           parachain.SessionInfo
	ControlledIndices map[parachain.ValidatorIndex]struct{}
}

func NewCandidateEnvironment(sessionIndex parachain.SessionIndex, session parachain.SessionInfo, controlledIndices map[parachain.ValidatorIndex]struct{}) CandidateEnvironment {
	return CandidateEnvironment{
		SessionIndex:      sessionIndex,
		Session:           session,
		ControlledIndices: controlledIndices,
	}
}
