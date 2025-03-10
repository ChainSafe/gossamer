// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ProcessActiveLeavesUpdateSignal updates the state of the CandidateBacking struct based on the
// provided ActiveLeavesUpdateSignal.
// It manages the activation and deactivation of relay chain block, performs cleanup operations
// on the perRelayParent and perCandidate maps, and adds entries to perRelayParent for
// new relay-parents introduced by the update.
func (cb *CandidateBacking) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	var implicitViewFetchError error
	activatedLeaf := update.Activated
	// activate in implicit view before deactivate, per the docs on ImplicitView, this is more efficient.
	if activatedLeaf != nil {
		implicitViewFetchError = cb.ImplicitView.ActivateLeaf(activatedLeaf.Hash)
	}

	for _, deactivated := range update.Deactivated {
		cb.ImplicitView.deactivateLeaf(deactivated)
	}

	// clean up `perRelayParent` according to ancestry of leaves.
	// we do this so we can clean up candidates right after as a result.
	cb.cleanUpPerRelayParentByLeafAncestry()

	// clean up `perCandidate` according to which relay-parents are known.
	cb.removeUnknownRelayParentsFromPerCandidate()

	if activatedLeaf == nil {
		return nil
	}

	if implicitViewFetchError != nil {
		return fmt.Errorf("failed to load implicit view for leaf %s: %w", activatedLeaf.Hash, implicitViewFetchError)
	}

	// Get relay parents which might be fresh but might be known already
	// that are explicit or implicit from the new active leaf.
	freshRelayParents := cb.ImplicitView.KnownAllowedRelayParentsUnder(activatedLeaf.Hash, nil)
	if len(freshRelayParents) == 0 {
		logger.Warnf("implicit view gave no relay-parents under leaf-hash %s", activatedLeaf.Hash)
		freshRelayParents = []common.Hash{activatedLeaf.Hash}
	}

	// add entries in `perRelayParent`. for all new relay-parents.
	for _, maybeNewRP := range freshRelayParents {
		if _, ok := cb.perRelayParent[maybeNewRP]; ok {
			continue
		}

		// construct a `PerRelayParent` from the runtime API and insert it.
		rpState, err := cb.constructPerRelayParentState(maybeNewRP)
		if err != nil {
			return fmt.Errorf("constructing per relay parent state for relay-parent %s: %w", maybeNewRP, err)
		}

		if rpState != nil {
			cb.perRelayParent[maybeNewRP] = rpState
		}
	}
	return nil
}

func (cb *CandidateBacking) cleanUpPerRelayParentByLeafAncestry() {
	allowedRelayParents := cb.ImplicitView.AllAllowedRelayParents()

	uniqueAllowedRelayParents := make(map[common.Hash]struct{})
	for _, relayParent := range allowedRelayParents {
		uniqueAllowedRelayParents[relayParent] = struct{}{}
	}

	keysToDelete := []common.Hash{}
	for rp := range cb.perRelayParent {
		if _, ok := uniqueAllowedRelayParents[rp]; !ok {
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
