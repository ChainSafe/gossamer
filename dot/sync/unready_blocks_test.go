// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
)

func TestUnreadyBlocks_removeIrrelevantFragments(t *testing.T) {
	t.Run("removing_all_disjoint_fragment", func(t *testing.T) {
		ub := newUnreadyBlocks()
		ub.disjointFragments = []*Fragment{
			NewFragment([]*types.BlockData{
				{
					Header: &types.Header{
						Number: 99,
					},
				},
			}),
			NewFragment([]*types.BlockData{
				{
					Header: &types.Header{
						Number: 92,
					},
				},
			}),
		}

		ub.pruneDisjointFragments(LowerThanOrEq(100))
		require.Empty(t, ub.disjointFragments)
	})

	t.Run("removing_irrelevant_fragments", func(t *testing.T) {
		ub := newUnreadyBlocks()
		ub.disjointFragments = []*Fragment{
			// first fragment
			NewFragment([]*types.BlockData{
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
			}),

			// second fragment
			NewFragment([]*types.BlockData{
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
			}),

			// third fragment
			NewFragment([]*types.BlockData{
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
			}),
		}

		// the first fragment should be removed
		// the second fragment should have only 2 items
		// the third frament shold not be affected
		ub.pruneDisjointFragments(LowerThanOrEq(253))
		require.Len(t, ub.disjointFragments, 1)

		expectedThirdFragment := NewFragment([]*types.BlockData{
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
		})

		require.Equal(t, ub.disjointFragments[0], expectedThirdFragment)
	})

	t.Run("keep_all_fragments", func(t *testing.T) {
		ub := newUnreadyBlocks()
		ub.disjointFragments = []*Fragment{
			NewFragment([]*types.BlockData{
				{
					Header: &types.Header{
						Number: 101,
					},
				},
			}),
			NewFragment([]*types.BlockData{
				{
					Header: &types.Header{
						Number: 103,
					},
				},
			}),
			NewFragment([]*types.BlockData{
				{
					Header: &types.Header{
						Number: 104,
					},
				},
			}),
		}
		ub.pruneDisjointFragments(LowerThanOrEq(100))
		require.Len(t, ub.disjointFragments, 3)
	})
}
