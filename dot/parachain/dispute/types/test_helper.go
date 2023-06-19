package types

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/stretchr/testify/require"
)

type disputeStatusEnum uint

const (
	DisputeStatusActive disputeStatusEnum = iota
	DisputeStatusConcludedFor
	DisputeStatusConcludedAgainst
	DisputeStatusConfirmed
)

// NewTestDispute returns a new dispute with the given session index, candidate hash, and status
func NewTestDispute(session parachain.SessionIndex,
	candidateHash common.Hash,
	status disputeStatusEnum,
) (*Dispute, error) {
	disputeStatus, err := NewDisputeStatus()
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

// NewTestCandidateVotes returns a new candidate votes with valid and invalid votes
func NewTestCandidateVotes(t *testing.T) *CandidateVotes {
	receipt := parachain.CandidateReceipt{
		Descriptor: parachain.CandidateDescriptor{
			ParaID:                      1,
			RelayParent:                 common.Hash{2},
			Collator:                    parachain.CollatorID{2},
			PersistedValidationDataHash: common.Hash{2},
			PovHash:                     common.Hash{2},
			ErasureRoot:                 common.Hash{2},
			Signature:                   parachain.CollatorSignature{2},
			ParaHead:                    common.Hash{2},
			ValidationCodeHash:          parachain.ValidationCodeHash{2},
		},
		CommitmentsHash: common.Hash{1},
	}

	validVotes := make(map[parachain.ValidatorIndex]Vote)
	validVotes[1] = Vote{
		ValidatorIndex:     1,
		DisputeStatement:   GetValidDisputeStatement(t),
		ValidatorSignature: [64]byte{1},
	}
	validVotes[2] = Vote{
		ValidatorIndex:     2,
		DisputeStatement:   GetValidDisputeStatement(t),
		ValidatorSignature: [64]byte{2},
	}

	invalidVotes := make(map[parachain.ValidatorIndex]Vote)
	invalidVotes[2] = Vote{
		ValidatorIndex:     2,
		DisputeStatement:   GetInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{2},
	}
	invalidVotes[3] = Vote{
		ValidatorIndex:     3,
		DisputeStatement:   GetInvalidDisputeStatement(t),
		ValidatorSignature: [64]byte{3},
	}

	return &CandidateVotes{
		CandidateReceipt: receipt,
		Valid:            validVotes,
		Invalid:          invalidVotes,
	}
}

// GetValidDisputeStatement returns a valid dispute statement
func GetValidDisputeStatement(t *testing.T) inherents.DisputeStatement {
	validDisputeStatement := inherents.NewDisputeStatement()
	disputeStatementKind := inherents.NewValidDisputeStatementKind()
	err := disputeStatementKind.Set(inherents.ExplicitValidDisputeStatementKind{})
	require.NoError(t, err)

	err = validDisputeStatement.Set(inherents.ValidDisputeStatementKind(disputeStatementKind))
	require.NoError(t, err)
	return validDisputeStatement
}

// GetInvalidDisputeStatement returns an invalid dispute statement
func GetInvalidDisputeStatement(t *testing.T) inherents.DisputeStatement {
	invalidDisputeStatement := inherents.NewDisputeStatement()
	invalidDisputeStatementKind := inherents.NewInvalidDisputeStatementKind()
	err := invalidDisputeStatementKind.Set(inherents.ExplicitInvalidDisputeStatementKind{})
	require.NoError(t, err)

	err = invalidDisputeStatement.Set(inherents.InvalidDisputeStatementKind(invalidDisputeStatementKind))
	require.NoError(t, err)
	return invalidDisputeStatement
}
