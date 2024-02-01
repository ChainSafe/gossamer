// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// This package implements the Candidate Backing subsystem.
// It ensures every parablock considered for relay block inclusion has been seconded by at least
// one validator, and approved by a quorum. Parablocks for which not enough validators will assert
// correctness are discarded. If the block later proves invalid, the initial backers are slashable;
// this gives Polkadot a rational threat model during subsequent stages.

// Its role is to produce backable candidates for inclusion in new relay-chain blocks. It does so
// by issuing signed Statements and tracking received statements signed by other validators. Once
// enough statements are received, they can be combined into backing for specific candidates.

// Note that though the candidate backing subsystem attempts to produce as many backable candidates
// as possible, it does not attempt to choose a single authoritative one. The choice of which
// actually gets included is ultimately up to the block author, by whatever metrics it may use;
// those are opaque to this subsystem.

// Once a sufficient quorum has agreed that a candidate is valid, this subsystem notifies the
// Provisioner, which in turn engages block production mechanisms to include the parablock.

package backing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/tidwall/btree"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

var (
	errRejectedByProspectiveParachains = errors.New("candidate rejected by prospective parachains subsystem")
	errInvalidErasureRoot              = errors.New("erasure root doesn't match the announced by the candidate receipt")
	errStatementForUnknownRelayParent  = errors.New("received statement for unknown relay parent")
	errNilRelayParentState             = errors.New("relay parent state is nil")
	errCandidateStateNotFound          = errors.New("candidate state not found")
	errAttestingDataNotFound           = errors.New("attesting data not found")
)

// CandidateBacking represents the state of the subsystem responsible for managing candidate backing.
type CandidateBacking struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

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
	perLeaf map[common.Hash]*activeLeafState
	// The utility for managing the implicit and explicit views in a consistent way.
	// We only feed leaves which have prospective parachains enabled to this view.
	implicitView ImplicitView
	// The handle to the keystore used for signing.
	keystore keystore.Keystore
}

type activeLeafState struct {
	prospectiveParachainsMode parachaintypes.ProspectiveParachainsMode
	secondedAtDepth           map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]
	perCandidate              map[parachaintypes.CandidateHash]*perCandidateState //nolint:unused
}

// perCandidateState represents the state information for a candidate in the subsystem.
type perCandidateState struct {
	persistedValidationData parachaintypes.PersistedValidationData
	secondedLocally         bool
	paraID                  parachaintypes.ParaID
	relayParent             common.Hash
}

// attestingData contains the data needed to retry validation with other backing validators
// in case a validator does not provide a PoV.
type attestingData struct {
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
	validator  *validator
	groups     map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex
	validators []parachaintypes.ValidatorID
}

// validator represents local validator information.
// It can be created if the local node is a validator in the context of a particular relay chain block.
type validator struct {
	signingContext parachaintypes.SigningContext
	key            parachaintypes.ValidatorID
	index          parachaintypes.ValidatorIndex
}

// sign method signs a given payload with the validator and returns a SignedFullStatement.
func (v validator) sign(keystore keystore.Keystore, payload parachaintypes.StatementVDT,
) (parachaintypes.SignedFullStatement, error) {
	statement := parachaintypes.SignedFullStatement{
		Payload:        payload,
		ValidatorIndex: v.index,
	}
	return statement.Sign(keystore, v.signingContext, v.key)
}

// GetBackedCandidatesMessage is a message received from overseer that requests a set of backable
// candidates that could be backed in a child of the given relay-parent.
type GetBackedCandidatesMessage struct {
	Candidates []*CandidateHashAndRelayParent
	ResCh      chan []*parachaintypes.BackedCandidate
}

type CandidateHashAndRelayParent struct {
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
	ResponseCh           chan bool
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
	SignedFullStatement parachaintypes.SignedFullStatementWithPVD
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
	cb.wg.Add(1)
	go cb.runUtil()
}

func (cb *CandidateBacking) runUtil() {
	chRelayParentAndCommand := make(chan relayParentAndCommand)

	for {
		select {
		case rpAndCmd := <-chRelayParentAndCommand:
			if err := cb.processValidatedCandidateCommand(rpAndCmd); err != nil {
				logger.Errorf("processing validated candidated command: %s", err.Error())
			}
		case msg := <-cb.OverseerToSubSystem:
			if err := cb.processMessage(msg, chRelayParentAndCommand); err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-cb.ctx.Done():
			close(chRelayParentAndCommand)
			if err := cb.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			cb.wg.Done()
			return
		}
	}
}

func (cb *CandidateBacking) Stop() {
	cb.cancel()
	cb.wg.Wait()
}

func (*CandidateBacking) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateBacking
}

// processMessage processes incoming messages from overseer
func (cb *CandidateBacking) processMessage(msg any, chRelayParentAndCommand chan relayParentAndCommand) error {
	// process these received messages by referencing
	// https://github.com/paritytech/polkadot-sdk/blob/769bdd3ff33a291cbc70a800a3830638467e42a2/polkadot/node/core/backing/src/lib.rs#L741
	switch msg := msg.(type) {
	case GetBackedCandidatesMessage:
		cb.handleGetBackedCandidatesMessage(msg)
	case CanSecondMessage:
		err := cb.handleCanSecondMessage(msg)
		if err != nil {
			logger.Debug(fmt.Sprintf("can't second the candidate: %s", err))
		}
	case SecondMessage:
		return cb.handleSecondMessage(msg.CandidateReceipt, msg.PersistedValidationData, msg.PoV, chRelayParentAndCommand)
	case StatementMessage:
		return cb.handleStatementMessage(msg.RelayParent, msg.SignedFullStatement, chRelayParentAndCommand)
	case parachaintypes.ActiveLeavesUpdateSignal:
		cb.ProcessActiveLeavesUpdateSignal()
	case parachaintypes.BlockFinalizedSignal:
		cb.ProcessBlockFinalizedSignal()
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (cb *CandidateBacking) ProcessActiveLeavesUpdateSignal() {
	// TODO #3503
}

func (cb *CandidateBacking) ProcessBlockFinalizedSignal() {
	// TODO #3644
}

// Import the statement and kick off validation work if it is a part of our assignment.
func (cb *CandidateBacking) handleStatementMessage(
	relayParent common.Hash,
	signedStatementWithPVD parachaintypes.SignedFullStatementWithPVD,
	chRelayParentAndCommand chan relayParentAndCommand,
) error {
	rpState, ok := cb.perRelayParent[relayParent]
	if !ok {
		return fmt.Errorf("%w: %s", errStatementForUnknownRelayParent, relayParent)
	}

	if rpState == nil {
		return errNilRelayParentState
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

	if summary.GroupID != rpState.assignment {
		logger.Debugf("The ParaId: %d is not assigned to the local validator at relay parent: %s",
			summary.GroupID, relayParent)
		return nil
	}

	// already ensured in importStatement that the value of the statementVDT has been set.
	// that is why there is no chance we can get an error here.
	statementVDT, _ := signedStatementWithPVD.SignedFullStatement.Payload.Value()

	var attesting attestingData
	switch statementVDT := statementVDT.(type) {
	case parachaintypes.Seconded:
		commitedCandidateReceipt, err := rpState.table.getCandidate(summary.Candidate)
		if err != nil {
			return fmt.Errorf("getting candidate: %w", err)
		}

		attesting = attestingData{
			candidate:     commitedCandidateReceipt.ToPlain(),
			povHash:       statementVDT.Descriptor.PovHash,
			fromValidator: signedStatementWithPVD.SignedFullStatement.ValidatorIndex,
			backing:       []parachaintypes.ValidatorIndex{},
		}
	case parachaintypes.Valid:
		candidateHash := parachaintypes.CandidateHash(statementVDT)
		attesting, ok = rpState.fallbacks[candidateHash]
		if !ok {
			// polkadot-sdk returs nil error here
			return errAttestingDataNotFound
		}

		ourIndex := rpState.tableContext.validator.index
		if signedStatementWithPVD.SignedFullStatement.ValidatorIndex == ourIndex {
			return nil
		}

		if rpState.awaitingValidation[candidateHash] {
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
		return errCandidateStateNotFound
	}

	return rpState.kickOffValidationWork(
		cb.SubSystemToOverseer,
		chRelayParentAndCommand,
		pc.persistedValidationData,
		attesting,
	)
}

func getPovFromValidator() parachaintypes.PoV {
	// TODO: Implement this #3545
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/core/backing/src/lib.rs#L1744 //nolint:lll
	return parachaintypes.PoV{}
}
