// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
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
			ParaID:                      1,
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
		enumValue     any
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
			expectedErr: fmt.Errorf("unsupported type"),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewStatementVDT()
				err := vdt.SetValue(c.enumValue)

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

func TestStatementVDT_Sign(t *testing.T) {
	statement := NewStatementVDT()
	err := statement.SetValue(Seconded{})
	require.NoError(t, err)

	signingContext := SigningContext{
		SessionIndex: 1,
		ParentHash:   getDummyHash(1),
	}

	ks := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	keyPair := keyring.Alice()
	err = ks.Insert(keyPair)
	require.NoError(t, err)

	publicKeyBytes := keyPair.Public().Encode()
	validatorID := ValidatorID(publicKeyBytes)

	valSign, err := statement.Sign(ks, signingContext, validatorID)
	require.NoError(t, err)

	ok, err := statement.VerifySignature(validatorID, signingContext, *valSign)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCompactStatement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		compactStatement any
		encodingValue    []byte
		expectedErr      error
	}{
		{
			name: "SecondedCandidateHash",
			compactStatement: CompactStatement[SecondedCandidateHash]{
				Value: SecondedCandidateHash{Value: getDummyHash(6)},
			},
			encodingValue: []byte{66, 75, 78, 71, 1,
				6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6},
		},
		{
			name: "Valid",
			compactStatement: CompactStatement[Valid]{
				Value: Valid{Value: getDummyHash(7)},
			},
			encodingValue: []byte{
				66, 75, 78, 71, 2,
				7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				compactStatementBytes, err := scale.Marshal(c.compactStatement)
				require.NoError(t, err)
				require.Equal(t, c.encodingValue, compactStatementBytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				switch expectedSatetement := c.compactStatement.(type) {
				case CompactStatement[Valid]:
					var actualStatement CompactStatement[Valid]
					err := scale.Unmarshal(c.encodingValue, &actualStatement)
					require.NoError(t, err)
					require.EqualValues(t, expectedSatetement, actualStatement)
				case CompactStatement[SecondedCandidateHash]:
					var actualStatement CompactStatement[SecondedCandidateHash]
					err := scale.Unmarshal(c.encodingValue, &actualStatement)
					require.NoError(t, err)
					require.EqualValues(t, expectedSatetement, actualStatement)
				}
			})

		})
	}
}
