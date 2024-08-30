package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
)

func TestUnreadyBlocks_removeIrrelevantFragments(t *testing.T) {
	ub := newUnreadyBlocks()
	ub.disjointFragments = [][]*types.BlockData{
		// first fragment
		{
			{
				Header: &types.Header{
					Number: 192,
				},
			},

			{
				Header: &types.Header{
					Number: 191,
				},
			},

			{
				Header: &types.Header{
					Number: 190,
				},
			},
		},

		// second fragment
		{
			{
				Header: &types.Header{
					Number: 253,
				},
			},

			{
				Header: &types.Header{
					Number: 254,
				},
			},

			{
				Header: &types.Header{
					Number: 255,
				},
			},
		},

		// third fragment
		{
			{
				Header: &types.Header{
					Number: 1022,
				},
			},

			{
				Header: &types.Header{
					Number: 1023,
				},
			},

			{
				Header: &types.Header{
					Number: 1024,
				},
			},
		},
	}

	// the first fragment should be removed
	// the second fragment should have only 2 items
	// the third frament shold not be affected
	ub.removeIrrelevantFragments(253)
	require.Len(t, ub.disjointFragments, 2)

	expectedSecondFrag := []*types.BlockData{
		{
			Header: &types.Header{
				Number: 254,
			},
		},

		{
			Header: &types.Header{
				Number: 255,
			},
		},
	}

	expectedThirdFragment := []*types.BlockData{
		{
			Header: &types.Header{
				Number: 1022,
			},
		},

		{
			Header: &types.Header{
				Number: 1023,
			},
		},

		{
			Header: &types.Header{
				Number: 1024,
			},
		},
	}
	require.Equal(t, ub.disjointFragments[0], expectedSecondFrag)
	require.Equal(t, ub.disjointFragments[1], expectedThirdFragment)
}
