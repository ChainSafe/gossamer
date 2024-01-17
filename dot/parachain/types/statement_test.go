// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	_ "embed"
	"fmt"
	"math"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/statement.yaml
var testDataStatementRaw []byte

var testDataStatement map[string]string

func init() {
	err := yaml.Unmarshal(testDataStatementRaw, &testDataStatement)
	if err != nil {
		fmt.Printf("Error unmarshaling test data: %s\n", err)
		return
	}
}

type invalidVayingDataTypeValue struct{}

func (invalidVayingDataTypeValue) Index() uint {
	return math.MaxUint
}

func getDummyHash(num byte) common.Hash {
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func TestStatementVDT(t *testing.T) {
	t.Parallel()

	var collatorID CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature CollatorSignature
	tempSignature := common.MustHexToBytes(testDataStatement["collatorSignature"])
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
			UpwardMessages:    []UpwardMessage{{1, 2, 3}},
			NewValidationCode: &ValidationCode{1, 2, 3},
			HeadData: HeadData{
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
		expectedErr   error
	}{
		{
			name:          "Seconded",
			enumValue:     secondedEnumValue,
			encodingValue: common.MustHexToBytes(testDataStatement["statementVDTSeconded"]),
			expectedErr:   nil,
		},
		{
			name:          "Valid",
			enumValue:     Valid{Value: hash5},
			encodingValue: common.MustHexToBytes("0x020505050505050505050505050505050505050505050505050505050505050505"),
			expectedErr:   nil,
		},
		{
			name:        "invalid struct",
			enumValue:   invalidVayingDataTypeValue{},
			expectedErr: scale.ErrUnsupportedVaryingDataTypeValue,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewStatementVDT()
				err := vdt.Set(c.enumValue)

				if c.expectedErr != nil {
					require.ErrorContains(t, err, c.expectedErr.Error())
					return
				}

				require.NoError(t, err)
				bytes, err := scale.Marshal(vdt)
				require.NoError(t, err)

				require.Equal(t, c.encodingValue, bytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()
				if c.expectedErr != nil {
					return
				}

				vdt := NewStatementVDT()
				err := scale.Unmarshal(c.encodingValue, &vdt)
				require.NoError(t, err)

				actualData, err := vdt.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}
