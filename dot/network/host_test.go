// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"net"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

func TestExternalAddrs(t *testing.T) {
	config := &Config{
		BasePath:    utils.NewTestBasePath(t, "node"),
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)

	addrInfo := node.host.addrInfo()
	privateIPs := ma.NewFilters()
	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr) //nolint
		require.NoError(t, err)
		privateIPs.AddFilter(*ipnet, ma.ActionDeny)
	}

	for _, addr := range addrInfo.Addrs {
		require.False(t, privateIPs.AddrBlocked(addr))
	}
}

// test host connect method
func TestConnect(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")

	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	if peerCountA != 1 {
		t.Error(
			"node A does not have expected peer count",
			"\nexpected:", 1,
			"\nreceived:", peerCountA,
		)
	}

	if peerCountB != 1 {
		t.Error(
			"node B does not have expected peer count",
			"\nexpected:", 1,
			"\nreceived:", peerCountB,
		)
	}
}

// test host bootstrap method on start
func TestBootstrap(t *testing.T) {
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

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	if peerCountA == 0 {
		// check peerstore for disconnected peers
		peerCountA := len(nodeA.host.h.Peerstore().Peers())
		if peerCountA == 0 {
			t.Error(
				"node A does not have expected peer count",
				"\nexpected:", "not zero",
				"\nreceived:", peerCountA,
			)
		}
	}

	if peerCountB == 0 {
		// check peerstore for disconnected peers
		peerCountB := len(nodeB.host.h.Peerstore().Peers())
		if peerCountB == 0 {
			t.Error(
				"node B does not have expected peer count",
				"\nexpected:", "not zero",
				"\nreceived:", peerCountB,
			)
		}
	}
}

// test host send method
func TestSend(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handler := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	_, err = nodeA.host.send(addrInfoB.ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)

	msg, ok := handler.messages[nodeA.host.id()]
	require.True(t, ok)
	require.Equal(t, 1, len(msg))
	require.Equal(t, testBlockRequestMessage, msg[0])
}

// test host send method with existing stream
func TestExistingStream(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	handlerA := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeA.host.registerStreamHandler(nodeA.host.protocolID, handlerA.handleStream)

	addrInfoA := nodeA.host.addrInfo()
	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handlerB := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handlerB.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	// node A opens the stream to send the first message
	stream, err := nodeA.host.send(addrInfoB.ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	// node A uses the stream to send a second message
	err = nodeA.host.writeToStream(stream, testBlockRequestMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	// node B opens the stream to send the first message
	stream, err = nodeB.host.send(addrInfoA.ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerA.messages[nodeB.host.id()], "node A timeout waiting for message from node B")

	// node B uses the stream to send a second message
	err = nodeB.host.writeToStream(stream, testBlockRequestMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerA.messages[nodeB.host.id()], "node A timeout waiting for message from node B")
}

func TestStreamCloseMetadataCleanup(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	handlerA := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeA.host.registerStreamHandler(blockAnnounceID, handlerA.handleStream)

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handlerB := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(blockAnnounceID, handlerB.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	testHandshake := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     nodeB.blockState.GenesisHash(),
	}

	// node A opens the stream to send the first message
	_, err = nodeA.host.send(nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID, testHandshake)
	require.NoError(t, err)

	info := nodeA.notificationsProtocols[BlockAnnounceMsgType]

	// Set handshake data to received
	info.inboundHandshakeData.Store(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
	})

	// Verify that handshake data exists.
	_, ok := info.getInboundHandshakeData(nodeB.host.id())
	require.True(t, ok)

	time.Sleep(time.Second)
	nodeB.host.close()

	// Wait for cleanup
	time.Sleep(time.Second)

	// Verify that handshake data is cleared.
	_, ok = info.getInboundHandshakeData(nodeB.host.id())
	require.False(t, ok)
}

func Test_PeerSupportsProtocol(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	tests := []struct {
		protocol protocol.ID
		expect   bool
	}{
		{
			protocol: protocol.ID("/gossamer/test/0/sync/2"),
			expect:   true,
		},
		{
			protocol: protocol.ID("/gossamer/test/0/light/2"),
			expect:   true,
		},
		{
			protocol: protocol.ID("/gossamer/test/0/block-announces/1"),
			expect:   true,
		},
		{
			protocol: protocol.ID("/gossamer/test/0/transactions/1"),
			expect:   true,
		},
		{
			protocol: protocol.ID("/gossamer/not_supported/protocol"),
			expect:   false,
		},
	}

	for _, test := range tests {
		output, err := nodeA.host.supportsProtocol(nodeB.host.id(), test.protocol)
		require.NoError(t, err)
		require.Equal(t, test.expect, output)
	}
}

func Test_AddReservedPeers(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	nodeBPeerAddr := nodeB.host.multiaddrs()[0].String()
	err := nodeA.host.addReservedPeers(nodeBPeerAddr)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
}

func Test_RemoveReservedPeers(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	nodeBPeerAddr := nodeB.host.multiaddrs()[0].String()
	err := nodeA.host.addReservedPeers(nodeBPeerAddr)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	pID := nodeB.host.addrInfo().ID.String()

	err = nodeA.host.removeReservedPeers(pID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	isProtected := nodeA.host.h.ConnManager().IsProtected(nodeB.host.addrInfo().ID, "")
	require.False(t, isProtected)

	err = nodeA.host.removeReservedPeers("failing peer ID")
	require.Error(t, err)
}

func TestStreamCloseEOF(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handler := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handler.handleStream)
	require.False(t, handler.exit)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := nodeA.host.send(addrInfoB.ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)
	require.False(t, handler.exit)

	err = stream.Close()
	require.NoError(t, err)

	time.Sleep(TestBackoffTimeout)

	require.True(t, handler.exit)
}

// Test to check the nodes connection by peer set manager
func TestPeerConnect(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    2,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	nodeA.host.h.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
	nodeA.host.cm.peerSetHandler.AddPeer(0, addrInfoB.ID)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	require.Equal(t, 1, nodeB.host.peerCount())
}

// Test to check banned peer disconnection by peer set manager
func TestBannedPeer(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")

	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    2,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	nodeA.host.h.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
	nodeA.host.cm.peerSetHandler.AddPeer(0, addrInfoB.ID)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	require.Equal(t, 1, nodeB.host.peerCount())

	nodeA.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
		Value:  peerset.BannedThresholdValue - 1,
		Reason: peerset.BannedReason,
	}, addrInfoB.ID)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 0, nodeA.host.peerCount())
	require.Equal(t, 0, nodeB.host.peerCount())

	time.Sleep(3 * time.Second)

	require.Equal(t, 1, nodeA.host.peerCount())
	require.Equal(t, 1, nodeB.host.peerCount())
}

// Test to check reputation updated by peer set manager
func TestPeerReputation(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")

	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	nodeA.host.h.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
	nodeA.host.cm.peerSetHandler.AddPeer(0, addrInfoB.ID)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	require.Equal(t, 1, nodeB.host.peerCount())

	nodeA.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
		Value:  peerset.GoodTransactionValue,
		Reason: peerset.GoodTransactionReason,
	}, addrInfoB.ID)

	time.Sleep(100 * time.Millisecond)

	rep, err := nodeA.host.cm.peerSetHandler.PeerReputation(addrInfoB.ID)
	require.NoError(t, err)
	require.Greater(t, rep, int32(0))
}
