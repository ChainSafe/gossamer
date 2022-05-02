// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

func padRightChildren(slice []*Node) (paddedSlice []*Node) {
	paddedSlice = make([]*Node, ChildrenCapacity)
	copy(paddedSlice, slice)
	return paddedSlice
}
