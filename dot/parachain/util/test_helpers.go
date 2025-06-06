package util

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

const MaxPoVSize = 1_000_000

var genesisHash = common.Hash{0}

// GetBlockHeader returns a block header for the given hash from the chain of hashes.
func GetBlockHeader(t *testing.T, chain []common.Hash, hash common.Hash) *types.Header {
	t.Helper()

	idx := -1
	for i, h := range chain {
		if h == hash {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}

	var parentHash common.Hash
	if idx > 0 {
		parentHash = chain[idx-1]
	} else {
		parentHash = genesisHash
	}

	var number uint
	if hash == genesisHash {
		number = 0
	} else {
		number = uint(idx) + 1
	}

	return &types.Header{
		ParentHash: parentHash,
		Number:     number,
	}
}

func DummyPVD(
	parentHead parachaintypes.HeadData,
	relayParentNumber uint32,
) parachaintypes.PersistedValidationData {
	return parachaintypes.PersistedValidationData{
		ParentHead:             parentHead,
		RelayParentNumber:      relayParentNumber,
		RelayParentStorageRoot: common.EmptyHash,
		MaxPovSize:             MaxPoVSize,
	}
}

func DummyCandidateReceiptBadSig(
	relayParentHash common.Hash,
	commitments *common.Hash,
) parachaintypes.CandidateReceipt {
	var commitmentsHash common.Hash

	if commitments != nil {
		commitmentsHash = *commitments
	} else {
		commitmentsHash = common.EmptyHash
	}

	descriptor := parachaintypes.CandidateDescriptor{
		ParaID:                      parachaintypes.ParaID(0),
		RelayParent:                 relayParentHash,
		Collator:                    parachaintypes.CollatorID{},
		PovHash:                     common.EmptyHash,
		ErasureRoot:                 common.EmptyHash,
		Signature:                   parachaintypes.CollatorSignature{},
		ParaHead:                    common.EmptyHash,
		ValidationCodeHash:          parachaintypes.ValidationCodeHash{},
		PersistedValidationDataHash: common.EmptyHash,
	}

	return parachaintypes.CandidateReceipt{
		CommitmentsHash: commitmentsHash,
		Descriptor:      descriptor,
	}
}

func MakeCandidate(
	relayParent common.Hash,
	relayParentNumber uint32,
	paraID parachaintypes.ParaID,
	parentHead parachaintypes.HeadData,
	headData parachaintypes.HeadData,
	validationCodeHash parachaintypes.ValidationCodeHash,
) parachaintypes.CommittedCandidateReceiptV2 {
	pvd := DummyPVD(parentHead, relayParentNumber)

	commitments := parachaintypes.CandidateCommitments{
		HeadData:                  headData,
		HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
		UpwardMessages:            []parachaintypes.UpwardMessage{},
		NewValidationCode:         nil,
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             relayParentNumber,
	}

	commitmentsHash := commitments.Hash()

	candidate := DummyCandidateReceiptBadSig(relayParent, &commitmentsHash)
	candidate.Descriptor.ParaID = paraID

	pvdh, err := pvd.Hash()
	if err != nil {
		panic(err)
	}

	candidate.Descriptor.PersistedValidationDataHash = pvdh
	candidate.Descriptor.ValidationCodeHash = validationCodeHash

	result := parachaintypes.CommittedCandidateReceipt{
		Descriptor:  candidate.Descriptor,
		Commitments: commitments,
	}

	return result.V2()
}
