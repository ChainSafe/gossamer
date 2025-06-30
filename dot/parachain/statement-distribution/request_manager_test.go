// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestRequestManager(t *testing.T) {
	parentA := common.Hash{1}
	parentB := common.Hash{2}
	parentC := common.Hash{3}

	candidateA1 := parachaintypes.CandidateHash{Value: common.Hash{11}}
	candidateA2 := parachaintypes.CandidateHash{Value: common.Hash{12}}
	candidateB1 := parachaintypes.CandidateHash{Value: common.Hash{21}}
	candidateB2 := parachaintypes.CandidateHash{Value: common.Hash{22}}
	candidateC1 := parachaintypes.CandidateHash{Value: common.Hash{31}}
	duplicateHash := parachaintypes.CandidateHash{Value: common.Hash{31}}

	t.Run("removeByRelayParent", func(t *testing.T) {
		manager := newRequestManager()
		manager.getOrInsert(parentA, candidateA1, 1)
		manager.getOrInsert(parentA, candidateA2, 1)
		manager.getOrInsert(parentB, candidateB1, 1)
		manager.getOrInsert(parentB, candidateB2, 2)
		manager.getOrInsert(parentC, candidateC1, 2)
		manager.getOrInsert(parentA, duplicateHash, 1)

		require.Len(t, manager.requests, 6)
		require.Len(t, manager.byPriority, 6)
		require.Len(t, manager.uniqueIdentifiers, 5)

		manager.removeByRelayParent(parentA)

		require.Len(t, manager.requests, 3)
		require.Len(t, manager.byPriority, 3)
		require.Len(t, manager.uniqueIdentifiers, 3)

		_, ok := manager.uniqueIdentifiers[candidateA1]
		require.False(t, ok)

		_, ok = manager.uniqueIdentifiers[candidateA2]
		require.False(t, ok)

		// Duplicate hash should still be there (under a different parent).
		_, ok = manager.uniqueIdentifiers[duplicateHash]
		require.True(t, ok)

		manager.removeByRelayParent(parentB)

		require.Len(t, manager.requests, 1)
		require.Len(t, manager.byPriority, 1)
		require.Len(t, manager.uniqueIdentifiers, 1)

		_, ok = manager.uniqueIdentifiers[candidateB1]
		require.False(t, ok)

		_, ok = manager.uniqueIdentifiers[candidateB2]
		require.False(t, ok)

		manager.removeByRelayParent(parentC)

		require.Empty(t, manager.requests)
		require.Empty(t, manager.byPriority)
		require.Empty(t, manager.uniqueIdentifiers)
	})

	t.Run("test_priority_ordering", func(t *testing.T) {
		manager := newRequestManager()

		// Add some entries, set a couple of them to cluster (high) priority.
		identifierA1 := manager.getOrInsert(parentA, candidateA1, 1).identifier

		entry := manager.getOrInsert(parentA, candidateA2, 1)
		entry.setClusterPriority()
		identifierA2 := entry.identifier

		identifierB1 := manager.getOrInsert(parentB, candidateB1, 1).identifier

		identifierB2 := manager.getOrInsert(parentB, candidateB2, 2).identifier

		entry = manager.getOrInsert(parentC, candidateC1, 2)
		entry.setClusterPriority()
		identifierC1 := entry.identifier

		require.Equal(
			t,
			manager.byPriority,
			priorityCandidatePairs{
				{
					priority:            priority{origin: cluster, attempts: 0},
					candidateIdentifier: identifierA2,
				},
				{
					priority:            priority{origin: cluster, attempts: 0},
					candidateIdentifier: identifierC1,
				},
				{
					priority:            priority{origin: unspecified, attempts: 0},
					candidateIdentifier: identifierA1,
				},
				{
					priority:            priority{origin: unspecified, attempts: 0},
					candidateIdentifier: identifierB1,
				},
				{
					priority:            priority{origin: unspecified, attempts: 0},
					candidateIdentifier: identifierB2,
				},
			})
	})
}
