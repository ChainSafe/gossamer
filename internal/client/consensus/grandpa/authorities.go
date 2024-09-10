// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// generic representation of hash and number tuple
type HashNumber[H, N any] struct {
	Hash   H
	Number N
}
