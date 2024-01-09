package types

import (
	"fmt"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/emirpasic/gods/sets/treeset"
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

// OwnVoteStateVDT is the state of the vote for a candidate
type OwnVoteStateVDT scale.VaryingDataType

// New returns a new OwnVoteStateVDT
func (OwnVoteStateVDT) New() OwnVoteStateVDT {
	ownVoteState, err := NewOwnVoteStateVDT(CannotVote{})
	if err != nil {
		panic(err)
	}

	return ownVoteState
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *OwnVoteStateVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*v)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*v = OwnVoteStateVDT(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (v *OwnVoteStateVDT) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// VoteMissing returns true if a vote from us is missing for the candidate
func (v *OwnVoteStateVDT) VoteMissing() bool {
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

	return voted.Votes == nil || len(voted.Votes) == 0
}

// ApprovalVotes returns the approval votes for the candidate
func (v *OwnVoteStateVDT) ApprovalVotes() ([]Vote, error) {
	vdt := scale.VaryingDataType(*v)
	val, err := vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from OwnVoteStateVDT vdt: %w", err)
	}

	_, ok := val.(CannotVote)
	if ok {
		return nil, nil
	}

	voted, ok := val.(Voted)
	if !ok {
		return nil, fmt.Errorf("invalid type for OwnVoteStateVDT: expected Voted, got %T", val)
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
func (v *OwnVoteStateVDT) Votes() ([]Vote, error) {
	vdt := scale.VaryingDataType(*v)
	val, err := vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from OwnVoteStateVDT vdt: %w", err)
	}

	_, ok := val.(CannotVote)
	if ok {
		return nil, nil
	}

	voted, ok := val.(Voted)
	if !ok {
		return nil, fmt.Errorf("invalid type for OwnVoteStateVDT: expected Voted, got %T", val)
	}

	return voted.Votes, nil
}

// NewOwnVoteStateVDT returns a new OwnVoteStateVDT with the given value
func NewOwnVoteStateVDT(value scale.VaryingDataTypeValue) (OwnVoteStateVDT, error) {
	vdt, err := scale.NewVaryingDataType(Voted{}, CannotVote{})
	if err != nil {
		return OwnVoteStateVDT{}, fmt.Errorf("creating new OwnVoteStateVDT vdt: %w", err)
	}

	err = vdt.Set(value)
	if err != nil {
		return OwnVoteStateVDT{}, fmt.Errorf("setting value to OwnVoteStateVDT vdt: %w", err)
	}

	return OwnVoteStateVDT(vdt), nil
}

// NewOwnVoteStateVDTWithVotes returns a new OwnVoteStateVDT with the given votes
func NewOwnVoteStateVDTWithVotes(voteState CandidateVotes, env *CandidateEnvironment) (OwnVoteStateVDT, error) {
	if len(env.ControlledIndices) == 0 {
		return NewOwnVoteStateVDT(CannotVote{})
	}

	var (
		validVotes   []Vote
		invalidVotes []Vote
	)

	for validatorIndex := range env.ControlledIndices {
		if validVote, ok := voteState.Valid.Value.Get(validatorIndex); ok {
			validVotes = append(validVotes, validVote)
		}

		if invalidVote, ok := voteState.Invalid.Get(validatorIndex); ok {
			invalidVotes = append(invalidVotes, invalidVote)
		}
	}

	voted := Voted{
		Votes: append(validVotes, invalidVotes...),
	}
	voteStateVDT, err := NewOwnVoteStateVDT(voted)
	if err != nil {
		return OwnVoteStateVDT{}, fmt.Errorf("creating new OwnVoteStateVDT: %w", err)
	}

	return voteStateVDT, nil
}

// CandidateVoteState is the state of the votes for a candidate
type CandidateVoteState struct {
	Votes                     CandidateVotes
	Own                       OwnVoteStateVDT
	DisputeStatus             *DisputeStatusVDT
	ByzantineThresholdAgainst bool
}

// IsDisputed returns true if we have an ongoing dispute
func (c *CandidateVoteState) IsDisputed() bool {
	return c.DisputeStatus != nil
}

// IsConfirmed returns true if there is an ongoing confirmed dispute
func (c *CandidateVoteState) IsConfirmed() (bool, error) {
	if c.DisputeStatus == nil {
		return false, nil
	}

	return c.DisputeStatus.IsConfirmedConcluded()
}

// IsConcludedFor returns true if there is a dispute, and it has already enough valid votes to conclude
func (c *CandidateVoteState) IsConcludedFor() (bool, error) {
	if c.DisputeStatus == nil {
		return false, nil
	}

	return c.DisputeStatus.IsConcludedFor()
}

// IsConcludedAgainst returns true if there is a dispute, and it has already enough invalid votes to conclude.
func (c *CandidateVoteState) IsConcludedAgainst() (bool, error) {
	if c.DisputeStatus == nil {
		return false, nil
	}

	return c.DisputeStatus.IsConcludedAgainst()
}

// IntoOldState Extracts `CandidateVotes` for handling import of new statements.
func (c *CandidateVoteState) IntoOldState() (CandidateVotes, CandidateVoteState) {
	return c.Votes.Copy(), CandidateVoteState{
		Votes:                     NewCandidateVotes(),
		Own:                       c.Own,
		DisputeStatus:             c.DisputeStatus,
		ByzantineThresholdAgainst: c.ByzantineThresholdAgainst,
	}
}

// NewCandidateVoteState creates a new CandidateVoteState
func NewCandidateVoteState(votes CandidateVotes,
	env *CandidateEnvironment,
	now uint64, byzantineThreshold,
	superMajorityThreshold int,
) (CandidateVoteState, error) {
	var (
		disputeStatus             *DisputeStatusVDT
		byzantineThresholdAgainst bool
		err                       error
	)

	ownVoteState, err := NewOwnVoteStateVDTWithVotes(votes, env)
	if err != nil {
		return CandidateVoteState{}, fmt.Errorf("create own vote state vdt: %w", err)
	}

	isDisputed := !(votes.Invalid.Len() == 0) && !(votes.Valid.Value.Len() == 0)
	if isDisputed {
		status, err := NewDisputeStatusVDT()
		if err != nil {
			return CandidateVoteState{}, fmt.Errorf("failed to create dispute status: %w", err)
		}

		if err := status.Set(ActiveStatus{}); err != nil {
			return CandidateVoteState{}, fmt.Errorf("failed to set dispute status: %w", err)
		}

		isConfirmed := votes.VotedIndices().Size() > byzantineThreshold
		if isConfirmed {
			if err := status.Confirm(); err != nil {
				return CandidateVoteState{}, fmt.Errorf("failed to confirm dispute status: %w", err)
			}
		}

		isConcludedFor := votes.Valid.Value.Len() >= superMajorityThreshold
		if isConcludedFor {
			if err := status.ConcludeFor(now); err != nil {
				return CandidateVoteState{}, fmt.Errorf("failed to conclude dispute status for: %w", err)
			}
		}

		isConcludedAgainst := votes.Invalid.Len() >= superMajorityThreshold
		if isConcludedAgainst {
			if err := status.ConcludeAgainst(now); err != nil {
				return CandidateVoteState{}, fmt.Errorf("failed to conclude dispute status against: %w", err)
			}
		}
		disputeStatus = &status
		byzantineThresholdAgainst = votes.Invalid.Len() > byzantineThreshold
	}

	return CandidateVoteState{
		Votes:                     votes,
		Own:                       ownVoteState,
		DisputeStatus:             disputeStatus,
		ByzantineThresholdAgainst: byzantineThresholdAgainst,
	}, nil
}

// NewCandidateVoteStateFromReceipt creates a new CandidateVoteState from a CandidateReceipt
func NewCandidateVoteStateFromReceipt(receipt parachainTypes.CandidateReceipt) (CandidateVoteState, error) {
	votes := NewCandidateVotesFromReceipt(receipt)
	ownVoteState, err := NewOwnVoteStateVDT(CannotVote{})
	if err != nil {
		return CandidateVoteState{}, fmt.Errorf("failed to create own vote state: %w", err)
	}

	return CandidateVoteState{
		Votes: votes,
		Own:   ownVoteState,
	}, nil
}

// ValidCandidateVotes is a list of valid votes for a candidate.
type ValidCandidateVotes struct {
	Value scale.BTreeMap[parachainTypes.ValidatorIndex, Vote]
}

// NewValidCandidateVotes creates a new ValidCandidateVotes.
func NewValidCandidateVotes(degree int) ValidCandidateVotes {
	return ValidCandidateVotes{
		scale.NewBTreeMap[parachainTypes.ValidatorIndex, Vote](degree),
	}
}

// InsertVote Inserts a vote, replacing any already existing vote.
// Except, for backing votes: Backing votes are always kept, and will never get overridden.
// Import of other king of `valid` votes, will be ignored if a backing vote is already
// present. Any already existing `valid` vote, will be overridden by any given backing vote.
//
// Returns: true, if the insert had any effect.
func (vcv ValidCandidateVotes) InsertVote(vote Vote) (bool, error) {
	existingVote, ok := vcv.Value.Get(vote.ValidatorIndex)
	if !ok {
		vcv.Value.Set(vote.ValidatorIndex, vote)
		return true, nil
	}

	disputeStatement, err := existingVote.DisputeStatement.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatement vdt: %w", err)
	}

	switch disputeStatement.(type) {
	case inherents.ValidDisputeStatementKind:
		validStatement := disputeStatement.(inherents.ValidDisputeStatementKind)
		validValue, err := validStatement.Value()
		if err != nil {
			return false, fmt.Errorf("get valid dispute statement value: %w", err)
		}
		switch validValue.(type) {
		case inherents.BackingValid, inherents.BackingSeconded:
			return false, nil
		case inherents.ExplicitValidDisputeStatementKind, inherents.ApprovalChecking:
			vcv.Value.Set(vote.ValidatorIndex, vote)
			return false, nil
		default:
			return false, fmt.Errorf("invalid dispute statement type: %T", disputeStatement)
		}
	case inherents.InvalidDisputeStatementKind:
		invalidStatement := disputeStatement.(inherents.InvalidDisputeStatementKind)
		invalidValue, err := invalidStatement.Value()
		if err != nil {
			return false, fmt.Errorf("get invalid dispute statement value: %w", err)
		}
		switch invalidValue.(type) {
		case inherents.ExplicitInvalidDisputeStatementKind:
			vcv.Value.Set(vote.ValidatorIndex, vote)
			return true, nil
		default:
			return false, fmt.Errorf("invalid dispute statement type: %T", disputeStatement)
		}
	default:
		return false, fmt.Errorf("invalid dispute statement type: %T", disputeStatement)
	}
}

// NewInvalidCandidateVotes creates a new instance of BTreeMap for invalid votes.
func NewInvalidCandidateVotes(degree int) scale.BTreeMap[parachainTypes.ValidatorIndex, Vote] {
	return scale.NewBTreeMap[parachainTypes.ValidatorIndex, Vote](degree)
}

// CandidateVotes is a struct containing the votes for a candidate.
type CandidateVotes struct {
	CandidateReceipt parachainTypes.CandidateReceipt                     `scale:"1"`
	Valid            ValidCandidateVotes                                 `scale:"2"`
	Invalid          scale.BTreeMap[parachainTypes.ValidatorIndex, Vote] `scale:"3"`
}

// Copy returns a copy of the CandidateVotes
func (cv *CandidateVotes) Copy() CandidateVotes {
	return CandidateVotes{
		CandidateReceipt: cv.CandidateReceipt,
		Valid:            ValidCandidateVotes{cv.Valid.Value.Copy()},
		Invalid:          cv.Invalid.Copy(),
	}
}

// VotedIndices returns the set of all validators who have votes in the set, ascending.
func (cv *CandidateVotes) VotedIndices() *treeset.Set {
	votedIndices := treeset.NewWith(parachainTypes.ValidatorIndexComparator)
	cv.Valid.Value.Ascend(0, func(index parachainTypes.ValidatorIndex, vote Vote) bool {
		votedIndices.Add(vote.ValidatorIndex)
		return true
	})

	cv.Invalid.Ascend(0, func(index parachainTypes.ValidatorIndex, vote Vote) bool {
		votedIndices.Add(vote.ValidatorIndex)
		return true
	})

	return votedIndices
}

// Encode returns the SCALE encoding of the CandidateVotes
func (cv *CandidateVotes) Encode() ([]byte, error) {
	// Scale doesn't support BTreeMap encoding, so we need to encode it manually
	encodedReceipt, err := scale.Marshal(cv.CandidateReceipt)
	if err != nil {
		return nil, err
	}

	var encodedValidVotes []byte
	cv.Valid.Value.Ascend(0, func(key parachainTypes.ValidatorIndex, value Vote) bool {
		encodedVote, err := scale.Marshal(value)
		if err != nil {
			return false
		}

		encodedValidVotes = append(encodedValidVotes, encodedVote...)
		return true
	})

	var encodedInvalidVotes []byte
	cv.Invalid.Ascend(0, func(key parachainTypes.ValidatorIndex, value Vote) bool {
		encodedVote, err := scale.Marshal(value)
		if err != nil {
			return false
		}

		encodedInvalidVotes = append(encodedInvalidVotes, encodedVote...)
		return true
	})

	return append(append(encodedReceipt, encodedValidVotes...), encodedInvalidVotes...), nil
}

// NewCandidateVotes creates a new CandidateVotes.
func NewCandidateVotes() CandidateVotes {
	return CandidateVotes{
		Valid:   NewValidCandidateVotes(32),
		Invalid: scale.NewBTreeMap[parachainTypes.ValidatorIndex, Vote](32),
	}
}

// NewCandidateVotesFromReceipt creates a new CandidateVotes from a candidate receipt.
func NewCandidateVotesFromReceipt(receipt parachainTypes.CandidateReceipt) CandidateVotes {
	return CandidateVotes{
		CandidateReceipt: receipt,
		Valid:            NewValidCandidateVotes(32),
		Invalid:          scale.NewBTreeMap[parachainTypes.ValidatorIndex, Vote](32),
	}
}
