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
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

var (
	ErrRejectedByProspectiveParachains = errors.New("candidate rejected by prospective parachains subsystem")
	ErrValidationFailed                = errors.New("validation failed")
	ErrInvalidErasureRoot              = errors.New("erasure root doesn't match the announced by the candidate receipt")
)

type CandidateBacking struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	perRelayParent      map[common.Hash]perRelayParentState
	perCandidate        map[parachaintypes.CandidateHash]perCandidateState
}

type perCandidateState struct {
	persistedValidationData parachaintypes.PersistedValidationData
	SecondedLocally         bool
	ParaID                  parachaintypes.ParaID
	RelayParent             common.Hash
}

type perRelayParentState struct {
	ProspectiveParachainsMode ProspectiveParachainsMode
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
	validators []parachaintypes.ValidatorID
}

// Local validator information
//
// It can be created if the local node is a validator in the context of a particular
// relay chain block.
type Validator struct {
	index parachaintypes.ValidatorIndex
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

func New(overseerChan chan<- any) *CandidateBacking {
	return &CandidateBacking{
		SubSystemToOverseer: overseerChan,
	}
}

func (cb *CandidateBacking) Run(ctx context.Context, overseerToSubSystem chan any, subSystemToOverseer chan any) error {
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
		}
	}
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
		cb.handleCanSecondMessage()
	case SecondMessage:
		cb.handleSecondMessage()
	case StatementMessage:
		return cb.handleStatementMessage(msg.RelayParent, msg.SignedFullStatement, chRelayParentAndCommand)
	default:
		return errors.New("unknown message type")
	}
	return nil
}

func (cb *CandidateBacking) handleActiveLeavesUpdate() {
	// TODO: Implement this #3503
}

func (cb *CandidateBacking) handleGetBackedCandidatesMessage() {
	// TODO: Implement this #3504
}

func (cb *CandidateBacking) handleCanSecondMessage() {
	// TODO: Implement this #3505
}

func (cb *CandidateBacking) handleSecondMessage() {
	// TODO: Implement this #3506
}

// Import the statement and kick off validation work if it is a part of our assignment.
func (cb *CandidateBacking) handleStatementMessage(
	relayParent common.Hash,
	signedStatementWithPVD SignedFullStatementWithPVD,
	chRelayParentAndCommand chan RelayParentAndCommand,
) error {
	rpState, ok := cb.perRelayParent[relayParent]
	if !ok {
		return fmt.Errorf("Received statement for unknown relay parent %s", relayParent)
	}

	summery, err := rpState.importStatement(cb.SubSystemToOverseer, signedStatementWithPVD, cb.perCandidate)
	if err != nil {
		return fmt.Errorf("importing statement: %w", err)
	}

	if err := postImportStatement(); err != nil {
		return fmt.Errorf("post import statement: %w", err)
	}

	if summery == nil || summery.GroupID != uint32(rpState.Assignment) {
		return nil
	}

	statementVDT, err := signedStatementWithPVD.SignedFullStatement.Payload.Value()
	if err != nil {
		return fmt.Errorf("getting value from statementVDT: %w", err)
	}

	var attesting AttestingData
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

		attesting = AttestingData{
			candidate:     candidateReceipt,
			povHash:       statementVDT.(parachaintypes.Seconded).Descriptor.PovHash,
			fromValidator: signedStatementWithPVD.SignedFullStatement.ValidatorIndex,
			backing:       []parachaintypes.ValidatorIndex{},
		}

		rpState.fallbacks[summery.Candidate] = attesting

	case 2: // Valid
		attesting, ok = rpState.fallbacks[summery.Candidate]
		if !ok {
			return nil
		}

		ourIndex := rpState.TableContext.validator.index
		if signedStatementWithPVD.SignedFullStatement.ValidatorIndex == ourIndex {
			return nil
		}

		if rpState.AwaitingValidation[summery.Candidate] {
			// Job already running
			attesting.backing = append(attesting.backing, signedStatementWithPVD.SignedFullStatement.ValidatorIndex)
			return nil
		}
		// No job, so start another with current validator:
		attesting.fromValidator = signedStatementWithPVD.SignedFullStatement.ValidatorIndex

	default:
		return fmt.Errorf("invalid statementVDT index: %d", statementVDT.Index())
	}

	// After `import_statement` succeeds, the candidate entry is guaranteed to exist.
	if pc, ok := cb.perCandidate[summery.Candidate]; ok {
		rpState.kickOffValidationWork(
			cb.SubSystemToOverseer,
			chRelayParentAndCommand,
			pc.persistedValidationData,
			attesting,
		)
	}

	return nil
}

func (rpState *perRelayParentState) importStatement(
	subSystemToOverseer chan<- any,
	signedStatementWithPVD SignedFullStatementWithPVD,
	perCandidate map[parachaintypes.CandidateHash]perCandidateState,
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
	candidateHash := parachaintypes.CandidateHash{
		Value: common.MustBlake2bHash(scale.MustMarshal(statementVDTSeconded)),
	}

	if _, ok := perCandidate[candidateHash]; ok {
		return rpState.Table.importStatement(&rpState.TableContext, signedStatementWithPVD)
	}

	if rpState.ProspectiveParachainsMode.IsEnabled {
		chIntroduceCandidate := make(chan error)
		subSystemToOverseer <- ProspectiveParachainsMessage{
			Value: IntroduceCandidate{
				IntroduceCandidateRequest: IntroduceCandidateRequest{
					CandidateParaID:           parachaintypes.ParaID(statementVDTSeconded.Descriptor.ParaID),
					CommittedCandidateReceipt: parachaintypes.CommittedCandidateReceipt(statementVDTSeconded),
					PersistedValidationData:   *signedStatementWithPVD.PersistedValidationData,
				},
				Ch: chIntroduceCandidate,
			},
		}

		introduceCandidateErr := <-chIntroduceCandidate
		if introduceCandidateErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrRejectedByProspectiveParachains, introduceCandidateErr)
		}

		subSystemToOverseer <- ProspectiveParachainsMessage{
			Value: CandidateSeconded{
				ParaID:        parachaintypes.ParaID(statementVDTSeconded.Descriptor.ParaID),
				CandidateHash: candidateHash,
			},
		}
	}

	// Only save the candidate if it was approved by prospective parachains.
	perCandidate[candidateHash] = perCandidateState{
		persistedValidationData: *signedStatementWithPVD.PersistedValidationData,
		SecondedLocally:         false, // This is set after importing when seconding locally.
		ParaID:                  parachaintypes.ParaID(statementVDTSeconded.Descriptor.ParaID),
		RelayParent:             statementVDTSeconded.Descriptor.RelayParent,
	}

	return rpState.Table.importStatement(&rpState.TableContext, signedStatementWithPVD)
}

type ProspectiveParachainsMessage struct {
	Value any
}

// Inform the Prospective Parachains Subsystem of a new candidate.
//
// The response sender accepts the candidate membership, which is the existing
// membership of the candidate if it was already known.
type IntroduceCandidate struct {
	IntroduceCandidateRequest IntroduceCandidateRequest
	Ch                        chan error
}

// Inform the Prospective Parachains Subsystem that a previously introduced candidate
// has been seconded. This requires that the candidate was successfully introduced in
// the past.
type CandidateSeconded struct {
	ParaID        parachaintypes.ParaID
	CandidateHash parachaintypes.CandidateHash
}

type IntroduceCandidateRequest struct {
	// The para-id of the candidate.
	CandidateParaID parachaintypes.ParaID
	// The candidate receipt itself.
	CommittedCandidateReceipt parachaintypes.CommittedCandidateReceipt
	// The persisted validation data of the candidate.
	PersistedValidationData parachaintypes.PersistedValidationData
}

type ProspectiveParachainsMode struct {
	// Runtime API without support of `async_backing_params`: no prospective parachains.
	// v6 runtime API: prospective parachains.
	// NOTE: MaxCandidateDepth and AllowedAncestryLen need to be set if this is enabled.
	IsEnabled bool

	// The maximum number of para blocks between the para head in a relay parent
	// and a new candidate. Restricts nodes from building arbitrary long chains
	// and spamming other validators.
	MaxCandidateDepth uint
	// How many ancestors of a relay parent are allowed to build candidates on top of.
	AllowedAncestryLen uint
}

func postImportStatement() error {
	//	TODO: Implement this by referencing
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/core/backing/src/lib.rs#L1573-L1574
	return nil
}

// Kick off validation work and distribute the result as a signed statement.
func (rpState *perRelayParentState) kickOffValidationWork(
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan RelayParentAndCommand,
	pvd parachaintypes.PersistedValidationData,
	attesting AttestingData,
) {
	candidateHash := parachaintypes.CandidateHash{
		Value: common.MustBlake2bHash(scale.MustMarshal(attesting.candidate)),
	}

	if rpState.issuedStatements[candidateHash] {
		return
	}

	if !rpState.AwaitingValidation[candidateHash] {
		rpState.AwaitingValidation[candidateHash] = true

		pov := GetPovFromValidator()

		go backgroundValidateAndMakeAvailable(
			subSystemToOverseer,
			chRelayParentAndCommand,
			attesting.candidate,
			rpState.RelayParent,
			pvd,
			pov,
			uint32(len(rpState.TableContext.validators)),
			Attest,
		)
	}
}

func backgroundValidateAndMakeAvailable(
	subSystemToOverseer chan<- any,
	chRelayParentAndCommand chan RelayParentAndCommand,
	candidateReceipt parachaintypes.CandidateReceipt,
	relayPaent common.Hash,
	pvd parachaintypes.PersistedValidationData,
	pov parachaintypes.PoV,
	numValidator uint32,
	makeCommand ValidatedCandidateCommand,
) {
	validationCodeHash := candidateReceipt.Descriptor.ValidationCodeHash
	candidateHash := parachaintypes.CandidateHash{
		Value: common.MustBlake2bHash(scale.MustMarshal(candidateReceipt)),
	}

	chValidationCodeByHashRes := make(chan OverseerFuncRes[parachaintypes.ValidationCode])
	subSystemToOverseer <- RuntimeApiMessage{
		RelayParent: relayPaent,
		RuntimeApiRequest: ValidationCodeByHash{
			ValidationCodeHash: validationCodeHash,
			Ch:                 chValidationCodeByHashRes,
		},
	}

	ValidationCodeByHashRes := <-chValidationCodeByHashRes
	if ValidationCodeByHashRes.err != nil {
		logger.Error(ValidationCodeByHashRes.err.Error())
		return
	}

	executorParams, err := executorParamsAtRelayParent(relayPaent, subSystemToOverseer)
	if err != nil {
		logger.Errorf("could not get executor params at relay parent: %w", err)
	}

	chValidationResultRes := make(chan OverseerFuncRes[ValidationResult])
	subSystemToOverseer <- CandidateValidationMessage{
		Value: ValidateFromExhaustive{
			PersistedValidationData: pvd,
			ValidationCode:          ValidationCodeByHashRes.data,
			CandidateReceipt:        candidateReceipt,
			pov:                     pov,
			ExecutorParams:          executorParams,
			PvfPrepTimeoutKind:      Approval,
			Ch:                      chValidationResultRes,
		},
	}

	ValidationResultRes := <-chValidationResultRes
	if ValidationResultRes.err != nil {
		logger.Error(ValidationResultRes.err.Error())
	}

	var backgroundValidationResult BackgroundValidationResult

	if ValidationResultRes.data.IsValid { // Valid
		// Important: the `av-store` subsystem will check if the erasure root of the `available_data`
		// matches `expected_erasure_root` which was provided by the collator in the `CandidateReceipt`.
		// This check is consensus critical and the `backing` subsystem relies on it for ensuring
		// candidate validity.

		chStoreAvailableDataError := make(chan error)
		subSystemToOverseer <- AvailabilityStoreMessage{
			Value: StoreAvailableData{
				CandidateHash: candidateHash,
				NumValidators: numValidator,
				AvailableData: AvailableData{pov, pvd},
				Ch:            chStoreAvailableDataError,
			},
		}

		storeAvailableDataError := <-chStoreAvailableDataError
		switch storeAvailableDataError {
		case nil:
			backgroundValidationResult = BackgroundValidationResult{
				CandidateReceipt:        &candidateReceipt,
				CandidateCommitments:    &ValidationResultRes.data.CandidateCommitments,
				PersistedValidationData: &ValidationResultRes.data.PersistedValidationData,
				Err:                     nil,
			}
		case ErrInvalidErasureRoot:
			logger.Debug(ErrInvalidErasureRoot.Error())

			backgroundValidationResult = BackgroundValidationResult{
				CandidateReceipt: &candidateReceipt,
				Err:              ErrInvalidErasureRoot,
			}
		default:
			logger.Error(storeAvailableDataError.Error())
			return
		}

	} else { // Invalid
		logger.Error(ValidationResultRes.data.err.Error())
		backgroundValidationResult = BackgroundValidationResult{
			CandidateReceipt: &candidateReceipt,
			Err:              ErrInvalidErasureRoot,
		}
	}

	chRelayParentAndCommand <- RelayParentAndCommand{
		RelayParent:   relayPaent,
		Command:       makeCommand,
		ValidationRes: backgroundValidationResult,
		CandidateHash: candidateHash,
	}
}

func GetPovFromValidator() parachaintypes.PoV {
	//	TODO: Implement this
	//	https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/core/backing/src/lib.rs#L1744
	return parachaintypes.PoV{}
}

type ExecutorParams struct {
	// TODO: Implement this
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/primitives/src/v6/executor_params.rs#L97-L98
}

func executorParamsAtRelayParent(relayParent common.Hash, subSystemToOverseer chan<- any) (ExecutorParams, error) {
	// TODO: Implement this
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/subsystem-util/src/lib.rs#L241-L242
	return ExecutorParams{}, nil
}

func (cb *CandidateBacking) processValidatedCandidateCommand(rpAndCmd RelayParentAndCommand) error {
	// TODO: Implement this
	return nil
}

type RuntimeApiMessage struct {
	RelayParent       common.Hash
	RuntimeApiRequest any
}

type ValidationCodeByHash struct {
	ValidationCodeHash parachaintypes.ValidationCodeHash
	Ch                 chan OverseerFuncRes[parachaintypes.ValidationCode]
}

type CandidateValidationMessage struct {
	Value any
}

type ValidateFromExhaustive struct {
	PersistedValidationData parachaintypes.PersistedValidationData
	ValidationCode          parachaintypes.ValidationCode
	CandidateReceipt        parachaintypes.CandidateReceipt
	pov                     parachaintypes.PoV
	ExecutorParams          ExecutorParams
	PvfPrepTimeoutKind      PvfPrepTimeoutKind
	Ch                      chan OverseerFuncRes[ValidationResult]
}

// Type discriminator for PVF execution timeouts
type PvfPrepTimeoutKind byte

const (
	// The amount of time to spend on execution during backing.
	Backing PvfPrepTimeoutKind = iota
	/// The amount of time to spend on execution during approval or disputes.
	///
	/// This should be much longer than the backing execution timeout to ensure that in the
	/// absence of extremely large disparities between hardware, blocks that pass backing are
	/// considered executable by approval checkers or dispute participants.
	Approval
)

// ValidationResult coming from candidate validation subsystem
type ValidationResult struct {
	IsValid                 bool
	CandidateCommitments    parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
	err                     error
}

type AvailabilityStoreMessage struct {
	Value any
}

// Computes and checks the erasure root of `AvailableData` before storing all of its chunks in
// the AV store.
type StoreAvailableData struct {
	CandidateHash       parachaintypes.CandidateHash
	NumValidators       uint32
	AvailableData       AvailableData
	ExpectedErasureRoot common.Hash
	Ch                  chan error
}

// AvailableData represents the data that is kept available for each candidate included in the relay chain.
type AvailableData struct {
	// The Proof-of-Validation (PoV) of the candidate
	PoV parachaintypes.PoV `scale:"1"`

	// The persisted validation data needed for approval checks
	ValidationData parachaintypes.PersistedValidationData `scale:"2"`
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

type ValidatedCandidateCommand byte

const (
	// We were instructed to second the candidate that has been already validated.
	Second = ValidatedCandidateCommand(iota)
	// We were instructed to validate the candidate.
	Attest
	// We were not able to `Attest` because backing validator did not send us the PoV.
	AttestNoPoV
)

type OverseerFuncRes[T any] struct {
	err  error
	data T
}
