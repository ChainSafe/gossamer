package types

import (
	"fmt"
	"github.com/emirpasic/gods/sets/treeset"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Vote is a vote from a validator for a dispute statement
type Vote struct {
	ValidatorIndex     parachainTypes.ValidatorIndex `scale:"1"`
	DisputeStatement   inherents.DisputeStatement    `scale:"2"`
	ValidatorSignature [64]byte                      `scale:"3"`
}

// Voted represents the voted state with the votes for a dispute statement
type Voted struct {
	Votes []Vote
}

// Index returns the index of the Voted enum
func (Voted) Index() uint {
	return 0
}

// CannotVote represents the state where we cannot vote because we are not a parachain validator in the current session
type CannotVote struct{}

// Index returns the index of the CannotVote enum
func (CannotVote) Index() uint {
	return 1
}

// OwnVoteState is the state of the vote for a candidate
type OwnVoteState scale.VaryingDataType

// New returns a new OwnVoteState
func (OwnVoteState) New() OwnVoteState {
	ownVoteState, err := NewOwnVoteState(CannotVote{})
	if err != nil {
		panic(err)
	}

	return ownVoteState
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *OwnVoteState) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*v)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*v = OwnVoteState(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (v *OwnVoteState) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// VoteMissing returns true if a vote from us is missing for the candidate
func (v *OwnVoteState) VoteMissing() bool {
	vdt := scale.VaryingDataType(*v)
	val, err := vdt.Value()
	if err != nil {
		return false
	}

	_, ok := val.(CannotVote)
	if ok {
		return false
	}

	voted, ok := val.(Voted)
	if !ok {
		return false
	}

	return len(voted.Votes) == 0
}

// ApprovalVotes returns the approval votes for the candidate
func (v *OwnVoteState) ApprovalVotes() ([]Vote, error) {
	vdt := scale.VaryingDataType(*v)
	val, err := vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from OwnVoteState vdt: %w", err)
	}

	_, ok := val.(CannotVote)
	if ok {
		return nil, nil
	}

	voted, ok := val.(Voted)
	if !ok {
		return nil, fmt.Errorf("invalid type for OwnVoteState: expected Voted, got %T", val)
	}

	var votes []Vote
	for _, vote := range voted.Votes {
		disputeStatement, err := vote.DisputeStatement.Value()
		if err != nil {
			return nil, fmt.Errorf("getting value from DisputeStatement vdt: %w", err)
		}

		_, ok := disputeStatement.(inherents.ApprovalChecking)
		if !ok {
			continue
		}

		votes = append(votes, Vote{
			ValidatorIndex:     vote.ValidatorIndex,
			ValidatorSignature: vote.ValidatorSignature,
			DisputeStatement:   vote.DisputeStatement,
		})
	}

	return votes, nil
}

// Votes returns the votes for the candidate
func (v *OwnVoteState) Votes() ([]Vote, error) {
	vdt := scale.VaryingDataType(*v)
	val, err := vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from OwnVoteState vdt: %w", err)
	}

	_, ok := val.(CannotVote)
	if ok {
		return nil, nil
	}

	voted, ok := val.(Voted)
	if !ok {
		return nil, fmt.Errorf("invalid type for OwnVoteState: expected Voted, got %T", val)
	}

	return voted.Votes, nil
}

// NewOwnVoteState returns a new OwnVoteState with the given value
func NewOwnVoteState(value scale.VaryingDataTypeValue) (OwnVoteState, error) {
	vdt, err := scale.NewVaryingDataType(Voted{}, CannotVote{})
	if err != nil {
		return OwnVoteState{}, fmt.Errorf("creating new OwnVoteState vdt: %w", err)
	}

	err = vdt.Set(value)
	if err != nil {
		return OwnVoteState{}, fmt.Errorf("setting value to OwnVoteState vdt: %w", err)
	}

	return OwnVoteState(vdt), nil
}

// CandidateVoteState is the state of the votes for a candidate
type CandidateVoteState struct {
	Votes         CandidateVotes
	Own           OwnVoteState
	DisputeStatus *DisputeStatus
}

func (c *CandidateVoteState) IsDisputed() bool {
	return c.DisputeStatus != nil
}

func (c *CandidateVoteState) IsConfirmed() (bool, error) {
	if c.DisputeStatus == nil {
		return false, nil
	}

	return c.DisputeStatus.IsConfirmedConcluded()
}

func (c *CandidateVoteState) IsConcludedFor() (bool, error) {
	if c.DisputeStatus == nil {
		return false, nil
	}

	return c.DisputeStatus.IsConcludedFor()
}

func (c *CandidateVoteState) IsConcludedAgainst() (bool, error) {
	if c.DisputeStatus == nil {
		return false, nil
	}

	return c.DisputeStatus.IsConcludedAgainst()
}

func (c *CandidateVoteState) IntoOldState() (CandidateVotes, CandidateVoteState) {
	return c.Votes, CandidateVoteState{
		Votes:         CandidateVotes{},
		Own:           c.Own,
		DisputeStatus: c.DisputeStatus,
	}
}

// NewCandidateVoteState creates a new CandidateVoteState
// TODO: implement this later since nothing is using it yet
func NewCandidateVoteState(votes CandidateVotes, now uint64) (CandidateVoteState, error) {
	var (
		status DisputeStatus
		err    error
	)

	// TODO: initialize own vote state with the votes
	ownVoteState, err := NewOwnVoteState(CannotVote{})
	if err != nil {
		return CandidateVoteState{}, fmt.Errorf("failed to create own vote state: %w", err)
	}

	// TODO: get number of validators
	//numberOfValidators := 0

	// TODO: get supermajority threshold
	superMajorityThreshold := 0

	isDisputed := !(len(votes.Invalid) == 0) && !(len(votes.Valid) == 0)
	if isDisputed {
		status, err = NewDisputeStatus()
		if err != nil {
			return CandidateVoteState{}, fmt.Errorf("failed to create dispute status: %w", err)
		}

		// TODO: get byzantine threshold
		byzantineThreshold := 0

		isConfirmed := len(votes.Valid) > byzantineThreshold
		if isConfirmed {
			if err := status.Confirm(); err != nil {
				return CandidateVoteState{}, fmt.Errorf("failed to confirm dispute status: %w", err)
			}
		}

		isConcludedFor := len(votes.Valid) > superMajorityThreshold
		if isConcludedFor {
			if err := status.ConcludeFor(now); err != nil {
				return CandidateVoteState{}, fmt.Errorf("failed to conclude dispute status for: %w", err)
			}
		}

		isConcludedAgainst := len(votes.Invalid) >= superMajorityThreshold
		if isConcludedAgainst {
			if err := status.ConcludeAgainst(now); err != nil {
				return CandidateVoteState{}, fmt.Errorf("failed to conclude dispute status against: %w", err)
			}
		}
	}

	return CandidateVoteState{
		Votes:         votes,
		Own:           ownVoteState,
		DisputeStatus: &status,
	}, nil
}

// NewCandidateVoteStateFromReceipt creates a new CandidateVoteState from a CandidateReceipt
func NewCandidateVoteStateFromReceipt(receipt parachainTypes.CandidateReceipt) (*CandidateVoteState, error) {
	votes := NewCandidateVotesFromReceipt(receipt)
	ownVoteState, err := NewOwnVoteState(CannotVote{})
	if err != nil {
		return nil, fmt.Errorf("failed to create own vote state: %w", err)
	}

	return &CandidateVoteState{
		Votes: votes,
		Own:   ownVoteState,
	}, nil
}

// ValidCandidateVotes is a list of valid votes for a candidate.
type ValidCandidateVotes map[parachainTypes.ValidatorIndex]Vote

func (vcv ValidCandidateVotes) InsertVote(vote Vote) (bool, error) {
	existingVote, ok := vcv[vote.ValidatorIndex]
	if !ok {
		vcv[vote.ValidatorIndex] = vote
		return true, nil
	}

	disputeStatement, err := existingVote.DisputeStatement.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatement vdt: %w", err)
	}

	switch disputeStatement.(type) {
	case inherents.BackingValid, inherents.BackingSeconded:
		return false, nil
	case inherents.ExplicitValidDisputeStatementKind,
		inherents.ExplicitInvalidDisputeStatementKind,
		inherents.ApprovalChecking:
		vcv[vote.ValidatorIndex] = vote
		return true, nil
	default:
		return false, fmt.Errorf("invalid dispute statement type: %T", disputeStatement)
	}
}

// CandidateVotes is a struct containing the votes for a candidate.
type CandidateVotes struct {
	CandidateReceipt parachainTypes.CandidateReceipt `scale:"1"`
	// TODO: check if we need to use btree for this in the future
	Valid   ValidCandidateVotes                    `scale:"2"`
	Invalid map[parachainTypes.ValidatorIndex]Vote `scale:"3"`
}

func (cv *CandidateVotes) VotedIndices() *treeset.Set {
	votedIndices := treeset.NewWithIntComparator()
	for validatorIndex := range cv.Valid {
		votedIndices.Add(validatorIndex)
	}

	for validatorIndex := range cv.Invalid {
		votedIndices.Add(validatorIndex)
	}

	return votedIndices
}

func NewCandidateVotes() *CandidateVotes {
	return &CandidateVotes{
		Valid:   make(map[parachainTypes.ValidatorIndex]Vote),
		Invalid: make(map[parachainTypes.ValidatorIndex]Vote),
	}
}

// NewCandidateVotesFromReceipt creates a new CandidateVotes from a candidate receipt.
func NewCandidateVotesFromReceipt(receipt parachainTypes.CandidateReceipt) CandidateVotes {
	return CandidateVotes{
		CandidateReceipt: receipt,
	}
}
