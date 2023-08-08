// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
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

	var collatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature parachaintypes.CollatorSignature
	tempSignature := common.MustHexToBytes(testDataStatement["collatorSignature"])
	copy(collatorSignature[:], tempSignature)

	hash5 := getDummyHash(5)

	secondedEnumValue := Seconded{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(hash5),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:    []parachaintypes.UpwardMessage{{1, 2, 3}},
			NewValidationCode: &parachaintypes.ValidationCode{1, 2, 3},
			HeadData: parachaintypes.HeadData{
				Data: []byte{1, 2, 3},
			},
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
			encodingValue: common.MustHexToBytes(testDataStatement["statementSeconded"]),
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
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewStatement()
				err := vdt.Set(c.enumValue)
				require.NoError(t, err)

				bytes, err := scale.Marshal(vdt)
				require.NoError(t, err)

				require.Equal(t, c.encodingValue, bytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewStatement()
				err := scale.Unmarshal(c.encodingValue, &vdt)
				require.NoError(t, err)

				actualData, err := vdt.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}
