// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

type Extrinsics []uint32

func (e *Extrinsics) CopyExtrinsicsInto(dest map[uint32]struct{}) {
	for _, ex := range *e {
		dest[ex] = struct{}{}
	}
}

func (e *Extrinsics) Insert(ext uint32) {
	if len(*e) == 0 || ext != (*e)[len(*e)-1] {
		*e = append(*e, ext)
	}
}

func (e *Extrinsics) Extend(other Extrinsics) {
	*e = append(*e, other...)
}
