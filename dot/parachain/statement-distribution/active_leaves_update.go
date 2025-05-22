package statementdistribution

import (
	"fmt"
	"maps"
	"slices"

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
	rt, err := s.blockState.GetRuntime(rp)
	if err != nil {
		return fmt.Errorf("getting runtime: %w", err)
	}

	disabledValidators, err := rt.ParachainHostDisabledValidators()
	if err != nil {
		return fmt.Errorf("querying disabled validators: %w", err)
	}

	disableValidatorsSet := make(map[parachaintypes.ValidatorIndex]struct{}, len(disabledValidators))
	for _, dv := range disabledValidators {
		disableValidatorsSet[dv] = struct{}{}
	}

	sessionIdx, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return fmt.Errorf("querying session index for child: %w", err)
	}

	if _, ok := s.state.perSession[sessionIdx]; !ok {
		sessionInfo, err := rt.ParachainHostSessionInfo(sessionIdx)
		if err != nil {
			return fmt.Errorf("querying session info: %w", err)
		}

		if sessionInfo == nil {
			logger.Warnf("no session info provided for session %d, relay parent=%s", sessionIdx, rp)
			return nil
		}

		minBackingVotes, err := rt.ParachainHostMinimumBackingVotes()
		if err != nil {
			return fmt.Errorf("querying minimum backing votes: %w", err)
		}

		nodeFeatures, err := rt.ParachainHostNodeFeatures()
		if err != nil {
			return fmt.Errorf("querying node features: %w", err)
		}

		allowV2Descriptor, err := nodeFeatures.Get(uint(parachaintypes.CandidateReceiptV2Feature))
		if err != nil {
			return fmt.Errorf("getting candidate receipt v2 feature: %w", err)
		}

		perSessionState := newPerSessionState(
			sessionInfo, s.state.keystore, minBackingVotes, allowV2Descriptor)

		if top, ok := s.state.unusedTopologies[sessionIdx]; ok {
			delete(s.state.unusedTopologies, sessionIdx)
			perSessionState.supplyTopology(top.Topology, top.LocalIndex)
		}

		s.state.perSession[sessionIdx] = perSessionState
	}

	perSession := s.state.perSession[sessionIdx]
	if perSession == nil {
		panic("either existed or just inserted; qed.")
	}

	if len(disableValidatorsSet) > 0 {
		disabled := make([]parachaintypes.ValidatorIndex, len(disableValidatorsSet))
		for d := range disableValidatorsSet {
			disabled = append(disabled, d)
		}

		logger.Debugf("disabled validators detected: %v, "+
			"session index=%v, relay parent=%s", disabled, sessionIdx, rp.String())
	}

	validatorGroups, err := rt.ParachainHostValidatorGroups()
	if err != nil {
		return fmt.Errorf("querying validator groups: %w", err)
	}

	claimQueue, err := rt.ParachainHostClaimQueue()
	if err != nil {
		return fmt.Errorf("querying host claim queue")
	}

	groupsPerPara, assignmentsPerGroup := determineGroupAssignment(
		len(perSession.groups.all()),
		&validatorGroups.GroupRotationInfo,
		&claimQueue,
	)

	transposedCq := claimQueue.ToTransposed()

	s.state.perRelayParent[rp] = &perRelayParentState{
		localValidator:       nil, //todo
		statementStore:       nil, // todo
		session:              sessionIdx,
		groupsPerPara:        groupsPerPara,
		disabledValidators:   disableValidatorsSet,
		transposedClaimQueue: transposedCq,
		assignmentsPerGroup:  assignmentsPerGroup,
	}

	return nil
}

// Utility function to populate:
// - per relay parent `ParaId` to `GroupIndex` mappings.
// - per `GroupIndex` claim queue assignments
func determineGroupAssignment(numCores int,
	groupRotationInfo *parachaintypes.GroupRotationInfo,
	claimQueue *parachaintypes.ClaimQueue,
) (map[parachaintypes.ParaID][]parachaintypes.GroupIndex, map[parachaintypes.GroupIndex][]parachaintypes.ParaID) {
	// Determine the core indices occupied by each para at the current relay parent. To support
	// on-demand parachains we also consider the core indices at next blocks.
	ordered := claimQueue.Ordered()

	schedule := make(map[parachaintypes.CoreIndex][]parachaintypes.ParaID)
	for _, cqEntry := range ordered {
		schedule[cqEntry.Core] = cqEntry.Paras
	}

	groupsPerPara := make(map[parachaintypes.ParaID][]parachaintypes.GroupIndex)
	assignmentsPerGroup := make(map[parachaintypes.GroupIndex][]parachaintypes.ParaID, len(schedule))

	for coreIdx, paras := range schedule {
		groupIdx := groupRotationInfo.GroupForCore(coreIdx, uint(numCores))
		assignmentsPerGroup[groupIdx] = slices.Clone(paras)

		for _, para := range paras {
			groups, ok := groupsPerPara[para]
			if !ok {
				groups = make([]parachaintypes.GroupIndex, 0)
				groupsPerPara[para] = groups
			}

			groups = append(groups, groupIdx)
		}
	}

	return groupsPerPara, assignmentsPerGroup
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

	maps.DeleteFunc(s.state.perSession, func(key parachaintypes.SessionIndex, _value *perSessionState) bool {
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
