package types

import (
	"fmt"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
)

// CandidateEnvironment is the environment of a candidate.
type CandidateEnvironment struct {
	SessionIndex      parachainTypes.SessionIndex
	Session           parachainTypes.SessionInfo
	ControlledIndices map[parachainTypes.ValidatorIndex]struct{}
}

// NewCandidateEnvironment creates a new candidate environment.
func NewCandidateEnvironment(keystore keystore.Keystore,
	runtime parachain.RuntimeInstance,
	sessionIndex parachainTypes.SessionIndex,
	relayParent common.Hash,
) (*CandidateEnvironment, error) {
	sessionInfo, err := runtime.ParachainHostSessionInfo(relayParent, sessionIndex)
	if err != nil {
		return nil, fmt.Errorf("get session info: %w", err)
	}

	controlledIndices := getControlledIndices(keystore, sessionInfo.Validators)
	return &CandidateEnvironment{
		SessionIndex:      sessionIndex,
		Session:           *sessionInfo,
		ControlledIndices: controlledIndices,
	}, nil
}

func getControlledIndices(keystore keystore.Keystore,
	validators []parachainTypes.ValidatorID,
) map[parachainTypes.ValidatorIndex]struct{} {
	controlled := make(map[parachainTypes.ValidatorIndex]struct{})
	for index := range validators {
		if kp, err := GetValidatorKeyPair(keystore, validators, parachainTypes.ValidatorIndex(index)); kp != nil && err == nil {
			controlled[parachainTypes.ValidatorIndex(index)] = struct{}{}
		}
	}

	return controlled
}

func GetValidatorKeyPair(ks keystore.Keystore,
	validators []parachainTypes.ValidatorID,
	index parachainTypes.ValidatorIndex,
) (keystore.KeyPair, error) {
	if int(index) >= len(validators) {
		return nil, fmt.Errorf("invalid validator index: %d", index)
	}

	pubKey, err := sr25519.NewPublicKey(validators[index][:])
	if err != nil {
		return nil, fmt.Errorf("new public key: %w", err)
	}

	return ks.GetKeypairFromAddress(pubKey.Address()), nil
}
