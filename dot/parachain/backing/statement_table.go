// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"
	"math"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var errCandidateDataNotFound = errors.New("candidate data not found") //nolint:unused

// statementTable implements the Table interface.
type statementTable struct { //nolint:unused
	authorityData  map[parachaintypes.ValidatorIndex]authorityData
	candidateVotes map[parachaintypes.CandidateHash]candidateData
	config         tableConfig

	// TODO: Implement this
	// detected_misbehaviour: HashMap<Ctx::AuthorityId, Vec<MisbehaviorFor<Ctx>>>,
}

type authorityData []proposal //nolint:unused

type proposal struct { //nolint:unused
	candidateHash parachaintypes.CandidateHash
	signature     parachaintypes.Signature
}

type candidateData struct { //nolint:unused
	groupID       parachaintypes.ParaID
	candidate     parachaintypes.CommittedCandidateReceipt
	validityVotes map[parachaintypes.ValidatorIndex]validityVoteWithSign
}

// attested yields a full attestation for a candidate.
// If the candidate can be included, it will return attested candidate.
func (data candidateData) attested(validityThreshold uint) (*AttestedCandidate, error) { //nolint:unused
	numOfValidityVotes := uint(len(data.validityVotes))
	if numOfValidityVotes < validityThreshold {
		return nil, fmt.Errorf("not enough validity votes: %d < %d", numOfValidityVotes, validityThreshold)
	}

	validityAttestations := make([]validityAttestation, numOfValidityVotes)
	for validatorIndex, voteWithSign := range data.validityVotes {
		switch voteWithSign.validityVote {
		case valid:
			attestation := parachaintypes.NewValidityAttestation()
			err := attestation.Set(parachaintypes.Explicit(voteWithSign.signature))
			if err != nil {
				return nil, fmt.Errorf("failed to set validity attestation: %w", err)
			}

			validityAttestations = append(validityAttestations, validityAttestation{
				ValidatorIndex:      validatorIndex,
				ValidityAttestation: attestation,
			})
		case issued:
			attestation := parachaintypes.NewValidityAttestation()
			err := attestation.Set(parachaintypes.Implicit(voteWithSign.signature))
			if err != nil {
				return nil, fmt.Errorf("failed to set validity attestation: %w", err)
			}

			validityAttestations = append(validityAttestations, validityAttestation{
				ValidatorIndex:      validatorIndex,
				ValidityAttestation: attestation,
			})
		default:
			return nil, fmt.Errorf("unknown validity vote: %d", voteWithSign.validityVote)
		}
	}

	return &AttestedCandidate{
		GroupID:              data.groupID,
		Candidate:            data.candidate,
		ValidityAttestations: validityAttestations,
	}, nil
}

type validityVoteWithSign struct { //nolint:unused
	validityVote validityVote
	signature    parachaintypes.ValidatorSignature
}

type validityVote byte //nolint:unused

const (
	// Implicit validity vote.
	issued validityVote = iota //nolint:unused
	// Direct validity vote.
	valid //nolint:unused
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
func (table statementTable) attestedCandidate( //nolint:unused
	candidateHash parachaintypes.CandidateHash, tableContext *TableContext, minimumBackingVotes uint32,
) (*AttestedCandidate, error) {
	// size of the backing group.
	var groupLen uint

	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return nil, fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}

	group, ok := tableContext.groups[data.groupID]
	if ok {
		groupLen = uint(len(group))
	} else {
		groupLen = math.MaxUint
	}

	validityThreshold := effectiveMinimumBackingVotes(groupLen, minimumBackingVotes)
	return data.attested(validityThreshold)
}

// effectiveMinimumBackingVotes adjusts the configured needed backing votes with the size of the backing group.
//
// groupLen is the size of the backing group.
func effectiveMinimumBackingVotes(groupLen uint, configuredMinimumBackingVotes uint32) uint { //nolint:unused
	return min(groupLen, uint(configuredMinimumBackingVotes))
}

func (statementTable) drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport { //nolint:unused
	// TODO: Implement this method
	return nil
}

type Table interface {
	getCandidate(parachaintypes.CandidateHash) (parachaintypes.CommittedCandidateReceipt, error)
	importStatement(*TableContext, parachaintypes.SignedFullStatementWithPVD) (*Summary, error)
	attestedCandidate(parachaintypes.CandidateHash, *TableContext, uint32) (*AttestedCandidate, error)
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

// AttestedCandidate represents an attested-to candidate.
type AttestedCandidate struct {
	// The group ID that the candidate is in.
	GroupID parachaintypes.ParaID `scale:"1"`
	// The candidate data.
	Candidate parachaintypes.CommittedCandidateReceipt `scale:"2"`
	// Validity attestations.
	ValidityAttestations []validityAttestation `scale:"3"`
}

// validityAttestation represents a validity attestation for a candidate.
type validityAttestation struct {
	ValidatorIndex      parachaintypes.ValidatorIndex      `scale:"1"`
	ValidityAttestation parachaintypes.ValidityAttestation `scale:"2"`
}

// Table configuration.
type tableConfig struct {
	// When this is true, the table will allow multiple seconded candidates
	// per authority. This flag means that higher-level code is responsible for
	// bounding the number of candidates.
	allowMultipleSeconded bool
}
