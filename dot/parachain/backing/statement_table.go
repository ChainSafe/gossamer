// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
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
	config               tableConfig
}

type proposal struct {
	candidateHash parachaintypes.CandidateHash
	signature     parachaintypes.ValidatorSignature
}

type candidateData struct {
	groupID       parachaintypes.ParaID
	candidate     parachaintypes.CommittedCandidateReceipt
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
			err := attestation.Set(parachaintypes.Explicit(voteWithSign.signature))
			if err != nil {
				return nil, fmt.Errorf("failed to set validity attestation: %w", err)
			}

			validityAttestations = append(validityAttestations, validatorIndexWithAttestation{
				validatorIndex:      validatorIndex,
				validityAttestation: attestation,
			})
		case issued:
			attestation := parachaintypes.NewValidityAttestation()
			err := attestation.Set(parachaintypes.Implicit(voteWithSign.signature))
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
) (parachaintypes.CommittedCandidateReceipt, error) {
	data, ok := table.candidateVotes[candidateHash]
	if !ok {
		return parachaintypes.CommittedCandidateReceipt{},
			fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}
	return data.candidate, nil
}

func (table *statementTable) importStatement(
	tableCtx *tableContext, signedStatement parachaintypes.SignedFullStatement,
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
			parachaintypes.CommittedCandidateReceipt(statementVDT),
			signedStatement.Signature,
			tableCtx,
		)
	case parachaintypes.Valid:
		summary, misbehaviour, err = table.validityVote(
			signedStatement.ValidatorIndex,
			parachaintypes.CandidateHash(statementVDT),
			validityVoteWithSign{validityVote: valid, signature: signedStatement.Signature},
			tableCtx,
		)
	}

	if err != nil {
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
	candidate parachaintypes.CommittedCandidateReceipt,
	signature parachaintypes.ValidatorSignature,
	tableCtx *tableContext,
) (*Summary, parachaintypes.Misbehaviour, error) {
	paraID := parachaintypes.ParaID(candidate.Descriptor.ParaID)

	if !tableCtx.isMemberOf(authority, paraID) {
		statementSeconded := parachaintypes.NewStatementVDT()
		err := statementSeconded.Set(parachaintypes.Seconded(candidate))
		if err != nil {
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
		table.authorityData[authority] = []proposal{{candidateHash, signature}}
		table.addCandidateVote(candidateHash, paraID, candidate)

		return table.validityVote(authority, candidateHash,
			validityVoteWithSign{validityVote: issued, signature: signature}, tableCtx)
	}

	switch {
	case !table.config.allowMultipleSeconded && len(proposals) == 1:
		oldCandidateHash := proposals[0].candidateHash
		oldSignature := proposals[0].signature

		// if digest is different, fetch candidate and note misbehaviour.
		if oldCandidateHash != candidateHash {
			data, ok := table.candidateVotes[oldCandidateHash]
			if !ok {
				// when proposal first received from authority, candidate votes entry is created.
				// and here authData is not empty, so candidate votes entry should be present.
				// So, this error should never happen.
				return nil, nil, fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, oldCandidateHash)
			}

			oldCandidate := data.candidate

			misbehaviour := parachaintypes.MultipleCandidates{
				First: parachaintypes.CommittedCandidateReceiptAndSign{
					CommittedCandidateReceipt: oldCandidate,
					Signature:                 oldSignature,
				},
				Second: parachaintypes.CommittedCandidateReceiptAndSign{
					CommittedCandidateReceipt: candidate,
					Signature:                 signature,
				},
			}
			return nil, misbehaviour, nil
		}
	case table.config.allowMultipleSeconded && isCandidateAlreadyProposed(proposals, candidateHash):
		// nothing to do here.
	default:
		proposals = append(proposals, proposal{candidateHash, signature})
		table.authorityData[authority] = proposals

		table.addCandidateVote(candidateHash, paraID, candidate)
	}

	return table.validityVote(authority, candidateHash,
		validityVoteWithSign{validityVote: issued, signature: signature}, tableCtx)
}

func (table *statementTable) addCandidateVote(
	candidateHash parachaintypes.CandidateHash,
	paraID parachaintypes.ParaID,
	candidate parachaintypes.CommittedCandidateReceipt,
) {
	table.candidateVotes[candidateHash] = &candidateData{
		groupID:       paraID,
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
		return nil, nil, fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}

	// check that this authority actually can vote in this group.
	if !tableCtx.isMemberOf(from, data.groupID) {
		switch voteWithSign.validityVote {
		case valid:
			validStatement := parachaintypes.NewStatementVDT()
			err := validStatement.Set(parachaintypes.Valid(candidateHash))
			if err != nil {
				return nil, nil, fmt.Errorf("setting valid statement: %w", err)
			}

			misbehaviour := parachaintypes.UnauthorizedStatement{
				Payload:        validStatement,
				ValidatorIndex: from,
				Signature:      voteWithSign.signature,
			}

			return nil, misbehaviour, nil
		case issued:
			panic("implicit issuance vote must only cast from `importCandidate` after checking group membership of issuer.")
		default:
			return nil, nil, fmt.Errorf("%w: %d", errUnknownValidityVote, voteWithSign.validityVote)
		}
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
		case existingVoteWithSign.validityVote == issued && voteWithSign.validityVote == issued:
			misbehaviour = parachaintypes.DoubleSignOnSeconded{
				Candidate: data.candidate,
				Sign1:     existingVoteWithSign.signature,
				Sign2:     voteWithSign.signature,
			}
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
	group, ok := tableCtx.groups[data.groupID]
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

func (statementTable) drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport {
	// TODO: Implement this method
	return nil
}

type Table interface {
	getCommittedCandidateReceipt(parachaintypes.CandidateHash) (parachaintypes.CommittedCandidateReceipt, error)
	importStatement(*tableContext, parachaintypes.SignedFullStatement) (*Summary, error)
	attestedCandidate(parachaintypes.CandidateHash, *tableContext, uint32) (*attestedCandidate, error)
	drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport
}

func newTable(config tableConfig) Table {
	return &statementTable{
		authorityData:        make(map[parachaintypes.ValidatorIndex][]proposal),
		detectedMisbehaviour: make(map[parachaintypes.ValidatorIndex][]parachaintypes.Misbehaviour),
		candidateVotes:       make(map[parachaintypes.CandidateHash]*candidateData),
		config:               config,
	}
}

// Summary represents summary of import of a statement.
type Summary struct {
	// The digest of the candidate referenced.
	Candidate parachaintypes.CandidateHash
	// The group that the candidate is in.
	GroupID parachaintypes.ParaID
	// How many validity votes are currently witnessed.
	ValidityVotes uint
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

func (attested *attestedCandidate) toBackedCandidate(tableCtx *tableContext) *parachaintypes.BackedCandidate {
	group := tableCtx.groups[attested.groupID]
	validatorIndices := make([]bool, len(group))
	var validityAttestations []parachaintypes.ValidityAttestation

	// The order of the validity votes in the backed candidate must match
	// the order of bits set in the bitfield, which is not necessarily
	// the order of the `validity_votes` we got from the table.
	for positionInGroup, validatorIndex := range group {
		for _, validityVote := range attested.validityAttestations {
			if validityVote.validatorIndex == validatorIndex {
				validatorIndices[positionInGroup] = true
				validityAttestations = append(validityAttestations, validityVote.validityAttestation)
			}
		}

		if !validatorIndices[positionInGroup] {
			logger.Error("validity vote from unknown validator")
			return nil
		}
	}

	return &parachaintypes.BackedCandidate{
		Candidate:        attested.committedCandidateReceipt,
		ValidityVotes:    validityAttestations,
		ValidatorIndices: scale.NewBitVec(validatorIndices),
	}
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
