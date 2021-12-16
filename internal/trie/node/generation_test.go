// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Branch_SetGeneration(t *testing.T) {
	t.Parallel()

	branch := &Branch{
		generation: 1,
	}
	branch.SetGeneration(2)
	assert.Equal(t, &Branch{generation: 2}, branch)
}

func Test_Branch_GetGeneration(t *testing.T) {
	t.Parallel()

	const generation uint64 = 1
	branch := &Branch{
		generation: generation,
	}
	assert.Equal(t, branch.GetGeneration(), generation)
}

func Test_Leaf_SetGeneration(t *testing.T) {
	t.Parallel()

	leaf := &Leaf{
		generation: 1,
	}
	leaf.SetGeneration(2)
	assert.Equal(t, &Leaf{generation: 2}, leaf)
}

func Test_Leaf_GetGeneration(t *testing.T) {
	t.Parallel()

	const generation uint64 = 1
	leaf := &Leaf{
		generation: generation,
	}
	assert.Equal(t, leaf.GetGeneration(), generation)
}
