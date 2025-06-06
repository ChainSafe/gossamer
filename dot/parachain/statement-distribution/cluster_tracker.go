// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

// knowledge about a candidate
type knowledge interface {
	isKnowledge()
}

// general knowledge
type general struct {
	candidateHash parachaintypes.CandidateHash
}

func (general) isKnowledge() {}

// specific knowledge of a given statement (with its originator)
type specific struct {
	statement parachaintypes.CompactStatement
	validator parachaintypes.ValidatorIndex
}

func (specific) isKnowledge() {}

// taggedKnowledge is knowledge paired with its source.
type taggedKnowledge interface {
	isTaggedKnowledge()
}

// incomingP2P is knowledge we have received from the validator on the p2p layer.
type incomingP2P struct {
	knowledge knowledge
}

func (incomingP2P) isTaggedKnowledge() {}

// outgoingP2P is knowledge we have sent to the validator on the p2p layer.
type outgoingP2P struct {
	knowledge knowledge
}

func (outgoingP2P) isTaggedKnowledge() {}

// seconded is knowledge of candidates the validator has seconded.
// This is limited only to `Seconded` statements we have accepted
// _without prejudice_.
type seconded struct {
	candidateHash parachaintypes.CandidateHash
}

func (seconded) isTaggedKnowledge() {}

// clusterTracker is a utility for keeping track of limits on direct statements within a group.
type clusterTracker struct {
	validators     []parachaintypes.ValidatorIndex
	secondingLimit uint
	knowledge      map[parachaintypes.ValidatorIndex]map[taggedKnowledge]struct{}

	// pending contains statements known locally which haven't been sent to particular validators.
	// maps target validator to (originator, statement) pairs.
	pending map[parachaintypes.ValidatorIndex]map[originatorStatementPair]struct{}
}

func newClusterTracker(
	clusterValidators []parachaintypes.ValidatorIndex,
	secondingLimit uint,
) *clusterTracker {
	return &clusterTracker{
		validators:     clusterValidators,
		secondingLimit: secondingLimit,
		knowledge:      make(map[parachaintypes.ValidatorIndex]map[taggedKnowledge]struct{}),
		pending:        make(map[parachaintypes.ValidatorIndex]map[originatorStatementPair]struct{}),
	}
}
