// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// DisputesCoordinatorMessage Messages sent to the Prospective Parachains subsystem.
type DisputesCoordinatorMessage interface {
	isDisputesCoordinatorMessage()
}

type CandidateVotesResponse struct {
	SessionIndex   parachaintypes.SessionIndex
	CandidateHash  parachaintypes.CandidateHash
	CandidateVotes parachaintypes.CandidateVotes
}

// / Get candidate votes for a candidate.
type QueryCandidateVotes struct {
	Query    []parachaintypes.DisputeKey
	Response chan []CandidateVotesResponse
}

type RecentDispute struct {
	SessionIndex  parachaintypes.SessionIndex
	CandidateHash parachaintypes.CandidateHash
	DisputeStatus parachaintypes.DisputeStatus
}

type GetRecentDisputes struct {
	Response chan []RecentDispute
}
