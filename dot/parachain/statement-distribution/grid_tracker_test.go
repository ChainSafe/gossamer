// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

func TestGridTracker(t *testing.T) {
	t.Parallel()

	t.Run("reject_disallowed_manifest", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("reject_malformed_wrong_group_size", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("reject_malformed_no_seconders", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("reject_insufficient_below_threshold", func(t *testing.T) {
		t.Parallel()
	})

	// Test that when we add a candidate as backed and advertise it to the sending group, they can
	// provide an acknowledgement manifest in response.
	t.Run("senders_can_provide_manifests_in_acknowledgement", func(t *testing.T) {
		t.Parallel()
	})

	// Check that pending communication is set correctly when receiving a manifest on a confirmed
	// candidate.
	//
	// It should also overwrite any existing `Full` ManifestKind.
	t.Run("pending_communication_receiving_manifest_on_confirmed_candidate", func(t *testing.T) {
		t.Parallel()
	})

	// Check that pending communication is cleared correctly in `manifest_sent_to`
	//
	// Also test a scenario where manifest import returns `Ok(true)` (should acknowledge).
	t.Run("pending_communication_is_cleared", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("invalid_fresh_statement_import", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("pending_statements_updated_when_importing_fresh_statement", func(t *testing.T) {
		t.Parallel()
	})

	// After learning fresh statements, we should not generate pending statements for knowledge that
	// the validator already has.
	t.Run("pending_statements_respect_remote_knowledge", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("pending_statements_cleared_when_sending", func(t *testing.T) {
		t.Parallel()
	})
}

func dummyGroups(t *testing.T, size int) *groups {
	t.Helper()

	group := make([]parachaintypes.ValidatorIndex, size)
	for i := 0; i < size; i++ {
		group[i] = parachaintypes.ValidatorIndex(i)
	}

	return newGroups([][]parachaintypes.ValidatorIndex{group}, 2)
}
