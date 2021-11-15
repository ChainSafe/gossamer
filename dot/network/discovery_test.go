// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"
	badger "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"

	"github.com/stretchr/testify/require"
)

func newTestDiscovery(t *testing.T, num int) []*discovery {
	t.Helper()
	var discs []*discovery
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        uint16(7001 + i),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		srvc := createTestService(t, config)

		opts := badger.DefaultOptions
		opts.InMemory = true

		ds, err := badger.NewDatastore("", &opts)
		require.NoError(t, err)
		disc := &discovery{
			ctx: srvc.ctx,
			h:   srvc.host.h,
			ds:  ds,
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

// Set `NoMDNS` to true and test routing via kademlia DHT service.
func TestKadDHT(t *testing.T) {
	if testing.Short() {
		return
	}

	// setup 3 nodes
	nodes := newTestDiscovery(t, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// connects node 0 and node 2
	connectNoSync(t, ctx, nodes[2], nodes[0])

	time.Sleep(startDHTTimeout + 1)

	// node 2 doesnt know about node 1 then should return error
	_, err := nodes[2].dht.FindPeer(ctx, nodes[1].h.ID())
	require.ErrorIs(t, err, routing.ErrNotFound)

	// connects node 1 and node 0
	connectNoSync(t, ctx, nodes[1], nodes[0])

	time.Sleep(startDHTTimeout + 1)

	// node 2 should know node 1 because both are connected to 0
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
