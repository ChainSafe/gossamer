// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

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

func NewStateMachineStats() StateMachineStats {
	return StateMachineStats{}
}

func (sms *StateMachineStats) Clone() StateMachineStats {
	return StateMachineStats{
		ReadsModified:      sms.ReadsModified,
		BytesReadModified:  sms.BytesReadModified,
		WritesOverlay:      sms.WritesOverlay,
		BytesWritesOverlay: sms.BytesWritesOverlay,
	}
}

// Tally one read modified operation, of some length.
func (sms *StateMachineStats) TallyReadModified(bytes uint64) {
	sms.ReadsModified++
	sms.BytesReadModified += bytes
}

func (sms *StateMachineStats) TallyWriteOverlay(bytes uint64) {
	sms.WritesOverlay++
	sms.BytesWritesOverlay += bytes
}
