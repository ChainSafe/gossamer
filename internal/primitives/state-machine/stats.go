// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

type StateMachineStats struct {
	// Number of read query from runtime
	// that hit a modified value (in state
	// machine overlay).
	ReadsModified uint64
	// Size in byte of read queries that
	// hit a modified value.
	BytesReadModified uint64
	// Number of time a write operation
	// occurs into the state machine overlay.
	WritesOverlay uint64
	// Size in bytes of the writes overlay
	// operation.
	BytesWritesOverlay uint64
}
