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

// tests that cover clusterTracker.canSend() and clusterTracker.noteSent()
func TestClusterTracker_send_statements(t *testing.T) {
	group := []parachaintypes.ValidatorIndex{5, 200, 24, 146}
	secondingLimit := uint(2)
	hashA := parachaintypes.CandidateHash{Value: common.Hash{0x1}}
	hashB := parachaintypes.CandidateHash{Value: common.Hash{0x2}}
	hashC := parachaintypes.CandidateHash{Value: common.Hash{0x3}}

	t.Run("accepts_incoming_valid_after_outgoing_seconded", func(t *testing.T) {
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
