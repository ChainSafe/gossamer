// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
)

func TestKnownBackedCandidate_hasReceivedManifestFrom(t *testing.T) {
	t.Parallel()

	validatorIndex := parachaintypes.ValidatorIndex(42)

	t.Run("validator_not_in_mutual_knowledge", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		require.False(t, kbc.hasReceivedManifestFrom(validatorIndex))
	})

	t.Run("validator_in_mutual_knowledge_but_remote_knowledge_is_nil", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}
		require.False(t, kbc.hasReceivedManifestFrom(validatorIndex))

	})

	t.Run("validator_in_mutual_knowledge_and_remote_knowledge_is_not_nil", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		filter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   filter,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}

		require.True(t, kbc.hasReceivedManifestFrom(validatorIndex))
	})
}

func TestKnownBackedCandidate_hasSentManifestTo(t *testing.T) {
	t.Parallel()

	validatorIndex := parachaintypes.ValidatorIndex(42)

	t.Run("validator_not_in_mutual_knowledge", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		require.False(t, kbc.hasSentManifestTo(validatorIndex))
	})

	t.Run("validator_in_mutual_knowledge_but_local_knowledge_is_nil", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}

		require.False(t, kbc.hasSentManifestTo(validatorIndex))
	})

	t.Run("validator_in_mutual_knowledge_and_local_knowledge_is_not_nil", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		filter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    filter,
			receivedKnowledge: nil,
		}

		require.True(t, kbc.hasSentManifestTo(validatorIndex))
	})
}

func TestKnownBackedCandidate_sentManifestTo(t *testing.T) {
	t.Parallel()

	kbc := &knownBackedCandidate{
		groupIndex:      parachaintypes.GroupIndex(1),
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	validatorIndex := parachaintypes.ValidatorIndex(42)
	filter, err := newStatementFilter(5, false)
	require.NoError(t, err)

	kbc.sentManifestTo(validatorIndex, *filter)

	mk, ok := kbc.mutualKnowledge[validatorIndex]
	require.True(t, ok)
	require.NotNil(t, mk.localKnowledge)
	require.NotNil(t, mk.receivedKnowledge)
	require.Nil(t, mk.remoteKnowledge)
	require.Equal(t, filter, mk.localKnowledge)
}

func TestKnownBackedCandidate_manifestReceivedFrom(t *testing.T) {
	t.Parallel()

	kbc := &knownBackedCandidate{
		groupIndex:      parachaintypes.GroupIndex(1),
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	validatorIndex := parachaintypes.ValidatorIndex(42)
	filter, err := newStatementFilter(5, false)
	require.NoError(t, err)

	kbc.manifestReceivedFrom(validatorIndex, *filter)

	mk, ok := kbc.mutualKnowledge[validatorIndex]
	require.True(t, ok)
	require.NotNil(t, mk.remoteKnowledge)
	require.Nil(t, mk.localKnowledge)
	require.Nil(t, mk.receivedKnowledge)
	require.Equal(t, filter, mk.remoteKnowledge)
}

func TestKnownBackedCandidate_directStatementSenders(t *testing.T) {
	t.Parallel()

	// Create a knownBackedCandidate with some mutual knowledge
	kbc := &knownBackedCandidate{
		groupIndex:      parachaintypes.GroupIndex(1),
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	validatorIndex1 := parachaintypes.ValidatorIndex(1)
	validatorIndex2 := parachaintypes.ValidatorIndex(2)

	// set up a validator that should be included in the result
	remoteFilter, err := newStatementFilter(5, false)
	require.NoError(t, err)
	localFilter, err := newStatementFilter(5, false)
	require.NoError(t, err)
	receivedFilter, err := newStatementFilter(5, false)
	require.NoError(t, err)

	// set statement as received
	statementIndex := uint(2)
	statementKind := seconded
	receivedFilter.set(statementIndex, statementKind)
	localFilter.set(statementIndex, statementKind)

	kbc.mutualKnowledge[validatorIndex1] = mutualKnowledge{
		remoteKnowledge:   remoteFilter,
		localKnowledge:    localFilter,
		receivedKnowledge: receivedFilter,
	}

	// validator2 should not be included (missing received knowledge)
	kbc.mutualKnowledge[validatorIndex2] = mutualKnowledge{
		remoteKnowledge:   remoteFilter,
		localKnowledge:    localFilter,
		receivedKnowledge: nil,
	}

	t.Run("group_index_matches", func(t *testing.T) {
		t.Parallel()

		senders := kbc.directStatementSenders(parachaintypes.GroupIndex(1), statementIndex, statementKind)

		require.Len(t, senders, 1)
		require.True(t, senders[validatorIndex1])
	})

	t.Run("group_index_does_not_match", func(t *testing.T) {
		t.Parallel()

		senders := kbc.directStatementSenders(parachaintypes.GroupIndex(2), statementIndex, statementKind)

		require.Empty(t, senders)
	})
}

func TestKnownBackedCandidate_directStatementRecipients(t *testing.T) {
	t.Parallel()

	// Create a knownBackedCandidate with some mutual knowledge
	kbc := &knownBackedCandidate{
		groupIndex:      parachaintypes.GroupIndex(1),
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	validatorIndex1 := parachaintypes.ValidatorIndex(1)
	validatorIndex2 := parachaintypes.ValidatorIndex(2)
	validatorIndex3 := parachaintypes.ValidatorIndex(3)

	statementIndex := uint(2)
	statementKind := seconded

	// set up validator1 with local knowledge but no remote knowledge (should be included)
	localFilter, err := newStatementFilter(5, false)
	require.NoError(t, err)
	kbc.mutualKnowledge[validatorIndex1] = mutualKnowledge{
		remoteKnowledge:   nil,
		localKnowledge:    localFilter,
		receivedKnowledge: nil,
	}

	// set up validator2 with both local and remote knowledge
	// but remote doesn't contain the statement (should be included)
	remoteFilter, err := newStatementFilter(5, false)
	require.NoError(t, err)
	kbc.mutualKnowledge[validatorIndex2] = mutualKnowledge{
		remoteKnowledge:   remoteFilter,
		localKnowledge:    localFilter,
		receivedKnowledge: nil,
	}

	remoteFilter3, err := newStatementFilter(5, false)
	require.NoError(t, err)
	remoteFilter3.set(statementIndex, statementKind)

	// set up validator3 with both local and remote knowledge,
	// and remote does contain the statement (should not be included)
	kbc.mutualKnowledge[validatorIndex3] = mutualKnowledge{
		remoteKnowledge:   remoteFilter3,
		localKnowledge:    localFilter,
		receivedKnowledge: nil,
	}

	t.Run("group_index_matches", func(t *testing.T) {
		t.Parallel()

		recipients := kbc.directStatementRecipients(parachaintypes.GroupIndex(1), statementIndex, statementKind)

		require.Len(t, recipients, 2)
		require.Contains(t, recipients, validatorIndex1)
		require.Contains(t, recipients, validatorIndex2)
		require.NotContains(t, recipients, validatorIndex3)
	})

	t.Run("group_index_does_not_match", func(t *testing.T) {
		t.Parallel()

		recipients := kbc.directStatementRecipients(parachaintypes.GroupIndex(2), statementIndex, statementKind)

		require.Empty(t, recipients)
	})
}

func TestKnownBackedCandidate_noteFreshStatement(t *testing.T) {
	t.Parallel()

	filter, err := newStatementFilter(5, false)
	require.NoError(t, err)

	kbc := &knownBackedCandidate{
		groupIndex:      parachaintypes.GroupIndex(1),
		localKnowledge:  *filter,
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	statementIndex := uint(2)
	statementKind := seconded

	// First time should be fresh
	require.True(t, kbc.noteFreshStatement(statementIndex, statementKind))

	// Second time should not be fresh
	require.False(t, kbc.noteFreshStatement(statementIndex, statementKind))

	// Check that the statement was added to local knowledge
	require.True(t, kbc.localKnowledge.contains(statementIndex, statementKind))
}

func TestKnownBackedCandidate_sentOrReceivedDirectStatement(t *testing.T) {
	t.Parallel()

	validatorIndex := parachaintypes.ValidatorIndex(1)
	statementIndex := uint(2)
	statementKind := seconded

	t.Run("validator_not_in_mutual_knowledge", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		kbc.sentOrReceivedDirectStatement(parachaintypes.ValidatorIndex(999), statementIndex, statementKind, true)

		require.Empty(t, kbc.mutualKnowledge)
	})

	t.Run("local_and_remote_knowledge_available", func(t *testing.T) {
		t.Parallel()

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)
		remoteFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)
		receivedFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		t.Run("received", func(t *testing.T) {
			t.Parallel()

			kbc := &knownBackedCandidate{
				groupIndex:      parachaintypes.GroupIndex(1),
				mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
			}

			kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
				remoteKnowledge:   remoteFilter,
				localKnowledge:    localFilter,
				receivedKnowledge: receivedFilter,
			}

			kbc.sentOrReceivedDirectStatement(validatorIndex, statementIndex, statementKind, true)

			mk := kbc.mutualKnowledge[validatorIndex]
			require.True(t, mk.localKnowledge.contains(statementIndex, statementKind))
			require.True(t, mk.remoteKnowledge.contains(statementIndex, statementKind))
			require.True(t, mk.receivedKnowledge.contains(statementIndex, statementKind))
		})

		t.Run("not_received", func(t *testing.T) {
			t.Parallel()

			kbc := &knownBackedCandidate{
				groupIndex:      parachaintypes.GroupIndex(1),
				mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
			}

			kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
				remoteKnowledge:   remoteFilter,
				localKnowledge:    localFilter,
				receivedKnowledge: receivedFilter,
			}

			statementIndex2 := uint(3)

			kbc.sentOrReceivedDirectStatement(validatorIndex, statementIndex2, statementKind, false)

			mk := kbc.mutualKnowledge[validatorIndex]
			require.True(t, mk.localKnowledge.contains(statementIndex2, statementKind))
			require.True(t, mk.remoteKnowledge.contains(statementIndex2, statementKind))
			require.False(t, mk.receivedKnowledge.contains(statementIndex2, statementKind))
		})
	})
}

func TestKnownBackedCandidate_isPendingStatement(t *testing.T) {
	t.Parallel()

	validatorIndex := parachaintypes.ValidatorIndex(1)
	statementIndex := uint(2)
	statementKind := seconded

	t.Run("validator_is_not_in_mutual_knowledge", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		require.False(t, kbc.isPendingStatement(parachaintypes.ValidatorIndex(999), statementIndex, statementKind))

	})

	t.Run("has_no_local_knowledge", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}

		require.False(t, kbc.isPendingStatement(validatorIndex, statementIndex, statementKind))
	})

	t.Run("has_no_remote_knowledge", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    localFilter,
			receivedKnowledge: nil,
		}

		require.False(t, kbc.isPendingStatement(validatorIndex, statementIndex, statementKind))

	})

	t.Run("has_local_and_remote_knowledge_and_statement_is_in_remote", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		remoteFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		remoteFilter.set(statementIndex, statementKind)

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   remoteFilter,
			localKnowledge:    localFilter,
			receivedKnowledge: nil,
		}

		require.False(t, kbc.isPendingStatement(validatorIndex, statementIndex, statementKind))
	})

	t.Run("has_local_and_remote_knowledge_and_statement_is_not_in_remote", func(t *testing.T) {
		t.Parallel()

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		remoteFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   remoteFilter,
			localKnowledge:    localFilter,
			receivedKnowledge: nil,
		}

		require.True(t, kbc.isPendingStatement(validatorIndex, statementIndex, statementKind))
	})
}

func TestKnownBackedCandidate_pendingStatements(t *testing.T) {
	t.Parallel()

	validatorIndex := parachaintypes.ValidatorIndex(1)

	t.Run("validator_is_not_in_mutual_knowledge", func(t *testing.T) {
		t.Parallel()

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		statementIndex1 := uint(1)
		statementIndex2 := uint(2)
		statementKind := seconded

		localFilter.set(statementIndex1, statementKind)
		localFilter.set(statementIndex2, statementKind)

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			localKnowledge:  *localFilter,
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		require.Nil(t, kbc.pendingStatements(parachaintypes.ValidatorIndex(999)))
	})

	t.Run("has_no_local_knowledge", func(t *testing.T) {
		t.Parallel()

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		statementIndex1 := uint(1)
		statementIndex2 := uint(2)
		statementKind := seconded

		localFilter.set(statementIndex1, statementKind)
		localFilter.set(statementIndex2, statementKind)

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			localKnowledge:  *localFilter,
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}

		require.Nil(t, kbc.pendingStatements(validatorIndex))
	})

	t.Run("has_no_remote_knowledge", func(t *testing.T) {
		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			localKnowledge:  *localFilter,
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    localFilter,
			receivedKnowledge: nil,
		}

		require.Nil(t, kbc.pendingStatements(validatorIndex))
	})

	// Test when both local and remote knowledge are available with some statements in remote
	t.Run("has_local_and_remote_knowledge_with_some_statements_in_remote", func(t *testing.T) {
		t.Parallel()

		statementIndex1 := uint(1)
		statementIndex2 := uint(2)
		statementKind := seconded

		localFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		localFilter.set(statementIndex1, statementKind)
		localFilter.set(statementIndex2, statementKind)

		kbc := &knownBackedCandidate{
			groupIndex:      parachaintypes.GroupIndex(1),
			localKnowledge:  *localFilter,
			mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
		}

		mutualLocalFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)

		remoteFilter, err := newStatementFilter(5, false)
		require.NoError(t, err)
		remoteFilter.set(statementIndex1, statementKind)

		kbc.mutualKnowledge[validatorIndex] = mutualKnowledge{
			remoteKnowledge:   remoteFilter,
			localKnowledge:    mutualLocalFilter,
			receivedKnowledge: nil,
		}

		pendingStatements := kbc.pendingStatements(validatorIndex)

		require.NotNil(t, pendingStatements)
		require.False(t, pendingStatements.contains(statementIndex1, statementKind)) // Known by remote
		require.True(t, pendingStatements.contains(statementIndex2, statementKind))  // Not known by remote
	})
}
