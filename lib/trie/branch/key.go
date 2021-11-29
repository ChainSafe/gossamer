// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

// SetKey sets the key to the branch.
// Note it does not copy it so modifying the passed key
// will modify the key stored in the branch.
func (b *Branch) SetKey(key []byte) {
	b.Key = key
}
