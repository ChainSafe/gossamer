// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

// ChildrenBitmap returns the 16 bit bitmap
// of the children in the branch.
func (b *Branch) ChildrenBitmap() (bitmap uint16) {
	for i := uint(0); i < 16; i++ {
		if b.Children[i] != nil {
			bitmap = bitmap | 1<<i
		}
	}
	return bitmap
}

// NumChildren returns the total number of children
// in the branch.
func (b *Branch) NumChildren() (count int) {
	for i := 0; i < 16; i++ {
		if b.Children[i] != nil {
			count++
		}
	}
	return count
}
