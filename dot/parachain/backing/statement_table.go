// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

/*
------------------------------------------
*** Statement table implementation notes:
------------------------------------------

added table2 field of type Table2 in perRelayParentState, which is implementation of statement table.

table field of type Table needs to be removed after implementing the methods in Table2.

rename Table2 to Table after implementing the methods in Table2.


=> At the end check for all the types in this file, if they should be exported or not.
*/

var errCandidateDataNotFound = errors.New("candidate data not found")

type authorityData []proposal

type proposal struct {
	candidateHash parachaintypes.CandidateHash
	signature     parachaintypes.Signature
}

type candidateData struct {
	groupID       parachaintypes.ParaID
	candidate     parachaintypes.CommittedCandidateReceipt
	validityVotes map[parachaintypes.ValidatorIndex]validityVoteWithSign
}

type validityVoteWithSign struct {
	validityVote validityVote
	signature    parachaintypes.Signature
}

type validityVote byte

const (
	// Implicit validity vote.
	issued validityVote = iota
	// Direct validity vote.
	valid
)

// Table configuration.
type tableConfig struct {
	// When this is true, the table will allow multiple seconded candidates
	// per authority. This flag means that higher-level code is responsible for
	// bounding the number of candidates.
	allowMultipleSeconded bool
}

// after finishing implementing statement table, we can remove Table interface,
// and rename StatementTable to Table
type StatementTable struct {
	// TODO: types of fields needs to be identified as we implement the methods

	authorityData map[parachaintypes.ValidatorIndex]authorityData
	// detected_misbehaviour: HashMap<Ctx::AuthorityId, Vec<MisbehaviorFor<Ctx>>>,
	candidateVotes map[parachaintypes.CandidateHash]candidateData
	config         tableConfig
}

func (t StatementTable) getCandidate(candidateHash parachaintypes.CandidateHash,
) (parachaintypes.CommittedCandidateReceipt, error) {
	data, ok := t.candidateVotes[candidateHash]
	if !ok {
		return parachaintypes.CommittedCandidateReceipt{},
			fmt.Errorf("%w for candidate-hash: %s", errCandidateDataNotFound, candidateHash)
	}
	return data.candidate, nil
}

func (StatementTable) importStatement(ctx *TableContext, statement parachaintypes.SignedFullStatementWithPVD,
) (*Summary, error) {
	return nil, nil
}

func (StatementTable) attestedCandidate(candidateHash parachaintypes.CandidateHash, ctx *TableContext,
) (*AttestedCandidate, error) {
	return nil, nil
}

func (StatementTable) drainMisbehaviors() []parachaintypes.ProvisionableDataMisbehaviorReport {
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

// validityAttestation represents a vote on the validity of a candidate by a validator.
type validityAttestation struct {
	ValidatorIndex      parachaintypes.ValidatorIndex      `scale:"1"`
	ValidityAttestation parachaintypes.ValidityAttestation `scale:"2"`
}
