// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

func (cb *CandidateBacking) handleGetBackedCandidatesMessage(requestedCandidates GetBackedCandidatesMessage) {
	var backedCandidates []*parachaintypes.BackedCandidate

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

		attested, err := rpState.table.attestedCandidate(candidate.CandidateHash, &rpState.tableContext)
		if err != nil {
			logger.Debugf("getting attested candidate: %w", err)
			continue
		}

		if attested == nil {
			logger.Debug("requested candidate is not attested")
			continue
		}

		backed := attestedToBackedCandidate(*attested, &rpState.tableContext)
		backedCandidates = append(backedCandidates, backed)
	}

	requestedCandidates.ResCh <- backedCandidates
}
