// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"regexp"
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

	addrInfo := node.host.addrInfo()

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

	config := &Config{
		BasePath:    t.TempDir(),
		PublicIP:    "10.0.5.2",
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)
	addrInfo := node.host.addrInfo()

	privateIPs, err := newPrivateIPFilters()
	require.NoError(t, err)

	multiAddrRegex := regexp.MustCompile("^/ip4/(127.0.0.1|10.0.5.2)/tcp/[0-9]+$")

	for i, addr := range addrInfo.Addrs {
		switch i {
		case len(addrInfo.Addrs) - 1:
			// would be blocked by privateIPs, but this address injected from Config.PublicIP
			require.True(t, privateIPs.AddrBlocked(addr))
		default:
			require.False(t, privateIPs.AddrBlocked(addr))
		}

		require.True(t, multiAddrRegex.MatchString(addr.String()))
	}
}

func TestExternalAddrsPublicDNS(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		PublicDNS:   "alice",
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)
	addrInfo := node.host.addrInfo()

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
		peerCountA := len(nodeA.host.h.Peerstore().Peers())
		require.NotZero(t, peerCountA)
	}

	peerCountB := nodeB.host.peerCount()
	if peerCountB == 0 {
		peerCountB := len(nodeB.host.h.Peerstore().Peers())
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

	addrInfoB := nodeB.host.addrInfo()
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

	msg, ok := handler.messagesFrom(nodeA.host.id())
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

	addrInfoA := nodeA.host.addrInfo()
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

	addrInfoB := nodeB.host.addrInfo()
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

	messages, ok := handlerB.messagesFrom(nodeA.host.id())
	require.True(t, ok, "node B timed out waiting for message from node A")
	assert.Len(t, messages, 1)

	// node A uses the stream to send a second message
	err = nodeA.host.writeToStream(stream, testBlockReqMessage)
	require.NoError(t, err)

	_, ok = handlerB.messagesFrom(nodeA.host.id())
	require.True(t, ok, "node B timed out waiting for message from node A")

	// node B opens the stream to send the first message
	stream, err = nodeB.host.send(addrInfoA.ID, nodeB.host.protocolID, testBlockReqMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	_, ok = handlerA.messagesFrom(nodeB.host.id())
	require.True(t, ok, "node A timed out waiting for message from node B")

	// node B uses the stream to send a second message
	err = nodeB.host.writeToStream(stream, testBlockReqMessage)
	require.NoError(t, err)

	_, ok = handlerA.messagesFrom(nodeB.host.id())
	require.True(t, ok, "node A timed out waiting for message from node B")
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

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	const (
		roles           byte   = 4
		bestBlockNumber uint32 = 77
	)

	testHandshake := &BlockAnnounceHandshake{
		Roles:           roles,
		BestBlockNumber: bestBlockNumber,
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
	pID := nodeB.host.addrInfo().ID.String()

	err = nodeA.host.removeReservedPeers(pID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 1, nodeA.host.peerCount())
	isProtected := nodeA.host.h.ConnManager().IsProtected(nodeB.host.addrInfo().ID, "")
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

	addrInfoB := nodeB.host.addrInfo()
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

	err = stream.Close()
	require.NoError(t, err)

	timeout := time.NewTimer(TestBackoffTimeout)

	select {
	case <-timeout.C:
		t.Fatal("stream handler does not exit after stream closed")
	case <-handler.eofCh:
		if !timeout.Stop() {
			<-timeout.C
		}
	}
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

	addrInfoB := nodeB.host.addrInfo()
	nodeA.host.h.Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
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

	const zeroReputation int32 = 0

	require.NoError(t, err)
	require.Greater(t, rep, zeroReputation)
}
