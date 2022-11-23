// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import "github.com/ChainSafe/gossamer/lib/common"

// Noop is a noop implementation of the tracking.
type Noop struct {
}

// NewNoop returns a new Noop struct.
func NewNoop() *Noop {
	return &Noop{}
}

// RecordDeleted records a node hash as deleted.
func (n *Noop) RecordDeleted(nodeHash common.Hash) {}

// Deleted returns a set (map) of all the recorded deleted
// node hashes. Note the map returned is not deep copied for
// performance reasons and so it's not safe for mutation.
func (n *Noop) Deleted() (nodeHashes map[common.Hash]struct{}) {
	return nil
}

// MergeWith merges the deltas given as argument in the receiving
// deltas struct.
func (n *Noop) MergeWith(deltas DeletedGetter) {}

// DeepCopy returns a deep copy of the Noop.
func (n *Noop) DeepCopy() (deepCopy *Noop) {
	return &Noop{}
}
