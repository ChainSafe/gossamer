// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"

	"testing"
)

func TestStatementFilter_HasLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		groupSize uint
		full      bool
		input     int
		expected  bool
	}{
		{
			name:      "empty_filter_size_0",
			groupSize: 0,
			full:      false,
			input:     0,
			expected:  true,
		},
		{
			name:      "empty_filter_correct_size",
			groupSize: 5,
			full:      false,
			input:     5,
			expected:  true,
		},
		{
			name:      "empty_filter_wrong_size",
			groupSize: 5,
			full:      false,
			input:     3,
			expected:  false,
		},
		{
			name:      "full_filter_correct_size",
			groupSize: 8,
			full:      true,
			input:     8,
			expected:  true,
		},
		{
			name:      "full_filter_wrong_size",
			groupSize: 8,
			full:      true,
			input:     10,
			expected:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter, err := newStatementFilter(tt.groupSize, tt.full)
			require.NoError(t, err)

			got := filter.hasLen(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestStatementFilter_BackingValidators(t *testing.T) {
	t.Parallel()

	// Helper function to modify filter BitVecs
	setFilterBits := func(filter *statementFilter, seconded, validated []bool) {
		newSeconded, err := parachaintypes.NewBitVec(seconded)
		require.NoError(t, err)
		newValidated, err := parachaintypes.NewBitVec(validated)
		require.NoError(t, err)
		filter.secondedInGroup = newSeconded
		filter.validatedInGroup = newValidated
	}

	tests := []struct {
		name      string
		groupSize uint
		seconded  []bool
		validated []bool
		expected  int
	}{
		{
			name:      "empty_filter",
			groupSize: 4,
			seconded:  []bool{false, false, false, false},
			validated: []bool{false, false, false, false},
			expected:  0,
		},
		{
			name:      "only_seconded",
			groupSize: 4,
			seconded:  []bool{true, false, true, false},
			validated: []bool{false, false, false, false},
			expected:  2,
		},
		{
			name:      "only_validated",
			groupSize: 4,
			seconded:  []bool{false, false, false, false},
			validated: []bool{false, true, false, true},
			expected:  2,
		},
		{
			name:      "both_no_overlap",
			groupSize: 4,
			seconded:  []bool{true, false, false, false},
			validated: []bool{false, true, false, false},
			expected:  2,
		},
		{
			name:      "both_with_overlap",
			groupSize: 4,
			seconded:  []bool{true, true, false, false},
			validated: []bool{false, true, true, false},
			expected:  3,
		},
		{
			name:      "all_set",
			groupSize: 4,
			seconded:  []bool{true, true, true, true},
			validated: []bool{true, true, true, true},
			expected:  4,
		},
		{
			name:      "size_0",
			groupSize: 0,
			seconded:  []bool{},
			validated: []bool{},
			expected:  0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter, err := newStatementFilter(tt.groupSize, false)
			require.NoError(t, err)
			setFilterBits(filter, tt.seconded, tt.validated)

			got := filter.backingValidators()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestStatementFilter_HasSeconded(t *testing.T) {
	t.Parallel()

	// Helper function to set seconded bits
	setSecondedBits := func(filter *statementFilter, seconded []bool) {
		newSeconded, err := parachaintypes.NewBitVec(seconded)
		require.NoError(t, err)
		filter.secondedInGroup = newSeconded
	}

	tests := []struct {
		name      string
		groupSize uint
		seconded  []bool
		expected  bool
	}{
		{
			name:      "empty_filter",
			groupSize: 4,
			seconded:  []bool{false, false, false, false},
			expected:  false,
		},
		{
			name:      "single_bit_set",
			groupSize: 4,
			seconded:  []bool{true, false, false, false},
			expected:  true,
		},
		{
			name:      "multiple_bits_set",
			groupSize: 4,
			seconded:  []bool{true, false, true, false},
			expected:  true,
		},
		{
			name:      "all_bits_set",
			groupSize: 4,
			seconded:  []bool{true, true, true, true},
			expected:  true,
		},
		{
			name:      "size_0",
			groupSize: 0,
			seconded:  []bool{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter, err := newStatementFilter(tt.groupSize, false)
			require.NoError(t, err)
			setSecondedBits(filter, tt.seconded)

			got := filter.hasSeconded()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestStatementFilter_MaskSeconded(t *testing.T) {
	t.Parallel()

	// Helper function to set seconded bits
	setSecondedBits := func(filter *statementFilter, seconded []bool) {
		newSeconded, err := parachaintypes.NewBitVec(seconded)
		require.NoError(t, err)
		filter.secondedInGroup = newSeconded
	}

	tests := []struct {
		name            string
		groupSize       uint
		initialSeconded []bool
		mask            []bool
		expectedResult  []bool
	}{
		{
			name:            "empty_filter_empty_mask",
			groupSize:       4,
			initialSeconded: []bool{false, false, false, false},
			mask:            []bool{false, false, false, false},
			expectedResult:  []bool{false, false, false, false},
		},
		{
			name:            "all_set_filter_empty_mask",
			groupSize:       4,
			initialSeconded: []bool{true, true, true, true},
			mask:            []bool{false, false, false, false},
			expectedResult:  []bool{true, true, true, true},
		},
		{
			name:            "all_set_filter_full_mask",
			groupSize:       4,
			initialSeconded: []bool{true, true, true, true},
			mask:            []bool{true, true, true, true},
			expectedResult:  []bool{false, false, false, false},
		},
		{
			name:            "partial_set_partial_mask",
			groupSize:       4,
			initialSeconded: []bool{true, false, true, false},
			mask:            []bool{true, false, false, true},
			expectedResult:  []bool{false, false, true, false},
		},
		{
			name:            "size_0",
			groupSize:       0,
			initialSeconded: []bool{},
			mask:            []bool{},
			expectedResult:  []bool{},
		},
		{
			name:            "mask_with_different_size",
			groupSize:       4,
			initialSeconded: []bool{true, false, true, false},
			mask:            []bool{true, false},
			expectedResult:  []bool{false, false, true, false},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter, err := newStatementFilter(tt.groupSize, false)
			require.NoError(t, err)
			setSecondedBits(filter, tt.initialSeconded)

			mask, err := parachaintypes.NewBitVec(tt.mask)
			require.NoError(t, err)

			// Capture the initial validated bits to verify they remain unchanged
			initialValidated := filter.validatedInGroup.Bits()

			filter.maskSeconded(mask)

			// Check the result
			require.Equal(t, tt.expectedResult, filter.secondedInGroup.Bits())
			// Verify validatedInGroup was not modified
			require.Equal(t, initialValidated, filter.validatedInGroup.Bits())
		})
	}
}

func TestStatementFilter_MaskValid(t *testing.T) {
	t.Parallel()

	// Helper function to set validated bits
	setValidatedBits := func(filter *statementFilter, validated []bool) {
		newValidated, err := parachaintypes.NewBitVec(validated)
		require.NoError(t, err)
		filter.validatedInGroup = newValidated
	}

	tests := []struct {
		name             string
		groupSize        uint
		initialValidated []bool
		mask             []bool
		expectedResult   []bool
	}{
		{
			name:             "empty_filter_empty_mask",
			groupSize:        4,
			initialValidated: []bool{false, false, false, false},
			mask:             []bool{false, false, false, false},
			expectedResult:   []bool{false, false, false, false},
		},
		{
			name:             "all_set_filter_empty_mask",
			groupSize:        4,
			initialValidated: []bool{true, true, true, true},
			mask:             []bool{false, false, false, false},
			expectedResult:   []bool{true, true, true, true},
		},
		{
			name:             "all_set_filter_full_mask",
			groupSize:        4,
			initialValidated: []bool{true, true, true, true},
			mask:             []bool{true, true, true, true},
			expectedResult:   []bool{false, false, false, false},
		},
		{
			name:             "partial_set_partial_mask",
			groupSize:        4,
			initialValidated: []bool{true, false, true, false},
			mask:             []bool{true, false, false, true},
			expectedResult:   []bool{false, false, true, false},
		},
		{
			name:             "size_0",
			groupSize:        0,
			initialValidated: []bool{},
			mask:             []bool{},
			expectedResult:   []bool{},
		},
		{
			name:             "mask_with_different_size",
			groupSize:        4,
			initialValidated: []bool{true, false, true, false},
			mask:             []bool{true, false},
			expectedResult:   []bool{false, false, true, false},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter, err := newStatementFilter(tt.groupSize, false)
			require.NoError(t, err)
			setValidatedBits(filter, tt.initialValidated)

			mask, err := parachaintypes.NewBitVec(tt.mask)
			require.NoError(t, err)

			// Capture the initial seconded bits to verify they remain unchanged
			initialSeconded := filter.secondedInGroup.Bits()

			filter.maskValid(mask)

			// Check the result
			require.Equal(t, tt.expectedResult, filter.validatedInGroup.Bits())
			// Verify secondedInGroup was not modified
			require.Equal(t, initialSeconded, filter.secondedInGroup.Bits())
		})
	}
}
