// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// SetGeneration sets the generation given to the branch.
func (b *Branch) SetGeneration(generation uint64) {
	b.generation = generation
}

// GetGeneration returns the generation of the branch.
func (b *Branch) GetGeneration() (generation uint64) {
	return b.generation
}

// SetGeneration sets the generation given to the leaf.
func (l *Leaf) SetGeneration(generation uint64) {
	l.generation = generation
}

// GetGeneration returns the generation of the leaf.
func (l *Leaf) GetGeneration() (generation uint64) {
	return l.generation
}
