// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import "golang.org/x/exp/constraints"

// DelayedKinds Kinds of delays for pending changes.
type DelayedKinds[N constraints.Unsigned] interface {
	Finalized | Best[N]
}

// DelayKind struct to represent DelayedKinds
type DelayKind struct {
	value interface{}
}

func setDelayKind[N constraints.Unsigned, T DelayedKinds[N]](delayKind *DelayKind, val T) {
	delayKind.value = val
}

func newDelayKind[N constraints.Unsigned, T DelayedKinds[N]](val T) DelayKind {
	delayKind := DelayKind{}
	setDelayKind[N](&delayKind, val)
	return delayKind
}

// Finalized Depth in finalized chain.
type Finalized struct{}

// Best Depth in best chain. The median last finalized block is calculated at the time the
// hashNumber was signaled.
type Best[N constraints.Unsigned] struct {
	medianLastFinalized N
}
