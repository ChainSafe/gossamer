// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tidwall/btree"
)

// / Backing votes threshold used from the host prior to runtime API version 6 and from the runtime
// / prior to v9 configuration migration.
const LEGACY_MIN_BACKING_VOTES uint32 = 2

func (cb *CandidateBacking) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	var implicitViewFetchError error
	var prospectiveParachainsMode *parachaintypes.ProspectiveParachainsMode
	activatedLeaf := update.Activated

	if activatedLeaf != nil {
		var err error
		prospectiveParachainsMode, err = getProspectiveParachainsMode(cb.BlockState, activatedLeaf.Hash)
		if err != nil {
			return fmt.Errorf("getting prospective parachains mode: %w", err)
		}

		if prospectiveParachainsMode.IsEnabled {
			_, implicitViewFetchError = cb.implicitView.activeLeaf(activatedLeaf.Hash)
		}
	}

	for _, deactivated := range update.Deactivated {
		delete(cb.perLeaf, deactivated)
		cb.implicitView.deactivateLeaf(deactivated)
	}

	// we do this so we can clean up candidates right after as a result.
	cb.cleanUpPerRelayParentByLeafAncestry()

	cb.removeUnknownRelayParentsFromPerCandidate()

	if activatedLeaf == nil {
		return nil
	}

	var freshRelayParents []common.Hash

	switch prospectiveParachainsMode.IsEnabled {
	case false:
		if _, ok := cb.perLeaf[activatedLeaf.Hash]; ok {
			return nil
		}

		cb.perLeaf[activatedLeaf.Hash] = &activeLeafState{
			prospectiveParachainsMode: *prospectiveParachainsMode,
			secondedAtDepth:           make(map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]),
		}

		freshRelayParents = []common.Hash{activatedLeaf.Hash}
	case true:
		if implicitViewFetchError != nil {
			return fmt.Errorf("failed to load implicit view for leaf %s: %w", activatedLeaf.Hash, implicitViewFetchError)
		}

		freshRelayParents = cb.implicitView.knownAllowedRelayParentsUnder(activatedLeaf.Hash, nil)

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

// clean up perRelayParent according to ancestry of leaves.
//
// when prospective parachains are disabled, the implicit view is empty,
// which means we'll clean up everything that's not a leaf - the expected behaviour
// for pre-asynchronous backing.
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

// clean up `per_candidate` according to which relay-parents are known.
//
// when prospective parachains are disabled, we clean up all candidates
// because we've cleaned up all relay parents. this is correct.
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
	// TODO: implement this

	rt, err := blockstate.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	sessionIndex, validators, validatorGroups, cores, err := fetchParachainHostData(rt)
	if err != nil {
		return nil, fmt.Errorf("fetching parachain host data: %w", err)
	}

	// TODO: call minBackingVotes function here ParachainHostMinimumBackingVotes test passed
	minBackingVotes := LEGACY_MIN_BACKING_VOTES

	signingContext := parachaintypes.SigningContext{
		SessionIndex: *sessionIndex,
		ParentHash:   relayParent,
	}

	return nil, nil
}

// From the given set of validators, find the first key we can sign with,
// if any, and return it along with the validator index.
func signingKeyAndIndex(
	validators []parachaintypes.ValidatorID,
	keystore keystore.Keystore,
) (parachaintypes.ValidatorID, parachaintypes.ValidatorID, error) {
	for i, v := range validators {
		key := keystore.
	}
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
		case sessionIndex = <-paraHostSessionIndexForChind(ctx, cancel, rt, errCh):
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case validators = <-paraHostValidators(ctx, cancel, rt, errCh):
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case validatorGroups = <-paraHostValidatorGroups(ctx, cancel, rt, errCh):
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case cores = <-ParaHostAvailabilityCores(ctx, cancel, rt, errCh):
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

func paraHostSessionIndexForChind(ctx context.Context, cancel context.CancelFunc, rt runtime.Instance, errCh chan error) chan parachaintypes.SessionIndex {
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

func paraHostValidators(ctx context.Context, cancel context.CancelFunc, rt runtime.Instance, errCh chan error) chan []parachaintypes.ValidatorID {
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

func paraHostValidatorGroups(ctx context.Context, cancel context.CancelFunc, rt runtime.Instance, errCh chan error) chan *parachaintypes.ValidatorGroups {
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

func ParaHostAvailabilityCores(ctx context.Context, cancel context.CancelFunc, rt runtime.Instance, errCh chan error) chan *scale.VaryingDataTypeSlice {
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
