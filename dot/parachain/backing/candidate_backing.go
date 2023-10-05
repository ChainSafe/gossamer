// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

type CandidateBacking struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
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
		switch msg.(type) {
		case ActiveLeavesUpdate:
			cb.handleActiveLeavesUpdate()
		case GetBackedCandidates:
			cb.handleGetBackedCandidates()
		case CanSecond:
			cb.handleCanSecond()
		case Second:
			cb.handleSecond()
		case Statement:
			cb.handleStatement()
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

func (cb *CandidateBacking) handleStatement() {
	// TODO: Implement this #3507
}

// SignedFullStatementWithPVD represents a signed full statement along with associated Persisted Validation Data (PVD).
type SignedFullStatementWithPVD struct {
	SignedFullStatement     parachaintypes.UncheckedSignedFullStatement
	PersistedValidationData *parachaintypes.PersistedValidationData
}
