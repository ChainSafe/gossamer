package statementdistribution

import (
	"fmt"
	"maps"

	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	"github.com/ChainSafe/gossamer/lib/common"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

func (s *StatementDistribution) handleActiveLeavesUpdate(leaf *parachaintypes.ActivatedLeaf) error {
	err := s.state.implicitView.ActivateLeaf(leaf.Hash, s.SubSystemToOverseer)
	if err != nil {
		return fmt.Errorf("implicit view activating leaf: %w", err)
	}

	newRelayParents := s.state.implicitView.AllAllowedRelayParents()

	for _, nrp := range newRelayParents {
		if _, ok := s.state.perRelayParent[nrp]; ok {
			continue
		}

		if err := s.handleActiveLeafUpdate(nrp); err != nil {
			logger.Warnf("failed to handle active leaf %s: %s", nrp.String(), err.Error())
		}
	}

	logger.Debugf("activated leaves. Now tracking %d relay-parent across %d sessions",
		len(s.state.perRelayParent), len(s.state.perSession))

	// Reconcile all peers' views with the active leaf and any relay parents
	// it implies. If they learned about the block before we did, this reconciliation will give
	// non-empty results and we should send them messages concerning all activated relay-parents.
	updatePeers := make(map[string][]common.Hash)
	for pid, pState := range s.state.peers {
		fresh := pState.reconcileActiveLeaf(leaf.Hash, newRelayParents)
		if len(fresh) > 0 {
			updatePeers[pid] = fresh
		}
	}

	for pid, fresh := range updatePeers {
		for _, freshRp := range fresh {
			s.sendPeerMessageForRelayParent(pid, freshRp)
		}
	}

	s.newLeafFragmentChainUpdates(leaf.Hash)

	return nil
}

func (s *StatementDistribution) handleActiveLeafUpdate(rp common.Hash) error {
	return nil
}

// handleDeactivatedLeaves deactivate leaves in the implicit view
func (s *StatementDistribution) handleDeactivatedLeaves(leaves []common.Hash) {
	for _, l := range leaves {
		pruned := s.state.implicitView.DeactivateLeaf(l)

		for _, prunedRp := range pruned {
			// clean up per-relay-parent data based on everything removed.
			rpInfo, ok := s.state.perRelayParent[prunedRp]
			if !ok {
				continue
			}

			delete(s.state.perRelayParent, prunedRp)

			if activeValidatorState := rpInfo.activeValidatorState(); activeValidatorState != nil {
				activeValidatorState.clusterTracker.warningIfTooManyPendingStatements(prunedRp)
			}

			// clean up requests related to this relay parent.
			s.state.requestManager.removeByRelayParent(prunedRp)
		}
	}

	s.state.candidates.onDeactivateLeaves(leaves, func(h common.Hash) bool {
		_, ok := s.state.perRelayParent[h]
		return ok
	})

	// clean up sessions based on everything remaining.
	sessions := make(map[parachaintypes.SessionIndex]struct{})
	for _, v := range s.state.perRelayParent {
		sessions[v.session] = struct{}{}
	}

	maps.DeleteFunc(s.state.perSession, func(key parachaintypes.SessionIndex, _value perSessionState) bool {
		_, ok := sessions[key]
		return !ok
	})

	var lastSessionIndex *parachaintypes.SessionIndex
	for k := range s.state.unusedTopologies {
		if lastSessionIndex == nil || k > *lastSessionIndex {
			//pin so we don't get the address of a looping variable
			sessionIdx := k
			lastSessionIndex = &sessionIdx
		}
	}

	// Do not clean-up the last saved toplogy unless we moved to the next session
	// This is needed because handle_deactive_leaves, gets also called when
	// prospective_parachains APIs are not present, so we would actually remove
	// the topology without using it because `perRelayParent` is empty until
	// prospective_parachains gets enabled
	maps.DeleteFunc(s.state.unusedTopologies, func(s parachaintypes.SessionIndex, _v events.NewGossipTopology) bool {
		_, ok := sessions[s]
		// delete if:
		// The session index does not exists in the sessions map
		// Or the session index exists BUT is not the lastSessionIndex
		return !ok || (lastSessionIndex != nil && *lastSessionIndex != s)
	})
}
