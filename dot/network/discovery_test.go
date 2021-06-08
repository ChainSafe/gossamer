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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	kbucket "github.com/libp2p/go-libp2p-kbucket"

	"github.com/stretchr/testify/require"
)

func newTestDiscovery(t *testing.T, num int) []*discovery {
	t.Helper()
	var discs []*discovery
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        uint32(7001 + i),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		srvc := createTestService(t, config)
		disc := &discovery{
			ctx: srvc.ctx,
			h:   srvc.host.h,
		}
		go disc.start()
		discs = append(discs, disc)
	}
	return discs
}

// nolint
func connectNoSync(t *testing.T, ctx context.Context, a, b *discovery) {
	t.Helper()

	idB := b.h.ID()
	addrB := b.h.Peerstore().Addrs(idB)
	if len(addrB) == 0 {
		t.Fatal("peers setup incorrectly: no local address")
	}

	a.h.Peerstore().AddAddrs(idB, addrB, time.Minute)
	pi := peer.AddrInfo{ID: idB}

	err := a.h.Connect(ctx, pi)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = a.h.Connect(ctx, pi)
	}
	require.NoError(t, err)
}

// nolint
func wait(t *testing.T, ctx context.Context, a, b *dual.DHT) {
	t.Helper()

	// Loop until connection notification has been received.
	// Under high load, this may not happen as immediately as we would like.
	for a.LAN.RoutingTable().Find(b.LAN.PeerID()) == "" {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond * 5):
		}
	}
}

// Set `NoMDNS` to true and test routing via kademlia DHT service.
func TestKadDHT(t *testing.T) {
	if testing.Short() {
		return
	}

	nodes := newTestDiscovery(t, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := nodes[2].dht.FindPeer(ctx, nodes[1].h.ID())
	require.Equal(t, err, kbucket.ErrLookupFailure)

	connectNoSync(t, ctx, nodes[1], nodes[0])
	connectNoSync(t, ctx, nodes[2], nodes[0])

	// Can't use `connect` because b and c are only clients.
	wait(t, ctx, nodes[1].dht, nodes[0].dht)
	wait(t, ctx, nodes[2].dht, nodes[0].dht)

	_, err = nodes[2].dht.FindPeer(ctx, nodes[1].h.ID())
	require.NoError(t, err)
}

func TestBeginDiscovery(t *testing.T) {
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	err = nodeA.host.discovery.start()
	require.NoError(t, err)

	err = nodeB.host.discovery.start()
	require.NoError(t, err)
}

func TestBeginDiscovery_ThreeNodes(t *testing.T) {
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	configC := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeC"),
		Port:        7003,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeC := createTestService(t, configC)
	nodeC.noGossip = true

	// connect A and B
	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	// connect A and C
	addrInfoC := nodeC.host.addrInfo()
	err = nodeA.host.connect(addrInfoC)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoC)
	}
	require.NoError(t, err)

	// begin advertising and discovery for all nodes
	err = nodeA.host.discovery.start()
	require.NoError(t, err)

	err = nodeB.host.discovery.start()
	require.NoError(t, err)

	err = nodeC.host.discovery.start()
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	// assert B and C can discover each other
	addrs := nodeB.host.h.Peerstore().Addrs(nodeC.host.id())
	require.NotEqual(t, 0, len(addrs))
}
