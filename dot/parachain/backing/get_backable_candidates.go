// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// handleGetBackableCandidatesMessage send back the backable candidates via the response channel
func (cb *CandidateBacking) handleGetBackableCandidatesMessage(requestedCandidates GetBackableCandidatesMessage) {
	backedCandidates := make([]*parachaintypes.BackedCandidate, 0, len(requestedCandidates.Candidates))

	for _, candidate := range requestedCandidates.Candidates {
		rpState, ok := cb.perRelayParent[candidate.CandidateRelayParent]
		if !ok {
			logger.Debug("requested candidate's relay parent is out of view")
			continue
		}

		if rpState == nil {
			logger.Debug(errNilRelayParentState.Error())
			continue
		}

		attested, err := rpState.table.attestedCandidate(
			candidate.CandidateHash, &rpState.tableContext, rpState.minBackingVotes)
		if err != nil {
			logger.Debugf("getting attested candidate: %w", err)
			continue
		}

		if attested == nil {
			logger.Debug("requested candidate is not attested")
			continue
		}

		backed, err := attested.toBackedCandidate(&rpState.tableContext)
		if err != nil {
			logger.Debugf("converting attested candidate to backed candidate: %w", err)
			continue
		}
		backedCandidates = append(backedCandidates, backed)
	}

	requestedCandidates.ResCh <- backedCandidates
}
