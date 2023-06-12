// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// ForkTree A roots data structure that stores several children across multiple branches.
//
// Top-level branches are called roots. The roots has functionality for
// finalizing children, which means that node is traversed, and all competing
// branches are pruned. It also guarantees that children in the roots are finalized
// in order. Each node is uniquely identified by its hash but can be ordered by
// its number. In order to build the roots an external function must be provided
// when interacting with the roots to establish a node's ancestry.
type ForkTree interface {
	Import(hash common.Hash, number uint, change PendingChange, isDescendentOf IsDescendentOf) (bool, error)
	Roots() []*pendingChangeNode
	FinalizeAnyWithDescendentIf(hash *common.Hash, number uint, isDescendentOf IsDescendentOf, predicate Predicate[*PendingChange]) (*bool, error)
	FinalizeWithDescendentIf(hash *common.Hash, number uint, isDescendentOf IsDescendentOf, predicate Predicate[*PendingChange]) (*FinalizationResult, error)
	DrainFilter()

	// GetPreOrder This one is just inlined in rust so not part of substrate interface, but I thought would be good to expose here
	GetPreOrder() []PendingChange
}
