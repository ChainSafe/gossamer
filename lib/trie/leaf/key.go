// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

// SetKey sets the key to the leaf.
// Note it does not copy it so modifying the passed key
// will modify the key stored in the leaf.
func (l *Leaf) SetKey(key []byte) {
	l.Key = key
}
