//go:build integration
// +build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalAddrs(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)

	addrInfo := addrInfo(node.host)

	privateIPs, err := newPrivateIPFilters()
	require.NoError(t, err)

	for _, addr := range addrInfo.Addrs {
		require.False(t, privateIPs.AddrBlocked(addr))
	}
}

func mustNewMultiAddr(s string) (a ma.Multiaddr) {
	a, err := ma.NewMultiaddr(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestExternalAddrsPublicIP(t *testing.T) {
	t.Parallel()

	port := availablePort(t)
	config := &Config{
		BasePath:    t.TempDir(),
		PublicIP:    "10.0.5.2",
		Port:        port,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)
	addrInfo := addrInfo(node.host)

	privateIPs, err := newPrivateIPFilters()
	require.NoError(t, err)

	for i, addr := range addrInfo.Addrs {
		switch i {
		case len(addrInfo.Addrs) - 1:
			// would be blocked by privateIPs, but this address injected from Config.PublicIP
			require.True(t, privateIPs.AddrBlocked(addr))
		default:
			require.False(t, privateIPs.AddrBlocked(addr))
		}
	}

	expected := []ma.Multiaddr{
		mustNewMultiAddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		mustNewMultiAddr(fmt.Sprintf("/ip4/10.0.5.2/tcp/%d", port)),
	}
	assert.Equal(t, addrInfo.Addrs, expected)
}

func TestExternalAddrsPublicDNS(t *testing.T) {
	config := &Config{
		BasePath:    t.TempDir(),
		PublicDNS:   "alice",
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)
	addrInfo := addrInfo(node.host)

	expected := []ma.Multiaddr{
		mustNewMultiAddr("/ip4/127.0.0.1/tcp/7001"),
		mustNewMultiAddr("/dns/alice/tcp/7001"),
	}
	assert.Equal(t, addrInfo.Addrs, expected)

}

// test host connect method
func TestConnect(t *testing.T) {
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
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	require.Equal(t, 1, peerCountA)
	require.Equal(t, 1, peerCountB)
}

// test host bootstrap method on start
func TestBootstrap(t *testing.T) {
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

	peerCountA := nodeA.host.peerCount()
	if peerCountA == 0 {
		peerCountA := len(nodeA.host.p2pHost.Peerstore().Peers())
		require.NotZero(t, peerCountA)
	}

	peerCountB := nodeB.host.peerCount()
	if peerCountB == 0 {
		peerCountB := len(nodeB.host.p2pHost.Peerstore().Peers())
		require.NotZero(t, peerCountB)
	}
}

// test host send method
func TestSend(t *testing.T) {
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
	handler := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handler.handleStream)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	testBlockReqMessage := newTestBlockRequestMessage(t)
	_, err = nodeA.host.send(addrInfoB.ID, nodeB.host.protocolID, testBlockReqMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)

	msg, ok := handler.messages[nodeA.host.id()]
	require.True(t, ok)
	require.Equal(t, 1, len(msg))
	require.Equal(t, testBlockReqMessage, msg[0])
}

// test host send method with existing stream
func TestExistingStream(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	handlerA := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeA.host.registerStreamHandler(nodeA.host.protocolID, handlerA.handleStream)

	addrInfoA := addrInfo(nodeA.host)
	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handlerB := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handlerB.handleStream)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	testBlockReqMessage := newTestBlockRequestMessage(t)

	// node A opens the stream to send the first message
	stream, err := nodeA.host.send(addrInfoB.ID, nodeB.host.protocolID, testBlockReqMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	// node A uses the stream to send a second message
	err = nodeA.host.writeToStream(stream, testBlockReqMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	// node B opens the stream to send the first message
	stream, err = nodeB.host.send(addrInfoA.ID, nodeB.host.protocolID, testBlockReqMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerA.messages[nodeB.host.id()], "node A timeout waiting for message from node B")

	// node B uses the stream to send a second message
	err = nodeB.host.writeToStream(stream, testBlockReqMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerA.messages[nodeB.host.id()], "node A timeout waiting for message from node B")
}

func TestStreamCloseMetadataCleanup(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	handlerA := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeA.host.registerStreamHandler(blockAnnounceID, handlerA.handleStream)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handlerB := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(blockAnnounceID, handlerB.handleStream)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	const (
		bestBlockNumber uint32 = 77
	)

	testHandshake := &BlockAnnounceHandshake{
		Roles:           common.AuthorityRole,
		BestBlockNumber: bestBlockNumber,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     nodeB.blockState.GenesisHash(),
	}

	// node A opens the stream to send the first message
	_, err = nodeA.host.send(nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID, testHandshake)
	require.NoError(t, err)

	info := nodeA.notificationsProtocols[blockAnnounceMsgType]

	// Set handshake data to received
	info.peersData.setInboundHandshakeData(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
	})

	// Verify that handshake data exists.
	data := info.peersData.getInboundHandshakeData(nodeB.host.id())
	require.NotNil(t, data)

	nodeB.host.close()

	// Verify that handshake data is cleared.
	data = info.peersData.getInboundHandshakeData(nodeB.host.id())
	require.Nil(t, data)
}

func Test_PeerSupportsProtocol(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)

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

	nodeBPeerAddr := nodeB.host.multiaddrs()[0].String()
	err := nodeA.host.addReservedPeers(nodeBPeerAddr)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
}

func Test_RemoveReservedPeers(t *testing.T) {
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

	nodeBPeerAddr := nodeB.host.multiaddrs()[0].String()
	err := nodeA.host.addReservedPeers(nodeBPeerAddr)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	pID := addrInfo(nodeB.host).ID.String()

	err = nodeA.host.removeReservedPeers(pID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	isProtected := nodeA.host.p2pHost.ConnManager().IsProtected(addrInfo(nodeB.host).ID, "")
	require.False(t, isProtected)

	err = nodeA.host.removeReservedPeers("unknown_perr_id")
	require.Error(t, err)
}

func TestStreamCloseEOF(t *testing.T) {
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
	handler := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handler.handleStream)
	require.False(t, handler.exit)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	testBlockReqMessage := newTestBlockRequestMessage(t)

	stream, err := nodeA.host.send(addrInfoB.ID, nodeB.host.protocolID, testBlockReqMessage)
	require.NoError(t, err)
	require.False(t, handler.exit)

	err = stream.Close()
	require.NoError(t, err)

	time.Sleep(TestBackoffTimeout)

	require.True(t, handler.exit)
}

// Test to check the nodes connection by peer set manager
func TestPeerConnect(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    2,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := addrInfo(nodeB.host)
	nodeA.host.p2pHost.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
	nodeA.host.cm.peerSetHandler.AddPeer(0, addrInfoB.ID)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	require.Equal(t, 1, nodeB.host.peerCount())
}

// Test to check banned peer disconnection by peer set manager
func TestBannedPeer(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    2,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := addrInfo(nodeB.host)
	nodeA.host.p2pHost.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
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
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    3,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := addrInfo(nodeB.host)
	nodeA.host.p2pHost.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
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
