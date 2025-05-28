// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestGridTracker(t *testing.T) {
	t.Parallel()

	groups := dummyGroups(t, 3)
	groupIndex := parachaintypes.GroupIndex(0)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x42}}

	groupSize, threshold := groups.getSizeAndBackingThreshold(groupIndex)
	require.NotNil(t, groupSize)
	require.Equal(t, uint32(3), *groupSize)
	require.NotNil(t, threshold)
	require.Equal(t, uint32(2), *threshold)

	emptySessionTopology := sessionTopologyView{
		groupViews: map[parachaintypes.GroupIndex]groupSubView{
			groupIndex: {
				sending:   make(map[parachaintypes.ValidatorIndex]struct{}),
				receiving: map[parachaintypes.ValidatorIndex]struct{}{},
			},
		},
	}

	t.Run("reject_disallowed_manifest", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[0] = struct{}{}

		// Known group, disallowed receiving validator.

		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			parachaintypes.ValidatorIndex(1),
		)

		require.ErrorIs(t, err, errManifestImportDisallowed)
		require.False(t, ack)

		// Unknown group

		ack, err = tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: 1,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			parachaintypes.ValidatorIndex(0),
		)

		require.ErrorIs(t, err, errManifestImportDisallowed)
		require.False(t, ack)
	})

	t.Run("reject_malformed_wrong_group_size", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[0] = struct{}{}

		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false, true),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			parachaintypes.ValidatorIndex(0),
		)

		require.ErrorIs(t, err, errManifestImportMalformed)
		require.False(t, ack)
	})

	t.Run("reject_malformed_no_seconders", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[0] = struct{}{}

		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, false, false),
					validatedInGroup: newBitVec(t, true, true, true),
				},
			},
			full,
			parachaintypes.ValidatorIndex(0),
		)

		require.ErrorIs(t, err, errManifestImportMalformed)
		require.False(t, ack)
	})

	t.Run("reject_insufficient_below_threshold", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[0] = struct{}{}

		// only one vote

		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, false, true),
					validatedInGroup: newBitVec(t, false, false, false),
				},
			},
			full,
			parachaintypes.ValidatorIndex(0),
		)

		require.ErrorIs(t, err, errManifestImportInsufficient)
		require.False(t, ack)

		// seconding + validating still not enough to reach '2' threshold

		ack, err = tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, false, true),
					validatedInGroup: newBitVec(t, false, false, true),
				},
			},
			full,
			parachaintypes.ValidatorIndex(0),
		)

		require.ErrorIs(t, err, errManifestImportInsufficient)
		require.False(t, ack)

		// finally good.

		ack, err = tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, false, true),
					validatedInGroup: newBitVec(t, false, true, false),
				},
			},
			full,
			parachaintypes.ValidatorIndex(0),
		)

		require.NoError(t, err)
		require.False(t, ack)
	})

	// Test that when we add a candidate as backed and advertise it to the sending group, they can
	// provide an acknowledgement manifest in response.
	t.Run("senders_can_provide_manifests_in_acknowledgement", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].sending[validatorIndex] = struct{}{}
		sessionTopology.groupViews[groupIndex].receiving[1] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Add the candidate as backed.
		receivers := tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Validator 0 is in the sending group. Advertise onward to it.
		//
		// Validator 1 is in the receiving group, but we have not received from it, so we're not
		// expected to send it an acknowledgement.
		require.Len(t, receivers, 1)
		require.Equal(t, validatorIndex, receivers[0].validator)
		require.Equal(t, full, receivers[0].kind)

		// Note the manifest as 'sent' to validator 0.
		tracker.manifestSentTo(*groups, validatorIndex, candidateHash, *localKnowledge)

		// Import manifest of kind `Acknowledgement` from validator 0.
		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			acknowledgement,
			validatorIndex,
		)

		require.NoError(t, err)
		require.False(t, ack)
	})

	// Check that pending communication is set correctly when receiving a manifest on a confirmed
	// candidate.
	//
	// It should also overwrite any existing `Full` ManifestKind.
	t.Run("pending_communication_receiving_manifest_on_confirmed_candidate", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].sending[validatorIndex] = struct{}{}
		sessionTopology.groupViews[groupIndex].receiving[1] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Manifest should not be pending yet.
		pendingManifest := tracker.isManifestPendingFor(validatorIndex, candidateHash)
		require.Nil(t, pendingManifest)

		// Add the candidate as backed.
		_ = tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Manifest should be pending as `Full`.
		pendingManifest = tracker.isManifestPendingFor(validatorIndex, candidateHash)
		require.NotNil(t, pendingManifest)
		require.Equal(t, full, *pendingManifest)

		// Note the manifest as 'sent' to validator 0.
		tracker.manifestSentTo(*groups, validatorIndex, candidateHash, *localKnowledge)

		// Import manifest.
		//
		// Should overwrite existing `Full` manifest.
		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			acknowledgement,
			validatorIndex,
		)

		require.NoError(t, err)
		require.False(t, ack)

		pendingManifest = tracker.isManifestPendingFor(validatorIndex, candidateHash)
		require.Nil(t, pendingManifest)
	})

	// Check that pending communication is cleared correctly in `manifest_sent_to`
	//
	// Also test a scenario where manifest import returns `Ok(true)` (should acknowledge).
	t.Run("pending_communication_is_cleared", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[validatorIndex] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Add the candidate as backed.
		_ = tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Manifest should not be pending yet.
		pendingManifest := tracker.isManifestPendingFor(validatorIndex, candidateHash)
		require.Nil(t, pendingManifest)

		// Import manifest. The candidate is confirmed backed and we are expected to receive from
		// validator 0, so send it an acknowledgement.
		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			validatorIndex,
		)

		require.NoError(t, err)
		require.True(t, ack)

		// Acknowledgement manifest should be pending.
		pendingManifest = tracker.isManifestPendingFor(validatorIndex, candidateHash)
		require.NotNil(t, pendingManifest)
		require.Equal(t, acknowledgement, *pendingManifest)

		// Note the candidate as advertised.
		tracker.manifestSentTo(*groups, validatorIndex, candidateHash, *localKnowledge)

		// Pending manifest should be cleared.
		pendingManifest = tracker.isManifestPendingFor(validatorIndex, candidateHash)
		require.Nil(t, pendingManifest)
	})

	/// A manifest exchange means that both `manifest_sent_to` and `manifest_received_from` have
	/// been invoked.
	///
	/// In practice, it means that one of three things have happened:
	///
	/// - They announced, we acknowledged
	///
	/// - We announced, they acknowledged
	///
	/// - We announced, they announced (not sure if this can actually happen; it would happen if 2
	///   nodes had each other in their sending set and they sent manifests at the same time. The
	///   code accounts for this anyway)
	t.Run("pending_statements_are_updated_after_manifest_exchange", func(t *testing.T) {
		t.Parallel()

		sendTo := parachaintypes.ValidatorIndex(0)
		receiveFrom := parachaintypes.ValidatorIndex(1)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].sending[sendTo] = struct{}{}
		sessionTopology.groupViews[groupIndex].receiving[receiveFrom] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		t.Run("receiving_followed_by_sending_an_ack", func(t *testing.T) {
			t.Parallel()

			tracker := newGridTracker()

			// Confirm the candidate.
			receivers := tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())
			require.Len(t, receivers, 1)
			require.Equal(t, sendTo, receivers[0].validator)
			require.Equal(t, full, receivers[0].kind)

			// Learn a statement from a different validator.
			tracker.learnedFreshStatement(
				*groups,
				sessionTopology,
				parachaintypes.ValidatorIndex(2),
				parachaintypes.NewCompactSeconded(candidateHash),
			)

			// Should start with no pending statements.
			ensurePendingStatements(t, tracker, receiveFrom, receiveFrom, candidateHash, nil, nil)

			ack, err := tracker.importManifest(
				sessionTopology,
				*groups,
				candidateHash,
				3,
				manifestSummary{
					claimedParentHash: common.Hash{0x0},
					claimedGroupIndex: groupIndex,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, false, true, false),
						validatedInGroup: newBitVec(t, true, false, true),
					},
				},
				full,
				receiveFrom,
			)

			require.NoError(t, err)
			require.True(t, ack)

			// Send ack now.
			tracker.manifestSentTo(*groups, receiveFrom, candidateHash, *localKnowledge)

			// There should be pending statements now.
			expectedFilter := &statementFilter{
				secondedInGroup:  newBitVec(t, false, false, true),
				validatedInGroup: newBitVec(t, false, false, false),
			}

			ensurePendingStatements(
				t,
				tracker,
				receiveFrom,
				parachaintypes.ValidatorIndex(2),
				candidateHash,
				expectedFilter,
				parachaintypes.NewCompactSeconded(candidateHash),
			)
		})

		t.Run("sending_followed_by_receiving_an_ack", func(t *testing.T) {
			t.Parallel()

			tracker := newGridTracker()

			// Confirm the candidate.
			receivers := tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())
			require.Len(t, receivers, 1)
			require.Equal(t, sendTo, receivers[0].validator)
			require.Equal(t, full, receivers[0].kind)

			// Learn a statement from a different validator.
			tracker.learnedFreshStatement(
				*groups,
				sessionTopology,
				parachaintypes.ValidatorIndex(2),
				parachaintypes.NewCompactSeconded(candidateHash),
			)

			// Should start with no pending statements.
			ensurePendingStatements(t, tracker, sendTo, sendTo, candidateHash, nil, nil)

			tracker.manifestSentTo(*groups, sendTo, candidateHash, localKnowledge.clone())

			ack, err := tracker.importManifest(
				sessionTopology,
				*groups,
				candidateHash,
				3,
				manifestSummary{
					claimedParentHash: common.Hash{0x0},
					claimedGroupIndex: groupIndex,
					statementKnowledge: statementFilter{
						secondedInGroup:  newBitVec(t, false, true, false),
						validatedInGroup: newBitVec(t, false, false, true),
					},
				},
				acknowledgement,
				sendTo,
			)

			require.NoError(t, err)
			require.False(t, ack)

			// There should be pending statements now.
			expectedFilter := &statementFilter{
				secondedInGroup:  newBitVec(t, false, false, true),
				validatedInGroup: newBitVec(t, false, false, false),
			}

			ensurePendingStatements(
				t,
				tracker,
				sendTo,
				parachaintypes.ValidatorIndex(2),
				candidateHash,
				expectedFilter,
				parachaintypes.NewCompactSeconded(candidateHash),
			)
		})
	})

	t.Run("invalid_fresh_statement_import", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[validatorIndex] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Should start with no pending statements.
		ensurePendingStatements(t, tracker, validatorIndex, validatorIndex, candidateHash, nil, nil)

		// Try to import fresh statement. Candidate not backed.
		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			validatorIndex,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		ensurePendingStatements(t, tracker, validatorIndex, validatorIndex, candidateHash, nil, nil)

		// Add the candidate as backed.
		tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Try to import fresh statement. Unknown group for validator index.
		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			parachaintypes.ValidatorIndex(1),
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		ensurePendingStatements(t, tracker, validatorIndex, validatorIndex, candidateHash, nil, nil)
	})

	t.Run("pending_statements_updated_when_importing_fresh_statement", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[validatorIndex] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Should start with no pending statements.
		ensurePendingStatements(t, tracker, validatorIndex, validatorIndex, candidateHash, nil, nil)

		// Add the candidate as backed.
		tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Import fresh statement.

		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			validatorIndex,
		)
		require.NoError(t, err)
		require.True(t, ack)

		tracker.manifestSentTo(*groups, validatorIndex, candidateHash, *localKnowledge)

		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			validatorIndex,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		// There should be pending statements now.
		expectedFilter := &statementFilter{
			secondedInGroup:  newBitVec(t, true, false, false),
			validatedInGroup: newBitVec(t, false, false, false),
		}

		ensurePendingStatements(
			t,
			tracker,
			validatorIndex,
			validatorIndex,
			candidateHash,
			expectedFilter,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		// After successful import, try importing again. Nothing should change.
		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			validatorIndex,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		ensurePendingStatements(
			t,
			tracker,
			validatorIndex,
			validatorIndex,
			candidateHash,
			expectedFilter,
			parachaintypes.NewCompactSeconded(candidateHash),
		)
	})

	// After learning fresh statements, we should not generate pending statements for knowledge that
	// the validator already has.
	t.Run("pending_statements_respect_remote_knowledge", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[validatorIndex] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Should start with no pending statements.
		ensurePendingStatements(t, tracker, validatorIndex, validatorIndex, candidateHash, nil, nil)

		// Add the candidate as backed.
		tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Import fresh statement.
		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, true, false, true),
					validatedInGroup: newBitVec(t, false, false, false),
				},
			},
			full,
			validatorIndex,
		)

		require.NoError(t, err)
		require.True(t, ack)

		tracker.manifestSentTo(*groups, validatorIndex, candidateHash, *localKnowledge)

		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			validatorIndex,
			parachaintypes.NewCompactSeconded(candidateHash),
		)
		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			validatorIndex,
			parachaintypes.NewCompactValid(candidateHash),
		)

		// The pending statements should respect the remote knowledge (meaning the Seconded
		// statement is ignored, but not the Valid statement).
		expectedFilter := &statementFilter{
			secondedInGroup:  newBitVec(t, false, false, false),
			validatedInGroup: newBitVec(t, true, false, false),
		}

		ensurePendingStatements(
			t,
			tracker,
			validatorIndex,
			validatorIndex,
			candidateHash,
			expectedFilter,
			parachaintypes.NewCompactValid(candidateHash),
		)
	})

	t.Run("pending_statements_cleared_when_sending", func(t *testing.T) {
		t.Parallel()

		tracker := newGridTracker()
		validatorIndex := parachaintypes.ValidatorIndex(0)
		counterparty := parachaintypes.ValidatorIndex(1)

		sessionTopology := emptySessionTopology.clone()
		sessionTopology.groupViews[groupIndex].receiving[validatorIndex] = struct{}{}
		sessionTopology.groupViews[groupIndex].receiving[counterparty] = struct{}{}

		localKnowledge, err := newStatementFilter(uint(*groupSize), false)
		require.NoError(t, err)

		// Should start with no pending statements.
		require.Nil(t, tracker.pendingStatementsFor(validatorIndex, candidateHash))
		require.Empty(t, tracker.allPendingStatementsFor(validatorIndex))

		// Add the candidate as backed.
		tracker.addBackedCandidate(sessionTopology, candidateHash, groupIndex, localKnowledge.clone())

		// Import statement for originator.
		ack, err := tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			validatorIndex,
		)
		require.NoError(t, err)
		require.True(t, ack)

		tracker.manifestSentTo(*groups, validatorIndex, candidateHash, localKnowledge.clone())

		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			validatorIndex,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		// Import statement for counterparty.
		ack, err = tracker.importManifest(
			sessionTopology,
			*groups,
			candidateHash,
			3,
			manifestSummary{
				claimedParentHash: common.Hash{0x0},
				claimedGroupIndex: groupIndex,
				statementKnowledge: statementFilter{
					secondedInGroup:  newBitVec(t, false, true, false),
					validatedInGroup: newBitVec(t, true, false, true),
				},
			},
			full,
			counterparty,
		)
		require.NoError(t, err)
		require.True(t, ack)

		tracker.manifestSentTo(*groups, counterparty, candidateHash, localKnowledge.clone())

		tracker.learnedFreshStatement(
			*groups,
			sessionTopology,
			counterparty,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		// There should be pending statements now.
		expectedFilter := &statementFilter{
			secondedInGroup:  newBitVec(t, true, false, false),
			validatedInGroup: newBitVec(t, false, false, false),
		}

		ensurePendingStatements(
			t,
			tracker,
			validatorIndex,
			validatorIndex,
			candidateHash,
			expectedFilter,
			parachaintypes.NewCompactSeconded(candidateHash),
		)

		ensurePendingStatements(
			t,
			tracker,
			counterparty,
			validatorIndex,
			candidateHash,
			expectedFilter,
			parachaintypes.NewCompactSeconded(candidateHash),
		)
	})
}

func ensurePendingStatements(
	t *testing.T,
	tracker *gridTracker,
	validator parachaintypes.ValidatorIndex,
	originator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
	expectedFilter *statementFilter,
	expectedStatement parachaintypes.CompactStatement,
) {
	t.Helper()

	pendingStatements := tracker.pendingStatementsFor(validator, candidateHash)
	if expectedFilter != nil {
		require.NotNil(t, pendingStatements)
	}
	require.Equal(t, expectedFilter, pendingStatements)

	allPendingStatements := tracker.allPendingStatementsFor(validator)

	if expectedStatement == nil {
		require.Empty(t, allPendingStatements)
		return
	}

	require.NotNil(t, allPendingStatements)
	require.Len(t, allPendingStatements, 1)

	expectedPair := originatorStatementPair{
		validatorIndex: originator,
		statement:      expectedStatement,
	}
	require.Equal(t, expectedPair, allPendingStatements[0])
}

func dummyGroups(t *testing.T, size int) *groups {
	t.Helper()

	group := make([]parachaintypes.ValidatorIndex, size)
	for i := 0; i < size; i++ {
		group[i] = parachaintypes.ValidatorIndex(i)
	}

	return newGroups([][]parachaintypes.ValidatorIndex{group}, 2)
}

func (g *gridTracker) isManifestPendingFor(
	validatorIndex parachaintypes.ValidatorIndex, //nolint:unparam
	candidateHash parachaintypes.CandidateHash,
) *manifestKind {
	pm, ok := g.pendingManifests[validatorIndex]
	if !ok {
		return nil
	}

	manifestKind, ok := pm[candidateHash]
	if !ok {
		return nil
	}

	return &manifestKind
}
