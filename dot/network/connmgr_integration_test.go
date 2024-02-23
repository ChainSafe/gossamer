//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/peerset"
)

func TestMinPeers(t *testing.T) {
	t.Parallel()

	const min = 1

	nodes := make([]*Service, 2)
	for i := range nodes {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
		}
		node := createTestService(t, config)
		nodes[i] = node
	}

	addrs := nodes[0].host.multiaddrs()[0]
	addrs1 := nodes[1].host.multiaddrs()[0]

	configB := &Config{
		BasePath:  t.TempDir(),
		Port:      availablePort(t),
		Bootnodes: []string{addrs.String(), addrs1.String()},
		NoMDNS:    true,
		MinPeers:  min,
	}

	nodeB := createTestService(t, configB)
	require.GreaterOrEqual(t, nodeB.host.peerCount(), len(nodes))

	// check that peer count is at least greater than minimum number of peers,
	// even after trying to disconnect from all peers
	for _, node := range nodes {
		nodeB.host.cm.peerSetHandler.(*peerset.Handler).DisconnectPeer(0, node.host.id())
	}

	require.GreaterOrEqual(t, nodeB.host.peerCount(), min)
}

func TestMaxPeers(t *testing.T) {
	t.Parallel()

	const max = 3
	nodes := make([]*Service, max+2)

	for i := range nodes {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
			MaxPeers:    max,
		}
		node := createTestService(t, config)
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

		n.host.p2pHost.Peerstore().AddAddrs(ainfo.ID, ainfo.Addrs, peerstore.PermanentAddrTTL)
		n.host.cm.peerSetHandler.AddPeer(0, ainfo.ID)
	}

	time.Sleep(200 * time.Millisecond)
	p := nodes[0].host.p2pHost.Peerstore().Peers()
	require.LessOrEqual(t, max, len(p))
}

func TestProtectUnprotectPeer(t *testing.T) {
	t.Parallel()

	const (
		min                = 1
		max                = 4
		slotAllocationTime = time.Second * 2
	)

	peerCfgSet := peerset.NewConfigSet(uint32(max-min), uint32(max), false, slotAllocationTime)
	cm, err := newConnManager(max, peerCfgSet)
	require.NoError(t, err)

	p1 := peer.ID("a")
	p2 := peer.ID("b")
	p3 := peer.ID("c")
	p4 := peer.ID("d")

	cm.Protect(p1, "")
	cm.Protect(p2, "")

	require.True(t, cm.IsProtected(p1, ""))
	require.True(t, cm.IsProtected(p2, ""))

	unprot := unprotectedPeers(cm, []peer.ID{p1, p2, p3, p4})
	require.Equal(t, unprot, []peer.ID{p3, p4})

	cm.Unprotect(p1, "")
	cm.Unprotect(p2, "")

	unprot = unprotectedPeers(cm, []peer.ID{p1, p2, p3, p4})
	require.Equal(t, unprot, []peer.ID{p1, p2, p3, p4})
}

func TestPersistentPeers(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	nodeA := createTestService(t, configA)
	addrs := nodeA.host.multiaddrs()

	configB := &Config{
		BasePath:        t.TempDir(),
		Port:            availablePort(t),
		NoMDNS:          true,
		PersistentPeers: []string{addrs[0].String()},
	}
	nodeB := createTestService(t, configB)

	time.Sleep(time.Millisecond * 600)

	// B should have connected to A during bootstrap
	conns := nodeB.host.p2pHost.Network().ConnsToPeer(nodeA.host.id())
	require.NotEqual(t, 0, len(conns))

	// if A disconnects from B, B should reconnect
	nodeA.host.cm.peerSetHandler.(*peerset.Handler).DisconnectPeer(0, nodeB.host.id())

	time.Sleep(time.Millisecond * 500)

	conns = nodeB.host.p2pHost.Network().ConnsToPeer(nodeA.host.id())
	require.NotEqual(t, 0, len(conns))
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	addrA := nodeA.host.multiaddrs()[0]

	configB := &Config{
		BasePath:  t.TempDir(),
		Port:      availablePort(t),
		Bootnodes: []string{addrA.String()},
		NoMDNS:    true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	time.Sleep(time.Millisecond * 600)

	// nodeB will be connected to nodeA through bootnodes.
	require.Equal(t, 1, nodeB.host.peerCount())

	nodeB.host.cm.peerSetHandler.(*peerset.Handler).RemovePeer(0, nodeA.host.id())
	time.Sleep(time.Millisecond * 200)

	require.Equal(t, 0, nodeB.host.peerCount())
}

func TestSetReservedPeer(t *testing.T) {
	t.Parallel()

	nodes := make([]*Service, 3)
	for i := range nodes {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
		}
		node := createTestService(t, config)
		nodes[i] = node
	}

	addrA := nodes[0].host.multiaddrs()[0]
	addrB := nodes[1].host.multiaddrs()[0]
	addrC := addrInfo(nodes[2].host)

	config := &Config{
		BasePath:        t.TempDir(),
		Port:            availablePort(t),
		NoMDNS:          true,
		PersistentPeers: []string{addrA.String(), addrB.String()},
	}

	node3 := createTestService(t, config)
	node3.noGossip = true
	time.Sleep(time.Millisecond * 600)

	require.Equal(t, 2, node3.host.peerCount())

	node3.host.p2pHost.Peerstore().AddAddrs(addrC.ID, addrC.Addrs, peerstore.PermanentAddrTTL)
	node3.host.cm.peerSetHandler.(*peerset.Handler).SetReservedPeer(0, addrC.ID)
	time.Sleep(200 * time.Millisecond)

	// reservedOnly mode is not yet implemented, so nodeA and nodeB won't be disconnected (#1888).
	// TODO: once reservedOnly mode is implemented and reservedOnly is set to true, change expected value to 1 (nodeC)
	require.Equal(t, 3, node3.host.peerCount())
}
