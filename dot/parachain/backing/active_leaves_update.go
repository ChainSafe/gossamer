// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tidwall/btree"
)

// Backing votes threshold used from the host prior to runtime API version 6 and
// from the runtime prior to v9 configuration migration.
const LEGACY_MIN_BACKING_VOTES uint32 = 2

func (cb *CandidateBacking) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	var implicitViewFetchError error
	var prospectiveParachainsMode *parachaintypes.ProspectiveParachainsMode
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
	cleanUpPerRelayParentByLeafAncestry(cb)

	// clean up `perCandidate` according to which relay-parents are known.
	//
	// when prospective parachains are disabled, we clean up all candidates
	// because we've cleaned up all relay parents. this is correct.
	removeUnknownRelayParentsFromPerCandidate(cb)

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
			prospectiveParachainsMode: *prospectiveParachainsMode,

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

		secondedAtDepth := processRemainingSeconded(cb, remainingSeconded, activatedLeaf.Hash)

		cb.perLeaf[activatedLeaf.Hash] = &activeLeafState{
			prospectiveParachainsMode: *prospectiveParachainsMode,
			secondedAtDepth:           secondedAtDepth,
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
			mode = *prospectiveParachainsMode
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

func processRemainingSeconded(
	cb *CandidateBacking,
	remainingSeconded map[parachaintypes.CandidateHash]parachaintypes.ParaID,
	leafHash common.Hash,
) map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash] {
	var wg sync.WaitGroup
	var mut sync.Mutex
	secondedAtDepth := make(map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash])

	for candidateHash, parID := range remainingSeconded {
		wg.Add(1)
		go func(candidateHash parachaintypes.CandidateHash, parID parachaintypes.ParaID) {
			defer wg.Done()

			getTreeMembership := parachaintypes.ProspectiveParachainsMessageGetTreeMembership{
				ParaID:        parID,
				CandidateHash: candidateHash,
				ResponseCh:    make(chan []parachaintypes.FragmentTreeMembership),
			}

			cb.SubSystemToOverseer <- getTreeMembership
			membership := <-getTreeMembership.ResponseCh

			for _, m := range membership {
				if m.RelayParent == leafHash {
					mut.Lock()

					tree, ok := secondedAtDepth[parID]
					if !ok {
						tree = new(btree.Map[uint, parachaintypes.CandidateHash])
					}

					for _, depth := range m.Depths {
						tree.Load(depth, candidateHash)
					}
					mut.Unlock()
				}
			}
		}(candidateHash, parID)
	}
	wg.Wait()
	return secondedAtDepth
}

func cleanUpPerRelayParentByLeafAncestry(cb *CandidateBacking) {
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

func removeUnknownRelayParentsFromPerCandidate(cb *CandidateBacking) {
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
func getProspectiveParachainsMode(blockstate *state.BlockState, relayParent common.Hash,
) (*parachaintypes.ProspectiveParachainsMode, error) {
	rt, err := blockstate.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	params, err := rt.ParachainHostAsyncBackingParams()
	if err != nil {
		if errors.Is(err, wazero_runtime.ErrExportFunctionNotFound) {
			logger.Tracef(
				"%s is not supported by the current Runtime API of the relay parent %s",
				runtime.ParachainHostAsyncBackingParams, relayParent,
			)

			return &parachaintypes.ProspectiveParachainsMode{IsEnabled: false}, nil
		}
		return nil, fmt.Errorf("getting async backing params: %w", err)
	}

	enabled := parachaintypes.ProspectiveParachainsMode{
		IsEnabled:          true,
		MaxCandidateDepth:  uint(params.MaxCandidateDepth),
		AllowedAncestryLen: uint(params.AllowedAncestryLen),
	}

	return &enabled, nil
}

// Load the data necessary to do backing work on top of a relay-parent.
func constructPerRelayParentState(
	blockstate *state.BlockState,
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

	// TODO: call minBackingVotes function here once ParachainHostMinimumBackingVotes test passed
	minBackingVotes := LEGACY_MIN_BACKING_VOTES

	signingContext := parachaintypes.SigningContext{
		SessionIndex: *sessionIndex,
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

	var assignment parachaintypes.ParaID // should be pointer?

	groups := make(map[parachaintypes.ParaID][]parachaintypes.ValidatorIndex)

	numOfCores := uint32(len(cores.Types))

	for idx := uint32(0); idx < numOfCores; idx++ {
		coreValue, err := cores.Types[idx].Value()
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
			coreParaID = parachaintypes.ParaID(v.ParaID)
		case parachaintypes.Free:
			continue
		}

		coreIndex := parachaintypes.CoreIndex{Index: idx}
		groupIndex := validatorGroups.GroupRotationInfo.GroupForCore(coreIndex, numOfCores)
		validatorIndexes := validatorGroups.Validators[groupIndex]

		if validatorIndexes != nil {
			isIndexPresent := slices.Contains(validatorIndexes, localValidator.index)

			if localValidator != nil && isIndexPresent {
				assignment = coreParaID
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

/*
TODO: use this function once a PR to get the minBackingVotes is merged

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
*/

func fetchParachainHostData(rt runtime.Instance) ( //nolint:unused
	*parachaintypes.SessionIndex,
	[]parachaintypes.ValidatorID,
	*parachaintypes.ValidatorGroups,
	*scale.VaryingDataTypeSlice,
	error,
) {
	var (
		sessionIndex    parachaintypes.SessionIndex
		validators      []parachaintypes.ValidatorID
		validatorGroups *parachaintypes.ValidatorGroups
		cores           *scale.VaryingDataTypeSlice
	)

	// Create a context with cancellation capability.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation happens when function returns.

	// WaitGroup to wait for all goroutines to finish.
	var wg sync.WaitGroup

	// Error channel to receive errors from goroutines.
	errCh := make(chan error)

	// Start each goroutine with a separate function and wait for all of them to finish.
	wg.Add(4)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done(): // Check if context was canceled.
			return
		case sessionIndex = <-paraHostSessionIndexForChind(cancel, rt, errCh):
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case validators = <-paraHostValidators(cancel, rt, errCh):
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case validatorGroups = <-paraHostValidatorGroups(cancel, rt, errCh):
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case cores = <-paraHostAvailabilityCores(cancel, rt, errCh):
		}
	}()

	// Wait for all goroutines to finish.
	wg.Wait()

	// If any goroutine encountered an error, return the error.
	select {
	case err := <-errCh:
		return nil, nil, nil, nil, err
	default:
		return &sessionIndex, validators, validatorGroups, cores, nil
	}
}

func paraHostSessionIndexForChind( //nolint:unused
	cancel context.CancelFunc,
	rt runtime.Instance,
	errCh chan error,
) chan parachaintypes.SessionIndex {
	sessionIndexCh := make(chan parachaintypes.SessionIndex)

	go func() {
		sessionIndex, err := rt.ParachainHostSessionIndexForChild()
		if err != nil {
			errCh <- fmt.Errorf("getting session index: %w", err)
			cancel() // Cancel context to signal other goroutines to stop.
			return
		}
		sessionIndexCh <- sessionIndex
	}()
	return sessionIndexCh
}

func paraHostValidators( //nolint:unused
	cancel context.CancelFunc,
	rt runtime.Instance,
	errCh chan error,
) chan []parachaintypes.ValidatorID {
	validatorsCh := make(chan []parachaintypes.ValidatorID)

	go func() {
		validators, err := rt.ParachainHostValidators()
		if err != nil {
			errCh <- fmt.Errorf("getting validators: %w", err)
			cancel()
			return
		}
		validatorsCh <- validators
	}()
	return validatorsCh
}

func paraHostValidatorGroups( //nolint:unused
	cancel context.CancelFunc,
	rt runtime.Instance,
	errCh chan error,
) chan *parachaintypes.ValidatorGroups {
	validatorGroupsCh := make(chan *parachaintypes.ValidatorGroups)

	go func() {
		validatorGroups, err := rt.ParachainHostValidatorGroups()
		if err != nil {
			errCh <- fmt.Errorf("getting validator groups: %w", err)
			cancel()
			return
		}
		validatorGroupsCh <- validatorGroups
	}()
	return validatorGroupsCh
}

func paraHostAvailabilityCores( //nolint:unused
	cancel context.CancelFunc,
	rt runtime.Instance,
	errCh chan error,
) chan *scale.VaryingDataTypeSlice {
	coresCh := make(chan *scale.VaryingDataTypeSlice)

	go func() {
		cores, err := rt.ParachainHostAvailabilityCores()
		if err != nil {
			errCh <- fmt.Errorf("getting availability cores: %w", err)
			cancel()
			return
		}
		coresCh <- cores
	}()
	return coresCh
}
