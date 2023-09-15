//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"testing"
	"time"

	badger "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/stretchr/testify/require"
)

func newTestDiscovery(t *testing.T, num int) []*discovery {
	t.Helper()

	var discs []*discovery
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
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
			h:   srvc.host.p2pHost,
			ds:  ds,
			pid: protocol.ID("/testing"),
		}

		go disc.start()
		discs = append(discs, disc)
	}

	return discs
}

func connectNoSync(ctx context.Context, t *testing.T, a, b *discovery) {
	t.Helper()

	idB := b.h.ID()
	addrB := b.h.Peerstore().Addrs(idB)
	require.NotEqual(t, 0, len(addrB), "peers setup incorrectly: no local address")

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
	t.Parallel()

	// setup 3 nodes
	nodes := newTestDiscovery(t, 3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// connects node 0 and node 2
	connectNoSync(ctx, t, nodes[2], nodes[0])

	time.Sleep(startDHTTimeout + 1)

	// node 0 doesnt know about node 1 then should return error
	_, err := nodes[0].dht.FindPeer(ctx, nodes[1].h.ID())
	require.ErrorIs(t, err, routing.ErrNotFound)

	// connects node 2 and node 1
	connectNoSync(ctx, t, nodes[2], nodes[1])

	time.Sleep(startDHTTimeout + 1)

	// node 0 should know node 1 because both are connected to 2
	_, err = nodes[0].dht.FindPeer(ctx, nodes[1].h.ID())
	require.NoError(t, err)
}

func TestBeginDiscovery(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := addrInfo(nodeB.host)
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
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	configC := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeC := createTestService(t, configC)
	nodeC.noGossip = true

	// connect A and B
	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	// connect A and C
	addrInfoC := addrInfo(nodeC.host)
	err = nodeA.host.connect(addrInfoC)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoC)
	}
	require.NoError(t, err)

	err = nodeB.host.discovery.start()
	require.NoError(t, err)

	err = nodeC.host.discovery.start()
	require.NoError(t, err)

	// begin advertising and discovery for all nodes
	err = nodeA.host.discovery.start()
	require.NoError(t, err)

	time.Sleep(time.Second)

	// assert B and C can discover each other
	addrs := nodeB.host.p2pHost.Peerstore().Addrs(nodeC.host.id())
	require.NotEqual(t, 0, len(addrs))

}
