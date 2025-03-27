// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// handleGetBackableCandidatesMessage returns backable candidates of multiple parachains
func (cb *CandidateBacking) handleGetBackableCandidatesMessage(
	requestedCandidates map[parachaintypes.ParaID][]*parachaintypes.CandidateHashAndRelayParent,
) map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate {
	paraIdTobackableCandidates := make(map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate)

	for paraID, listOfCandidateAndRelayParent := range requestedCandidates {
		backableCandidates := cb.getBackableCandidatesOfAParachain(listOfCandidateAndRelayParent)
		if len(backableCandidates) > 0 {
			paraIdTobackableCandidates[paraID] = backableCandidates
		}
	}

	return paraIdTobackableCandidates
}

// getBackableCandidatesOfAParachain returns backable candidates of a parachain
func (cb *CandidateBacking) getBackableCandidatesOfAParachain(
	candidateAndRelayParentPairs []*parachaintypes.CandidateHashAndRelayParent,
) []*parachaintypes.BackedCandidate {
	backableCandidates := make([]*parachaintypes.BackedCandidate, 0, len(candidateAndRelayParentPairs))

	for _, candidateAndRelayParent := range candidateAndRelayParentPairs {
		rpState, ok := cb.perRelayParent[candidateAndRelayParent.CandidateRelayParent]
		if !ok {
			logger.Debug("requested candidate's relay parent is out of view")
			break
		}

		if rpState == nil {
			logger.Debug(errNilRelayParentState.Error())
			break
		}

		attested, err := rpState.table.attestedCandidate(
			candidateAndRelayParent.CandidateHash, &rpState.tableContext, rpState.minBackingVotes)
		if err != nil {
			logger.Debugf("getting attested candidate: %w", err)
			break
		}

		if attested == nil {
			logger.Debug("requested candidate is not attested")
			break
		}

		backed, err := attested.toBackedCandidate(&rpState.tableContext, rpState.injectCoreIndex)
		if err != nil {
			logger.Debugf("converting attested candidate to backed candidate: %w", err)
			break
		}

		backableCandidates = append(backableCandidates, backed)
	}
	return backableCandidates
}
