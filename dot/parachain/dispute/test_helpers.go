package dispute

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func dummyCandidateCommitments() parachainTypes.CandidateCommitments {
	return parachainTypes.CandidateCommitments{
		UpwardMessages:            nil,
		HorizontalMessages:        nil,
		NewValidationCode:         nil,
		HeadData:                  parachainTypes.HeadData{},
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             0,
	}
}

func dummyValidationCode() parachainTypes.ValidationCode {
	return parachainTypes.ValidationCode{1, 2, 3}
}

func dummyCollator() parachainTypes.CollatorID {
	return parachainTypes.CollatorID{}
}

func dummyCollatorSignature() parachainTypes.CollatorSignature {
	return parachainTypes.CollatorSignature{}
}

func dummyCandidateDescriptorBadSignature(relayParent common.Hash) parachainTypes.CandidateDescriptor {
	zeros := common.Hash{}
	validationCodeHash, err := dummyValidationCode().Hash()
	if err != nil {
		panic(err)
	}

	return parachainTypes.CandidateDescriptor{
		ParaID:                      0,
		RelayParent:                 relayParent,
		Collator:                    dummyCollator(),
		PersistedValidationDataHash: zeros,
		PovHash:                     zeros,
		ErasureRoot:                 zeros,
		ParaHead:                    zeros,
		ValidationCodeHash:          validationCodeHash,
		Signature:                   dummyCollatorSignature(),
	}
}

func DummyCandidateReceipt(relayParent common.Hash) parachainTypes.CandidateReceipt {
	descriptor := parachainTypes.CandidateDescriptor{
		ParaID:                      0,
		RelayParent:                 relayParent,
		Collator:                    parachainTypes.CollatorID{},
		PersistedValidationDataHash: common.Hash{},
		PovHash:                     common.Hash{},
		ErasureRoot:                 common.Hash{},
		Signature:                   parachainTypes.CollatorSignature{},
		ParaHead:                    common.Hash{},
		ValidationCodeHash:          parachainTypes.ValidationCodeHash{},
	}

	return parachainTypes.CandidateReceipt{
		Descriptor:      descriptor,
		CommitmentsHash: common.Hash{},
	}
}

func dummyCandidateReceiptBadSignature(
	relayParent common.Hash,
	commitments *common.Hash,
) (parachainTypes.CandidateReceipt, error) {
	var (
		err             error
		commitmentsHash common.Hash
	)
	if commitments == nil {
		commitmentsHash, err = dummyCandidateCommitments().Hash()
		if err != nil {
			return parachainTypes.CandidateReceipt{}, err
		}
	} else {
		commitmentsHash = *commitments
	}

	return parachainTypes.CandidateReceipt{
		Descriptor:      dummyCandidateDescriptorBadSignature(relayParent),
		CommitmentsHash: commitmentsHash,
	}, nil
}

func activateLeaf(
	participation Participation,
	blockNumber parachainTypes.BlockNumber,
) error {
	encodedBlockNumber, err := scale.Marshal(blockNumber)
	if err != nil {
		return fmt.Errorf("failed to encode block number: %w", err)
	}
	parentHash, err := common.Blake2bHash(encodedBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to hash block number: %w", err)
	}

	blockHeader := types.Header{
		ParentHash:     parentHash,
		Number:         uint(blockNumber),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         scale.VaryingDataTypeSlice{},
	}
	blockHash := blockHeader.Hash()

	update := overseer.ActiveLeavesUpdate{
		Activated: &overseer.ActivatedLeaf{
			Hash:   blockHash,
			Number: uint32(blockNumber),
		},
	}

	participation.ProcessActiveLeavesUpdate(update)
	return nil
}

func GetBlockNumberHash(blockNumber parachainTypes.BlockNumber) common.Hash {
	encodedBlockNumber, err := scale.Marshal(blockNumber)
	if err != nil {
		panic("failed to encode block number:" + err.Error())
	}

	blockHash, err := common.Blake2bHash(encodedBlockNumber)
	if err != nil {
		panic("failed to hash block number:" + err.Error())
	}

	return blockHash
}

func DummyActivatedLeaf(blockNumber parachainTypes.BlockNumber) overseer.ActivatedLeaf {
	return overseer.ActivatedLeaf{
		Hash:   GetBlockNumberHash(blockNumber),
		Number: uint32(blockNumber),
	}
}

func NextLeaf(chain *[]common.Hash) overseer.ActivatedLeaf {
	nextBlockNumber := len(*chain)
	nextHash := GetBlockNumberHash(parachainTypes.BlockNumber(nextBlockNumber))
	*chain = append(*(chain), nextHash)
	return DummyActivatedLeaf(parachainTypes.BlockNumber(nextBlockNumber))
}
