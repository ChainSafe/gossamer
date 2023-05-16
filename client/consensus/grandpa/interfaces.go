// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// ForkTree A tree data structure that stores several nodes across multiple branches.
//
// Top-level branches are called roots. The tree has functionality for
// finalizing nodes, which means that node is traversed, and all competing
// branches are pruned. It also guarantees that nodes in the tree are finalized
// in order. Each node is uniquely identified by its hash but can be ordered by
// its number. In order to build the tree an external function must be provided
// when interacting with the tree to establish a node's ancestry.
// TODO implement this rather than mock out
type ForkTree interface {
	Import(hash common.Hash, number uint, change PendingChange, isDescendentOf IsDescendentOf) error
}
