// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package testing

// / An opaque extrinsic wrapper type.
type ExtrinsicsWrapper[T any] struct {
	T T
}

func (ExtrinsicsWrapper[T]) IsSigned() *bool {
	return nil
}
