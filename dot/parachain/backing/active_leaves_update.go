// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/tidwall/btree"
)

// Backing votes threshold used from the host prior to runtime API version 6 and
// from the runtime prior to v9 configuration migration.
const LEGACY_MIN_BACKING_VOTES uint32 = 2

// ProcessActiveLeavesUpdateSignal updates the state of the CandidateBacking struct based on the
// provided ActiveLeavesUpdateSignal.
// It manages the activation and deactivation of relay chain block, performs cleanup operations
// on the perRelayParent and perCandidate maps, and adds entries to perRelayParent for
// new relay-parents introduced by the update.
func (cb *CandidateBacking) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	var implicitViewFetchError error
	var prospectiveParachainsMode parachaintypes.ProspectiveParachainsMode
	activatedLeaf := update.Activated

	// activate in implicit view before deactivate, per the docs on ImplicitView, this is more efficient.
	if activatedLeaf != nil {
		var err error
		prospectiveParachainsMode, err = getProspectiveParachainsMode(cb.BlockState, activatedLeaf.Hash)
		if err != nil {
			return fmt.Errorf("getting prospective parachains mode: %w", err)
		}

		// activate in implicit view only if prospective parachains are enabled.
		if prospectiveParachainsMode.IsEnabled {
			_, implicitViewFetchError = cb.implicitView.activeLeaf(activatedLeaf.Hash)
		}
	}

	for _, deactivated := range update.Deactivated {
		delete(cb.perLeaf, deactivated)
		cb.implicitView.deactivateLeaf(deactivated)
	}

	// clean up `perRelayParent` according to ancestry of leaves.
	// we do this so we can clean up candidates right after as a result.
	//
	// when prospective parachains are disabled, the implicit view is empty,
	// which means we'll clean up everything that's not a leaf - the expected behaviour
	// for pre-asynchronous backing.
	cb.cleanUpPerRelayParentByLeafAncestry()

	// clean up `perCandidate` according to which relay-parents are known.
	//
	// when prospective parachains are disabled, we clean up all candidates
	// because we've cleaned up all relay parents. this is correct.
	cb.removeUnknownRelayParentsFromPerCandidate()

	if activatedLeaf == nil {
		return nil
	}

	// Get relay parents which might be fresh but might be known already
	// that are explicit or implicit from the new active leaf.
	var freshRelayParents []common.Hash

	switch prospectiveParachainsMode.IsEnabled {
	case false:
		if _, ok := cb.perLeaf[activatedLeaf.Hash]; ok {
			return nil
		}

		cb.perLeaf[activatedLeaf.Hash] = &activeLeafState{
			prospectiveParachainsMode: prospectiveParachainsMode,

			// This is empty because the only allowed relay-parent and depth
			// when prospective parachains are disabled is the leaf hash and 0,
			// respectively. We've just learned about the leaf hash, so we cannot
			// have any candidates seconded with it as a relay-parent yet.
			secondedAtDepth: make(map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]),
		}

		freshRelayParents = []common.Hash{activatedLeaf.Hash}
	case true:
		if implicitViewFetchError != nil {
			return fmt.Errorf("failed to load implicit view for leaf %s: %w", activatedLeaf.Hash, implicitViewFetchError)
		}

		freshRelayParents = cb.implicitView.knownAllowedRelayParentsUnder(activatedLeaf.Hash, nil)

		// At this point, all candidates outside of the implicit view
		// have been cleaned up. For all which remain, which we've seconded,
		// we ask the prospective parachains subsystem where they land in the fragment
		// tree for the given active leaf. This comprises our `secondedAtDepth`.

		remainingSeconded := make(map[parachaintypes.CandidateHash]parachaintypes.ParaID)
		for candidateHash, candidateState := range cb.perCandidate {
			if candidateState.secondedLocally {
				remainingSeconded[candidateHash] = candidateState.paraID
			}
		}

		// update the candidates seconded at various depths under new active leaves.
		{
			var wg sync.WaitGroup
			var mut sync.Mutex
			secondedAtDepth := make(map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash])

			for candidateHash, paraID := range remainingSeconded {
				wg.Add(1)
				go updateCandidateSecondedAtDepth(
					&wg, &mut, cb.SubSystemToOverseer, candidateHash, paraID, activatedLeaf.Hash, secondedAtDepth,
				)
			}
			wg.Wait()

			cb.perLeaf[activatedLeaf.Hash] = &activeLeafState{
				prospectiveParachainsMode: prospectiveParachainsMode,
				secondedAtDepth:           secondedAtDepth,
			}
		}

		if len(freshRelayParents) == 0 {
			logger.Warnf("implicit view gave no relay-parents under leaf-hash %s", activatedLeaf.Hash)
			freshRelayParents = []common.Hash{activatedLeaf.Hash}
		}
	}

	// add entries in `perRelayParent`. for all new relay-parents.
	for _, maybeNewRP := range freshRelayParents {
		if _, ok := cb.perRelayParent[maybeNewRP]; ok {
			continue
		}

		var mode parachaintypes.ProspectiveParachainsMode
		leaf, ok := cb.perLeaf[maybeNewRP]
		if !ok {
			// If the relay-parent isn't a leaf itself,
			// then it is guaranteed by the prospective parachains
			// subsystem that it is an ancestor of a leaf which
			// has prospective parachains enabled and that the
			// block itself did.
			mode = prospectiveParachainsMode
		} else {
			mode = leaf.prospectiveParachainsMode
		}

		// construct a `PerRelayParent` from the runtime API and insert it.
		rpState, err := constructPerRelayParentState(cb.BlockState, maybeNewRP, &cb.keystore, mode)
		if err != nil {
			return fmt.Errorf("constructing per relay parent state for relay-parent %s: %w", maybeNewRP, err)
		}

		if rpState != nil {
			cb.perRelayParent[maybeNewRP] = rpState
		}
	}
	return nil
}

// updateCandidateSecondedAtDepth updates candidates seconded at depth under new active leaves.
func updateCandidateSecondedAtDepth(
	wg *sync.WaitGroup, mut *sync.Mutex, subSystemToOverseer chan<- any,
	candidateHash parachaintypes.CandidateHash, paraID parachaintypes.ParaID,
	leafHash common.Hash, secondedAtDepth map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash],
) {
	defer wg.Done()

	getTreeMembership := parachaintypes.ProspectiveParachainsMessageGetTreeMembership{
		ParaID:        paraID,
		CandidateHash: candidateHash,
		ResponseCh:    make(chan []parachaintypes.FragmentTreeMembership),
	}

	var membership []parachaintypes.FragmentTreeMembership

	subSystemToOverseer <- getTreeMembership
	select {
	case membership = <-getTreeMembership.ResponseCh:
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		logger.Errorf("getting fragment tree membership: %w; candidate: %s, para-id: %d",
			parachaintypes.ErrSubsystemRequestTimeout, candidateHash, paraID)
		return
	}

	for _, m := range membership {
		if m.RelayParent == leafHash {
			mut.Lock()

			tree, ok := secondedAtDepth[paraID]
			if !ok {
				tree = new(btree.Map[uint, parachaintypes.CandidateHash])
			}

			for _, depth := range m.Depths {
				tree.Load(depth, candidateHash)
			}
			mut.Unlock()
		}
	}

}

func (cb *CandidateBacking) cleanUpPerRelayParentByLeafAncestry() {
	remaining := make(map[common.Hash]bool)

	for hash := range cb.perLeaf {
		remaining[hash] = true
	}

	allowedRelayParents := cb.implicitView.allAllowedRelayParents()
	for _, relayParent := range allowedRelayParents {
		remaining[relayParent] = true
	}

	keysToDelete := []common.Hash{}
	for rp := range cb.perRelayParent {
		if _, ok := remaining[rp]; !ok {
			keysToDelete = append(keysToDelete, rp)
		}
	}

	for _, key := range keysToDelete {
		delete(cb.perRelayParent, key)
	}
}

func (cb *CandidateBacking) removeUnknownRelayParentsFromPerCandidate() {
	keysToDelete := []parachaintypes.CandidateHash{}

	for candidateHash, pc := range cb.perCandidate {
		if _, ok := cb.perRelayParent[pc.relayParent]; !ok {
			keysToDelete = append(keysToDelete, candidateHash)
		}
	}

	for _, key := range keysToDelete {
		delete(cb.perCandidate, key)
	}
}

// getProspectiveParachainsMode requests prospective parachains mode
// for a given relay parent based on the Runtime API version.
func getProspectiveParachainsMode(blockstate BlockState, relayParent common.Hash,
) (parachaintypes.ProspectiveParachainsMode, error) {
	var emptyMode parachaintypes.ProspectiveParachainsMode

	rt, err := blockstate.GetRuntime(relayParent)
	if err != nil {
		return emptyMode, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	params, err := rt.ParachainHostAsyncBackingParams()
	if err != nil {
		if errors.Is(err, wazero_runtime.ErrExportFunctionNotFound) {
			logger.Debugf(
				"%s is not supported by the current Runtime API of the relay parent %s",
				runtime.ParachainHostAsyncBackingParams, relayParent,
			)

			return parachaintypes.ProspectiveParachainsMode{IsEnabled: false}, nil
		}
		return emptyMode, fmt.Errorf("getting async backing params: %w", err)
	}

	enabled := parachaintypes.ProspectiveParachainsMode{
		IsEnabled:          true,
		MaxCandidateDepth:  uint(params.MaxCandidateDepth),
		AllowedAncestryLen: uint(params.AllowedAncestryLen),
	}

	return enabled, nil
}

// Load the data necessary to do backing work on top of a relay-parent.
func constructPerRelayParentState(
	blockstate BlockState,
	relayParent common.Hash,
	keystore *keystore.Keystore,
	mode parachaintypes.ProspectiveParachainsMode,
) (*perRelayParentState, error) {
	rt, err := blockstate.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	sessionIndex, validators, validatorGroups, cores, err := fetchParachainHostData(rt)
	if err != nil {
		return nil, fmt.Errorf("fetching parachain host data: %w", err)
	}

	minBackingVotes, err := minBackingVotes(rt)
	if err != nil {
		return nil, fmt.Errorf("getting minimum backing votes: %w", err)
	}

	signingContext := parachaintypes.SigningContext{
		SessionIndex: sessionIndex,
		ParentHash:   relayParent,
	}

	var localValidator *validator
	validatorID, validatorIndex := parachainutil.SigningKeyAndIndex(validators, *keystore)
	if validatorID != nil {
		//  local node is a validator
		localValidator = &validator{
			signingContext: signingContext,
			key:            *validatorID,
			index:          validatorIndex,
		}
	}

	var assignment *parachaintypes.ParaID

	groups := make(map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex)

	numOfCores := uint(len(cores))

	for idx := uint(0); idx < numOfCores; idx++ {
		coreValue, err := cores[idx].Value()
		if err != nil {
			return nil, fmt.Errorf("getting core value at index %d: %w", idx, err)
		}

		var coreParaID parachaintypes.ParaID
		switch v := coreValue.(type) {
		case parachaintypes.OccupiedCore:
			if mode.IsEnabled {
				// Async backing makes it legal to build on top of occupied core.
				coreParaID = parachaintypes.ParaID(v.CandidateDescriptor.ParaID)
			} else {
				continue
			}
		case parachaintypes.ScheduledCore:
			coreParaID = v.ParaID
		case parachaintypes.Free:
			continue
		}

		coreIndex := parachaintypes.CoreIndex{Index: uint32(idx)}
		groupIndex := validatorGroups.GroupRotationInfo.GroupForCore(coreIndex, numOfCores)
		validatorIndexes := validatorGroups.Validators[groupIndex]

		if validatorIndexes != nil {
			if localValidator != nil && slices.Contains(validatorIndexes, localValidator.index) {
				assignment = &coreParaID
			}
			groups[coreParaID] = validatorIndexes
		}
	}

	tableContext := TableContext{
		validator:  localValidator,
		groups:     groups,
		validators: validators,
	}

	tableConfig := Config{
		AllowMultipleSeconded: mode.IsEnabled,
	}

	newPerRelayParentState := perRelayParentState{
		prospectiveParachainsMode: mode,
		relayParent:               relayParent,
		assignment:                assignment,
		table:                     newTable(tableConfig),
		tableContext:              tableContext,
		fallbacks:                 make(map[parachaintypes.CandidateHash]attestingData),
		awaitingValidation:        make(map[parachaintypes.CandidateHash]bool),
		issuedStatements:          make(map[parachaintypes.CandidateHash]bool),
		backed:                    make(map[parachaintypes.CandidateHash]bool),
		minBackingVotes:           minBackingVotes,
	}

	return &newPerRelayParentState, nil
}

func minBackingVotes(rt runtime.Instance) (uint32, error) {
	votes, err := rt.ParachainHostMinimumBackingVotes()
	if err != nil && errors.Is(err, wazero_runtime.ErrExportFunctionNotFound) {
		logger.Tracef(
			"%s is not supported by the current Runtime API",
			runtime.ParachainHostMinimumBackingVotes,
		)
		return LEGACY_MIN_BACKING_VOTES, nil
	}
	return votes, err
}

func fetchParachainHostData(rt runtime.Instance) (
	parachaintypes.SessionIndex,
	[]parachaintypes.ValidatorID,
	parachaintypes.ValidatorGroups,
	[]parachaintypes.CoreState,
	error,
) {
	var (
		sessionIndex    parachaintypes.SessionIndex
		validators      []parachaintypes.ValidatorID
		validatorGroups *parachaintypes.ValidatorGroups
		cores           []parachaintypes.CoreState
	)

	// Error channel to receive errors from goroutines.
	errCh := make(chan error, 4)

	// WaitGroup to wait for all goroutines to finish.
	var wg sync.WaitGroup

	wg.Add(4)

	go func() {
		defer wg.Done()

		var err error
		sessionIndex, err = rt.ParachainHostSessionIndexForChild()
		if err != nil {
			errCh <- fmt.Errorf("getting session index: %w", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		var err error
		validators, err = rt.ParachainHostValidators()
		if err != nil {
			errCh <- fmt.Errorf("getting validators: %w", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		var err error
		validatorGroups, err = rt.ParachainHostValidatorGroups()
		if err != nil {
			errCh <- fmt.Errorf("getting validator groups: %w", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		var err error
		cores, err = rt.ParachainHostAvailabilityCores()
		if err != nil {
			errCh <- fmt.Errorf("getting availability cores: %w", err)
			return
		}
	}()

	wg.Wait()

	if len(errCh) > 0 {
		var joinedErrors error
		for err := range errCh {
			joinedErrors = errors.Join(joinedErrors, err)
		}

		return parachaintypes.SessionIndex(0), nil, parachaintypes.ValidatorGroups{}, []parachaintypes.CoreState(nil),
			joinedErrors
	}

	return sessionIndex, validators, *validatorGroups, cores, nil
}
