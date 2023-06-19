package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func getDummyHash(num byte) common.Hash {
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func TestStatement(t *testing.T) {
	t.Parallel()

	var collatorID CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature CollatorSignature
	tempSignature := common.MustHexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll
	copy(collatorSignature[:], tempSignature)

	hash5 := getDummyHash(5)

	secondedEnumValue := Seconded{
		Descriptor: CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          ValidationCodeHash(hash5),
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

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte
	}{
		{
			name:          "Seconded",
			enumValue:     secondedEnumValue,
			encodingValue: common.MustHexToBytes(expectedSDMHex["statementSeconded"]),
			// expected Hex stored in statement_distribution_message.yaml
		},
		{
			name:          "Valid",
			enumValue:     Valid{hash5},
			encodingValue: common.MustHexToBytes("0x020505050505050505050505050505050505050505050505050505050505050505"),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			vtd := NewStatement()

			err := vtd.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(vtd)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}
