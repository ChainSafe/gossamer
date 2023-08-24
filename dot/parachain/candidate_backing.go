package parachain

import (
	"errors"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var InvalidErasureRoot = errors.New("Invalid erasure root")

func ValidateAndMakeAvailable(
	nValidators uint,
	runtimeInstance parachainruntime.RuntimeInstance,
	povRequestor PoVRequestor,
	candidateReceipt parachaintypes.CandidateReceipt,
) error {

	// TODO: either use already available data (from candidate selection) if possible,
	// or request it from the validator.
	// https://github.com/paritytech/polkadot/blob/9b1fc27cec47f01a2c229532ee7ab79cc5bb28ef/node/core/backing/src/lib.rs#L697-L708
	pov := povRequestor.RequestPoV(candidateReceipt.Descriptor.PovHash) // temporary

	candidateCommitments, persistedValidationData, isValid, err := ValidateFromChainState(runtimeInstance, povRequestor, candidateReceipt)
	if err != nil {
		return err
	}

	if isValid {
		candidateHash := CandidateHash{common.MustBlake2bHash(scale.MustMarshal(candidateReceipt))}
		if err := MakePoVAvailable(
			nValidators,
			pov,
			candidateHash,
			*persistedValidationData,
			candidateReceipt.Descriptor.ErasureRoot,
		); err != nil {
			return err
		}
	}

	// TODO: If is not valid Report to collator protocol,
	// about the invalidity so that it can punish the collator that sent us this candidate

	return nil
}

func MakePoVAvailable(
	nValidators uint,
	pov PoV,
	candidateHash CandidateHash,
	validationData parachaintypes.PersistedValidationData,
	expectedErasureRoot common.Hash,
) error {
	availableData := AvailableData{pov, validationData}
	availableDataBytes, err := scale.Marshal(availableData)
	if err != nil {
		return err
	}

	chunks, err := erasure.ObtainChunks(nValidators, availableDataBytes)
	if err != nil {
		return err
	}

	chunksTrie, err := erasure.ChunksToTrie(chunks)
	if err != nil {
		return err
	}

	root, err := chunksTrie.Hash()
	if err != nil {
		return err
	}

	if root != expectedErasureRoot {
		return InvalidErasureRoot
	}

	// TODO: send a message to overseear to store the available data
	// https://github.com/paritytech/polkadot/blob/9b1fc27cec47f01a2c229532ee7ab79cc5bb28ef/node/core/backing/src/lib.rs#L566-L573

	return nil
}
