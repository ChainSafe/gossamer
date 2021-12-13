// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// IsDirty returns the dirty status of the branch.
func (b *Branch) IsDirty() bool {
	return b.dirty
}

// SetDirty sets the dirty status to the branch.
func (b *Branch) SetDirty(dirty bool) {
	b.dirty = dirty
}

// IsDirty returns the dirty status of the leaf.
func (l *Leaf) IsDirty() bool {
	return l.dirty
}

// SetDirty sets the dirty status to the leaf.
func (l *Leaf) SetDirty(dirty bool) {
	l.dirty = dirty
}
