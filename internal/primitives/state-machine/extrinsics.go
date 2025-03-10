// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import "github.com/tidwall/btree"

type Extrinsics []uint32

func (e *Extrinsics) CopyExtrinsicsInto(dest *btree.Set[uint32]) {
	for _, ex := range *e {
		dest.Insert(ex)
	}
}

func (e *Extrinsics) Insert(ext uint32) {
	if len(*e) == 0 || ext != (*e)[len(*e)-1] {
		*e = append(*e, ext)
	}
}

func (e *Extrinsics) Extend(other Extrinsics) {
	for _, ext := range other {
		*e = append(*e, ext)
	}
}
