// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"context"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

var ErrRejectedByProspectiveParachains error = fmt.Errorf("rejected by prospective parachains")

type CandidateBacking struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	perRelayParent      map[common.Hash]perRelayParentState
	perCandidate        map[parachaintypes.CandidateHash]perCandidateState
}

type perCandidateState struct {
	persistedValidationData parachaintypes.PersistedValidationData
	secondedLocally         bool
	paraID                  parachaintypes.ParaID
	relayParent             common.Hash
}

type perRelayParentState struct {
	// The `ParaId` assigned to the local validator at this relay parent.
	Assignment parachaintypes.ParaID
	// The table of candidates and statements under this relay-parent.
	Table Table
	// The table context, including groups.
	TableContext TableContext
	// Data needed for retrying in case of `ValidatedCandidateCommand::AttestNoPoV`.
	fallbacks map[parachaintypes.CandidateHash]AttestingData
	// These candidates are undergoing validation in the background.
	AwaitingValidation map[parachaintypes.CandidateHash]bool
}

// In case a backing validator does not provide a PoV, we need to retry with other backing
// validators.
//
// This is the data needed to accomplish this. Basically all the data needed for spawning a
// validation job and a list of backing validators, we can try.
type AttestingData struct {
	// The candidate to attest.
	candidate parachaintypes.CandidateReceipt
	// Hash of the PoV we need to fetch.
	povHash common.Hash
	// Validator we are currently trying to get the PoV from.
	fromValidator parachaintypes.ValidatorIndex
	// Other backing validators we can try in case `from_validator` failed.
	backing []parachaintypes.ValidatorIndex
}

type TableContext struct {
	validator  *Validator
	groups     map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex
	validators []parachaintypes.ValidatorID
}

// Local validator information
//
// It can be created if the local node is a validator in the context of a particular
// relay chain block.
type Validator struct {
	signing_context SigningContext
	key             parachaintypes.ValidatorID
	index           parachaintypes.ValidatorIndex
}

// A type returned by runtime with current session index and a parent hash.
type SigningContext struct {
	/// Current session index.
	SessionIndex parachaintypes.SessionIndex
	/// Hash of the parent.
	ParentHash common.Hash
}

// ActiveLeavesUpdate is a messages from overseer
type ActiveLeavesUpdate struct {
	// TODO: Complete this struct #3503
}

// GetBackedCandidates is a message received from overseer that requests a set of backable
// candidates that could be backed in a child of the given relay-parent.
type GetBackedCandidates []struct {
	CandidateHash        parachaintypes.CandidateHash
	CandidateRelayParent common.Hash
}

// CanSecond is a request made to the candidate backing subsystem to determine whether it is permissible
// to second a given candidate.
// The rule for seconding candidates is: Collations must either be built on top of the root of a fragment tree
// or have a parent node that represents the backed candidate.
type CanSecond struct {
	CandidateParaID      parachaintypes.ParaID
	CandidateRelayParent common.Hash
	CandidateHash        parachaintypes.CandidateHash
	ParentHeadDataHash   common.Hash
}

// Second is a message received from overseer. Candidate Backing subsystem should second the given
// candidate in the context of the given relay parent. This candidate must be validated.
type Second struct {
	RelayParent             common.Hash
	CandidateReceipt        parachaintypes.CandidateReceipt
	PersistedValidationData parachaintypes.PersistedValidationData
	PoV                     parachaintypes.PoV
}

// Statement represents a validator's assessment of a specific candidate. If there are disagreements
// regarding the validity of this assessment, they should be addressed through the Disputes Subsystem,
// with the actual escalation deferred until the approval voting stage to ensure its availability.
// Meanwhile, agreements are straightforwardly counted until a quorum is achieved.
type Statement struct {
	RelayParent         common.Hash
	SignedFullStatement SignedFullStatementWithPVD
}

// SignedFullStatementWithPVD represents a signed full statement along with associated Persisted Validation Data (PVD).
type SignedFullStatementWithPVD struct {
	SignedFullStatement     parachaintypes.UncheckedSignedFullStatement
	PersistedValidationData *parachaintypes.PersistedValidationData
}

func New(overseerChan chan<- any) *CandidateBacking {
	return &CandidateBacking{
		SubSystemToOverseer: overseerChan,
	}
}

func (cb *CandidateBacking) Run(ctx context.Context, overseerToSubSystem chan any, subSystemToOverseer chan any) error {
	// TODO: handle_validated_candidate_command
	// There is one more case where we handle results of candidate validation.
	// My feeling is that instead of doing it here, we would be able to do that along with processing
	// other backing related overseer message.
	// This would become more clear after we complete processMessages function. It would give us clarity
	// if we need background_validation_rx or background_validation_tx, as done in rust.
	cb.processMessages()
	return nil
}

func (cb *CandidateBacking) processMessages() {
	for msg := range cb.OverseerToSubSystem {
		// process these received messages by referencing
		// https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/core/backing/src/lib.rs#L741
		switch msg := msg.(type) {
		case ActiveLeavesUpdate:
			cb.handleActiveLeavesUpdate()
		case GetBackedCandidates:
			cb.handleGetBackedCandidates()
		case CanSecond:
			cb.handleCanSecond()
		case Second:
			cb.handleSecond()
		case Statement:
			cb.handleStatement(msg.RelayParent, msg.SignedFullStatement)
		default:
			logger.Error("unknown message type")
		}
	}
}

func (cb *CandidateBacking) handleActiveLeavesUpdate() {
	// TODO: Implement this #3503
}

func (cb *CandidateBacking) handleGetBackedCandidates() {
	// TODO: Implement this #3504
}

func (cb *CandidateBacking) handleCanSecond() {
	// TODO: Implement this #3505
}

func (cb *CandidateBacking) handleSecond() {
	// TODO: Implement this #3506
}

func (cb *CandidateBacking) handleStatement(relayParent common.Hash, statement SignedFullStatementWithPVD) error {
	// TODO: Implement this #3507

	rpState, ok := cb.perRelayParent[relayParent]
	if !ok {
		logger.Tracef("Received statement for unknown relay parent %s", relayParent)
		return nil
	}

	summery, err := importStatement()
	if err != nil {
		if err == ErrRejectedByProspectiveParachains {
			logger.Debug("Statement rejected by prospective parachains")
			return nil
		}
		return err
	}

	if err := postImportStatement(); err != nil {
		return err
	}

	if summery == nil || summery.GroupID != uint32(rpState.Assignment) {
		return nil
	}

	statementVDT, err := statement.SignedFullStatement.Payload.Value()
	if err != nil {
		return fmt.Errorf("getting statementVDT value: %w", err)
	}

	var attestingData AttestingData
	switch statementVDT.Index() {
	case 1: // Seconded
		commitedCandidateReceipt, err := rpState.Table.getCandidate(summery.Candidate)
		if err != nil {
			return fmt.Errorf("getting candidate: %w", err)
		}

		candidateReceipt := parachaintypes.CandidateReceipt{
			Descriptor:      commitedCandidateReceipt.Descriptor,
			CommitmentsHash: common.MustBlake2bHash(scale.MustMarshal(commitedCandidateReceipt.Commitments)),
		}

		attestingData = AttestingData{
			candidate:     candidateReceipt,
			povHash:       statementVDT.(parachaintypes.Seconded).Descriptor.PovHash,
			fromValidator: statement.SignedFullStatement.ValidatorIndex,
			backing:       []parachaintypes.ValidatorIndex{},
		}

		rpState.fallbacks[summery.Candidate] = attestingData

	case 2: // Valid
		attesting, ok := rpState.fallbacks[summery.Candidate]
		if !ok {
			return nil
		}

		ourIndex := rpState.TableContext.validator.index
		if statement.SignedFullStatement.ValidatorIndex == ourIndex {
			return nil
		}

		if rpState.AwaitingValidation[summery.Candidate] {
			// Job already running
			attesting.backing = append(attesting.backing, statement.SignedFullStatement.ValidatorIndex)
			return nil
		}
		// No job, so start another with current validator:
		attesting.fromValidator = statement.SignedFullStatement.ValidatorIndex
		attestingData = attesting

	default:
		return fmt.Errorf("invalid statementVDT index: %d", statementVDT.Index())
	}

	if pc, ok := cb.perCandidate[summery.Candidate]; ok {
		if err := kickOffValidationWork(pc.persistedValidationData); err != nil {
			return fmt.Errorf("validating candidate: %w", err)
		}
	}

	return nil
}

func importStatement() (*Summary, error) {
	// TODO: Implement this
	return &Summary{}, nil
}

func postImportStatement() error {
	// TODO: Implement this
	return nil
}

func kickOffValidationWork(parachaintypes.PersistedValidationData) error {
	// TODO: Implement this
	return nil
}
