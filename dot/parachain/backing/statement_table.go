// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var errCandidateDataNotFound = errors.New("candidate data not found")
var errNotEnoughValidityVotes = errors.New("not enough validity votes")

// statementTable implements the Table interface.
type statementTable struct {
	authorityData  map[parachaintypes.ValidatorIndex]authorityData //nolint:unused
	candidateVotes map[parachaintypes.CandidateHash]candidateData
	config         tableConfig //nolint:unused

	// TODO: Implement this
	// detected_misbehaviour: HashMap<Ctx::AuthorityId, Vec<MisbehaviorFor<Ctx>>>,
}

type authorityData []proposal //nolint:unused

type proposal struct { //nolint:unused
	candidateHash parachaintypes.CandidateHash
	signature     parachaintypes.Signature
}

type candidateData struct {
	groupID       parachaintypes.ParaID
	candidate     parachaintypes.CommittedCandidateReceipt
	validityVotes map[parachaintypes.ValidatorIndex]validityVoteWithSign
}

// attested yields a full attestation for a candidate.
// If the candidate can be included, it will return attested candidate.
func (data candidateData) attested(validityThreshold uint) (*attestedCandidate, error) {
	numOfValidityVotes := uint(len(data.validityVotes))
	if numOfValidityVotes < validityThreshold {
		return nil, fmt.Errorf("%w: %d < %d", errNotEnoughValidityVotes, numOfValidityVotes, validityThreshold)
	}

	validityAttestations := make([]validatorIndexWithAttestation, 0, numOfValidityVotes)
	for validatorIndex, voteWithSign := range data.validityVotes {
		switch voteWithSign.validityVote {
		case valid:
			attestation := parachaintypes.NewValidityAttestation()
			err := attestation.SetValue(parachaintypes.Explicit(voteWithSign.signature))
			if err != nil {
				return nil, fmt.Errorf("failed to set validity attestation: %w", err)
			}

			validityAttestations = append(validityAttestations, validatorIndexWithAttestation{
				validatorIndex:      validatorIndex,
				validityAttestation: attestation,
			})
		case issued:
			attestation := parachaintypes.NewValidityAttestation()
			err := attestation.SetValue(parachaintypes.Implicit(voteWithSign.signature))
			if err != nil {
				return nil, fmt.Errorf("failed to set validity attestation: %w", err)
			}

			validityAttestations = append(validityAttestations, validatorIndexWithAttestation{
				validatorIndex:      validatorIndex,
				validityAttestation: attestation,
			})
		default:
			return nil, fmt.Errorf("unknown validity vote: %d", voteWithSign.validityVote)
		}
	}

	slices.SortFunc(validityAttestations, func(i, j validatorIndexWithAttestation) int {
		return cmp.Compare(i.validatorIndex, j.validatorIndex)
	})

	return &attestedCandidate{
		groupID:                   data.groupID,
		committedCandidateReceipt: data.candidate,
		validityAttestations:      validityAttestations,
	}, nil
}

type validityVoteWithSign struct {
	validityVote validityVote
	signature    parachaintypes.ValidatorSignature // NOTE: should never be empty
}

type validityVote byte

// To make sure the validity vote has a value assigned, we use iota + 1.
const (
	// Implicit validity vote.
	issued validityVote = iota + 1
	// Direct validity vote.
	valid
)

// getCommittedCandidateReceipt returns the committed candidate receipt for the given candidate hash.
func (table *statementTable) getCommittedCandidateReceipt(candidateHash parachaintypes.CandidateHash, //nolint:unused
) (parachaintypes.CommittedCandidateReceipt, error) {
	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return parachaintypes.CommittedCandidateReceipt{},
			fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}
	return data.candidate, nil
}

func (statementTable) importStatement( //nolint:unused
	ctx *TableContext, statement parachaintypes.SignedFullStatementWithPVD,
) (*Summary, error) {
	// TODO: Implement this method
	return nil, nil
}

// attestedCandidate retrieves the attested candidate for the given candidate hash.
// returns attested candidate  if the candidate exists and is includable.
func (table statementTable) attestedCandidate(
	candidateHash parachaintypes.CandidateHash, tableContext *TableContext, minimumBackingVotes uint32,
) (*attestedCandidate, error) {
	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return nil, fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}

	var validityThreshold uint
	group, ok := tableContext.groups[data.groupID]
	if ok {
		// size of the backing group.
		groupLen := uint(len(group))
		validityThreshold = effectiveMinimumBackingVotes(groupLen, minimumBackingVotes)
	} else {
		validityThreshold = uint(minimumBackingVotes)
	}

	return data.attested(validityThreshold)
}

// effectiveMinimumBackingVotes adjusts the configured needed backing votes with the size of the backing group.
//
// groupLen is the size of the backing group.
func effectiveMinimumBackingVotes(groupLen uint, configuredMinimumBackingVotes uint32) uint {
	return min(groupLen, uint(configuredMinimumBackingVotes))
}

func (statementTable) drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport { //nolint:unused
	// TODO: Implement this method
	return nil
}

type Table interface {
	getCandidate(parachaintypes.CandidateHash) (parachaintypes.CommittedCandidateReceipt, error)
	importStatement(*TableContext, parachaintypes.SignedFullStatementWithPVD) (*Summary, error)
	attestedCandidate(parachaintypes.CandidateHash, *TableContext, uint32) (*attestedCandidate, error)
	drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport
}

func newTable(tableConfig) Table {
	// TODO: Implement this function
	return nil
}

// Summary represents summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate parachaintypes.CandidateHash
	// The group that the candidate is in.
	GroupID parachaintypes.ParaID
	// How many validity votes are currently witnessed.
	ValidityVotes uint64
}

// attestedCandidate represents an attested-to candidate.
type attestedCandidate struct {
	// The group ID that the candidate is in.
	groupID parachaintypes.ParaID
	// The committedCandidateReceipt data.
	committedCandidateReceipt parachaintypes.CommittedCandidateReceipt
	// Validity attestations.
	validityAttestations []validatorIndexWithAttestation
}

// validatorIndexWithAttestation represents a validity attestation for a candidate.
type validatorIndexWithAttestation struct {
	validatorIndex      parachaintypes.ValidatorIndex
	validityAttestation parachaintypes.ValidityAttestation
}

// Table configuration.
type tableConfig struct {
	// When this is true, the table will allow multiple seconded candidates
	// per authority. This flag means that higher-level code is responsible for
	// bounding the number of candidates.
	allowMultipleSeconded bool
}
