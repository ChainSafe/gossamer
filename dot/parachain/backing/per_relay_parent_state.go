// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"
	"time"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	candidatevalidation "github.com/ChainSafe/gossamer/dot/parachain/candidate-validation"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	statementedistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/lib/common"
)

var errNilPersistedValidationData = errors.New("persisted validation data is nil")

// PerRelayParentState represents the state information for a relay-parent in the subsystem.
type perRelayParentState struct {
	prospectiveParachainsMode parachaintypes.ProspectiveParachainsMode
	// The hash of the relay parent on top of which this job is doing it's work.
	relayParent common.Hash
	// The `ParaId` assigned to the local validator at this relay parent.
	assignment *parachaintypes.ParaID
	// The table of candidates and statements under this relay-parent.
	table Table
	// The table context, including groups.
	tableContext tableContext
	// Data needed for retrying in case of `ValidatedCandidateCommand::AttestNoPoV`.
	fallbacks map[parachaintypes.CandidateHash]attestingData
	// These candidates are undergoing validation in the background.
	awaitingValidation map[parachaintypes.CandidateHash]bool
	// We issued `Seconded` or `Valid` statements on about these candidates.
	issuedStatements map[parachaintypes.CandidateHash]bool
	// The candidates that are backed by enough validators in their group, by hash.
	backed map[parachaintypes.CandidateHash]bool
	// The minimum backing votes threshold.
	minBackingVotes uint32
}

// importStatement imports a statement into the statement table and returns the summary of the import.
func (rpState *perRelayParentState) importStatement(
	subSystemToOverseer chan<- any,
	signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD,
	perCandidate map[parachaintypes.CandidateHash]*perCandidateState,
) (*Summary, error) {
	index, statementVDT, err := signedStatementWithPVD.SignedFullStatement.Payload.IndexValue()
	if err != nil {
		return nil, fmt.Errorf("getting value from statementVDT: %w", err)
	}

	if index != 1 { // Not Seconded
		return rpState.table.importStatement(&rpState.tableContext, signedStatementWithPVD.SignedFullStatement)
	}

	committedCandidateReceipt := parachaintypes.CommittedCandidateReceipt(statementVDT.(parachaintypes.Seconded))
	candidateHash, err := parachaintypes.GetCandidateHash(committedCandidateReceipt)
	if err != nil {
		return nil, fmt.Errorf("getting candidate hash: %w", err)
	}

	if _, ok := perCandidate[candidateHash]; ok {
		return rpState.table.importStatement(&rpState.tableContext, signedStatementWithPVD.SignedFullStatement)
	}

	// PersistedValidationData should not be nil if the statementVDT is Seconded.
	if signedStatementWithPVD.PersistedValidationData == nil {
		return nil, errNilPersistedValidationData
	}

	paraID := committedCandidateReceipt.Descriptor.ParaID

	if rpState.prospectiveParachainsMode.IsEnabled {
		chIntroduceCandidate := make(chan error)
		subSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageIntroduceCandidate{
			IntroduceCandidateRequest: parachaintypes.IntroduceCandidateRequest{
				CandidateParaID:           paraID,
				CommittedCandidateReceipt: committedCandidateReceipt,
				PersistedValidationData:   *signedStatementWithPVD.PersistedValidationData,
			},
			Ch: chIntroduceCandidate,
		}

		introduceCandidateErr, ok := <-chIntroduceCandidate
		if !ok {
			return nil, fmt.Errorf("%w: %s",
				errRejectedByProspectiveParachains,
				"Could not reach the Prospective Parachains subsystem.",
			)
		}
		if introduceCandidateErr != nil {
			return nil, fmt.Errorf("%w: %w", errRejectedByProspectiveParachains, introduceCandidateErr)
		}

		subSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageCandidateSeconded{
			ParaID:        paraID,
			CandidateHash: candidateHash,
		}
	}

	// Only save the candidate if it was approved by prospective parachains.
	perCandidate[candidateHash] = &perCandidateState{
		persistedValidationData: *signedStatementWithPVD.PersistedValidationData,
		secondedLocally:         false, // This is set after importing when seconding locally.
		paraID:                  paraID,
		relayParent:             committedCandidateReceipt.Descriptor.RelayParent,
	}

	return rpState.table.importStatement(&rpState.tableContext, signedStatementWithPVD.SignedFullStatement)
}

// postImportStatement handles a summary received from importStatement func and dispatches `Backed` notifications and
// misbehaviors as a result of importing a statement.
func (rpState *perRelayParentState) postImportStatement(subSystemToOverseer chan<- any, summary *Summary) {
	defer issueNewMisbehaviors(subSystemToOverseer, rpState.relayParent, rpState.table)

	// Return, If the summary is nil.
	if summary == nil {
		return
	}

	attested, err := rpState.table.attestedCandidate(summary.Candidate, &rpState.tableContext, rpState.minBackingVotes)
	if err != nil {
		logger.Error(err.Error())
	}

	// Return, If the candidate is not attested.
	if attested == nil {
		return
	}

	candidateHash, err := parachaintypes.GetCandidateHash(attested.committedCandidateReceipt)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// Return, If the candidate is already backed.
	if rpState.backed[candidateHash] {
		return
	}

	// Mark the candidate as backed.
	rpState.backed[candidateHash] = true

	// Convert the attested candidate to a backed candidate.
	backedCandidate, err := attested.toBackedCandidate(&rpState.tableContext)
	if err != nil {
		logger.Errorf("converting attested candidate to backed candidate: %w", err)
		return
	}

	paraID := backedCandidate.Candidate.Descriptor.ParaID

	if rpState.prospectiveParachainsMode.IsEnabled {

		// Inform the prospective parachains subsystem that the candidate is now backed.
		subSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageCandidateBacked{
			ParaID:        paraID,
			CandidateHash: candidateHash,
		}

		// Backed candidate potentially unblocks new advertisements, notify collator protocol.
		subSystemToOverseer <- collatorprotocolmessages.Backed{
			ParaID:   paraID,
			ParaHead: backedCandidate.Candidate.Descriptor.ParaHead,
		}

		// Notify statement distribution of backed candidate.
		subSystemToOverseer <- statementedistributionmessages.Backed(candidateHash)

	} else {
		// TODO: figure out what this comment means by 'avoid cycles'.
		//
		// The provisioner waits on candidate-backing, which means
		// that we need to send unbounded messages to avoid cycles.
		//
		// Backed candidates are bounded by the number of validators,
		// parachains, and the block production rate of the relay chain.
		subSystemToOverseer <- provisionermessages.ProvisionableData{
			RelayParent: rpState.relayParent,
			Data:        provisionermessages.ProvisionableDataBackedCandidate(backedCandidate.Candidate.ToPlain()),
		}
	}
}

// issueNewMisbehaviors checks for new misbehaviors and sends necessary messages to the Overseer subsystem.
func issueNewMisbehaviors(subSystemToOverseer chan<- any, relayParent common.Hash, table Table) {
	// collect the validatorsToMisbehaviors to avoid double mutable self borrow issues
	validatorsToMisbehaviors := table.drainMisbehaviors()

	for validatorIndex, misbehaviours := range validatorsToMisbehaviors {
		// TODO: figure out what this comment means by 'avoid cycles'.
		//
		// The provisioner waits on candidate-backing, which means
		// that we need to send unbounded messages to avoid cycles.
		//
		// Misbehaviors are bounded by the number of validators and
		// the block production protocol.
		for _, misbehaviour := range misbehaviours {
			subSystemToOverseer <- provisionermessages.ProvisionableData{
				RelayParent: relayParent,
				Data: provisionermessages.ProvisionableDataMisbehaviorReport{
					ValidatorIndex: validatorIndex,
					Misbehaviour:   misbehaviour,
				},
			}
		}

	}
}

// Kick off validation work and distribute the result as a signed statement.
func (rpState *perRelayParentState) kickOffValidationWork(
	blockState BlockState,
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan relayParentAndCommand,
	pvd parachaintypes.PersistedValidationData,
	attesting attestingData,
) error {
	hash, err := attesting.candidate.Hash()
	if err != nil {
		return fmt.Errorf("getting candidate hash: %w", err)
	}

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	if rpState.issuedStatements[candidateHash] {
		return nil
	}

	pov, err := getPovFromValidator(subSystemToOverseer, chRelayParentAndCommand,
		rpState.relayParent, candidateHash, &attesting)
	if err != nil {
		if errors.Is(err, parachaintypes.ErrFetchPoV) {
			return nil
		}
		return err
	}

	return rpState.validateAndMakeAvailable(
		blockState,
		subSystemToOverseer,
		chRelayParentAndCommand,
		attesting.candidate,
		rpState.relayParent,
		pvd,
		pov,
		uint32(len(rpState.tableContext.validators)),
		attest,
		candidateHash,
	)
}

func (rpState *perRelayParentState) validateAndMakeAvailable(
	blockState BlockState,
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan relayParentAndCommand,
	candidateReceipt parachaintypes.CandidateReceipt,
	relayParent common.Hash,
	pvd parachaintypes.PersistedValidationData,
	pov parachaintypes.PoV,
	numValidator uint32,
	makeCommand validatedCandidateCommand,
	candidateHash parachaintypes.CandidateHash,
) error {
	if rpState.awaitingValidation[candidateHash] {
		return nil
	}

	rpState.awaitingValidation[candidateHash] = true
	validationCodeHash := candidateReceipt.Descriptor.ValidationCodeHash

	rt, err := blockState.GetRuntime(relayParent)
	if err != nil {
		return fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	validationCode, err := rt.ParachainHostValidationCodeByHash(common.Hash(validationCodeHash))
	if err != nil {
		return fmt.Errorf("getting validation code by hash: %w", err)
	}

	executorParams, err := util.ExecutorParamsAtRelayParent(rt, relayParent)
	if err != nil {
		return fmt.Errorf("getting executor params for relay parent %s: %w", relayParent, err)
	}

	pvfExecTimeoutKind := parachaintypes.NewPvfExecTimeoutKind()
	err = pvfExecTimeoutKind.SetValue(parachaintypes.Backing{})
	if err != nil {
		return fmt.Errorf("setting pvfExecTimeoutKind: %w", err)
	}

	chValidationResultRes := make(chan parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult])
	subSystemToOverseer <- candidatevalidation.ValidateFromExhaustive{
		PersistedValidationData: pvd,
		ValidationCode:          *validationCode,
		CandidateReceipt:        candidateReceipt,
		PoV:                     pov,
		ExecutorParams:          *executorParams,
		PvfExecTimeoutKind:      pvfExecTimeoutKind,
		Ch:                      chValidationResultRes,
	}

	validationResultRes := <-chValidationResultRes
	if validationResultRes.Err != nil {
		return fmt.Errorf("getting validation result: %w", validationResultRes.Err)
	}

	var bgValidationResult backgroundValidationResult
	if validationResultRes.Data.IsValid() { // Valid
		// Important: the `av-store` subsystem will check if the erasure root of the `available_data`
		// matches `expected_erasure_root` which was provided by the collator in the `CandidateReceipt`.
		// This check is consensus critical and the `backing` subsystem relies on it for ensuring
		// candidate validity.

		logger.Debugf("validation successful! candidateHash=%s", candidateHash)

		chStoreAvailableDataError := make(chan error)
		subSystemToOverseer <- availabilitystore.StoreAvailableData{
			CandidateHash: candidateHash,
			NumValidators: numValidator,
			AvailableData: availabilitystore.AvailableData{
				PoV:            pov,
				ValidationData: pvd,
			},
			ExpectedErasureRoot: candidateReceipt.Descriptor.ErasureRoot,
			Sender:              chStoreAvailableDataError,
		}

		storeAvailableDataError := <-chStoreAvailableDataError
		switch {
		case storeAvailableDataError == nil:
			bgValidationResult = backgroundValidationResult{
				outputs: &backgroundValidationOutputs{
					candidateReceipt:        candidateReceipt,
					candidateCommitments:    validationResultRes.Data.Valid.CandidateCommitments,
					persistedValidationData: validationResultRes.Data.Valid.PersistedValidationData,
				},
				candidate: nil,
				err:       nil,
			}
		case errors.Is(storeAvailableDataError, errInvalidErasureRoot):
			logger.Debug(errInvalidErasureRoot.Error())
			bgValidationResult = backgroundValidationResult{
				outputs:   nil,
				candidate: &candidateReceipt,
				err:       errInvalidErasureRoot,
			}
		default:
			return fmt.Errorf("storing available data: %w", storeAvailableDataError)
		}

	} else { // Invalid
		logger.Error(validationResultRes.Data.Invalid.Error())
		bgValidationResult = backgroundValidationResult{
			outputs:   nil,
			candidate: &candidateReceipt,
			err:       fmt.Errorf(validationResultRes.Data.Invalid.Error()),
		}
	}

	// nil if command is not attestNoPoV
	var candidateHashAccordingToCommand *parachaintypes.CandidateHash
	if makeCommand == attestNoPoV {
		candidateHashAccordingToCommand = &candidateHash
	}

	chRelayParentAndCommand <- relayParentAndCommand{
		relayParent:   relayParent,
		command:       makeCommand,
		validationRes: &bgValidationResult,
		candidateHash: candidateHashAccordingToCommand,
	}
	return nil
}

func getPovFromValidator(
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan relayParentAndCommand,
	relayParent common.Hash,
	candidateHash parachaintypes.CandidateHash,
	attesting *attestingData,
) (parachaintypes.PoV, error) {
	var PovRes parachaintypes.OverseerFuncRes[parachaintypes.PoV]

	fetchPov := parachaintypes.AvailabilityDistributionMessageFetchPoV{
		RelayParent:   relayParent,
		FromValidator: attesting.fromValidator,
		ParaID:        attesting.candidate.Descriptor.ParaID,
		CandidateHash: candidateHash,
		PovHash:       attesting.povHash,
		PovCh:         make(chan parachaintypes.OverseerFuncRes[parachaintypes.PoV]),
	}

	subSystemToOverseer <- fetchPov
	select {
	case PovRes = <-fetchPov.PovCh:
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		return parachaintypes.PoV{}, parachaintypes.ErrSubsystemRequestTimeout
	}

	if PovRes.Err != nil {
		if errors.Is(PovRes.Err, parachaintypes.ErrFetchPoV) {
			chRelayParentAndCommand <- relayParentAndCommand{
				relayParent:   relayParent,
				command:       attestNoPoV,
				candidateHash: &candidateHash,
			}
		}
		return parachaintypes.PoV{}, PovRes.Err
	}
	return PovRes.Data, nil
}
