// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// DelayedKinds Kinds of delays for pending changes.
type DelayedKinds interface {
	Finalized | Best
}

// DelayKind struct to represent DelayedKinds
type DelayKind struct {
	value interface{}
}

func setDelayKind[T DelayedKinds](delayKind *DelayKind, val T) {
	delayKind.value = val
}

func newDelayKind[T DelayedKinds](val T) DelayKind {
	delayKind := DelayKind{}
	setDelayKind(&delayKind, val)
	return delayKind
}

// Finalized Depth in finalized chain.
type Finalized struct{}

// Best Depth in best chain. The median last finalized block is calculated at the time the
// hashNumber was signaled.
type Best struct {
	medianLastFinalized uint
}
