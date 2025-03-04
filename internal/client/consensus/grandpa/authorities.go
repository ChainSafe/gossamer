// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// HashNumber is a generic representation of hash and number
type HashNumber[H, N any] struct {
	Hash   H
	Number N
}
