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
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-candidate-backing"))

var (
	errRejectedByProspectiveParachains = errors.New("candidate rejected by prospective parachains subsystem")
	errInvalidErasureRoot              = errors.New("erasure root doesn't match the announced by the candidate receipt")
	errStatementForUnknownRelayParent  = errors.New("received statement for unknown relay parent")
	errNilRelayParentState             = errors.New("relay parent state is nil")
	errCandidateStateNotFound          = errors.New("candidate state not found")
	errFallbackNotAvailable            = errors.New("no fallback available for the candidate")
)

// CandidateBacking represents the state of the subsystem responsible for managing candidate backing.
type CandidateBacking struct {
	SubSystemToOverseer chan<- any
	// State tracked for all relay-parents backing work is ongoing for. This includes
	// all active leaves.
	perRelayParent map[common.Hash]*perRelayParentState
	// State tracked for all candidates relevant to the implicit view.
	//
	// This is guaranteed to have an entry for each candidate with a relay parent in the implicit
	// or explicit view for which a `Seconded` statement has been successfully imported.
	perCandidate map[parachaintypes.CandidateHash]*perCandidateState
	// The utility for managing the implicit and explicit views in a consistent way.
	// We only feed leaves which have prospective parachains enabled to this view.
	ImplicitView ImplicitView
	// The handle to the Keystore used for signing.
	Keystore   keystore.Keystore
	BlockState BlockState
	// A local cache for storing per-session data. This cache helps to
	// reduce repeated calls to the runtime and avoid redundant computations.
	perSessionCache perSessionCache
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// perCandidateState represents the state information for a candidate in the subsystem.
type perCandidateState struct {
	persistedValidationData parachaintypes.PersistedValidationData
	secondedLocally         bool
	relayParent             common.Hash
}

// attestingData contains the data needed to retry validation with other backing validators
// in case a validator does not provide a PoV.
type attestingData struct {
	// The candidate to attest.
	candidate parachaintypes.CandidateReceiptV2
	// Hash of the PoV we need to fetch.
	povHash common.Hash
	// Validator we are currently trying to get the PoV from.
	fromValidator parachaintypes.ValidatorIndex
	// Other backing validators we can try in case `from_validator` failed.
	backing []parachaintypes.ValidatorIndex
}

// tableContext represents the contextual information associated with a validator and groups
// for a table under a relay-parent.
type tableContext struct {
	validator          *parachaintypes.Validator
	groups             map[parachaintypes.CoreIndex][]parachaintypes.ValidatorIndex
	validators         []parachaintypes.ValidatorID
	disabledValidators []parachaintypes.ValidatorIndex
}

// isMemberOf returns true if the validator is a member of the group of validators assigned to the parachain.
func (tc *tableContext) isMemberOf(validatorIndex parachaintypes.ValidatorIndex, core parachaintypes.CoreIndex) bool {
	validators, ok := tc.groups[core]
	if !ok {
		return false
	}

	return slices.Contains(validators, validatorIndex)
}

// GetBackableCandidatesMessage is a message received from overseer that requests a set of backable
// candidates that could be backed in a child of the given relay-parent.
// The order of candidates of the same para must be preserved in the response.
// So, If a backable candidate of a parachain cannot be retrieved,
// the response should not contain any candidates of the same parachain that follow it in the input slice.
type GetBackableCandidatesMessage struct {
	Candidates map[parachaintypes.ParaID][]*parachaintypes.CandidateHashAndRelayParent
	ResCh      chan map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate
}

// CanSecondMessage Request the candidate backing subsystem to check whether it's
// allowed to second given candidate.
// The rule is to only fetch collations that can either be directly chained to any
// FragmentChain in the view or there is at least one FragmentChain where this candidate is a
// potentially unconnected candidate (we predict that it may become connected to a
// FragmentChain in the future).
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
	CandidateReceipt        parachaintypes.CandidateReceiptV2
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
		perSessionCache:     newPerSessionCache(2),
	}
}

func (cb *CandidateBacking) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	chRelayParentAndCommand := make(chan relayParentAndCommand, 1)

	for {
		select {
		case rpAndCmd := <-chRelayParentAndCommand:
			if err := cb.processValidatedCandidateCommand(rpAndCmd, chRelayParentAndCommand); err != nil {
				logger.Errorf("processing validated candidated command: %s", err.Error())
			}
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			if err := cb.processMessage(msg, chRelayParentAndCommand); err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			close(chRelayParentAndCommand)
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (cb *CandidateBacking) Stop() {}

func (*CandidateBacking) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateBacking
}

// processMessage processes incoming messages from overseer
func (cb *CandidateBacking) processMessage(msg any, chRelayParentAndCommand chan relayParentAndCommand) error {
	switch msg := msg.(type) {
	case GetBackableCandidatesMessage:
		msg.ResCh <- cb.handleGetBackableCandidatesMessage(msg.Candidates)
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
		err := cb.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		return cb.ProcessBlockFinalizedSignal(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (cb *CandidateBacking) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// Nothing to do here
	return nil
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

	senderValidatorIndex := signedStatementWithPVD.SignedFullStatement.ValidatorIndex

	// Don't import statement if the sender is disabled
	if slices.Contains(rpState.tableContext.disabledValidators, senderValidatorIndex) {
		logger.Debugf("sender validator is disabled; validator index: %d",
			senderValidatorIndex)
		return nil
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

	// importStatement already takes care of communicating with the prospective parachains subsystem.
	// At this point, the candidate has already been accepted by the subsystem.

	if uint32(summary.GroupID) != rpState.assignedCore.Index {
		logger.Debugf("The GroupID: %d is not assigned to the local validator at relay parent: %s",
			summary.GroupID, relayParent)
		return nil
	}

	// already ensured in importStatement that the value of the statementVDT has been set.
	// that is why there is no chance we can get an error here.
	statementVDT, _ := signedStatementWithPVD.SignedFullStatement.Payload.Value()

	var attesting attestingData
	switch statementVDT := statementVDT.(type) {
	case parachaintypes.Seconded:
		commitedCandidateReceipt, err := rpState.table.getCommittedCandidateReceipt(summary.Candidate)
		if err != nil {
			return fmt.Errorf("getting candidate: %w", err)
		}

		attesting = attestingData{
			candidate:     commitedCandidateReceipt.ToPlain(),
			povHash:       statementVDT.Descriptor.PovHash,
			fromValidator: senderValidatorIndex,
			backing:       []parachaintypes.ValidatorIndex{},
		}
	case parachaintypes.Valid:
		candidateHash := parachaintypes.CandidateHash(statementVDT)
		attesting, ok = rpState.fallbacks[candidateHash]
		if !ok {
			// polkadot-sdk returs nil error here
			return errFallbackNotAvailable
		}

		ourIndex := rpState.tableContext.validator.Index
		if senderValidatorIndex == ourIndex {
			return nil
		}

		if rpState.awaitingValidation[candidateHash] {
			logger.Debug("Job already running")
			attesting.backing = append(attesting.backing, senderValidatorIndex)
			return nil
		}

		logger.Debug("No job, so start another with current validator")
		attesting.fromValidator = senderValidatorIndex
	}

	// in case of `Seconded` statement we add new fallback
	// in case of `Valid` statement we update existing fallback
	rpState.fallbacks[summary.Candidate] = attesting

	// After `import_statement` succeeds, the candidate entry is guaranteed to exist.
	pc, ok := cb.perCandidate[summary.Candidate]
	if !ok {
		return errCandidateStateNotFound
	}

	return rpState.kickOffValidationWork(
		cb.BlockState,
		cb.SubSystemToOverseer,
		chRelayParentAndCommand,
		pc.persistedValidationData,
		attesting,
	)
}

// perSessionCache is a cache for storing data per-session to reduce repeated runtime API calls
// and avoid redundant computations.
type perSessionCache struct {
	// Cache for storing validators list, retrieved from the runtime.
	validatorsCache *lrucache.LRUCache[parachaintypes.SessionIndex, []parachaintypes.ValidatorID]
	// Cache for storing node features, retrieved from the runtime.
	nodeFeaturesCache *lrucache.LRUCache[parachaintypes.SessionIndex, *parachaintypes.BitVec]
	// Cache for storing executor parameters, retrieved from the runtime.
	executorParamsCache *lrucache.LRUCache[parachaintypes.SessionIndex, parachaintypes.ExecutorParams]
	// Cache for storing the minimum backing votes threshold, retrieved from the runtime.
	minimumBackingVotesCache *lrucache.LRUCache[parachaintypes.SessionIndex, uint32]
	// Cache for storing validator-to-group mappings, computed from validator groups.
	validatorToGroupCache *lrucache.LRUCache[parachaintypes.SessionIndex,
		map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex] // indexed vector in Rust
}

func newPerSessionCache(capacity uint) perSessionCache {
	return perSessionCache{
		validatorsCache:          lrucache.NewLRUCache[parachaintypes.SessionIndex, []parachaintypes.ValidatorID](capacity),
		nodeFeaturesCache:        lrucache.NewLRUCache[parachaintypes.SessionIndex, *parachaintypes.BitVec](capacity),
		executorParamsCache:      lrucache.NewLRUCache[parachaintypes.SessionIndex, parachaintypes.ExecutorParams](capacity),
		minimumBackingVotesCache: lrucache.NewLRUCache[parachaintypes.SessionIndex, uint32](capacity),
		validatorToGroupCache: lrucache.NewLRUCache[parachaintypes.SessionIndex,
			map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex](capacity),
	}
}

// getValidators retrieves validators from the cache or fetches them from the runtime if not present.
func (cache *perSessionCache) getValidators(
	sessionIndex parachaintypes.SessionIndex,
	rt runtime.Instance,
) ([]parachaintypes.ValidatorID, error) {
	validators := cache.validatorsCache.Get(sessionIndex)
	if validators != nil {
		return validators, nil
	}

	validators, err := rt.ParachainHostValidators()
	if err != nil {
		return nil, err
	}

	cache.validatorsCache.Put(sessionIndex, validators)
	return validators, nil
}

// getNodeFeatures retrieves the node features from the cache or fetches it from the runtime if not present.
func (cache *perSessionCache) getNodeFeatures(
	sessionIndex parachaintypes.SessionIndex,
	rt runtime.Instance,
) (*parachaintypes.BitVec, error) {
	cachedNodeFeatures := cache.nodeFeaturesCache.Get(sessionIndex)
	if cachedNodeFeatures != nil {
		return cachedNodeFeatures, nil
	}

	nodeFeatures, err := rt.ParachainHostNodeFeatures()
	if err != nil {
		return nil, err
	}

	cache.nodeFeaturesCache.Put(sessionIndex, &nodeFeatures)
	return &nodeFeatures, nil
}

// getExecutorParams retrieves the executor parameters from the cache or fetches them from the runtime if not present.
func (cache *perSessionCache) getExecutorParams(
	sessionIndex parachaintypes.SessionIndex,
	rt runtime.Instance,
) (parachaintypes.ExecutorParams, error) {
	cachedExecParams := cache.executorParamsCache.Get(sessionIndex)
	if cachedExecParams != nil {
		return cachedExecParams, nil
	}

	params, err := rt.ParachainHostSessionExecutorParams(sessionIndex)
	if err != nil {
		return nil, err
	}

	cache.executorParamsCache.Put(sessionIndex, *params)
	return *params, nil
}

// getMinimumBackingVotes retrieves the minimum backing votes threshold from the cache or
// fetches it from the runtime if not present.
func (cache *perSessionCache) getMinimumBackingVotes(
	sessionIndex parachaintypes.SessionIndex,
	rt runtime.Instance,
) (uint32, error) {
	minBackingVotes := cache.minimumBackingVotesCache.Get(sessionIndex)
	if minBackingVotes != 0 {
		return minBackingVotes, nil
	}

	minBackingVotes, err := rt.ParachainHostMinimumBackingVotes()
	if err != nil {
		return 0, err
	}

	cache.minimumBackingVotesCache.Put(sessionIndex, minBackingVotes)
	return minBackingVotes, nil
}

// getValidatorToGroup retrieves a mapping of validators to their respective groups for a given session index.
// If the mapping is already cached, it returns the cached value.
// Otherwise, it constructs the mapping, caches it, and then returns it.
func (cache *perSessionCache) getValidatorToGroup(
	sessionIndex parachaintypes.SessionIndex,
	validators []parachaintypes.ValidatorID,
	validatorGroups [][]parachaintypes.ValidatorIndex,
) map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex {
	validatorToGroup := cache.validatorToGroupCache.Get(sessionIndex)
	if validatorToGroup != nil {
		return validatorToGroup
	}

	numOfValidators := len(validators)
	validatorToGroup = make(map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex, numOfValidators)

	for groupID, group := range validatorGroups {
		for _, validator := range group {
			validatorToGroup[validator] = parachaintypes.GroupIndex(groupID)
		}
	}

	cache.validatorToGroupCache.Put(sessionIndex, validatorToGroup)
	return validatorToGroup
}
