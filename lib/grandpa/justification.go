package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func DecodeAndVerifyFinalizes(
	encodedJustification []byte, finalizedTarget *types.Header, setID uint64, voters []types.GrandpaVoter) (
	justification Justification, err error) {
	justification = Justification{}
	err = scale.Unmarshal(encodedJustification, &justification)
	if err != nil {
		return justification, fmt.Errorf("while decoding justification: %w", err)
	}

	if finalizedTarget.Hash() != justification.Commit.Hash || finalizedTarget.Number != uint(justification.Commit.Number) {
		return justification, fmt.Errorf("%w: justification %s and block hash %s",
			ErrJustificationMismatch, justification.Commit.Hash, finalizedTarget.Hash())
	}

	err = verifyWithVoterSet(justification, setID, voters)
	if err != nil {
		return justification, fmt.Errorf("while verifying with voter set: %w", err)
	}

	return justification, nil
}

func verifyWithVoterSet(justification Justification, setID uint64, voters []types.GrandpaVoter) error {

}
