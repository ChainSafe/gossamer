// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package validationprotocol

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/statement.yaml
var testDataStatementRaw string

var testDataStatement map[string]string

func init() {
	err := yaml.Unmarshal([]byte(testDataStatementRaw), &testDataStatement)
	if err != nil {
		fmt.Printf("Error unmarshaling test data: %s\n", err)
		return
	}
}

func TestStatementDistributionMessage(t *testing.T) {
	t.Parallel()

	tempSignature := common.MustHexToBytes(testDataStatement["collatorSignature"])
	var validatorSignature parachaintypes.ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	hash5 := common.Hash(bytes.Repeat([]byte{10}, 32))
	compactValidStmt := parachaintypes.NewCompactValid(
		parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))}).ToEncodable()

	signedStatementWithValid := Statement{
		RelayParent: hash5,
		Compact: parachaintypes.UncheckedSignedCompactStatement{
			Payload:        *compactValidStmt,
			ValidatorIndex: parachaintypes.ValidatorIndex(5),
			Signature:      validatorSignature,
		},
	}

	compactSecondedStmt := parachaintypes.NewCompactSeconded(
		parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))}).ToEncodable()

	signedStatementWithSeconded := Statement{
		RelayParent: hash5,
		Compact: parachaintypes.UncheckedSignedCompactStatement{
			Payload:        *compactSecondedStmt,
			ValidatorIndex: parachaintypes.ValidatorIndex(5),
			Signature:      validatorSignature,
		},
	}

	stmtFilter, err := parachaintypes.NewStatementFilter(0, false)
	require.NoError(t, err)

	backedCandidate := BackedCandidateManifest{
		RelayParent:        common.Hash(bytes.Repeat([]byte{10}, 32)),
		CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))},
		GroupIndex:         parachaintypes.GroupIndex(0),
		ParaID:             parachaintypes.ParaID(0),
		ParentHeadDataHash: common.Hash(bytes.Repeat([]byte{10}, 32)),
		StatementKnowledge: *stmtFilter,
	}

	candidateKnown := BackedCandidateKnown{
		CandidateHash:      parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{15}, 32))},
		StatementKnowledge: *stmtFilter,
	}

	testCases := []struct {
		name          string
		enumValue     any
		encodingValue []byte
		expectedErr   error
	}{
		{
			name:          "signed_statement_valid",
			enumValue:     signedStatementWithValid,
			encodingValue: common.MustHexToBytes(testDataStatement["statementValid"]),
		},
		{
			name:          "signed_statement_seconded",
			enumValue:     signedStatementWithSeconded,
			encodingValue: common.MustHexToBytes(testDataStatement["statementSeconded"]),
		},
		{
			name:          "backed_candidate_manifest",
			enumValue:     backedCandidate,
			encodingValue: common.MustHexToBytes(testDataStatement["backedCandidate"]),
		},
		{
			name:          "backed_candidate_known",
			enumValue:     candidateKnown,
			encodingValue: common.MustHexToBytes(testDataStatement["backedKnown"]),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewStatementDistributionMessage()
				err := vdt.SetValue(c.enumValue)

				if c.expectedErr != nil {
					require.EqualError(t, err, c.expectedErr.Error())
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

				vdt := NewStatementDistributionMessage()
				err := scale.Unmarshal(c.encodingValue, &vdt)
				require.NoError(t, err)

				actualData, err := vdt.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}
