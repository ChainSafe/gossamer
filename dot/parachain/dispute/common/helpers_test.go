package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetByzantineThreshold(t *testing.T) {
	cases := []struct {
		n, expected int
	}{
		{0, 0},
		{1, 0},
		{2, 0},
		{3, 0},
		{4, 1},
		{5, 1},
		{6, 1},
		{9, 2},
		{10, 3},
		// Additional cases can be added here
	}

	for _, c := range cases {
		got := GetByzantineThreshold(c.n)
		require.Equal(t, c.expected, got)
	}
}

func TestGetSuperMajorityThreshold(t *testing.T) {
	cases := []struct {
		n, expected int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 3},
		{5, 4},
		{6, 5},
		{9, 7},
		{10, 7},
		// Additional cases can be added here
	}

	for _, c := range cases {
		got := GetSuperMajorityThreshold(c.n)
		require.Equal(t, c.expected, got)
	}
}
