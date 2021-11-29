// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

// IsDirty returns the dirty status of the branch.
func (b *Branch) IsDirty() bool {
	return b.Dirty
}

// SetDirty sets the dirty status to the branch.
func (b *Branch) SetDirty(dirty bool) {
	b.Dirty = dirty
}
