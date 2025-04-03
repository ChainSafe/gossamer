// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var errCandidateDataNotFound = errors.New("candidate data not found")
var errNotEnoughValidityVotes = errors.New("not enough validity votes")
var errUnknownValidityVote = errors.New("unknown validity vote")

var _ Table = (*statementTable)(nil)

// statementTable implements the Table interface.
type statementTable struct {
	authorityData        map[parachaintypes.ValidatorIndex][]proposal
	detectedMisbehaviour map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour
	candidateVotes       map[parachaintypes.CandidateHash]*candidateData
}

type proposal struct {
	candidateHash parachaintypes.CandidateHash
	signature     parachaintypes.ValidatorSignature
}

type candidateData struct {
	groupID       parachaintypes.GroupIndex
	candidate     parachaintypes.CommittedCandidateReceiptV2
	validityVotes map[parachaintypes.ValidatorIndex]validityVoteWithSign
}

func (data *candidateData) getSummary(candidateHash parachaintypes.CandidateHash) *Summary {
	return &Summary{
		GroupID:       data.groupID,
		Candidate:     candidateHash,
		ValidityVotes: uint(len(data.validityVotes)),
	}
}

// attested yields a full attestation for a candidate.
// If the candidate can be included, it will return attested candidate.
func (data *candidateData) attested(validityThreshold uint) (*attestedCandidate, error) {
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
			return nil, fmt.Errorf("%w: %d", errUnknownValidityVote, voteWithSign.validityVote)
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
func (table *statementTable) getCommittedCandidateReceipt(candidateHash parachaintypes.CandidateHash,
) (parachaintypes.CommittedCandidateReceiptV2, error) {
	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return parachaintypes.CommittedCandidateReceiptV2{},
			fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}
	return data.candidate, nil
}

// importStatement imports a statement into the table.
func (table *statementTable) importStatement(
	tableCtx *tableContext, groupID parachaintypes.GroupIndex,
	signedStatement parachaintypes.SignedFullStatement,
) (*Summary, error) {
	var summary *Summary
	var misbehaviour parachaintypes.Misbehaviour

	statementVDT, err := signedStatement.Payload.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from statement: %w", err)
	}

	switch statementVDT := statementVDT.(type) {
	case parachaintypes.Seconded:
		summary, misbehaviour, err = table.importCandidate(
			signedStatement.ValidatorIndex,
			parachaintypes.CommittedCandidateReceiptV2(statementVDT),
			signedStatement.Signature,
			tableCtx,
			groupID,
		)
	case parachaintypes.Valid:
		summary, misbehaviour, err = table.validityVote(
			signedStatement.ValidatorIndex,
			parachaintypes.CandidateHash(statementVDT),
			validityVoteWithSign{validityVote: valid, signature: signedStatement.Signature},
			tableCtx,
		)
	}

	if err != nil && !errors.Is(err, errCandidateDataNotFound) {
		return nil, err
	}

	// If misbehaviour is detected, store it.
	if misbehaviour != nil {
		misbehaviors, ok := table.detectedMisbehaviour[signedStatement.ValidatorIndex]
		if !ok {
			misbehaviors = []parachaintypes.Misbehaviour{misbehaviour}
		} else {
			misbehaviors = append(misbehaviors, misbehaviour)
		}

		table.detectedMisbehaviour[signedStatement.ValidatorIndex] = misbehaviors
	}

	return summary, nil
}

func isCandidateAlreadyProposed(proposals []proposal, candidateHash parachaintypes.CandidateHash) bool {
	return slices.ContainsFunc(proposals, func(p proposal) bool {
		return p.candidateHash == candidateHash
	})
}

func (table *statementTable) importCandidate(
	authority parachaintypes.ValidatorIndex,
	candidate parachaintypes.CommittedCandidateReceiptV2,
	signature parachaintypes.ValidatorSignature,
	tableCtx *tableContext,
	group parachaintypes.GroupIndex,
) (*Summary, parachaintypes.Misbehaviour, error) {
	if !tableCtx.isMemberOf(authority, parachaintypes.CoreIndex{Index: uint32(group)}) {
		statementSeconded := parachaintypes.NewStatementVDT()
		if err := statementSeconded.SetValue(parachaintypes.Seconded(candidate)); err != nil {
			return nil, nil, fmt.Errorf("setting seconded statement: %w", err)
		}

		misbehaviour := parachaintypes.UnauthorizedStatement{
			Payload:        statementSeconded,
			ValidatorIndex: authority,
			Signature:      signature,
		}

		return nil, misbehaviour, nil
	}

	candidateHash, err := parachaintypes.GetCandidateHash(candidate)
	if err != nil {
		return nil, nil, fmt.Errorf("getting candidate hash: %w", err)
	}

	proposals, ok := table.authorityData[authority]
	if !ok {
		// Add the new proposal to the authorityData and update the candidateVotes in the statement table.
		table.authorityData[authority] = []proposal{{candidateHash, signature}}
		table.addCandidateVote(candidateHash, group, candidate)

		return table.validityVote(authority, candidateHash,
			validityVoteWithSign{validityVote: issued, signature: signature}, tableCtx)
	}

	if isCandidateAlreadyProposed(proposals, candidateHash) {
		return table.validityVote(authority, candidateHash,
			validityVoteWithSign{validityVote: issued, signature: signature}, tableCtx)
	}

	// Add the new proposal to the authorityData and update the candidateVotes in the statement table.
	proposals = append(proposals, proposal{candidateHash: candidateHash, signature: signature})
	table.authorityData[authority] = proposals

	table.addCandidateVote(candidateHash, group, candidate)

	return table.validityVote(authority, candidateHash,
		validityVoteWithSign{validityVote: issued, signature: signature}, tableCtx)
}

func (table *statementTable) addCandidateVote(
	candidateHash parachaintypes.CandidateHash,
	groupID parachaintypes.GroupIndex,
	candidate parachaintypes.CommittedCandidateReceiptV2,
) {
	table.candidateVotes[candidateHash] = &candidateData{
		groupID:       groupID,
		candidate:     candidate,
		validityVotes: make(map[parachaintypes.ValidatorIndex]validityVoteWithSign),
	}
}

func (table *statementTable) validityVote(
	from parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
	voteWithSign validityVoteWithSign,
	tableCtx *tableContext,
) (*Summary, parachaintypes.Misbehaviour, error) {
	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return nil, nil, errCandidateDataNotFound
	}

	// check that this authority actually can vote in this group.
	if !tableCtx.isMemberOf(from, parachaintypes.CoreIndex{Index: uint32(data.groupID)}) {
		var sig parachaintypes.ValidatorSignature
		switch voteWithSign.validityVote {
		case valid:
			sig = voteWithSign.signature
		case issued:
			sig = voteWithSign.signature
		}

		validStatement := parachaintypes.NewStatementVDT()
		err := validStatement.SetValue(parachaintypes.Valid(candidateHash))
		if err != nil {
			return nil, nil, fmt.Errorf("setting valid statement: %w", err)
		}

		misbehaviour := parachaintypes.UnauthorizedStatement{
			Payload:        validStatement,
			ValidatorIndex: from,
			Signature:      sig,
		}

		return nil, misbehaviour, nil
	}

	existingVoteWithSign, ok := data.validityVotes[from]
	if !ok {
		data.validityVotes[from] = voteWithSign
		return data.getSummary(candidateHash), nil, nil
	}

	// check for double votes.
	if existingVoteWithSign != voteWithSign {
		var misbehaviour parachaintypes.Misbehaviour

		switch {
		// valid vote conflicting with candidate statement
		case existingVoteWithSign.validityVote == issued && voteWithSign.validityVote == valid,
			existingVoteWithSign.validityVote == valid && voteWithSign.validityVote == issued:
			misbehaviour = parachaintypes.ValidityDoubleVoteIssuedAndValidity{
				CommittedCandidateReceiptAndSign: parachaintypes.CommittedCandidateReceiptAndSign{
					CommittedCandidateReceipt: data.candidate,
					Signature:                 existingVoteWithSign.signature,
				},
				CandidateHashAndSign: parachaintypes.CandidateHashAndSign{
					CandidateHash: candidateHash,
					Signature:     voteWithSign.signature,
				},
			}

		// two signatures on same candidate
		case existingVoteWithSign.validityVote == issued && voteWithSign.validityVote == issued:
			misbehaviour = parachaintypes.DoubleSignOnSeconded{
				Candidate: data.candidate,
				Sign1:     existingVoteWithSign.signature,
				Sign2:     voteWithSign.signature,
			}

		// two signatures on same validity vote
		case existingVoteWithSign.validityVote == valid && voteWithSign.validityVote == valid:
			misbehaviour = parachaintypes.DoubleSignOnValidity{
				CandidateHash: candidateHash,
				Sign1:         existingVoteWithSign.signature,
				Sign2:         voteWithSign.signature,
			}
		}

		return nil, misbehaviour, nil
	}

	return nil, nil, nil
}

// attestedCandidate retrieves the attested candidate for the given candidate hash.
// returns attested candidate  if the candidate exists and is includable.
func (table *statementTable) attestedCandidate(
	candidateHash parachaintypes.CandidateHash, tableCtx *tableContext, minimumBackingVotes uint32,
) (*attestedCandidate, error) {
	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return nil, fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}

	var validityThreshold uint
	if group, ok := tableCtx.groups[parachaintypes.CoreIndex{Index: uint32(data.groupID)}]; ok {
		validityThreshold = effectiveMinimumBackingVotes(uint(len(group)), minimumBackingVotes)
	} else {
		validityThreshold = math.MaxUint
	}

	return data.attested(validityThreshold)
}

// effectiveMinimumBackingVotes adjusts the configured needed backing votes with the size of the backing group.
//
// groupLen is the size of the backing group.
func effectiveMinimumBackingVotes(groupLen uint, configuredMinimumBackingVotes uint32) uint {
	return min(groupLen, uint(configuredMinimumBackingVotes))
}

// drainMisbehaviors returns the current detected misbehaviors and resets the internal map.
func (table *statementTable) drainMisbehaviors() map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour {
	mapToReturn := table.detectedMisbehaviour
	table.detectedMisbehaviour = make(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour)
	return mapToReturn
}

type Table interface {
	getCommittedCandidateReceipt(parachaintypes.CandidateHash) (parachaintypes.CommittedCandidateReceiptV2, error)
	importStatement(*tableContext, parachaintypes.GroupIndex, parachaintypes.SignedFullStatement) (*Summary, error)
	attestedCandidate(parachaintypes.CandidateHash, *tableContext, uint32) (*attestedCandidate, error)
	drainMisbehaviors() map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour
}

func newTable() *statementTable {
	return &statementTable{
		authorityData:        make(map[parachaintypes.ValidatorIndex][]proposal),
		detectedMisbehaviour: make(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour),
		candidateVotes:       make(map[parachaintypes.CandidateHash]*candidateData),
	}
}

// Summary represents summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate parachaintypes.CandidateHash
	// The group that the candidate is in.
	GroupID parachaintypes.GroupIndex
	// How many validity votes are currently witnessed.
	ValidityVotes uint
}

// attestedCandidate represents an attested-to candidate.
type attestedCandidate struct {
	// The group ID that the candidate is in.
	groupID parachaintypes.GroupIndex
	// The committedCandidateReceipt data.
	committedCandidateReceipt parachaintypes.CommittedCandidateReceiptV2
	// Validity attestations.
	validityAttestations []validatorIndexWithAttestation
}

func (attested *attestedCandidate) toBackedCandidate(
	tableCtx *tableContext,
	injectCoreIndex bool,
) (*parachaintypes.BackedCandidate, error) {
	if tableCtx == nil {
		return nil, errors.New("table context is nil")
	}

	// Get validator group for this candidate
	coreIndex := parachaintypes.CoreIndex{Index: uint32(attested.groupID)}
	group, ok := tableCtx.groups[coreIndex]
	if !ok {
		return nil, fmt.Errorf("validator group not found for group ID %d", attested.groupID)
	}

	// Create map of validator indices to their positions in the group
	validatorPositions := make(map[parachaintypes.ValidatorIndex]int, len(group))
	for i, validator := range group {
		validatorPositions[validator] = i
	}

	validatorIndices := make([]bool, len(group))
	attestationsByPosition := make(map[int]parachaintypes.ValidityAttestation)

	for _, attestation := range attested.validityAttestations {
		pos, exists := validatorPositions[attestation.validatorIndex]
		if !exists {
			return nil, fmt.Errorf("validator %d not found in backing group", attestation.validatorIndex)
		}

		validatorIndices[pos] = true
		attestationsByPosition[pos] = attestation.validityAttestation
	}

	// Build sorted attestations list matching order of validator indices
	sortedAttestations := make([]parachaintypes.ValidityAttestation, 0, len(attested.validityAttestations))
	for i := range group {
		if validatorIndices[i] {
			sortedAttestations = append(sortedAttestations, attestationsByPosition[i])
		}
	}

	var coreIndexToInject *parachaintypes.CoreIndex
	if injectCoreIndex {
		coreIndexToInject = &coreIndex
	}

	// The order of the validity votes in the backed candidate must match
	// the order of bits set in the validatorIndices, which is not necessarily
	// the order of the `validityAttestations` we got from the statement table.
	return parachaintypes.NewBackedCandidate(
		attested.committedCandidateReceipt,
		sortedAttestations,
		validatorIndices,
		coreIndexToInject,
	)
}

// validatorIndexWithAttestation represents a validity attestation for a candidate.
type validatorIndexWithAttestation struct {
	validatorIndex      parachaintypes.ValidatorIndex
	validityAttestation parachaintypes.ValidityAttestation
}
