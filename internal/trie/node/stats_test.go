// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Branch_GetDescendants(t *testing.T) {
	t.Parallel()

	const descendants uint32 = 10
	branch := &Branch{
		Descendants: descendants,
	}
	result := branch.GetDescendants()

	assert.Equal(t, descendants, result)
}

func Test_Branch_AddDescendants(t *testing.T) {
	t.Parallel()

	const (
		initialDescendants uint32 = 10
		addDescendants     uint32 = 2
		finalDescendants   uint32 = 12
	)
	branch := &Branch{
		Descendants: initialDescendants,
	}
	branch.AddDescendants(addDescendants)
	expected := &Branch{
		Descendants: finalDescendants,
	}

	assert.Equal(t, expected, branch)
}

func Test_Branch_SubDescendants(t *testing.T) {
	t.Parallel()

	const (
		initialDescendants uint32 = 10
		subDescendants     uint32 = 2
		finalDescendants   uint32 = 8
	)
	branch := &Branch{
		Descendants: initialDescendants,
	}
	branch.SubDescendants(subDescendants)
	expected := &Branch{
		Descendants: finalDescendants,
	}

	assert.Equal(t, expected, branch)
}
