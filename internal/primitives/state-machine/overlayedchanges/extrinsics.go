// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

// extrinsics is a slice of uint32 values that represent the extrinsics in which a change is applied.
type extrinsics []uint32

// insert inserts an extrinsic into the slice.
func (e *extrinsics) insert(ext uint32) {
	if len(*e) == 0 || ext != (*e)[len(*e)-1] {
		*e = append(*e, ext)
	}
}

// extend appends a slice of extrinsics to the current slice.
func (e *extrinsics) extend(other extrinsics) {
	*e = append(*e, other...)
}
