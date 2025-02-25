package parachaintypes

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestValidDisputeStatementKind(t *testing.T) {
	inputs := []string{
		"0x00",
		"0x010101010101010101010101010101010101010101010101010101010101010101",
		"0x020202020202020202020202020202020202020202020202020202020202020202",
		"0x03",
		"0x04040303030303030303030303030303030303030303030303030303030303030303",
	}
	expected := []ValidDisputeStatementKind{
		{inner: ExplicitStatement{}},
		{inner: SecondedCandidateHash{Value: common.Hash(bytes.Repeat([]byte{0x01}, 32))}},
		{inner: Valid{Value: common.Hash(bytes.Repeat([]byte{0x02}, 32))}},
		{inner: ApprovalChecking{}},
		{inner: ApprovalCheckingMultipleCandidates{
			CandidateHashes: []CandidateHash{
				{Value: common.Hash(bytes.Repeat([]byte{0x03}, 32))},
			},
		}},
	}

	for idx, in := range inputs {
		var vdk ValidDisputeStatementKind
		scale.Unmarshal(common.MustHexToBytes(in), &vdk)

		require.Equal(t, expected[idx], vdk)
	}
}

func TestInvalidDisputeStatementKind(t *testing.T) {
	inputs := []string{
		"0x00", // there is only one variant
	}
	expected := []InvalidDisputeStatementKind{
		{inner: ExplicitStatement{}},
	}

	for idx, in := range inputs {
		var vdk InvalidDisputeStatementKind
		scale.Unmarshal(common.MustHexToBytes(in), &vdk)

		require.Equal(t, expected[idx], vdk)
	}
}

func TestDisputeStatus(t *testing.T) {
	inputs := []string{
		"0x00",
		"0x01704fdf6000000000",
		"0x02704fdf6000000000",
		"0x03",
	}
	expected := []DisputeStatus{
		{inner: Active{}},
		{inner: ConcludedFor{Timestamp: 1625247600}},
		{inner: ConcludedAgainst{Timestamp: 1625247600}},
		{inner: Confirmed{}},
	}

	for idx, in := range inputs {
		var ds DisputeStatus
		scale.Unmarshal(common.MustHexToBytes(in), &ds)

		require.Equal(t, expected[idx], ds)
	}
}

func TestDisputeStatement(t *testing.T) {
	inputs := []string{
		"0x0000",
		"0x00010101010101010101010101010101010101010101010101010101010101010101",
		"0x00020202020202020202020202020202020202020202020202020202020202020202",
		"0x0003",
		"0x0004040303030303030303030303030303030303030303030303030303030303030303",
		"0x0100",
	}
	expected := []DisputeStatement{
		{inner: ValidDisputeStatement{Kind: ValidDisputeStatementKind{inner: ExplicitStatement{}}}},
		{inner: ValidDisputeStatement{
			Kind: ValidDisputeStatementKind{
				inner: SecondedCandidateHash{Value: common.Hash(bytes.Repeat([]byte{0x01}, 32))},
			},
		}},
		{inner: ValidDisputeStatement{
			Kind: ValidDisputeStatementKind{
				inner: Valid{Value: common.Hash(bytes.Repeat([]byte{0x02}, 32))},
			},
		}},
		{inner: ValidDisputeStatement{Kind: ValidDisputeStatementKind{inner: ApprovalChecking{}}}},
		{inner: ValidDisputeStatement{
			Kind: ValidDisputeStatementKind{inner: ApprovalCheckingMultipleCandidates{
				CandidateHashes: []CandidateHash{
					{Value: common.Hash(bytes.Repeat([]byte{0x03}, 32))},
				},
			}},
		}},
		{inner: InvalidDisputeStatement{
			Kind: InvalidDisputeStatementKind{inner: ExplicitStatement{}},
		}},
	}

	for idx, in := range inputs {
		var ds DisputeStatement
		scale.Unmarshal(common.MustHexToBytes(in), &ds)

		require.Equal(t, expected[idx], ds)
	}
}

func TestDisputeStatementSet(t *testing.T) {
	inputs := []string{
		"0x010101010101010101010101010101010101010101010101010101010101010101000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", //nolint:lll
	}
	expected := []DisputeStatementSet{
		{
			CandidateHash: CandidateHash{Value: common.Hash(bytes.Repeat([]byte{0x01}, 32))},
			Session:       1,
			Statements: []DisputeStatementEntry{
				{
					Statement: DisputeStatement{
						inner: ValidDisputeStatement{
							Kind: ValidDisputeStatementKind{inner: ExplicitStatement{}},
						},
					},
					Index:     0,
					Signature: ValidatorSignature([64]byte{}),
				},
				{
					Statement: DisputeStatement{
						inner: InvalidDisputeStatement{
							Kind: InvalidDisputeStatementKind{inner: ExplicitStatement{}},
						},
					},
					Index:     1,
					Signature: ValidatorSignature([64]byte{}),
				},
			},
		},
	}

	for idx, in := range inputs {
		var dss DisputeStatementSet
		scale.Unmarshal(common.MustHexToBytes(in), &dss)

		require.Equal(t, expected[idx], dss)
	}
}
