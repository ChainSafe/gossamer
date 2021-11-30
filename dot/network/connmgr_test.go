// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/utils"
)

func TestMinPeers(t *testing.T) {
	const min = 1

	nodes := make([]*Service, 2)
	for i := range nodes {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        7000 + uint16(i),
			NoBootstrap: true,
			NoMDNS:      true,
		}
		node := createTestService(t, config)
		nodes[i] = node
	}

	addrs := nodes[0].host.multiaddrs()[0]
	addrs1 := nodes[1].host.multiaddrs()[0]

	configB := &Config{
		BasePath:  utils.NewTestBasePath(t, "nodeB"),
		Port:      7002,
		Bootnodes: []string{addrs.String(), addrs1.String()},
		NoMDNS:    true,
		MinPeers:  min,
	}

	nodeB := createTestService(t, configB)
	require.Equal(t, min, nodeB.host.peerCount())

	nodeB.host.cm.peerSetHandler.DisconnectPeer(0, nodes[0].host.id())
	require.GreaterOrEqual(t, min, nodeB.host.peerCount())
}

func TestMaxPeers(t *testing.T) {
	const max = 3
	nodes := make([]*Service, max+2)
	for i := range nodes {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        7000 + uint16(i),
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

		n.host.h.Peerstore().AddAddrs(ainfo.ID, ainfo.Addrs, peerstore.PermanentAddrTTL)
		n.host.cm.peerSetHandler.AddPeer(0, ainfo.ID)
	}

	time.Sleep(200 * time.Millisecond)
	p := nodes[0].host.h.Peerstore().Peers()
	require.LessOrEqual(t, max, len(p))
}

func TestProtectUnprotectPeer(t *testing.T) {
	const (
		min                = 1
		max                = 4
		slotAllocationTime = time.Second * 2
	)

	peerCfgSet := peerset.NewConfigSet(uint32(max-min), uint32(max), false, slotAllocationTime)
	cm, err := newConnManager(min, max, peerCfgSet)
	require.NoError(t, err)

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

func TestPersistentPeers(t *testing.T) {
	if testing.Short() {
		t.Skip() // this sometimes fails on CI
	}

	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "node-a"),
		Port:        7000,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	nodeA := createTestService(t, configA)

	addrs := nodeA.host.multiaddrs()
	configB := &Config{
		BasePath:        utils.NewTestBasePath(t, "node-b"),
		Port:            7001,
		NoMDNS:          true,
		PersistentPeers: []string{addrs[0].String()},
	}
	nodeB := createTestService(t, configB)

	time.Sleep(time.Millisecond * 600)
	// B should have connected to A during bootstrap
	conns := nodeB.host.h.Network().ConnsToPeer(nodeA.host.id())
	require.NotEqual(t, 0, len(conns))

	// if A disconnects from B, B should reconnect
	nodeA.host.cm.peerSetHandler.DisconnectPeer(0, nodeB.host.id())

	time.Sleep(time.Millisecond * 500)

	conns = nodeB.host.h.Network().ConnsToPeer(nodeA.host.id())
	require.NotEqual(t, 0, len(conns))
}

func TestRemovePeer(t *testing.T) {
	if testing.Short() {
		t.Skip() // this sometimes fails on CI
	}

	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	addrA := nodeA.host.multiaddrs()[0]

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:  basePathB,
		Port:      7002,
		Bootnodes: []string{addrA.String()},
		NoMDNS:    true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	time.Sleep(time.Millisecond * 600)

	// nodeB will be connected to nodeA through bootnodes.
	require.Equal(t, 1, nodeB.host.peerCount())

	nodeB.host.cm.peerSetHandler.RemovePeer(0, nodeA.host.id())
	time.Sleep(time.Millisecond * 200)

	require.Equal(t, 0, nodeB.host.peerCount())
}

func TestSetReservedPeer(t *testing.T) {
	if testing.Short() {
		t.Skip() // this sometimes fails on CI
	}

	nodes := make([]*Service, 3)
	for i := range nodes {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        7000 + uint16(i),
			NoBootstrap: true,
			NoMDNS:      true,
		}
		node := createTestService(t, config)
		nodes[i] = node
	}

	addrA := nodes[0].host.multiaddrs()[0]
	addrB := nodes[1].host.multiaddrs()[0]
	addrC := nodes[2].host.addrInfo()

	basePathD := utils.NewTestBasePath(t, "node3")
	config := &Config{
		BasePath:        basePathD,
		Port:            7004,
		NoMDNS:          true,
		PersistentPeers: []string{addrA.String(), addrB.String()},
	}

	node3 := createTestService(t, config)
	node3.noGossip = true
	time.Sleep(time.Millisecond * 600)

	require.Equal(t, 2, node3.host.peerCount())

	node3.host.h.Peerstore().AddAddrs(addrC.ID, addrC.Addrs, peerstore.PermanentAddrTTL)
	node3.host.cm.peerSetHandler.SetReservedPeer(0, addrC.ID)
	time.Sleep(200 * time.Millisecond)

	// reservedOnly mode is not yet implemented, so nodeA and nodeB won't be disconnected (#1888).
	// TODO: once reservedOnly mode is implemented and reservedOnly is set to true, change expected value to 1 (nodeC)
	require.Equal(t, 3, node3.host.peerCount())
}
