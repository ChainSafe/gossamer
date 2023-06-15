package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestStatementDistributionMessage(t *testing.T) {
	t.Parallel()

	var collatorSignature CollatorSignature
	tempSignature := common.MustHexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll
	copy(collatorSignature[:], tempSignature)

	var validatorSignature ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	var collatorID CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	hash1 := getDummyHash(5)

	statementWithValid := NewStatement()
	err := statementWithValid.Set(Valid{hash1})
	require.NoError(t, err)

	secondedEnumValue := Seconded{
		Descriptor: CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash1,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash1,
			PovHash:                     hash1,
			ErasureRoot:                 hash1,
			Signature:                   collatorSignature,
			ParaHead:                    hash1,
			ValidationCodeHash:          ValidationCodeHash(hash1),
		},
		Commitments: CandidateCommitments{
			UpwardMessages:            []UpwardMessage{{1, 2, 3}},
			HorizontalMessages:        []OutboundHrmpMessage{},
			NewValidationCode:         &ValidationCode{1, 2, 3},
			HeadData:                  headData{1, 2, 3},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	statementWithSeconded := NewStatement()
	err = statementWithSeconded.Set(secondedEnumValue)
	require.NoError(t, err)

	signedFullStatementWithValid := SignedFullStatement{
		Hash: hash1,
		UncheckedSignedFullStatement: UncheckedSignedFullStatement{
			Payload:        statementWithValid,
			ValidatorIndex: ValidatorIndex{5},
			Signature:      validatorSignature,
		},
	}

	signedFullStatementWithSeconded := SignedFullStatement{
		Hash: hash1,
		UncheckedSignedFullStatement: UncheckedSignedFullStatement{
			Payload:        statementWithSeconded,
			ValidatorIndex: ValidatorIndex{5},
			Signature:      validatorSignature,
		},
	}

	secondedStatementWithLargePayload := SecondedStatementWithLargePayload{
		RelayParent:   hash1,
		CandidateHash: CandidateHash{hash1},
		SignedBy:      ValidatorIndex{5},
		Signature:     validatorSignature,
	}
	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte
	}{
		{
			name:          "SignedFullStatement with valid statement",
			enumValue:     signedFullStatementWithValid,
			encodingValue: []byte{0, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 2, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 0, 0, 0, 198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184, 148, 207, 27, 134}, //nolint:lll
		},
		{
			name:          "SignedFullStatement with Seconded statement",
			enumValue:     signedFullStatementWithSeconded,
			encodingValue: []byte{0, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 1, 1, 0, 0, 0, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 72, 33, 91, 157, 50, 38, 1, 229, 177, 169, 81, 100, 206, 160, 220, 70, 38, 245, 69, 249, 131, 67, 208, 127, 21, 81, 235, 149, 67, 196, 177, 71, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184, 148, 207, 27, 134, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 4, 12, 1, 2, 3, 0, 1, 12, 1, 2, 3, 12, 1, 2, 3, 5, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184, 148, 207, 27, 134}, //nolint:lll
		},
		{
			name:          "SecondedStatementWithLargePayload",
			enumValue:     secondedStatementWithLargePayload,
			encodingValue: []byte{1, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 0, 0, 0, 198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184, 148, 207, 27, 134}, //nolint:lll
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			vtd := NewStatementDistributionMessage()

			err := vtd.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(vtd)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}
