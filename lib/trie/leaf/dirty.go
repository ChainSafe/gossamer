// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

// IsDirty returns the dirty status of the leaf.
func (l *Leaf) IsDirty() bool {
	return l.Dirty
}

// SetDirty sets the dirty status to the leaf.
func (l *Leaf) SetDirty(dirty bool) {
	l.Dirty = dirty
}
