package grid_test

import (
	"github.com/ChainSafe/gossamer/dot/parachain/grid"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"testing"
)

// FixtureTopologyPeerInfo returns a slice of 11 TopologyPeerInfo for testing purposes.
func FixtureTopologyPeerInfo() []grid.TopologyPeerInfo {
	return []grid.TopologyPeerInfo{
		{
			Peers:          []peer.ID{"peer1", "peer2"},
			ValidatorIndex: parachaintypes.ValidatorIndex(0),
			DiscoveryID:    types.AuthorityID{1},
		},
		{
			Peers:          []peer.ID{"peer3", "peer4"},
			ValidatorIndex: parachaintypes.ValidatorIndex(1),
			DiscoveryID:    types.AuthorityID{2},
		},
		{
			Peers:          []peer.ID{"peer5", "peer6"},
			ValidatorIndex: parachaintypes.ValidatorIndex(2),
			DiscoveryID:    types.AuthorityID{3},
		},
		{
			Peers:          []peer.ID{"peer7", "peer8"},
			ValidatorIndex: parachaintypes.ValidatorIndex(3),
			DiscoveryID:    types.AuthorityID{4},
		},
		{
			Peers:          []peer.ID{"peer9", "peer10"},
			ValidatorIndex: parachaintypes.ValidatorIndex(4),
			DiscoveryID:    types.AuthorityID{5},
		},
		{
			Peers:          []peer.ID{"peer11", "peer12"},
			ValidatorIndex: parachaintypes.ValidatorIndex(5),
			DiscoveryID:    types.AuthorityID{6},
		},
		{
			Peers:          []peer.ID{"peer13", "peer14"},
			ValidatorIndex: parachaintypes.ValidatorIndex(6),
			DiscoveryID:    types.AuthorityID{7},
		},
		{
			Peers:          []peer.ID{"peer15", "peer16"},
			ValidatorIndex: parachaintypes.ValidatorIndex(7),
			DiscoveryID:    types.AuthorityID{8},
		},
		{
			Peers:          []peer.ID{"peer17", "peer18"},
			ValidatorIndex: parachaintypes.ValidatorIndex(8),
			DiscoveryID:    types.AuthorityID{9},
		},
		{
			Peers:          []peer.ID{"peer19", "peer20"},
			ValidatorIndex: parachaintypes.ValidatorIndex(9),
			DiscoveryID:    types.AuthorityID{10},
		},
		{
			Peers:          []peer.ID{"peer21", "peer22"},
			ValidatorIndex: parachaintypes.ValidatorIndex(10),
			DiscoveryID:    types.AuthorityID{11},
		},
	}
}

func Test_SessionGridTopology(t *testing.T) {
	gt := grid.NewSessionGridTopology([]uint{1, 2, 3}, []grid.TopologyPeerInfo{grid.TopologyPeerInfo{
		Peers:          []peer.ID{"peer1", "peer2"},
		ValidatorIndex: parachaintypes.ValidatorIndex(1),
		DiscoveryID:    types.AuthorityID{1},
	},
	})
	assert.Equal(t, 2, len(gt.Peers))

	updated := gt.UpdateAuthoritiesIDs(peer.ID("peer2"), map[types.AuthorityID]struct{}{types.AuthorityID{1}: {}})
	assert.False(t, updated)

	updated = gt.UpdateAuthoritiesIDs(peer.ID("peer3"), map[types.AuthorityID]struct{}{types.AuthorityID{1}: {}})
	assert.True(t, updated)
	assert.Equal(t, 3, len(gt.Peers))
	assert.Equal(t,
		peer.IDSlice(peer.IDSlice{"peer1", "peer2", "peer3"}),
		gt.CanonicalShuffling[0].Peers,
	)
}

func Test_SessionGridTopologyNeighbors(t *testing.T) {
	gt := grid.NewSessionGridTopology([]uint{1, 2, 3}, []grid.TopologyPeerInfo{grid.TopologyPeerInfo{
		Peers:          []peer.ID{"peer1", "peer2"},
		ValidatorIndex: parachaintypes.ValidatorIndex(1),
		DiscoveryID:    types.AuthorityID{1},
	},
	})
	_, err := gt.ComputeGridNeighborsFor(1)
	assert.NotNil(t, err)

	gt = grid.NewSessionGridTopology([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, FixtureTopologyPeerInfo())
	gn, err := gt.ComputeGridNeighborsFor(10)
	assert.Nil(t, err)
	assert.Equal(t, len(gn.ValidatorIndicesRow), 1)
	assert.Equal(t, len(gn.ValidatorIndicesCol), 3)
	assert.Equal(t, len(gn.PeersRow), 2)
	assert.Equal(t, len(gn.PeersCol), 6)

}

func Test_MatrixNeighbors(t *testing.T) {
	m, err := grid.CalculateMatrixNeighbors(10, 11)
	assert.Nil(t, err)
	assert.Equal(t, len(m.ColumnNeighbors), 3)
	assert.Equal(t, len(m.RowNeighbors), 1)

	m, err = grid.CalculateMatrixNeighbors(4, 12)
	assert.Nil(t, err)
	assert.Equal(t, len(m.ColumnNeighbors), 3)
	assert.Equal(t, len(m.RowNeighbors), 2)
}

func Test_GridNeighbors(t *testing.T) {
	// e.g. for size 11 the matrix would be
	//
	// 0  1  2
	// 3  4  5
	// 6  7  8
	// 9 10
	//
	// and for index 10, the neighbors would be 1, 4, 7, 9
	gt := grid.NewSessionGridTopology([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, FixtureTopologyPeerInfo())
	gn, err := gt.ComputeGridNeighborsFor(10)
	assert.Nil(t, err)
	assert.Equal(t, len(gn.ValidatorIndicesRow), 1)
	assert.Equal(t, len(gn.ValidatorIndicesCol), 3)
	assert.Equal(t, len(gn.PeersRow), 2)
	assert.Equal(t, len(gn.PeersCol), 6)
	assert.Nil(t, err)
	routing := gn.RequiredRoutingByIndex(4, false)
	// The origin of the message was from column so we should rout in a row
	assert.Equal(t, grid.RequiredRoutingGridX, routing)

	// Since routing is for rows we have only validator with index 9 there and its peers are peer19 and peer20
	assert.False(t, gn.ShouldRouteToPeer(grid.RequiredRoutingGridX, peer.ID("peer1")))
	assert.True(t, gn.ShouldRouteToPeer(grid.RequiredRoutingGridX, peer.ID("peer19")))
	assert.True(t, gn.ShouldRouteToPeer(grid.RequiredRoutingGridX, peer.ID("peer20")))
}

func Test_SessionGridTopologyEntry(t *testing.T) {
	gt := grid.NewSessionGridTopology([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, FixtureTopologyPeerInfo())
	gn, err := gt.ComputeGridNeighborsFor(10)
	assert.Nil(t, err)
	sgte := &grid.SessionGridTopologyEntry{
		Topology:       gt,
		LocalNeighbors: gn,
		LocalIndex:     10,
		SessionIndex:   99,
	}
	// Since routing is for rows we have only validator with index 9 there and its peers are peer19 and peer20
	assert.Equal(t, len(sgte.PeersToRoute(grid.RequiredRoutingGridX)), 2)
	// Since routing is for rows we have only validator with index 1,4,7 there and its peers are peer19 and peer20
	//[]peer.ID{"peer3", "peer4", "peer9", "peer10", "peer15", "peer16"},
	assert.Equal(t, len(sgte.PeersToRoute(grid.RequiredRoutingGridY)), 6)
	assert.Equal(t, len(sgte.PeersToRoute(grid.RequiredRoutingAll)), 22)

	updated, err := sgte.UpdateAuthoritiesIDs(peer.ID("peer99"), map[types.AuthorityID]struct{}{types.AuthorityID{10}: {}})
	assert.True(t, updated)
	assert.Nil(t, err)
	// Now we added one more peer to validator with AuthorityID 10, index 9. Sine we are actins as validator with index 10
	// we should route to this peer if strategy RequiredRoutingGridX
	assert.Equal(t, len(sgte.PeersToRoute(grid.RequiredRoutingGridX)), 3)
}

func Test_SessionGridTopologyStorage(t *testing.T) {
	gt := grid.NewSessionGridTopology([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, FixtureTopologyPeerInfo())
	gn, err := gt.ComputeGridNeighborsFor(10)
	assert.Nil(t, err)
	sgteCurrent := &grid.SessionGridTopologyEntry{
		Topology:       gt,
		LocalNeighbors: gn,
		LocalIndex:     10,
		SessionIndex:   99,
	}
	sgtePrev := &grid.SessionGridTopologyEntry{
		Topology:       gt,
		LocalNeighbors: gn,
		LocalIndex:     10,
		SessionIndex:   98,
	}
	storage := grid.SessionGridTopologyStorage{
		CurrentTopology: sgteCurrent,
		PrevTopology:    sgtePrev,
	}

	err = storage.UpdateCurrentTopology(100, gt, 10)
	assert.Nil(t, err)
	assert.Equal(t, storage.PrevTopology.SessionIndex, parachaintypes.SessionIndex(99))
	assert.Equal(t, storage.CurrentTopology.SessionIndex, parachaintypes.SessionIndex(100))
}
