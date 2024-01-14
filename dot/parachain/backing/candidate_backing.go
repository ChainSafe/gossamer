// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"context"
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tidwall/btree"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

var (
	ErrRejectedByProspectiveParachains = errors.New("candidate rejected by prospective parachains subsystem")
	ErrInvalidErasureRoot              = errors.New("erasure root doesn't match the announced by the candidate receipt")
	ErrStatementForUnknownRelayParent  = errors.New("received statement for unknown relay parent")
	ErrCandidateStateNotFound          = errors.New("candidate state not found")
	ErrAttestingDataNotFound           = errors.New("attesting data not found")
)

// CandidateBacking represents the state of the subsystem responsible for managing candidate backing.
type CandidateBacking struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	// State tracked for all relay-parents backing work is ongoing for. This includes
	// all active leaves.
	//
	// relay-parents fall into one of 3 categories.
	//   1. active leaves which do support prospective parachains
	//   2. active leaves which do not support prospective parachains
	//   3. relay-chain blocks which are ancestors of an active leaf and do support prospective
	//      parachains.
	//
	// Relay-chain blocks which don't support prospective parachains are
	// never included in the fragment trees of active leaves which do.
	//
	// While it would be technically possible to support such leaves in
	// fragment trees, it only benefits the transition period when asynchronous
	// backing is being enabled and complicates code complexity.
	perRelayParent map[common.Hash]*perRelayParentState
	// State tracked for all candidates relevant to the implicit view.
	//
	// This is guaranteed to have an entry for each candidate with a relay parent in the implicit
	// or explicit view for which a `Seconded` statement has been successfully imported.
	perCandidate map[parachaintypes.CandidateHash]*perCandidateState
	// State tracked for all active leaves, whether or not they have prospective parachains enabled.
	perLeaf map[common.Hash]ActiveLeafState
	// The utility for managing the implicit and explicit views in a consistent way.
	// We only feed leaves which have prospective parachains enabled to this view.
	implicitView ImplicitView
}

type ActiveLeafState struct {
	ProspectiveParachainsMode parachaintypes.ProspectiveParachainsMode
	SecondedAtDepth           map[parachaintypes.ParaID]btree.Map[uint, parachaintypes.CandidateHash]
	perCandidate              map[parachaintypes.CandidateHash]*perCandidateState //nolint:unused
}

// perCandidateState represents the state information for a candidate in the subsystem.
type perCandidateState struct {
	persistedValidationData parachaintypes.PersistedValidationData
	SecondedLocally         bool
	ParaID                  parachaintypes.ParaID
	RelayParent             common.Hash
}

// PerRelayParentState represents the state information for a relay-parent in the subsystem.
type perRelayParentState struct {
	ProspectiveParachainsMode parachaintypes.ProspectiveParachainsMode
	// The hash of the relay parent on top of which this job is doing it's work.
	RelayParent common.Hash
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
	// We issued `Seconded` or `Valid` statements on about these candidates.
	issuedStatements map[parachaintypes.CandidateHash]bool
	// The candidates that are backed by enough validators in their group, by hash.
	backed map[parachaintypes.CandidateHash]bool
}

// AttestingData contains the data needed to retry validation with other backing validators
// in case a validator does not provide a PoV.
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

// TableContext represents the contextual information associated with a validator and groups
// for a table under a relay-parent.
type TableContext struct {
	validator  *Validator
	groups     map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex
	validators []parachaintypes.ValidatorID
}

// Validator represents local validator information.
// It can be created if the local node is a validator in the context of a particular relay chain block.
type Validator struct {
	index parachaintypes.ValidatorIndex
}

// SigningContext represents a type returned by the runtime, including the current session index
// and the hash of the parent.
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

// GetBackedCandidatesMessage is a message received from overseer that requests a set of backable
// candidates that could be backed in a child of the given relay-parent.
type GetBackedCandidatesMessage []struct {
	CandidateHash        parachaintypes.CandidateHash
	CandidateRelayParent common.Hash
}

// CanSecondMessage is a request made to the candidate backing subsystem to determine whether it is permissible
// to second a given candidate.
// The rule for seconding candidates is: Collations must either be built on top of the root of a fragment tree
// or have a parent node that represents the backed candidate.
type CanSecondMessage struct {
	CandidateParaID      parachaintypes.ParaID
	CandidateRelayParent common.Hash
	CandidateHash        parachaintypes.CandidateHash
	ParentHeadDataHash   common.Hash
	resCh                chan bool
}

// SecondMessage is a message received from overseer. Candidate Backing subsystem should second the given
// candidate in the context of the given relay parent. This candidate must be validated.
type SecondMessage struct {
	RelayParent             common.Hash
	CandidateReceipt        parachaintypes.CandidateReceipt
	PersistedValidationData parachaintypes.PersistedValidationData
	PoV                     parachaintypes.PoV
}

// StatementMessage represents a validator's assessment of a specific candidate. If there are disagreements
// regarding the validity of this assessment, they should be addressed through the Disputes Subsystem,
// with the actual escalation deferred until the approval voting stage to ensure its availability.
// Meanwhile, agreements are straightforwardly counted until a quorum is achieved.
type StatementMessage struct {
	RelayParent         common.Hash
	SignedFullStatement SignedFullStatementWithPVD
}

// SignedFullStatementWithPVD represents a signed full statement along with associated Persisted Validation Data (PVD).
type SignedFullStatementWithPVD struct {
	SignedFullStatement     parachaintypes.UncheckedSignedFullStatement
	PersistedValidationData *parachaintypes.PersistedValidationData
}

// New creates a new CandidateBacking instance and initialises it with the provided overseer channel.
func New(overseerChan chan<- any) *CandidateBacking {
	return &CandidateBacking{
		SubSystemToOverseer: overseerChan,
		perRelayParent:      map[common.Hash]*perRelayParentState{},
		perCandidate:        map[parachaintypes.CandidateHash]*perCandidateState{},
	}
}

func (cb *CandidateBacking) Run(ctx context.Context, overseerToSubSystem chan any, subSystemToOverseer chan any) {
	chRelayParentAndCommand := make(chan RelayParentAndCommand)
	for {
		select {
		case rpAndCmd := <-chRelayParentAndCommand:
			if err := cb.processValidatedCandidateCommand(rpAndCmd); err != nil {
				logger.Error(err.Error())
			}
		case msg := <-cb.OverseerToSubSystem:
			if err := cb.processOverseerMessage(msg, chRelayParentAndCommand); err != nil {
				logger.Error(err.Error())
			}
		case <-ctx.Done():
			close(cb.SubSystemToOverseer)
			close(chRelayParentAndCommand)
			return
		}
	}
}

func (*CandidateBacking) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateBacking
}

// processOverseerMessage processes incoming messages from overseer
func (cb *CandidateBacking) processOverseerMessage(msg any, chRelayParentAndCommand chan RelayParentAndCommand) error {
	// process these received messages by referencing
	// https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/core/backing/src/lib.rs#L741
	switch msg := msg.(type) {
	case ActiveLeavesUpdate:
		cb.handleActiveLeavesUpdate()
	case GetBackedCandidatesMessage:
		cb.handleGetBackedCandidatesMessage()
	case CanSecondMessage:
		cb.handleCanSecondMessage(msg)
	case SecondMessage:
		err := cb.handleSecondMessage(msg.CandidateReceipt, msg.PersistedValidationData, msg.PoV, chRelayParentAndCommand)
		switch err {
		case errWrongPVDForSecondingCandidate, errUnknownRelayParentForSecondingCandidate,
			errParaOutsideAssignmentForSeconding, errAlreadySignedValidStatement:
			logger.Error(err.Error())
			return nil
		default:
			return err
		}

	case StatementMessage:
		err := cb.handleStatementMessage(msg.RelayParent, msg.SignedFullStatement, chRelayParentAndCommand)

		if errors.Is(err, ErrRejectedByProspectiveParachains) || errors.Is(err, ErrAttestingDataNotFound) {
			logger.Error(err.Error())
			return nil
		}
		return err
	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
	return nil
}

func (cb *CandidateBacking) handleActiveLeavesUpdate() {
	// TODO: Implement this #3503
}

func (cb *CandidateBacking) handleGetBackedCandidatesMessage() {
	// TODO: Implement this #3504
}

// Import the statement and kick off validation work if it is a part of our assignment.
func (cb *CandidateBacking) handleStatementMessage(
	relayParent common.Hash,
	signedStatementWithPVD SignedFullStatementWithPVD,
	chRelayParentAndCommand chan RelayParentAndCommand,
) error {
	rpState, ok := cb.perRelayParent[relayParent]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStatementForUnknownRelayParent, relayParent)
	}

	summary, err := rpState.importStatement(cb.SubSystemToOverseer, signedStatementWithPVD, cb.perCandidate)
	if err != nil {
		return fmt.Errorf("importing statement: %w", err)
	}

	rpState.postImportStatement(cb.SubSystemToOverseer, summary)

	if summary == nil {
		logger.Debug("summary is nil")
		return nil
	}

	if summary.GroupID != rpState.Assignment {
		logger.Debugf("The ParaId: %d is not assigned to the local validator at relay parent: %s",
			summary.GroupID, relayParent)
		return nil
	}

	// already ensured in importStatement that the value of the statementVDT has been set.
	// that is why there is no chance we can get an error here.
	statementVDT, _ := signedStatementWithPVD.SignedFullStatement.Payload.Value()

	var attesting AttestingData
	switch statementVDT := statementVDT.(type) {
	case parachaintypes.Seconded:
		commitedCandidateReceipt, err := rpState.Table.getCandidate(summary.Candidate)
		if err != nil {
			return fmt.Errorf("getting candidate: %w", err)
		}

		attesting = AttestingData{
			candidate:     commitedCandidateReceipt.ToPlain(),
			povHash:       statementVDT.Descriptor.PovHash,
			fromValidator: signedStatementWithPVD.SignedFullStatement.ValidatorIndex,
			backing:       []parachaintypes.ValidatorIndex{},
		}
	case parachaintypes.Valid:
		candidateHash := parachaintypes.CandidateHash(statementVDT)
		attesting, ok = rpState.fallbacks[candidateHash]
		if !ok {
			return ErrAttestingDataNotFound
		}

		ourIndex := rpState.TableContext.validator.index
		if signedStatementWithPVD.SignedFullStatement.ValidatorIndex == ourIndex {
			return nil
		}

		if rpState.AwaitingValidation[candidateHash] {
			logger.Debug("Job already running")
			attesting.backing = append(attesting.backing, signedStatementWithPVD.SignedFullStatement.ValidatorIndex)
			return nil
		}

		logger.Debug("No job, so start another with current validator")
		attesting.fromValidator = signedStatementWithPVD.SignedFullStatement.ValidatorIndex
	}

	rpState.fallbacks[summary.Candidate] = attesting

	// After `import_statement` succeeds, the candidate entry is guaranteed to exist.
	pc, ok := cb.perCandidate[summary.Candidate]
	if !ok {
		return ErrCandidateStateNotFound
	}

	return rpState.kickOffValidationWork(
		cb.SubSystemToOverseer,
		chRelayParentAndCommand,
		pc.persistedValidationData,
		attesting,
	)
}

// importStatement imports a statement into the statement table and returns the summary of the import.
func (rpState *perRelayParentState) importStatement(
	subSystemToOverseer chan<- any,
	signedStatementWithPVD SignedFullStatementWithPVD,
	perCandidate map[parachaintypes.CandidateHash]*perCandidateState,
) (*Summary, error) {
	statementVDT, err := signedStatementWithPVD.SignedFullStatement.Payload.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from statementVDT: %w", err)
	}

	if statementVDT.Index() == 2 { // Valid
		return rpState.Table.importStatement(&rpState.TableContext, signedStatementWithPVD)
	}

	// PersistedValidationData should not be nil if the statementVDT is Seconded.
	if signedStatementWithPVD.PersistedValidationData == nil {
		return nil, fmt.Errorf("persisted validation data is nil")
	}

	statementVDTSeconded := statementVDT.(parachaintypes.Seconded)
	hash, err := parachaintypes.CommittedCandidateReceipt(statementVDTSeconded).Hash()
	if err != nil {
		return nil, fmt.Errorf("getting candidate hash: %w", err)
	}

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	if _, ok := perCandidate[candidateHash]; ok {
		return rpState.Table.importStatement(&rpState.TableContext, signedStatementWithPVD)
	}

	if rpState.ProspectiveParachainsMode.IsEnabled {
		chIntroduceCandidate := make(chan error)
		subSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageIntroduceCandidate{
			IntroduceCandidateRequest: parachaintypes.IntroduceCandidateRequest{
				CandidateParaID:           parachaintypes.ParaID(statementVDTSeconded.Descriptor.ParaID),
				CommittedCandidateReceipt: parachaintypes.CommittedCandidateReceipt(statementVDTSeconded),
				PersistedValidationData:   *signedStatementWithPVD.PersistedValidationData,
			},
			Ch: chIntroduceCandidate,
		}

		introduceCandidateErr, ok := <-chIntroduceCandidate
		if !ok {
			return nil, fmt.Errorf("%w: %s",
				ErrRejectedByProspectiveParachains,
				"Could not reach the Prospective Parachains subsystem.",
			)
		}
		if introduceCandidateErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrRejectedByProspectiveParachains, introduceCandidateErr)
		}

		subSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageCandidateSeconded{
			ParaID:        parachaintypes.ParaID(statementVDTSeconded.Descriptor.ParaID),
			CandidateHash: candidateHash,
		}
	}

	// Only save the candidate if it was approved by prospective parachains.
	perCandidate[candidateHash] = &perCandidateState{
		persistedValidationData: *signedStatementWithPVD.PersistedValidationData,
		SecondedLocally:         false, // This is set after importing when seconding locally.
		ParaID:                  parachaintypes.ParaID(statementVDTSeconded.Descriptor.ParaID),
		RelayParent:             statementVDTSeconded.Descriptor.RelayParent,
	}

	return rpState.Table.importStatement(&rpState.TableContext, signedStatementWithPVD)
}

// postImportStatement handles a summary received from importStatement func and dispatches `Backed` notifications and
// misbehaviors as a result of importing a statement.
func (rpState *perRelayParentState) postImportStatement(subSystemToOverseer chan<- any, summary *Summary) {
	// If the summary is nil, issue new misbehaviors and return.
	if summary == nil {
		issueNewMisbehaviors(subSystemToOverseer, rpState.RelayParent, rpState.Table)
		return
	}

	attested, err := rpState.Table.attestedCandidate(&summary.Candidate, &rpState.TableContext)
	if err != nil {
		logger.Error(err.Error())
	}

	// If the candidate is not attested, issue new misbehaviors and return.
	if attested == nil {
		issueNewMisbehaviors(subSystemToOverseer, rpState.RelayParent, rpState.Table)
		return
	}

	hash, err := attested.Candidate.Hash()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	// If the candidate is already backed, issue new misbehaviors and return.
	if rpState.backed[candidateHash] {
		issueNewMisbehaviors(subSystemToOverseer, rpState.RelayParent, rpState.Table)
		return
	}

	// Mark the candidate as backed.
	rpState.backed[candidateHash] = true

	// Convert the attested candidate to a backed candidate.
	backedCandidate := attestedToBackedCandidate(*attested, &rpState.TableContext)
	if backedCandidate == nil {
		issueNewMisbehaviors(subSystemToOverseer, rpState.RelayParent, rpState.Table)
		return
	}

	paraID := backedCandidate.Candidate.Descriptor.ParaID

	if rpState.ProspectiveParachainsMode.IsEnabled {

		// Inform the prospective parachains subsystem that the candidate is now backed.
		subSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageCandidateBacked{
			ParaID:        parachaintypes.ParaID(paraID),
			CandidateHash: candidateHash,
		}

		// Backed candidate potentially unblocks new advertisements, notify collator protocol.
		subSystemToOverseer <- parachaintypes.CollatorProtocolMessageBacked{
			ParaID:   parachaintypes.ParaID(paraID),
			ParaHead: backedCandidate.Candidate.Descriptor.ParaHead,
		}

		// Notify statement distribution of backed candidate.
		subSystemToOverseer <- parachaintypes.StatementDistributionMessageBacked(candidateHash)

	} else {
		// TODO: figure out what this comment means by 'avoid cycles'.
		//
		// The provisioner waits on candidate-backing, which means
		// that we need to send unbounded messages to avoid cycles.
		//
		// Backed candidates are bounded by the number of validators,
		// parachains, and the block production rate of the relay chain.
		subSystemToOverseer <- parachaintypes.ProvisionerMessageProvisionableData{
			RelayParent:       rpState.RelayParent,
			ProvisionableData: parachaintypes.ProvisionableDataBackedCandidate(backedCandidate.Candidate.ToPlain()),
		}
	}

	issueNewMisbehaviors(subSystemToOverseer, rpState.RelayParent, rpState.Table)
}

// issueNewMisbehaviors checks for new misbehaviors and sends necessary messages to the Overseer subsystem.
func issueNewMisbehaviors(subSystemToOverseer chan<- any, relayParent common.Hash, table Table) {
	// collect the misbehaviors to avoid double mutable self borrow issues
	misbehaviors := table.drainMisbehaviors()

	for _, m := range misbehaviors {
		// TODO: figure out what this comment means by 'avoid cycles'.
		//
		// The provisioner waits on candidate-backing, which means
		// that we need to send unbounded messages to avoid cycles.
		//
		// Misbehaviors are bounded by the number of validators and
		// the block production protocol.
		subSystemToOverseer <- parachaintypes.ProvisionerMessageProvisionableData{
			RelayParent: relayParent,
			ProvisionableData: parachaintypes.ProvisionableDataMisbehaviorReport{
				ValidatorIndex: m.ValidatorIndex,
				Misbehaviour:   m.Misbehaviour,
			},
		}
	}
}

func attestedToBackedCandidate(
	attested AttestedCandidate,
	tableContext *TableContext,
) *parachaintypes.BackedCandidate {
	group := tableContext.groups[attested.GroupID]
	validatorIndices := make([]bool, len(group))
	var validityAttestations []parachaintypes.ValidityAttestation

	// The order of the validity votes in the backed candidate must match
	// the order of bits set in the bitfield, which is not necessarily
	// the order of the `validity_votes` we got from the table.
	for positionInGroup, validatorIndex := range group {
		for _, validityVote := range attested.ValidityVotes {
			if validityVote.ValidatorIndex == validatorIndex {
				validatorIndices[positionInGroup] = true
				validityAttestations = append(validityAttestations, validityVote.ValidityAttestation)
			}
		}

		if !validatorIndices[positionInGroup] {
			logger.Error("validity vote from unknown validator")
			return nil
		}
	}

	return &parachaintypes.BackedCandidate{
		Candidate:        attested.Candidate,
		ValidityVotes:    validityAttestations,
		ValidatorIndices: scale.NewBitVec(validatorIndices),
	}
}

// Kick off validation work and distribute the result as a signed statement.
func (rpState *perRelayParentState) kickOffValidationWork(
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan RelayParentAndCommand,
	pvd parachaintypes.PersistedValidationData,
	attesting AttestingData,
) error {
	hash, err := attesting.candidate.Hash()
	if err != nil {
		return fmt.Errorf("getting candidate hash: %w", err)
	}

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	if rpState.issuedStatements[candidateHash] {
		return nil
	}

	pov := getPovFromValidator()

	return rpState.validateAndMakeAvailable(
		executorParamsAtRelayParent,
		subSystemToOverseer,
		chRelayParentAndCommand,
		attesting.candidate,
		rpState.RelayParent,
		pvd,
		pov,
		uint32(len(rpState.TableContext.validators)),
		Attest,
		candidateHash,
	)
}

// this is temporary until we implement executorParamsAtRelayParent #3544
type ExecutorParamsGetter func(common.Hash, chan<- any) (parachaintypes.ExecutorParams, error)

func (rpState *perRelayParentState) validateAndMakeAvailable(
	executorParamsAtRelayParentFunc ExecutorParamsGetter, // remove after executorParamsAtRelayParent is implemented #3544
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan RelayParentAndCommand,
	candidateReceipt parachaintypes.CandidateReceipt,
	relayParent common.Hash,
	pvd parachaintypes.PersistedValidationData,
	pov parachaintypes.PoV,
	numValidator uint32,
	makeCommand ValidatedCandidateCommand,
	candidateHash parachaintypes.CandidateHash,
) error {
	if rpState.AwaitingValidation[candidateHash] {
		return nil
	}

	rpState.AwaitingValidation[candidateHash] = true
	validationCodeHash := candidateReceipt.Descriptor.ValidationCodeHash

	chValidationCodeByHashRes := make(chan parachaintypes.OverseerFuncRes[parachaintypes.ValidationCode])
	subSystemToOverseer <- parachaintypes.RuntimeApiMessageRequest{
		RelayParent: relayParent,
		RuntimeApiRequest: parachaintypes.RuntimeApiRequestValidationCodeByHash{
			ValidationCodeHash: validationCodeHash,
			Ch:                 chValidationCodeByHashRes,
		},
	}

	validationCodeByHashRes := <-chValidationCodeByHashRes
	if validationCodeByHashRes.Err != nil {
		return fmt.Errorf("getting validation code by hash: %w", validationCodeByHashRes.Err)
	}

	// executorParamsAtRelayParent() should be called after it is implemented #3544
	executorParams, err := executorParamsAtRelayParentFunc(relayParent, subSystemToOverseer)
	if err != nil {
		return fmt.Errorf("getting executor params at relay parent: %w", err)
	}

	pvfExecTimeoutKind := parachaintypes.NewPvfExecTimeoutKind()
	err = pvfExecTimeoutKind.Set(parachaintypes.Backing{})
	if err != nil {
		return fmt.Errorf("setting pvfExecTimeoutKind: %w", err)
	}

	chValidationResultRes := make(chan parachaintypes.OverseerFuncRes[parachaintypes.ValidationResult])
	subSystemToOverseer <- parachaintypes.CandidateValidationMessageValidateFromExhaustive{
		PersistedValidationData: pvd,
		ValidationCode:          validationCodeByHashRes.Data,
		CandidateReceipt:        candidateReceipt,
		PoV:                     pov,
		ExecutorParams:          executorParams,
		PvfExecTimeoutKind:      pvfExecTimeoutKind,
		Ch:                      chValidationResultRes,
	}

	ValidationResultRes := <-chValidationResultRes
	if ValidationResultRes.Err != nil {
		return fmt.Errorf("getting validation result: %w", ValidationResultRes.Err)
	}

	var backgroundValidationResult BackgroundValidationResult

	if ValidationResultRes.Data.IsValid { // Valid
		// Important: the `av-store` subsystem will check if the erasure root of the `available_data`
		// matches `expected_erasure_root` which was provided by the collator in the `CandidateReceipt`.
		// This check is consensus critical and the `backing` subsystem relies on it for ensuring
		// candidate validity.

		logger.Debugf("validation successful! candidateHash=%s", candidateHash)

		chStoreAvailableDataError := make(chan error)
		subSystemToOverseer <- parachaintypes.AvailabilityStoreMessageStoreAvailableData{
			CandidateHash: candidateHash,
			NumValidators: numValidator,
			AvailableData: parachaintypes.AvailableData{
				PoV:            pov,
				ValidationData: pvd,
			},
			ExpectedErasureRoot: candidateReceipt.Descriptor.ErasureRoot,
			Ch:                  chStoreAvailableDataError,
		}

		storeAvailableDataError := <-chStoreAvailableDataError
		switch {
		case storeAvailableDataError == nil:
			backgroundValidationResult = BackgroundValidationResult{
				CandidateReceipt:        &candidateReceipt,
				CandidateCommitments:    &ValidationResultRes.Data.CandidateCommitments,
				PersistedValidationData: &ValidationResultRes.Data.PersistedValidationData,
				Err:                     nil,
			}
		case errors.Is(storeAvailableDataError, ErrInvalidErasureRoot):
			logger.Debug(ErrInvalidErasureRoot.Error())

			backgroundValidationResult = BackgroundValidationResult{
				CandidateReceipt: &candidateReceipt,
				Err:              ErrInvalidErasureRoot,
			}
		default:
			return fmt.Errorf("storing available data: %w", storeAvailableDataError)
		}

	} else { // Invalid
		logger.Error(ValidationResultRes.Data.Err.Error())
		backgroundValidationResult = BackgroundValidationResult{
			CandidateReceipt: &candidateReceipt,
			Err:              ValidationResultRes.Data.Err,
		}
	}

	chRelayParentAndCommand <- RelayParentAndCommand{
		RelayParent:   relayParent,
		Command:       makeCommand,
		ValidationRes: backgroundValidationResult,
		CandidateHash: candidateHash,
	}
	return nil
}

func getPovFromValidator() parachaintypes.PoV {
	// TODO: Implement this #3545
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/core/backing/src/lib.rs#L1744 //nolint:lll
	return parachaintypes.PoV{}
}

func executorParamsAtRelayParent(
	relayParent common.Hash, subSystemToOverseer chan<- any,
) (parachaintypes.ExecutorParams, error) {
	// TODO: Implement this #3544
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/subsystem-util/src/lib.rs#L241-L242
	return parachaintypes.ExecutorParams{}, nil
}

func (cb *CandidateBacking) processValidatedCandidateCommand(rpAndCmd RelayParentAndCommand) error {
	// TODO: Implement this #3571
	return nil
}

type BackgroundValidationResult struct {
	CandidateReceipt        *parachaintypes.CandidateReceipt
	CandidateCommitments    *parachaintypes.CandidateCommitments
	PersistedValidationData *parachaintypes.PersistedValidationData
	Err                     error
}

// RelayParentAndCommand contains the relay parent and the command to be executed on validated candidate,
// along with the result of the background validation.
type RelayParentAndCommand struct {
	RelayParent   common.Hash
	Command       ValidatedCandidateCommand
	ValidationRes BackgroundValidationResult
	CandidateHash parachaintypes.CandidateHash
}

// ValidatedCandidateCommand represents commands for handling validated candidates.
type ValidatedCandidateCommand byte

const (
	// We were instructed to second the candidate that has been already validated.
	Second = ValidatedCandidateCommand(iota)
	// We were instructed to validate the candidate.
	Attest
	// We were not able to `Attest` because backing validator did not send us the PoV.
	AttestNoPoV
)
