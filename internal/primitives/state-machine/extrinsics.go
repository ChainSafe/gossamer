// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

type extrinsics []uint32

func (e extrinsics) CopyExtrinsicsInto(dest map[uint32]struct{}) {
	for _, ex := range e {
		dest[ex] = struct{}{}
	}
}

func (e *extrinsics) Insert(ext uint32) {
	if len(*e) == 0 || ext != (*e)[len(*e)-1] {
		*e = append(*e, ext)
	}
}

func (e *extrinsics) Extend(other extrinsics) {
	*e = append(*e, other...)
}
