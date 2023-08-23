package types

import (
	"testing"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func Test_CandidateVotes(t *testing.T) {
	t.Parallel()
	// with
	receipt := parachainTypes.CandidateReceipt{
		Descriptor: parachainTypes.CandidateDescriptor{
			ParaID:                      100,
			RelayParent:                 getRandomHash(),
			Collator:                    parachainTypes.CollatorID{2},
			PersistedValidationDataHash: getRandomHash(),
			PovHash:                     getRandomHash(),
			ErasureRoot:                 getRandomHash(),
			Signature:                   parachainTypes.CollatorSignature{2},
			ParaHead:                    getRandomHash(),
			ValidationCodeHash:          parachainTypes.ValidationCodeHash(getRandomHash()),
		},
		CommitmentsHash: getRandomHash(),
	}

	validVotes := make(map[parachainTypes.ValidatorIndex]Vote)
	validVotes[1] = Vote{
		ValidatorIndex:     1,
		DisputeStatement:   DummyInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{1},
	}
	validVotes[2] = Vote{
		ValidatorIndex:     2,
		DisputeStatement:   DummyValidDisputeStatement(t),
		ValidatorSignature: [64]byte{2},
	}

	invalidVotes := make(map[parachainTypes.ValidatorIndex]Vote)
	invalidVotes[2] = Vote{
		ValidatorIndex:     2,
		DisputeStatement:   DummyInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{2},
	}
	invalidVotes[3] = Vote{
		ValidatorIndex:     3,
		DisputeStatement:   DummyInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{3},
	}

	votes := CandidateVotes{CandidateReceipt: receipt, Valid: validVotes, Invalid: invalidVotes}

	// when
	encoded, err := scale.Marshal(votes)
	require.NoError(t, err)

	decoded := CandidateVotes{
		Valid:   make(map[parachainTypes.ValidatorIndex]Vote),
		Invalid: make(map[parachainTypes.ValidatorIndex]Vote),
	}
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// then
	require.Equal(t, votes, decoded)
}

func Test_Vote(t *testing.T) {
	t.Parallel()
	validVote := Vote{
		ValidatorIndex:     1,
		DisputeStatement:   DummyValidDisputeStatement(t),
		ValidatorSignature: getRandomSignature(),
	}

	encoded, err := scale.Marshal(validVote)
	require.NoError(t, err)

	decoded := Vote{}
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	require.Equal(t, validVote, decoded)

	invalidVote := Vote{
		ValidatorIndex:     1,
		DisputeStatement:   DummyInvalidDisputeStatement(t),
		ValidatorSignature: getRandomSignature(),
	}

	encoded, err = scale.Marshal(invalidVote)
	require.NoError(t, err)

	decoded = Vote{}
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	require.Equal(t, invalidVote, decoded)
}

func TestOwnVoteState_CannotVote(t *testing.T) {
	t.Parallel()
	// with
	ownVoteState, err := NewOwnVoteState(CannotVote{})
	require.NoError(t, err)

	// when
	encoded, err := scale.Marshal(ownVoteState)
	require.NoError(t, err)

	decoded := OwnVoteState{}
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// then
	require.Equal(t, ownVoteState, decoded)
}

func TestOwnVoteState_Voted(t *testing.T) {
	t.Parallel()
	// with
	votes := []Vote{
		{
			ValidatorIndex:     1,
			DisputeStatement:   DummyValidDisputeStatement(t),
			ValidatorSignature: getRandomSignature(),
		},
		{
			ValidatorIndex:     2,
			DisputeStatement:   DummyInvalidDisputeStatement(t),
			ValidatorSignature: getRandomSignature(),
		},
	}

	ownVoteState, err := NewOwnVoteState(Voted{Votes: votes})
	require.NoError(t, err)

	// when
	encoded, err := scale.Marshal(ownVoteState)
	require.NoError(t, err)

	decoded, err := NewOwnVoteState(CannotVote{})
	require.NoError(t, err)
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// then
	require.Equal(t, ownVoteState, decoded)
}
