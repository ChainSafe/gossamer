// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

// SetGeneration sets the generation given to the leaf.
func (l *Leaf) SetGeneration(generation uint64) {
	l.Generation = generation
}

// GetGeneration returns the generation of the leaf.
func (l *Leaf) GetGeneration() uint64 {
	return l.Generation
}
