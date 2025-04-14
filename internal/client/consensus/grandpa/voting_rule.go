// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// A channel returned by a [VotingRule] to restrict a given vote, if any restriction is necessary.
type VotingRuleResult[H, N any] <-chan *HashNumber[H, N]

// VotingRule is an interface for custom voting rules in GRANDPA.
type VotingRule[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] interface {
	// Restrict the given currentTarget vote, returning the block hash and number of the block to vote on, and nil
	// in case the vote should not be restricted. base is the block that we're basing our votes on in order to pick
	// our target (e.g. last round estimate), and bestTarget is the initial best vote target before any vote rules
	// were applied. When applying multiple VotingRule associated base and bestTarget should remain unchanged.
	//
	// The contract of this interface requires that when restricting a vote, the returned value **must** be an
	// ancestor of the given currentTarget, this also means that a variant must be maintained throughout the execution
	// of voting rules wherein currentTarget <= bestTarget.
	RestrictVote(
		backend blockchain.HeaderBackend[H, N, Header],
		base Header,
		bestTarget Header,
		currentTarget Header,
	) VotingRuleResult[H, N]
}
