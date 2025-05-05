// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/assert"
)

func TestNewGroups(t *testing.T) {
	t.Parallel()

	t.Run("empty_groups", func(t *testing.T) {
		t.Parallel()

		emptyGroups := [][]parachaintypes.ValidatorIndex{}

		g := newGroups(emptyGroups, 2)

		assert.NotNil(t, g)
		assert.Empty(t, g.groups)
		assert.Empty(t, g.byValidatorIdx)
		assert.Equal(t, uint32(2), g.backingThreshold)
	})

	t.Run("nil_groups", func(t *testing.T) {
		t.Parallel()

		g := newGroups(nil, 2)

		assert.NotNil(t, g)
		assert.NotNil(t, g.groups)
		assert.Empty(t, g.groups)
		assert.Empty(t, g.byValidatorIdx)
		assert.Equal(t, uint32(2), g.backingThreshold)
	})

	t.Run("non_empty_groups", func(t *testing.T) {
		t.Parallel()

		initGroups := [][]parachaintypes.ValidatorIndex{
			{0, 1, 2},
			{3, 4, 5},
		}

		g := newGroups(initGroups, 3)

		assert.NotNil(t, g)
		assert.Equal(t, initGroups, g.groups)
		assert.Len(t, g.byValidatorIdx, 6)
		assert.Equal(t, uint32(3), g.backingThreshold)
	})
}

func TestGroups_All(t *testing.T) {
	t.Parallel()

	initGroups := [][]parachaintypes.ValidatorIndex{
		{0, 1, 2},
		{3, 4, 5},
	}
	g := newGroups(initGroups, 2)

	result := g.all()

	assert.Equal(t, initGroups, result)
}

func TestGroups_Group(t *testing.T) {
	t.Parallel()

	initGroups := [][]parachaintypes.ValidatorIndex{
		{0, 1, 2},
		{3, 4, 5},
	}
	g := newGroups(initGroups, 2)

	t.Run("valid_group_indices", func(t *testing.T) {
		t.Parallel()

		group0 := g.group(0)

		assert.Equal(t, initGroups[0], group0)

		group1 := g.group(1)

		assert.Equal(t, initGroups[1], group1)
	})

	t.Run("invalid_group_index", func(t *testing.T) {
		t.Parallel()

		invalidGroup := g.group(2)

		assert.Nil(t, invalidGroup)
	})
}

func TestGroups_GetSizeAndBackingThreshold(t *testing.T) {
	t.Parallel()

	initGroups := [][]parachaintypes.ValidatorIndex{
		{0, 1, 2},
	}
	g := newGroups(initGroups, 2)

	t.Run("valid_group_index", func(t *testing.T) {
		t.Parallel()

		size, threshold := g.getSizeAndBackingThreshold(0)

		assert.NotNil(t, size)
		assert.NotNil(t, threshold)
		assert.Equal(t, uint32(3), *size)
		assert.NotNil(t, threshold)
	})

	t.Run("invalid_group_index", func(t *testing.T) {
		t.Parallel()

		size, threshold := g.getSizeAndBackingThreshold(3)

		assert.Nil(t, size)
		assert.Nil(t, threshold)
	})
}

func TestGroups_ByValidatorIndex(t *testing.T) {
	t.Parallel()

	initGroups := [][]parachaintypes.ValidatorIndex{
		{0, 1, 2},
		{3, 4, 5},
	}
	g := newGroups(initGroups, 2)

	t.Run("valid_validator_index", func(t *testing.T) {
		t.Parallel()

		groupIndex := g.byValidatorIndex(4)

		assert.NotNil(t, groupIndex)
		expectedGroupIdx := parachaintypes.GroupIndex(1)
		assert.Equal(t, expectedGroupIdx, *groupIndex)
	})

	t.Run("invalid_validator_index", func(t *testing.T) {
		t.Parallel()

		groupIndex := g.byValidatorIndex(6)

		assert.Nil(t, groupIndex)
	})
}
