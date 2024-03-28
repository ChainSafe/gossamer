// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"fmt"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/tidwall/btree"
)

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

		rpState, err := constructPerRelayParentState(maybeNewRP, &cb.keystore, mode)
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
	var isAsyncBackingSupported bool
	// `ParachainHost` is the runtime api witch has the method to get the async backing params.
	// and this api supports async backing from version 7.
	apiNameHash := common.MustBlake2b8([]byte("ParachainHost"))
	asyncBackingSupportVersion := uint32(7)

	rt, err := blockstate.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	currentVersion, err := rt.Version()
	if err != nil {
		return nil, fmt.Errorf("getting runtime version: %w", err)
	}

	// check if the current runtime api supports async backing
	for _, api := range currentVersion.APIItems {
		if api.Name == apiNameHash && api.Ver >= asyncBackingSupportVersion {
			isAsyncBackingSupported = true
			break
		}
	}

	if !isAsyncBackingSupported {
		logger.Tracef("async backing is not supported by the current Runtime API of the relay parent %s", relayParent)
		return &parachaintypes.ProspectiveParachainsMode{IsEnabled: false}, nil
	}

	params, err := rt.ParachainHostAsyncBackingParams()
	if err != nil {
		return nil, fmt.Errorf("getting async backing params: %w", err)
	}

	enabled := parachaintypes.ProspectiveParachainsMode{
		IsEnabled:          true,
		MaxCandidateDepth:  uint(params.MaxCandidateDepth),
		AllowedAncestryLen: uint(params.AllowedAncestryLen),
	}

	return &enabled, nil
}

func constructPerRelayParentState(
	relayParent common.Hash,
	keystore *keystore.Keystore,
	mode parachaintypes.ProspectiveParachainsMode,
) (*perRelayParentState, error) {
	// TODO: implement this
	return nil, nil
}
