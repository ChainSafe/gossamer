// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

//go:embed testdata/collation_protocol.yaml
var testDataCollationProtocolRaw string

var testDataCollationProtocol map[string]string

func init() {
	err := yaml.Unmarshal([]byte(testDataCollationProtocolRaw), &testDataCollationProtocol)
	if err != nil {
		fmt.Println("Error unmarshaling test data:", err)
		return
	}
}

func TestCollationProtocol(t *testing.T) {
	t.Parallel()

	var collatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature parachaintypes.CollatorSignature
	tempSignature := common.MustHexToBytes(testDataStatement["collatorSignature"])
	copy(collatorSignature[:], tempSignature)

	var validatorSignature parachaintypes.ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	hash5 := getDummyHash(5)

	secondedEnumValue := parachaintypes.Seconded{
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

	statementVDTWithSeconded := parachaintypes.NewStatementVDT()
	err := statementVDTWithSeconded.Set(secondedEnumValue)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte
	}{
		{
			name: "Declare",
			enumValue: Declare{
				CollatorId:        collatorID,
				ParaID:            uint32(5),
				CollatorSignature: collatorSignature,
			},
			encodingValue: common.MustHexToBytes(testDataCollationProtocol["declare"]),
		},
		{
			name:          "AdvertiseCollation",
			enumValue:     AdvertiseCollation(hash5),
			encodingValue: common.MustHexToBytes("0x00010505050505050505050505050505050505050505050505050505050505050505"),
		},
		{
			name: "CollationSeconded",
			enumValue: CollationSeconded{
				Hash: hash5,
				UncheckedSignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
					Payload:        statementVDTWithSeconded,
					ValidatorIndex: parachaintypes.ValidatorIndex(5),
					Signature:      validatorSignature,
				},
			},
			encodingValue: common.MustHexToBytes(testDataCollationProtocol["collationSeconded"]),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt_parent := NewCollationProtocol()
				vdt_child := NewCollatorProtocolMessage()

				err := vdt_child.Set(c.enumValue)
				require.NoError(t, err)

				err = vdt_parent.Set(vdt_child)
				require.NoError(t, err)

				bytes, err := scale.Marshal(vdt_parent)
				require.NoError(t, err)

				require.Equal(t, c.encodingValue, bytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				vdt_parent := NewCollationProtocol()
				err := scale.Unmarshal(c.encodingValue, &vdt_parent)
				require.NoError(t, err)

				vdt_child_temp, err := vdt_parent.Value()
				require.NoError(t, err)
				require.Equal(t, uint(0), vdt_child_temp.Index())

				vdt_child := vdt_child_temp.(CollatorProtocolMessage)
				require.NoError(t, err)

				actualData, err := vdt_child.Value()
				require.NoError(t, err)

				require.Equal(t, c.enumValue.Index(), actualData.Index())
				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}

func TestDecodeCollationHandshake(t *testing.T) {
	t.Parallel()

	testHandshake := &collatorHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)
	require.Equal(t, []byte{}, enc)

	msg, err := decodeCollatorHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}
