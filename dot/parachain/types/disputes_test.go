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
		ValidDisputeStatementKind{inner: ExplicitStatement{}},
		ValidDisputeStatementKind{inner: BackingSeconded{Hash: common.Hash(bytes.Repeat([]byte{0x01}, 32))}},
		ValidDisputeStatementKind{inner: BackingValid{Hash: common.Hash(bytes.Repeat([]byte{0x02}, 32))}},
		ValidDisputeStatementKind{inner: ApprovalChecking{}},
		ValidDisputeStatementKind{inner: ApprovalCheckingMultipleCandidates{
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
