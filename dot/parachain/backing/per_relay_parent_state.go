// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	candidatevalidation "github.com/ChainSafe/gossamer/dot/parachain/candidate-validation"
	prospectiveparachains "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	statementedistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/lib/common"
)

var errNilPersistedValidationData = errors.New("persisted validation data is nil")
var errNoAssignedCore = errors.New("no assigned core")

// constructPerRelayParentState constructs and returns the perRelayParentState for a given relay parent hash,
// initialising various parameters and caches required for candidate backing.
func (cb *CandidateBacking) constructPerRelayParentState(relayParent common.Hash) (*perRelayParentState, error) {
	rt, err := cb.BlockState.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	sessionIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return nil, fmt.Errorf("getting session index: %w", err)
	}

	validators, err := cb.perSessionCache.getValidators(sessionIndex, rt)
	if err != nil {
		return nil, fmt.Errorf("getting validators: %w", err)
	}

	features, err := cb.perSessionCache.getNodeFeatures(sessionIndex, rt)
	if err != nil {
		return nil, fmt.Errorf("getting node features: %w", err)
	}

	if features == nil {
		return nil, fmt.Errorf("invalid node features for relay parent %s", relayParent)
	}

	injectCoreIndex, err := features.Get(uint(parachaintypes.ElasticScalingMVP))
	if err != nil {
		return nil, fmt.Errorf("getting inject core index: %w", err)
	}

	executorParams, err := cb.perSessionCache.getExecutorParams(sessionIndex, rt)
	if err != nil {
		return nil, fmt.Errorf("getting executor params: %w", err)
	}

	validatorGroups, err := rt.ParachainHostValidatorGroups()
	if err != nil {
		return nil, fmt.Errorf("getting validator groups: %w", err)
	}

	minBackingVotes, err := cb.perSessionCache.getMinimumBackingVotes(sessionIndex, rt)
	if err != nil {
		return nil, fmt.Errorf("getting minimum backing votes: %w", err)
	}

	claimQueue, err := rt.ParachainHostClaimQueue()
	if err != nil {
		return nil, fmt.Errorf("getting claim queue: %w", err)
	}

	disabledValidators, err := rt.ParachainHostDisabledValidators()
	if err != nil {
		return nil, fmt.Errorf("getting disabled validators: %w", err)
	}

	signingContext := parachaintypes.SigningContext{
		SessionIndex: sessionIndex,
		ParentHash:   relayParent,
	}

	var localValidator *parachaintypes.Validator

	validatorID, validatorIndex := parachainutil.SigningKeyAndIndex(validators, cb.Keystore)
	if validatorID != nil {
		//  local node is a validator
		localValidator = &parachaintypes.Validator{
			SigningContext: signingContext,
			Key:            *validatorID,
			Index:          validatorIndex,
			Disabled:       slices.Contains(disabledValidators, validatorIndex),
		}
	}

	numOfCores := uint32(len(validatorGroups.Validators))

	groups := make(map[parachaintypes.CoreIndex][]parachaintypes.ValidatorIndex)
	var assignedCore parachaintypes.CoreIndex

	for idx := uint32(0); idx < numOfCores; idx++ {
		coreIndex := parachaintypes.CoreIndex{Index: idx}

		if _, ok := claimQueue[coreIndex]; !ok {
			continue
		}

		groupIndex := validatorGroups.GroupRotationInfo.GroupForCore(coreIndex, uint(numOfCores))
		if uint32(groupIndex) < numOfCores {
			validatorsOfGroup := validatorGroups.Validators[groupIndex]
			if localValidator != nil && slices.Contains(validatorsOfGroup, localValidator.Index) {
				assignedCore = coreIndex
			}
			groups[coreIndex] = validatorsOfGroup
		}
	}

	validatorToGroup := cb.perSessionCache.getValidatorToGroup(sessionIndex, validators, validatorGroups.Validators)

	tableCtx := tableContext{
		validator:          localValidator,
		groups:             groups,
		validators:         validators,
		disabledValidators: disabledValidators,
	}

	newPerRelayParentState := perRelayParentState{
		relayParent:        relayParent,
		nodeFeatures:       *features,
		executorParams:     &executorParams,
		assignedCore:       &assignedCore,
		table:              newTable(),
		tableContext:       tableCtx,
		fallbacks:          make(map[parachaintypes.CandidateHash]attestingData),
		awaitingValidation: make(map[parachaintypes.CandidateHash]bool),
		issuedStatements:   make(map[parachaintypes.CandidateHash]bool),
		backed:             make(map[parachaintypes.CandidateHash]bool),
		minBackingVotes:    minBackingVotes,
		injectCoreIndex:    injectCoreIndex,
		numOfCores:         numOfCores,
		claimQueue:         claimQueue,
		validatorToGroup:   validatorToGroup,
		groupRotationInfo:  validatorGroups.GroupRotationInfo,
	}

	return &newPerRelayParentState, nil
}

// PerRelayParentState represents the state information for a relay-parent in the subsystem.
type perRelayParentState struct {
	// The hash of the relay parent on top of which this job is doing its work.
	relayParent common.Hash
	// The node features.
	nodeFeatures parachaintypes.BitVec
	// The executor parameters.
	executorParams *parachaintypes.ExecutorParams
	// The `CoreIndex` assigned to the local validator at this relay parent.
	assignedCore *parachaintypes.CoreIndex
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
	// If true, we're appending extra bits in the BackedCandidate validator indices bitfield,
	// which represent the assigned core index. True if ElasticScalingMVP is enabled.
	injectCoreIndex bool
	// The number of cores.
	numOfCores uint32
	// Claim queue state. If the runtime API is not available, it'll be populated with info from
	// availability cores.
	claimQueue parachaintypes.ClaimQueue
	// The validator index -> group mapping at this relay parent.
	validatorToGroup map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex
	// The associated group rotation information.
	groupRotationInfo parachaintypes.GroupRotationInfo
}

func (rpState *perRelayParentState) coreIndexFromStatement(
	signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD,
) (*parachaintypes.CoreIndex, error) {
	groupIndex, ok := rpState.validatorToGroup[signedStatementWithPVD.SignedFullStatement.ValidatorIndex]
	if !ok {
		return nil, fmt.Errorf("validator index not found in validator to group mapping")
	}

	coreIndex := rpState.groupRotationInfo.CoreForGroup(groupIndex, uint(rpState.numOfCores))
	if coreIndex.Index > rpState.numOfCores {
		return nil, fmt.Errorf("invalid core index %d, expected core index to be less than %d",
			coreIndex.Index, rpState.numOfCores)
	}

	statementVDTValue, err := signedStatementWithPVD.SignedFullStatement.Payload.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from statementVDT: %w", err)
	}

	secondedStatement, ok := statementVDTValue.(parachaintypes.Seconded)
	if !ok {
		return &coreIndex, nil
	}

	paraID := secondedStatement.Descriptor.ParaID

	// Check if the core is assigned to the paraID of the candidate.
	paraIDs := rpState.claimQueue[coreIndex]
	if !slices.Contains(paraIDs, paraID) {
		return nil, fmt.Errorf("invalid core index %d, core is not assigned to the paraID %d",
			coreIndex.Index, paraID)
	}

	return &coreIndex, nil
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
		return rpState.findCoreIndexAndImportStatement(signedStatementWithPVD)
	}

	return rpState.importSecondedStatement(
		subSystemToOverseer,
		signedStatementWithPVD,
		perCandidate,
		statementVDT.(parachaintypes.Seconded),
	)
}

// findCoreIndexAndImportStatement finds the core index from a statement and imports the statement into the statement
// table.
func (rpState *perRelayParentState) findCoreIndexAndImportStatement(
	signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD,
) (*Summary, error) {
	core, err := rpState.coreIndexFromStatement(signedStatementWithPVD)
	if err != nil {
		return nil, fmt.Errorf("getting core index from statement: %w", err)
	}

	return rpState.table.importStatement(
		&rpState.tableContext,
		parachaintypes.GroupIndex(core.Index),
		signedStatementWithPVD.SignedFullStatement,
	)
}

// importSecondedStatement imports a Seconded statement into the statement table and returns the summary of the import.
func (rpState *perRelayParentState) importSecondedStatement(
	subSystemToOverseer chan<- any,
	signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD,
	perCandidate map[parachaintypes.CandidateHash]*perCandidateState,
	statement parachaintypes.Seconded,
) (*Summary, error) {
	committedCandidateReceipt := parachaintypes.CommittedCandidateReceiptV2(statement)
	candidateHash, err := parachaintypes.GetCandidateHash(committedCandidateReceipt)
	if err != nil {
		return nil, fmt.Errorf("getting candidate hash: %w", err)
	}

	if _, ok := perCandidate[candidateHash]; ok {
		return rpState.findCoreIndexAndImportStatement(signedStatementWithPVD)
	}

	// PersistedValidationData should not be nil if the statementVDT is Seconded.
	if signedStatementWithPVD.PersistedValidationData == nil {
		return nil, errNilPersistedValidationData
	}

	// Introduce candidate to the prospective parachain subsystem.
	if err := rpState.introduceCandidate(
		subSystemToOverseer, committedCandidateReceipt, signedStatementWithPVD.PersistedValidationData); err != nil {
		return nil, err
	}

	perCandidate[candidateHash] = &perCandidateState{
		persistedValidationData: *signedStatementWithPVD.PersistedValidationData,
		secondedLocally:         false,
		relayParent:             committedCandidateReceipt.Descriptor.RelayParent,
	}

	return rpState.findCoreIndexAndImportStatement(signedStatementWithPVD)
}

// introduceCandidate sends a message to the Prospective Parachains subsystem to introduce a candidate.
func (rpState *perRelayParentState) introduceCandidate(
	subSystemToOverseer chan<- any,
	committedCandidateReceipt parachaintypes.CommittedCandidateReceiptV2,
	persistedValidationData *parachaintypes.PersistedValidationData,
) error {
	paraID := committedCandidateReceipt.Descriptor.ParaID

	chIntroduceCandidate := make(chan bool)
	subSystemToOverseer <- prospectiveparachains.IntroduceSecondedCandidate{
		Request: prospectiveparachains.IntroduceSecondedCandidateRequest{
			CandidateParaID:         paraID,
			CandidateReceipt:        committedCandidateReceipt,
			PersistedValidationData: *persistedValidationData,
		},
		Response: chIntroduceCandidate,
	}

	select {
	case isIntroduced, ok := <-chIntroduceCandidate:
		if ok && isIntroduced {
			// candidate introduced successfully
			return nil
		}
		if !ok {
			logger.Warn("response channel of introduce candidate message closed")
		}
	case <-time.After(time.Second * 10): // Add a timeout to avoid potential deadlocks
		logger.Warn("timeout waiting for introduce candidate response")
	}

	return errRejectedByProspectiveParachains
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
	backedCandidate, err := attested.toBackedCandidate(&rpState.tableContext, rpState.injectCoreIndex)
	if err != nil {
		logger.Errorf("converting attested candidate to backed candidate: %w", err)
		return
	}

	paraID := backedCandidate.Candidate.Descriptor.ParaID

	// Inform the prospective parachains subsystem that the candidate is now backed.
	subSystemToOverseer <- prospectiveparachains.CandidateBacked{
		ParaID:        paraID,
		CandidateHash: candidateHash,
	}

	// Notify statement distribution of backed candidate.
	subSystemToOverseer <- statementedistributionmessages.Backed(candidateHash)
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
	localValidator := rpState.tableContext.validator

	if localValidator == nil {
		return fmt.Errorf("we are not a validator - don't kick off validation")
	}

	if localValidator.Disabled {
		return fmt.Errorf("local validator disabled - don't kick off validation")
	}

	candidateHash, err := parachaintypes.GetCandidateHash(attesting.candidate)
	if err != nil {
		return fmt.Errorf("getting candidate hash: %w", err)
	}

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
	candidateReceipt parachaintypes.CandidateReceiptV2,
	pvd parachaintypes.PersistedValidationData,
	pov parachaintypes.PoV,
	numValidator uint32,
	makeCommand validatedCandidateCommand,
	candidateHash parachaintypes.CandidateHash,
) error {
	if rpState.assignedCore == nil {
		return errNoAssignedCore
	}

	if rpState.awaitingValidation[candidateHash] {
		return nil
	}

	rpState.awaitingValidation[candidateHash] = true
	validationCodeHash := candidateReceipt.Descriptor.ValidationCodeHash

	rt, err := blockState.GetRuntime(rpState.relayParent)
	if err != nil {
		return fmt.Errorf("getting runtime for relay parent %s: %w", rpState.relayParent, err)
	}

	validationCode, err := rt.ParachainHostValidationCodeByHash(common.Hash(validationCodeHash))
	if err != nil {
		return fmt.Errorf("getting validation code by hash: %w", err)
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
		ExecutorParams:          *rpState.executorParams,
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
			err:       validationResultRes.Data.Invalid,
		}
	}

	// nil if command is not attestNoPoV
	var candidateHashAccordingToCommand *parachaintypes.CandidateHash
	if makeCommand == attestNoPoV {
		candidateHashAccordingToCommand = &candidateHash
	}

	chRelayParentAndCommand <- relayParentAndCommand{
		relayParent:   rpState.relayParent,
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
