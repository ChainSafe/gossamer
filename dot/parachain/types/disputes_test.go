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
		{inner: BackingSeconded{Hash: common.Hash(bytes.Repeat([]byte{0x01}, 32))}},
		{inner: BackingValid{Hash: common.Hash(bytes.Repeat([]byte{0x02}, 32))}},
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
