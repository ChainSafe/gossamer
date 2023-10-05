package types

import (
	"fmt"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type CandidateEnvironment struct {
	SessionIndex      parachainTypes.SessionIndex
	Session           parachainTypes.SessionInfo
	ControlledIndices map[parachainTypes.ValidatorIndex]struct{}
}

func NewCandidateEnvironment(sessionIndex parachainTypes.SessionIndex,
	relayParent common.Hash,
	runtime parachain.RuntimeInstance,
) (*CandidateEnvironment, error) {
	sessionInfo, err := runtime.ParachainHostSessionInfo(relayParent, sessionIndex)
	if err != nil {
		return nil, fmt.Errorf("get session info: %w", err)
	}

	// TODO: get controlled indices

	return &CandidateEnvironment{
		SessionIndex:      sessionIndex,
		Session:           *sessionInfo,
		ControlledIndices: nil,
	}, nil
}
