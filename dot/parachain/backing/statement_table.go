// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

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

type validityVoteWithSign struct { //nolint:unused
	validityVote validityVote
	signature    parachaintypes.Signature
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

func (statementTable) attestedCandidate(candidateHash parachaintypes.CandidateHash, ctx *TableContext, //nolint:unused
) (*AttestedCandidate, error) {
	// TODO: Implement this method
	return nil, nil
}

func (statementTable) drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport { //nolint:unused
	// TODO: Implement this method
	return nil
}

type Table interface {
	getCandidate(parachaintypes.CandidateHash) (parachaintypes.CommittedCandidateReceipt, error)
	importStatement(*TableContext, parachaintypes.SignedFullStatementWithPVD) (*Summary, error)
	attestedCandidate(parachaintypes.CandidateHash, *TableContext) (*AttestedCandidate, error)
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
