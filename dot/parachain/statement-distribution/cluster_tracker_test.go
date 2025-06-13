// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestNewClusterTracker(t *testing.T) {
	clusterTracker := newClusterTracker([]parachaintypes.ValidatorIndex{4, 6, 3, 9}, 3)
	require.NotNil(t, clusterTracker)
}

// tests that cover clusterTracker.canReceive() and clusterTracker.noteReceived()
func TestClusterTracker_receive_statements(t *testing.T) {
	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(2)
	hashA := parachaintypes.CandidateHash{Value: common.Hash{0x1}}
	hashB := parachaintypes.CandidateHash{Value: common.Hash{0x2}}
	hashC := parachaintypes.CandidateHash{Value: common.Hash{0x3}}

	t.Run("rejects_incoming_outside_of_group", func(t *testing.T) {
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

func TestClusterTracker_pendingStatementsFor(t *testing.T) {
	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(2)
	tracker := newClusterTracker(group, secondingLimit)

	tracker.pending[parachaintypes.ValidatorIndex(5)] = map[originatorStatementPair]struct{}{
		{
			validatorIndex: parachaintypes.ValidatorIndex(24),
			compactStmt:    parachaintypes.NewCompactSeconded(parachaintypes.CandidateHash{Value: common.Hash{0xab}}),
		}: {},
		{
			validatorIndex: parachaintypes.ValidatorIndex(200),
			compactStmt:    parachaintypes.NewCompactValid(parachaintypes.CandidateHash{Value: common.Hash{0xab}}),
		}: {},
		{
			validatorIndex: parachaintypes.ValidatorIndex(146),
			compactStmt:    parachaintypes.NewCompactValid(parachaintypes.CandidateHash{Value: common.Hash{0xab}}),
		}: {},
		{
			validatorIndex: parachaintypes.ValidatorIndex(200),
			compactStmt:    parachaintypes.NewCompactSeconded(parachaintypes.CandidateHash{Value: common.Hash{0x1}}),
		}: {},
		{
			validatorIndex: parachaintypes.ValidatorIndex(24),
			compactStmt:    parachaintypes.NewCompactValid(parachaintypes.CandidateHash{Value: common.Hash{0x1}}),
		}: {},
		{
			validatorIndex: parachaintypes.ValidatorIndex(146),
			compactStmt:    parachaintypes.NewCompactValid(parachaintypes.CandidateHash{Value: common.Hash{0x1}}),
		}: {},
	}

	pairs := tracker.pendingStatementsFor(parachaintypes.ValidatorIndex(5))
	require.Len(t, pairs, 6)

	for i := 0; i < 2; i++ {
		_, ok := pairs[i].compactStmt.(*parachaintypes.CompactSeconded)
		require.True(t, ok)
	}

	for i := 2; i < 6; i++ {
		_, ok := pairs[i].compactStmt.(*parachaintypes.CompactValid)
		require.True(t, ok)
	}
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
