// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"cmp"
	"slices"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestNewClusterTracker(t *testing.T) {
	t.Parallel()

	clusterTracker := newClusterTracker([]parachaintypes.ValidatorIndex{4, 6, 3, 9}, 3)
	require.NotNil(t, clusterTracker)
}

// tests that cover clusterTracker.canReceive() and clusterTracker.noteReceived()
func TestClusterTracker_receive_statements(t *testing.T) {
	t.Parallel()

	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(2)
	hashA := parachaintypes.CandidateHash{Value: common.Hash{0x1}}
	hashB := parachaintypes.CandidateHash{Value: common.Hash{0x2}}
	hashC := parachaintypes.CandidateHash{Value: common.Hash{0x3}}

	t.Run("rejects_incoming_outside_of_group", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		require.Equal(
			t,
			tracker.canReceive(
				parachaintypes.ValidatorIndex(100),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(parachaintypes.CandidateHash{Value: common.Hash{0x1}}),
			),
			notInGroupIncoming{},
		)

		require.Equal(
			t,
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(100),
				parachaintypes.NewCompactSeconded(parachaintypes.CandidateHash{Value: common.Hash{0x1}}),
			),
			notInGroupIncoming{},
		)
	})

	t.Run("begrudgingly_accepts_too_many_seconded_from_multiple_peers", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashB),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashB),
		)

		require.Equal(
			t,
			excessiveSecondedIncoming{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashC),
			),
		)
	})

	t.Run("rejects_too_many_seconded_from_sender", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashB),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashB),
		)

		require.Equal(
			t,
			withPrejudice{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashC),
			),
		)
	})

	t.Run("rejects_duplicates", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(200),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			duplicateIncoming{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		require.Equal(
			t,
			duplicateIncoming{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(200),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)
	})

	t.Run("rejects_incoming_valid_without_seconded", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		require.Equal(
			t,
			candidateUnknownIncoming{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactValid(hashA),
			),
		)
	})

	t.Run("accepts_incoming_valid_after_receiving_seconded", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactValid(hashA),
			),
		)
	})
}

func TestClusterTracker_noteSent(t *testing.T) {
	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(2)
	tracker := newClusterTracker(group, secondingLimit)

	secondedStmt := parachaintypes.NewCompactSeconded(
		parachaintypes.CandidateHash{Value: common.Hash{0xab}})

	// noteSent should not panic if the validator is not in the group
	tracker.noteSent(
		parachaintypes.ValidatorIndex(100),
		parachaintypes.ValidatorIndex(5),
		secondedStmt,
	)

	expectedSpecific := outgoingP2P{specific{secondedStmt, parachaintypes.ValidatorIndex(5)}}
	_, ok := tracker.knowledge[parachaintypes.ValidatorIndex(100)][expectedSpecific]
	require.True(t, ok)

	expectedGeneral := outgoingP2P{general{secondedStmt.CandidateHash()}}
	_, ok = tracker.knowledge[parachaintypes.ValidatorIndex(100)][expectedGeneral]
	require.True(t, ok)

	// since the compact statement is seconded, the originator will also be part of the knowledge
	expectedSecondedOriginator := seconded{secondedStmt.CandidateHash()}
	_, ok = tracker.knowledge[parachaintypes.ValidatorIndex(5)][expectedSecondedOriginator]
	require.True(t, ok)

	// add a pending statement that should be deleted after noteSent.
	validStmt := parachaintypes.NewCompactValid(
		parachaintypes.CandidateHash{Value: common.Hash{0xab}})
	tracker.pending[parachaintypes.ValidatorIndex(5)] = map[originatorStatementPair]struct{}{
		{
			validatorIndex: parachaintypes.ValidatorIndex(24),
			compactStmt:    validStmt,
		}: {},
	}

	tracker.noteSent(
		parachaintypes.ValidatorIndex(5),
		parachaintypes.ValidatorIndex(24),
		validStmt,
	)

	// since the compact statement is valid, we dont add the originator to knowledge
	expectedSpecific = outgoingP2P{specific{validStmt, parachaintypes.ValidatorIndex(24)}}
	_, ok = tracker.knowledge[parachaintypes.ValidatorIndex(5)][expectedSpecific]
	require.True(t, ok)

	require.Len(t, tracker.pending[parachaintypes.ValidatorIndex(5)], 0)
}

// tests that cover clusterTracker.canSend() and clusterTracker.noteSent()
func TestClusterTracker_send_statements(t *testing.T) {
	t.Parallel()

	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(2)
	hashA := parachaintypes.CandidateHash{Value: common.Hash{0x1}}
	hashB := parachaintypes.CandidateHash{Value: common.Hash{0x2}}
	hashC := parachaintypes.CandidateHash{Value: common.Hash{0x3}}

	t.Run("accepts_incoming_valid_after_outgoing_seconded", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		tracker.noteSent(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(200),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)
	})

	t.Run("cannot_send_too_many_seconded_even_to_multiple_peers", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		tracker.noteSent(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		tracker.noteSent(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashB),
		)

		require.Equal(
			t,
			excessiveSecondedOutgoing{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashC),
			),
		)

		require.Equal(
			t,
			excessiveSecondedOutgoing{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashC),
			),
		)
	})

	t.Run("cannot_send_duplicate", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		tracker.noteSent(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			knownOutgoing{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)
	})

	t.Run("cannot_send_what_was_received", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			knownOutgoing{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)
	})

	// Ensure statements received with prejudice don't prevent sending later.
	t.Run("can_send_statements_received_with_prejudice", func(t *testing.T) {
		t.Parallel()

		secondingLimit := uint(1)
		tracker := newClusterTracker(group, secondingLimit)

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			withPrejudice{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashB),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(24),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashB),
		)

		require.Equal(
			t,
			ok{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)
	})
}

func TestClusterTracker_pendingStatementsFor(t *testing.T) {
	t.Parallel()

	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(1)
	hashA := parachaintypes.CandidateHash{Value: common.Hash{0x1}}
	hashB := parachaintypes.CandidateHash{Value: common.Hash{0x2}}

	// Test that the `pending_statements` are set whenever we receive a fresh statement.
	//
	// Also test that pending statements are sorted, with `Seconded` statements in the front.
	t.Run("pending_statements_set_when_receiving_fresh_statements", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		// Receive a 'Seconded' statement for candidate A.
		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5)),
		)

		require.Equal(
			t,
			[]originatorStatementPair(nil),
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(200)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(146)),
		)

		// Receive a 'Valid' statement for candidate A.

		// First, send a `Seconded` statement for the candidate.
		require.Equal(
			t,
			ok{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(200),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		tracker.noteSent(
			parachaintypes.ValidatorIndex(24),
			parachaintypes.ValidatorIndex(200),
			parachaintypes.NewCompactSeconded(hashA),
		)

		// We have to see that the candidate is known by the sender, e.g. we sent them
		// 'Seconded' above.
		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(200),
				parachaintypes.NewCompactValid(hashA),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(24),
			parachaintypes.ValidatorIndex(200),
			parachaintypes.NewCompactValid(hashA),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(200)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(146)),
		)

		// Receive a 'Seconded' statement for candidate B.

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(5),
				parachaintypes.ValidatorIndex(146),
				parachaintypes.NewCompactSeconded(hashB),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(5),
			parachaintypes.ValidatorIndex(146),
			parachaintypes.NewCompactSeconded(hashB),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(146), parachaintypes.NewCompactSeconded(hashB)},
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(200)),
		)

		sorter := func(a, b originatorStatementPair) int { return cmp.Compare(a.validatorIndex, b.validatorIndex) }
		{
			pendingStatements := tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24))
			slices.SortFunc(pendingStatements, sorter)

			require.Equal(
				t,
				[]originatorStatementPair{
					{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
					{parachaintypes.ValidatorIndex(146), parachaintypes.NewCompactSeconded(hashB)},
				},
				pendingStatements,
			)
		}

		{
			pendingStatements := tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(146))
			slices.SortFunc(pendingStatements, sorter)

			require.Equal(
				t,
				[]originatorStatementPair{
					{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
					{parachaintypes.ValidatorIndex(146), parachaintypes.NewCompactSeconded(hashB)},
					{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashA)},
				},
				pendingStatements,
			)
		}
	})

	// Test that the `pending_statements` are updated when we send or receive statements from others
	// in the cluster.
	t.Run("pending_statements_updated_when_sending_statements", func(t *testing.T) {
		t.Parallel()

		tracker := newClusterTracker(group, secondingLimit)

		// Receive a 'Seconded' statement for candidate A.

		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(200),
				parachaintypes.ValidatorIndex(5),
				parachaintypes.NewCompactSeconded(hashA),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(200),
			parachaintypes.ValidatorIndex(5),
			parachaintypes.NewCompactSeconded(hashA),
		)

		// Pending statements should be updated.

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5)),
		)

		require.Equal(
			t,
			[]originatorStatementPair(nil),
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(200)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(146)),
		)

		// Receive a 'Valid' statement for candidate B.

		// First, send a `Seconded` statement for the candidate.
		require.Equal(
			t,
			ok{},
			tracker.canSend(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(200),
				parachaintypes.NewCompactSeconded(hashB),
			),
		)

		tracker.noteSent(
			parachaintypes.ValidatorIndex(24),
			parachaintypes.ValidatorIndex(200),
			parachaintypes.NewCompactSeconded(hashB),
		)

		// We have to see the candidate is known by the sender, e.g. we sent them 'Seconded'.
		require.Equal(
			t,
			ok{},
			tracker.canReceive(
				parachaintypes.ValidatorIndex(24),
				parachaintypes.ValidatorIndex(200),
				parachaintypes.NewCompactValid(hashB),
			),
		)

		tracker.noteReceived(
			parachaintypes.ValidatorIndex(24),
			parachaintypes.ValidatorIndex(200),
			parachaintypes.NewCompactValid(hashB),
		)

		// Pending statements should be updated.

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashB)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashB)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(200)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24)),
		)

		require.Equal(
			t,
			[]originatorStatementPair{
				{parachaintypes.ValidatorIndex(5), parachaintypes.NewCompactSeconded(hashA)},
				{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactValid(hashB)},
			},
			tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(146)),
		)
	})
}

func TestClusterTracker_noteIssued(t *testing.T) {
	t.Parallel()

	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	hashA := parachaintypes.CandidateHash{Value: common.Hash{0x1}}
	tracker := newClusterTracker(group, 2)

	// validator 24 knows the seconded statement from validator 200 about hashA
	tracker.knowledge[parachaintypes.ValidatorIndex(24)] = map[taggedKnowledge]struct{}{
		incomingP2P{
			specific{
				statement: parachaintypes.NewCompactSeconded(hashA),
				validator: parachaintypes.ValidatorIndex(200),
			},
		}: {},
	}

	tracker.noteIssued(parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactSeconded(hashA))

	require.Equal(
		t,
		[]originatorStatementPair{
			{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactSeconded(hashA)},
		},
		tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5)),
	)

	require.Equal(
		t,
		[]originatorStatementPair{
			{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactSeconded(hashA)},
		},
		tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(200)),
	)

	// pending statement for validator 24 was not added because of preexisting knowledge
	require.Equal(
		t,
		[]originatorStatementPair(nil),
		tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(24)),
	)

	require.Equal(
		t,
		[]originatorStatementPair{
			{parachaintypes.ValidatorIndex(200), parachaintypes.NewCompactSeconded(hashA)},
		},
		tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(146)),
	)
}
