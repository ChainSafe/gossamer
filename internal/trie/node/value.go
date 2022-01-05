// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// GetValue returns the value of the branch.
// Note it does not copy the byte slice so modifying the returned
// byte slice will modify the byte slice of the branch.
func (b *Branch) GetValue() (value []byte) {
	return b.Value
}

// GetValue returns the value of the leaf.
// Note it does not copy the byte slice so modifying the returned
// byte slice will modify the byte slice of the leaf.
func (l *Leaf) GetValue() (value []byte) {
	return l.Value
}
