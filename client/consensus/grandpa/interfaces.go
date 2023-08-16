// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"golang.org/x/exp/constraints"
)

// ForkTree A roots data structure that stores several children across multiple branches.
//
// Top-level branches are called roots. The roots has functionality for
// finalizing children, which means that node is traversed, and all competing
// branches are pruned. It also guarantees that children in the roots are finalized
// in order. Each node is uniquely identified by its hash but can be ordered by
// its number. In order to build the roots an external function must be provided
// when interacting with the roots to establish a node's ancestry.
type ForkTree[H comparable, N constraints.Unsigned] interface {
	Import(hash H, number N, change PendingChange[H, N], isDescendentOf IsDescendentOf[H]) (bool, error)
	Roots() []*pendingChangeNode[H, N]
	FinalizesAnyWithDescendentIf(hash *H, number N, isDescendentOf IsDescendentOf[H], predicate func(*PendingChange[H, N]) bool) (*bool, error)
	FinalizeWithDescendentIf(hash *H, number N, isDescendentOf IsDescendentOf[H], predicate func(*PendingChange[H, N]) bool) (*FinalizationResult[H, N], error)
	DrainFilter()

	// PendingChanges gets pending changes in PreOrder
	PendingChanges() []PendingChange[H, N]
}

// PublicKey interface
type PublicKey interface {
	Verify(msg, sig []byte) (bool, error)
	Encode() []byte
	Decode([]byte) error
	Hex() string
}

type Telemetry interface {
}
