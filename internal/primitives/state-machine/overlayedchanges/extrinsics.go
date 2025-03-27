// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

type extrinsics []uint32

func (e *extrinsics) insert(ext uint32) {
	if len(*e) == 0 || ext != (*e)[len(*e)-1] {
		*e = append(*e, ext)
	}
}

func (e *extrinsics) extend(other extrinsics) {
	*e = append(*e, other...)
}
