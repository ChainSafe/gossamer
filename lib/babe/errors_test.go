package babe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
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
			expected: "custom module error: index: 4 code: 5 message: ",
		},
		{
			name:     "Dispatch custom module error",
			test:     []byte{0, 1, 3, 4, 5, 1, 0x04, 0x65},
			expected: "custom module error: index: 4 code: 5 message: 65",
		},
		{
			name:     "Dispatch unknown error",
			test:     []byte{0, 1, 0, 0x04, 65},
			expected: "unknown error: A",
		},
		{
			name:     "Invalid txn payment error",
			test:     []byte{1, 0, 1},
			expected: "invalid payment",
		},
		{
			name:     "Invalid txn payment error",
			test:     []byte{1, 0, 7, 65},
			expected: "unknown error: 65",
		},
		{
			name:     "Unknown txn lookup failed error",
			test:     []byte{1, 1, 0},
			expected: "lookup failed",
		},
		{
			name:     "Invalid txn unknown error",
			test:     []byte{1, 1, 2, 75},
			expected: "unknown error: 75",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			ok, err := determineErr(c.test)
			if c.test[0] == 0 {
				require.True(t, ok)
			} else {
				require.False(t, ok)
			}

			if c.expected == "" {
				require.NoError(t, err)
				return
			}
			require.Equal(t, err.Error(), c.expected)
		})
	}
}
