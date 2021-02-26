// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestMaxPeers(t *testing.T) {
	max := 3
	nodes := make([]*Service, max+2)

	for i := range nodes {
		config := &Config{
			Port:        7000 + uint32(i),
			RandSeed:    1 + int64(i),
			NoBootstrap: true,
			NoMDNS:      true,
			MaxPeers:    max,
		}
		node := createTestService(t, config)
		defer node.Stop()
		nodes[i] = node
	}

	addrs := nodes[0].host.multiaddrs()
	ainfo, err := peer.AddrInfoFromP2pAddr(addrs[0])
	require.NoError(t, err)

	for i, n := range nodes {
		if i == 0 {
			// connect other nodes to first node
			continue
		}

		err = n.host.connect(*ainfo)
		require.NoError(t, err, i)
	}

	p := nodes[0].host.h.Peerstore().Peers()
	require.LessOrEqual(t, max, len(p))
}

func TestProtectUnprotectPeer(t *testing.T) {
	cm := newConnManager(1, 4)
	require.Zero(t, len(cm.protectedPeerMap))

	p1 := peer.ID("a")
	p2 := peer.ID("b")
	p3 := peer.ID("c")
	p4 := peer.ID("d")

	cm.Protect(p1, "")
	cm.Protect(p2, "")

	require.True(t, cm.IsProtected(p1, ""))
	require.True(t, cm.IsProtected(p2, ""))

	unprot := cm.unprotectedPeers([]peer.ID{p1, p2, p3, p4})
	require.Equal(t, unprot, []peer.ID{p3, p4})

	cm.Unprotect(p1, "")
	cm.Unprotect(p2, "")

	unprot = cm.unprotectedPeers([]peer.ID{p1, p2, p3, p4})
	require.Equal(t, unprot, []peer.ID{p1, p2, p3, p4})
}
