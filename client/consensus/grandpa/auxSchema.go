// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"golang.org/x/exp/constraints"
)

var (
	VERSION_KEY        = []byte("grandpa_schema_version")
	SET_STATE_KEY      = []byte("grandpa_completed_round")
	CONCLUDED_ROUNDS   = []byte("grandpa_concluded_rounds")
	AUTHORITY_SET_KEY  = []byte("grandpa_voters")
	BEST_JUSTIFICATION = []byte("grandpa_best_justification")
)

const CURRENT_VERSION = uint32(3)

type roundInfo[H comparable, N constraints.Unsigned] struct {
	roundNumber uint64
	roundState  finalityGrandpa.RoundState[H, N]
}

// V1VoterSetState The voter set state.
type V1VoterSetState[H comparable, N constraints.Unsigned] struct {
	isLive    bool // indicates live or paused state
	roundInfo roundInfo[H, N]
}

type V0VoterSetState[H comparable, N constraints.Unsigned] roundInfo[H, N]

type V0PendingChange[H comparable, N constraints.Unsigned] struct {
	nextAuthorities AuthorityList
	delay           N
	canonHeight     N
	canonHash       H
}

type V0AuthoritySet[H comparable, N constraints.Unsigned] struct {
	currentAuthorities AuthorityList
	setID              uint64
	pendingChanges     []V0PendingChange[H, N]
}

func (authSet V0AuthoritySet[H, N]) into() AuthoritySet[H, N] {
	pendingStandardChanges := NewChangeTree[H, N]()

	for _, oldChange := range authSet.pendingChanges {
		finalizedKind := Finalized{}
		newChange := PendingChange[H, N]{
			nextAuthorities: oldChange.nextAuthorities,
			delay:           oldChange.delay,
			canonHeight:     oldChange.canonHeight,
			canonHash:       oldChange.canonHash,
			delayKind:       newDelayKind(finalizedKind),
		}

		isDescOf := func(H, H) (bool, error) { return false, nil }
		_, err := pendingStandardChanges.Import(newChange.canonHash, newChange.canonHeight, newChange, isDescOf)
		if err != nil {
			logger.Warnf("migrating pending authority set change: %s", err)
			logger.Warn("node is in a potentially inconsistent state")
		}
	}

	authoritySet := NewAuthoritySet[H, N](authSet.currentAuthorities, authSet.setID, pendingStandardChanges, []PendingChange[H, N]{}, nil)
	if authoritySet == nil {
		panic("current_authorities is non-empty and weights are non-zero; qed.")
	}
	return *authoritySet
}
