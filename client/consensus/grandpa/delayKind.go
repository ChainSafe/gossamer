// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// DelayedKinds Kinds of Delays for pending changes.
type DelayedKinds interface {
	Finalized | Best
}

// DelayKind struct to represent DelayedKinds
// TODO this needs to be a vdt I think
type DelayKind struct {
	Value interface{}
}

func setDelayKind[T DelayedKinds](delayKind *DelayKind, val T) {
	delayKind.Value = val
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
	MedianLastFinalized uint
}
