// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// IsDirty returns the dirty status of the branch.
func (b *Branch) IsDirty() bool {
	return b.Dirty
}

// SetDirty sets the dirty status to the branch.
func (b *Branch) SetDirty(dirty bool) {
	b.Dirty = dirty
	if dirty {
		// A node is marked dirty if its key or value is modified.
		// This means its cached encoding and hash fields are no longer
		// valid. To improve memory usage, we clear these fields.
		b.Encoding = nil
		b.HashDigest = nil
	}
}

// IsDirty returns the dirty status of the leaf.
func (l *Leaf) IsDirty() bool {
	return l.Dirty
}

// SetDirty sets the dirty status to the leaf.
func (l *Leaf) SetDirty(dirty bool) {
	l.Dirty = dirty
	if dirty {
		// A node is marked dirty if its key or value is modified.
		// This means its cached encoding and hash fields are no longer
		// valid. To improve memory usage, we clear these fields.
		l.Encoding = nil
		l.HashDigest = nil
	}
}
