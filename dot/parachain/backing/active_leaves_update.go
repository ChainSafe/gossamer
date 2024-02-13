// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

func (cb *CandidateBacking) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) {
	// TODO #3503
	var leafHasProspectiveParachains bool
	var implicitViewFetchError error
	var prospectiveParachainMode parachaintypes.ProspectiveParachainsMode
	activatedLeaf := update.Activated

	if activatedLeaf != nil {
		mode, err := getProspectiveParachainsMode()
		if err != nil {
			logger.Errorf("getting prospective parachains mode: %s", err)
			return
		}

		if mode.IsEnabled {
			leafHasProspectiveParachains = true
			_, implicitViewFetchError = cb.implicitView.activeLeaf(activatedLeaf.Hash)
			if implicitViewFetchError == nil {
				prospectiveParachainMode = mode
			}
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
		return
	}

	switch {
	case leafHasProspectiveParachains == false:
		if _, ok := cb.perLeaf[activatedLeaf.Hash]; ok {
			return
		}

		cb.perLeaf[activatedLeaf.Hash] = &activeLeafState{
			prospectiveParachainsMode: prospectiveParachainMode,
			secondedAtDepth:           make(map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]),
		}

		// TODO: something returned in rust here

	case leafHasProspectiveParachains == true && implicitViewFetchError == nil:
	case leafHasProspectiveParachains == true && implicitViewFetchError != nil:
	}
}

// clean up perRelayParent according to ancestry of leaves.
//
// when prospective parachains are disabled, the implicit view is empty,
// which means we'll clean up everything that's not a leaf - the expected behavior
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

// Requests prospective parachains mode for a given relay parent based on the Runtime API version.
func getProspectiveParachainsMode() (parachaintypes.ProspectiveParachainsMode, error) {
	// TODO: implement
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/subsystem-util/src/runtime/mod.rs#L453-L456

	return parachaintypes.ProspectiveParachainsMode{}, nil
}
