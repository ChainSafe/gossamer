// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

// SetGeneration sets the generation given to the branch.
func (b *Branch) SetGeneration(generation uint64) {
	b.Generation = generation
}

// GetGeneration returns the generation of the branch.
func (b *Branch) GetGeneration() uint64 {
	return b.Generation
}
