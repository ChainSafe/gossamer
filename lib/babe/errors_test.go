package babe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyExtrinsicErrors(t *testing.T) {
	testCases := []struct {
		name     string
		test     []byte
		expected string
	}{
		{
			name:     "Valid extrinsic",
			test:     []byte{0, 0},
			expected: "",
		},
		{
			name:     "Dispatch custom module error empty",
			test:     []byte{0, 1, 3, 4, 5, 1, 0},
			expected: "dispatch outcome error: custom module error: index: 4 code: 5 message: ",
		},
		{
			name:     "Dispatch custom module error",
			test:     []byte{0, 1, 3, 4, 5, 1, 0x04, 0x65},
			expected: "dispatch outcome error: custom module error: index: 4 code: 5 message: 65",
		},
		{
			name:     "Dispatch unknown error",
			test:     []byte{0, 1, 0, 0x04, 65},
			expected: "dispatch outcome error: unknown error: A",
		},
		{
			name:     "Dispatch failed lookup",
			test:     []byte{0, 1, 1},
			expected: "dispatch outcome error: failed lookup",
		},
		{
			name:     "Dispatch bad origin",
			test:     []byte{0, 1, 2},
			expected: "dispatch outcome error: bad origin",
		},
		{
			name:     "Invalid txn payment error",
			test:     []byte{1, 0, 1},
			expected: "transaction validity error: invalid payment",
		},
		{
			name:     "Invalid txn payment error",
			test:     []byte{1, 0, 7, 65},
			expected: "transaction validity error: unknown error: 65",
		},
		{
			name:     "Unknown txn lookup failed error",
			test:     []byte{1, 1, 0},
			expected: "transaction validity error: lookup failed",
		},
		{
			name:     "Invalid txn unknown error",
			test:     []byte{1, 1, 2, 75},
			expected: "transaction validity error: unknown error: 75",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			err := determineErr(c.test)
			if c.expected == "" {
				require.NoError(t, err)
				return
			}

			if c.test[0] == 0 {
				_, ok := err.(*DispatchOutcomeError)
				require.True(t, ok)
			} else {
				_, ok := err.(*TransactionValidityError)
				require.True(t, ok)
			}
			require.Equal(t, c.expected, err.Error())
		})
	}
}
