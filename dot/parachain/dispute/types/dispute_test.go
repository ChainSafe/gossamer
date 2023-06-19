package types

import (
	"crypto/rand"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func getRandomHash() common.Hash {
	var hash [32]byte
	randomBytes := make([]byte, len(hash))
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	copy(hash[:], randomBytes)
	return hash
}

func getRandomSignature() [64]byte {
	var hash [64]byte
	randomBytes := make([]byte, len(hash))
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	copy(hash[:], randomBytes)
	return hash
}

func TestDispute_Codec(t *testing.T) {
	t.Parallel()

	disputeStatus, err := NewDisputeStatus()
	require.NoError(t, err)
	err = disputeStatus.Set(ActiveStatus{})
	require.NoError(t, err)

	// with
	dispute := Dispute{
		Comparator: Comparator{
			SessionIndex:  1,
			CandidateHash: getRandomHash(),
		},
		DisputeStatus: disputeStatus,
	}

	// when
	encoded, err := scale.Marshal(dispute)
	require.NoError(t, err)

	// then
	decoded, err := NewDispute()
	require.NoError(t, err)

	err = scale.Unmarshal(encoded, decoded)
	require.NoError(t, err)
	require.Equal(t, &dispute, decoded)
}

func TestDispute_Less(t *testing.T) {
	t.Parallel()

	status, err := NewDisputeStatus()
	require.NoError(t, err)
	err = status.Set(ActiveStatus{})
	require.NoError(t, err)

	// with
	dispute1 := Dispute{
		Comparator: Comparator{
			SessionIndex:  1,
			CandidateHash: common.Hash{1},
		},
		DisputeStatus: status,
	}

	dispute2 := Dispute{
		Comparator: Comparator{
			SessionIndex:  2,
			CandidateHash: common.Hash{2},
		},
		DisputeStatus: status,
	}

	dispute3 := Dispute{
		Comparator: Comparator{
			SessionIndex:  2,
			CandidateHash: common.Hash{3},
		},
		DisputeStatus: status,
	}

	// when
	less12 := dispute1.Less(&dispute2)
	less23 := dispute2.Less(&dispute3)

	// then
	require.True(t, less12)
	require.True(t, less23)
}
