package types

import (
	"crypto/rand"
	"fmt"
	"github.com/tidwall/btree"
	"testing"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

type disputeStatusEnum uint

const (
	DisputeStatusActive disputeStatusEnum = iota
	DisputeStatusConcludedFor
	DisputeStatusConcludedAgainst
	DisputeStatusConfirmed
)

// DummyDispute returns a new dummy dispute with the given session index, candidate hash, and status
func DummyDispute(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
	status disputeStatusEnum,
) (*Dispute, error) {
	disputeStatus, err := NewDisputeStatusVDT()
	if err != nil {
		return nil, err
	}

	switch status {
	case DisputeStatusActive:
		err := disputeStatus.Set(ActiveStatus{})
		if err != nil {
			return nil, err
		}
	case DisputeStatusConcludedFor:
		err := disputeStatus.Set(ConcludedForStatus{})
		if err != nil {
			return nil, err
		}
	case DisputeStatusConcludedAgainst:
		err := disputeStatus.Set(ConcludedAgainstStatus{})
		if err != nil {
			return nil, err
		}
	case DisputeStatusConfirmed:
		err := disputeStatus.Set(ConfirmedStatus{})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid dispute status")
	}

	return &Dispute{
		Comparator: Comparator{
			SessionIndex:  session,
			CandidateHash: candidateHash,
		},
		DisputeStatus: disputeStatus,
	}, nil
}

// DummyCandidateVotes returns a new candidate votes with valid and invalid votes
func DummyCandidateVotes(t *testing.T) *CandidateVotes {
	receipt := parachainTypes.CandidateReceipt{
		Descriptor: parachainTypes.CandidateDescriptor{
			ParaID:                      1,
			RelayParent:                 common.Hash{2},
			Collator:                    parachainTypes.CollatorID{2},
			PersistedValidationDataHash: common.Hash{2},
			PovHash:                     common.Hash{2},
			ErasureRoot:                 common.Hash{2},
			Signature:                   parachainTypes.CollatorSignature{2},
			ParaHead:                    common.Hash{2},
			ValidationCodeHash:          parachainTypes.ValidationCodeHash{2},
		},
		CommitmentsHash: common.Hash{1},
	}

	validVotes := ValidCandidateVotes{
		VotedValidators: make(map[parachainTypes.ValidatorIndex]struct{}),
		Value:           btree.New(CompareVoteIndices),
	}
	validVotes.Value.Set(Vote{
		ValidatorIndex:     1,
		DisputeStatement:   DummyValidDisputeStatement(t),
		ValidatorSignature: [64]byte{1},
	})
	validVotes.VotedValidators[1] = struct{}{}
	validVotes.Value.Set(Vote{
		ValidatorIndex:     2,
		DisputeStatement:   DummyValidDisputeStatement(t),
		ValidatorSignature: [64]byte{2},
	})
	validVotes.VotedValidators[2] = struct{}{}

	invalidVotes := btree.New(CompareVoteIndices)
	invalidVotes.Set(Vote{
		ValidatorIndex:     2,
		DisputeStatement:   DummyInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{2},
	})
	invalidVotes.Set(Vote{
		ValidatorIndex:     3,
		DisputeStatement:   DummyInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{3},
	})

	return &CandidateVotes{
		CandidateReceipt: receipt,
		Valid:            validVotes,
		Invalid:          invalidVotes,
	}
}

// DummyValidDisputeStatement returns a dummy valid dispute statement
func DummyValidDisputeStatement(t *testing.T) inherents.DisputeStatement {
	validDisputeStatement := inherents.NewDisputeStatement()
	disputeStatementKind := inherents.NewValidDisputeStatementKind()
	err := disputeStatementKind.Set(inherents.ExplicitValidDisputeStatementKind{})
	require.NoError(t, err)

	err = validDisputeStatement.Set(inherents.ValidDisputeStatementKind(disputeStatementKind))
	require.NoError(t, err)
	return validDisputeStatement
}

// DummyInvalidDisputeStatement returns an invalid dispute statement
func DummyInvalidDisputeStatement(t *testing.T) inherents.DisputeStatement {
	invalidDisputeStatement := inherents.NewDisputeStatement()
	invalidDisputeStatementKind := inherents.NewInvalidDisputeStatementKind()
	err := invalidDisputeStatementKind.Set(inherents.ExplicitInvalidDisputeStatementKind{})
	require.NoError(t, err)

	err = invalidDisputeStatement.Set(inherents.InvalidDisputeStatementKind(invalidDisputeStatementKind))
	require.NoError(t, err)
	return invalidDisputeStatement
}

func GetRandomHash() common.Hash {
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
