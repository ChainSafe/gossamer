package dispute

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
)

// ImportResult is an ongoing statement/vote import
type ImportResult interface {
	// VotesChanged returns true if any votes were changed during the import
	VotesChanged() bool
	// DisputeStateChanged returns true if the dispute state changed during the import
	DisputeStateChanged() (bool, error)
	// IsFreshlyDisputed returns true if the dispute state changed from undisputed to disputed during the import
	IsFreshlyDisputed() bool
	// IsFreshlyConfirmed returns true if the dispute state changed to confirmed during the import
	IsFreshlyConfirmed() (bool, error)
	// IsFreshlyConcludedFor returns true if the dispute state changed to concluded for during the import
	IsFreshlyConcludedFor() (bool, error)
	// IsFreshlyConcludedAgainst returns true if the dispute state changed to concluded against during the import
	IsFreshlyConcludedAgainst() (bool, error)
	// IsFreshlyConcluded returns true if the dispute state changed to concluded during the import
	IsFreshlyConcluded() (bool, error)
	// ImportApprovalVotes imports the given approval votes into the current import
	ImportApprovalVotes(approvalVotes []overseer.ApprovalSignature, now uint64) (ImportResult, error)
	// IntoUpdatedVotes returns the updated votes after the import
	IntoUpdatedVotes() types.CandidateVotes
}

// ImportResultHandler implements ImportResult interface
type ImportResultHandler struct {
	// oldState the state before the import
	oldState types.CandidateVoteState
	// newState the state after the importing new statements
	newState types.CandidateVoteState
	// newInvalidVoters the new invalid voters as of this import
	newInvalidVoters []parachainTypes.ValidatorIndex
	// importedInvalidVotes number of invalid voters
	importedInvalidVotes uint32
	// importedValidVotes number of valid voters
	importedValidVotes uint32
	// importedApprovalVotes number of approval votes imported via ImportApprovalVotes()
	importedApprovalVotes uint32
}

func (i ImportResultHandler) VotesChanged() bool {
	return i.importedValidVotes != 0 || i.importedInvalidVotes != 0
}

func (i ImportResultHandler) DisputeStateChanged() (bool, error) {
	isFreshlyConfirmed, err := i.IsFreshlyConfirmed()
	if err != nil {
		return false, fmt.Errorf("checking if freshly confirmed: %w", err)
	}

	isFreshlyConcluded, err := i.IsFreshlyConcluded()
	if err != nil {
		return false, fmt.Errorf("checking if freshly concluded: %w", err)
	}

	return i.IsFreshlyDisputed() || isFreshlyConfirmed || isFreshlyConcluded, nil
}

func (i ImportResultHandler) IsFreshlyDisputed() bool {
	return !i.oldState.IsDisputed() && i.newState.IsDisputed()
}

func (i ImportResultHandler) IsFreshlyConfirmed() (bool, error) {
	isOldStateConfirmed, err := i.oldState.IsConfirmed()
	if err != nil {
		return false, fmt.Errorf("checking if old state is confirmed: %w", err)
	}

	isNewStateConfirmed, err := i.newState.IsConfirmed()
	if err != nil {
		return false, fmt.Errorf("checking if new state is confirmed: %w", err)
	}

	return !isOldStateConfirmed && isNewStateConfirmed, nil
}

func (i ImportResultHandler) IsFreshlyConcludedFor() (bool, error) {
	isOldStateConcludedFor, err := i.oldState.IsConcludedFor()
	if err != nil {
		return false, fmt.Errorf("checking if old state is concluded for: %w", err)
	}

	isNewStateConcludedFor, err := i.newState.IsConcludedFor()
	if err != nil {
		return false, fmt.Errorf("checking if new state is concluded for: %w", err)
	}

	return !isOldStateConcludedFor && isNewStateConcludedFor, nil
}

func (i ImportResultHandler) IsFreshlyConcludedAgainst() (bool, error) {
	isOldStateConcludedAgainst, err := i.oldState.IsConcludedAgainst()
	if err != nil {
		return false, fmt.Errorf("checking if old state is concluded against: %w", err)
	}

	isNewStateConcludedAgainst, err := i.newState.IsConcludedAgainst()
	if err != nil {
		return false, fmt.Errorf("checking if new state is concluded against: %w", err)
	}

	return !isOldStateConcludedAgainst && isNewStateConcludedAgainst, nil
}

func (i ImportResultHandler) IsFreshlyConcluded() (bool, error) {
	isFreshlyConcludedFor, err := i.IsFreshlyConcludedFor()
	if err != nil {
		return false, fmt.Errorf("checking if freshly concluded for: %w", err)
	}

	isFreshlyConcludedAgainst, err := i.IsFreshlyConcludedAgainst()
	if err != nil {
		return false, fmt.Errorf("checking if freshly concluded against: %w", err)
	}

	return isFreshlyConcludedFor || isFreshlyConcludedAgainst, nil
}

func (i ImportResultHandler) ImportApprovalVotes(approvalVotes []overseer.ApprovalSignature, now uint64) (ImportResult, error) {
	votes := i.newState.Votes

	for _, approvalVote := range approvalVotes {
		// TODO: validate signature

		existingVote := votes.Valid.Value.Get(approvalVote)
		if existingVote == nil {
			votes.Valid.Value.Set(approvalVote)
			i.importedValidVotes++
			i.importedApprovalVotes++
		}
	}

	newState, err := types.NewCandidateVoteState(votes, now)
	if err != nil {
		return nil, fmt.Errorf("creating new candidate vote state: %w", err)
	}

	return &ImportResultHandler{
		oldState:              i.oldState,
		newState:              newState,
		newInvalidVoters:      i.newInvalidVoters,
		importedInvalidVotes:  i.importedInvalidVotes,
		importedValidVotes:    i.importedValidVotes,
		importedApprovalVotes: i.importedApprovalVotes,
	}, nil
}

func (i ImportResultHandler) IntoUpdatedVotes() types.CandidateVotes {
	if !i.VotesChanged() {
		return types.CandidateVotes{}
	}

	return i.newState.Votes
}

var _ ImportResult = (*ImportResultHandler)(nil)

func NewImportResultFromStatements(
	env *types.CandidateEnvironment,
	statements []types.Statement,
	candidateVoteState types.CandidateVoteState,
	now uint64,
) (*ImportResultHandler, error) {
	votes, oldState, err := candidateVoteState.IntoOldState()
	if err != nil {
		return nil, fmt.Errorf("get old state: %w", err)
	}

	var (
		newInvalidVoters     []parachainTypes.ValidatorIndex
		importedInvalidVotes uint32
		importedValidVotes   uint32
	)
	expectedCandidateHash, err := votes.CandidateReceipt.Hash()
	if err != nil {
		return nil, fmt.Errorf("get candidate receipt hash: %w", err)
	}

	for _, statement := range statements {
		if statement.ValidatorIndex < parachainTypes.ValidatorIndex(len(env.Session.Validators)) {
			validator := env.Session.Validators[statement.ValidatorIndex]
			if statement.SignedDisputeStatement.ValidatorPublic != validator {
				continue
			}
		}

		if statement.SignedDisputeStatement.CandidateHash != expectedCandidateHash {
			continue
		}

		if statement.SignedDisputeStatement.SessionIndex != env.SessionIndex {
			continue
		}

		disputeStatement, err := statement.SignedDisputeStatement.DisputeStatement.Value()
		if err != nil {
			continue
		}
		switch disputeStatement.(type) {
		case inherents.ValidDisputeStatementKind:
			vote := types.Vote{
				ValidatorIndex:     statement.ValidatorIndex,
				ValidatorSignature: statement.SignedDisputeStatement.ValidatorSignature,
				DisputeStatement:   statement.SignedDisputeStatement.DisputeStatement,
			}
			fresh, err := votes.Valid.InsertVote(vote)
			if err != nil {
				return nil, fmt.Errorf("inserting valid vote: %w", err)
			}

			if fresh {
				importedValidVotes++
			}
		case inherents.InvalidDisputeStatementKind:
			vote := types.Vote{
				ValidatorIndex:     statement.ValidatorIndex,
				ValidatorSignature: statement.SignedDisputeStatement.ValidatorSignature,
				DisputeStatement:   statement.SignedDisputeStatement.DisputeStatement,
			}
			if v := votes.Invalid.Get(vote); v == nil {
				votes.Invalid.Set(vote)
				importedInvalidVotes++
				newInvalidVoters = append(newInvalidVoters, statement.ValidatorIndex)
			}
		}
	}

	newState, err := types.NewCandidateVoteState(votes, now)
	if err != nil {
		return nil, fmt.Errorf("creating new candidate vote state: %w", err)
	}

	return &ImportResultHandler{
		oldState:              oldState,
		newState:              newState,
		newInvalidVoters:      newInvalidVoters,
		importedInvalidVotes:  importedInvalidVotes,
		importedValidVotes:    importedValidVotes,
		importedApprovalVotes: 0,
	}, nil
}
