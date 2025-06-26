package statementdistribution

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/grid"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionTopology(t *testing.T) {
	topology := newSessionTopologyView()
	assert.NotNil(t, topology)
	assert.NotNil(t, topology.groupViews)
	assert.Empty(t, topology.groupViews)
}

func TestBuildSessionTopology(t *testing.T) {
	t.Parallel()

	t.Run("our_index_nil", func(t *testing.T) {
		t.Parallel()

		baseTopology := &grid.SessionGridTopology{
			ShuffledIndices: []uint{0, 1, 2},
			CanonicalShuffling: []grid.TopologyPeerInfo{
				{
					ValidatorIndex: 0,
					DiscoveryID:    types.AuthorityID{},
					Peers:          []peer.ID{},
				},
				{
					ValidatorIndex: 1,
					DiscoveryID:    types.AuthorityID{},
					Peers:          []peer.ID{},
				},
				{
					ValidatorIndex: 2,
					DiscoveryID:    types.AuthorityID{},
					Peers:          []peer.ID{},
				},
			},
		}

		groups := [][]parachaintypes.ValidatorIndex{
			{0, 1, 2},
		}

		view, err := buildSessionTopology(groups, baseTopology, nil)

		require.NoError(t, err)
		assert.NotNil(t, view)
		assert.Empty(t, view.groupViews)
	})

	t.Run("topology_setup", func(t *testing.T) {
		t.Parallel()

		baseTopology := &grid.SessionGridTopology{
			ShuffledIndices:    []uint{0, 1, 2, 3, 4, 5, 6, 7, 8},
			CanonicalShuffling: make([]grid.TopologyPeerInfo, 9),
		}

		for i := 0; i < 9; i++ {
			baseTopology.CanonicalShuffling[i] = grid.TopologyPeerInfo{
				ValidatorIndex: parachaintypes.ValidatorIndex(i),
				DiscoveryID:    types.AuthorityID{},
				Peers:          []peer.ID{},
			}
		}

		groups := [][]parachaintypes.ValidatorIndex{
			{0, 3, 6},
			{4, 2, 7},
			{8, 5, 1},
		}

		ourIndex := parachaintypes.ValidatorIndex(0)

		view, err := buildSessionTopology(groups, baseTopology, &ourIndex)

		require.NoError(t, err)
		assert.NotNil(t, view)
		require.Len(t, view.groupViews, 3)

		// 0 1 2
		// 3 4 5
		// 6 7 8

		// our group: we send to all row/column neighbours which are not in our
		// group and receive nothing.
		require.Equal(
			t,
			map[parachaintypes.ValidatorIndex]struct{}{
				1: {},
				2: {},
			},
			view.groupViews[parachaintypes.GroupIndex(0)].sending,
		)
		require.Equal(
			t,
			map[parachaintypes.ValidatorIndex]struct{}{},
			view.groupViews[parachaintypes.GroupIndex(0)].receiving,
		)

		// we share a row with '2' and have indirect connections to '4' and '7'.

		require.Equal(
			t,
			map[parachaintypes.ValidatorIndex]struct{}{
				3: {},
				6: {},
			},
			view.groupViews[parachaintypes.GroupIndex(1)].sending,
		)
		require.Equal(
			t,
			map[parachaintypes.ValidatorIndex]struct{}{
				1: {},
				2: {},
				3: {},
				6: {},
			},
			view.groupViews[parachaintypes.GroupIndex(1)].receiving,
		)

		// we share a row with '1' and have indirect connections to '5' and '8'.

		require.Equal(
			t,
			map[parachaintypes.ValidatorIndex]struct{}{
				3: {},
				6: {},
			},
			view.groupViews[parachaintypes.GroupIndex(2)].sending,
		)
		require.Equal(
			t,
			map[parachaintypes.ValidatorIndex]struct{}{
				1: {},
				2: {},
				3: {},
				6: {},
			},
			view.groupViews[parachaintypes.GroupIndex(2)].receiving,
		)
	})
}

func TestSessionTopologyView_IterSendingForGroup(t *testing.T) {
	t.Parallel()

	t.Run("non_existent_group", func(t *testing.T) {
		t.Parallel()

		view := newSessionTopologyView()

		var results []parachaintypes.ValidatorIndex
		for i := range view.iterSendingForGroup(42, full) {
			results = append(results, i)
		}

		// Should be empty since group doesn't exist
		require.Empty(t, results)
	})

	t.Run("manifestKind_full", func(t *testing.T) {
		t.Parallel()

		view := newSessionTopologyView()

		groupIdx := parachaintypes.GroupIndex(1)
		view.groupViews[groupIdx] = groupSubView{
			sending: map[parachaintypes.ValidatorIndex]struct{}{
				5: {},
				7: {},
			},
			receiving: map[parachaintypes.ValidatorIndex]struct{}{
				1: {},
				2: {},
				3: {},
			},
		}

		var results []parachaintypes.ValidatorIndex
		for i := range view.iterSendingForGroup(groupIdx, full) {
			results = append(results, i)
		}

		// For full manifest kind, should return indices from receiving map
		require.Len(t, results, 3)
		require.ElementsMatch(t, []parachaintypes.ValidatorIndex{1, 2, 3}, results)
	})

	t.Run("manifestKind_acknowledgement", func(t *testing.T) {
		t.Parallel()

		view := newSessionTopologyView()

		groupIdx := parachaintypes.GroupIndex(1)
		view.groupViews[groupIdx] = groupSubView{
			sending: map[parachaintypes.ValidatorIndex]struct{}{
				5: {},
				7: {},
			},
			receiving: map[parachaintypes.ValidatorIndex]struct{}{
				1: {},
				2: {},
				3: {},
			},
		}

		var results []parachaintypes.ValidatorIndex
		for i := range view.iterSendingForGroup(groupIdx, acknowledgement) {
			results = append(results, i)
		}

		// For acknowledgement kind, should return indices from sending map
		require.Len(t, results, 2)
		require.ElementsMatch(t, []parachaintypes.ValidatorIndex{5, 7}, results)
	})
}
