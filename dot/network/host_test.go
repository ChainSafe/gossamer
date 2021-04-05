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
	"net"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/require"
)

func TestExternalAddrs(t *testing.T) {
	config := &Config{
		BasePath:    utils.NewTestBasePath(t, "node"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)

	addrInfos, err := node.host.addrInfos()
	require.NoError(t, err)

	privateIPs := ma.NewFilters()
	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr) //nolint
		require.NoError(t, err)
		privateIPs.AddFilter(*ipnet, ma.ActionDeny)
	}

	for _, info := range addrInfos {
		for _, addr := range info.Addrs {
			require.False(t, privateIPs.AddrBlocked(addr))
		}
	}
}

// test host connect method
func TestConnect(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")

	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
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
		RandSeed:    1,
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
		RandSeed:  2,
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
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handler := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler("", handler.handleStream)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	err = nodeA.host.send(addrInfosB[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.Equal(t, testBlockRequestMessage, handler.messages[nodeA.host.id()])
}

// test host send method with existing stream
func TestExistingStream(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	handlerA := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeA.host.registerStreamHandler("", handlerA.handleStream)

	addrInfosA, err := nodeA.host.addrInfos()
	require.NoError(t, err)

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handlerB := newTestStreamHandler(testBlockRequestMessageDecoder)
	nodeB.host.registerStreamHandler("", handlerB.handleStream)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	stream := nodeA.host.getOutboundStream(nodeB.host.id(), nodeB.host.protocolID)
	require.Nil(t, stream, "node A should not have an outbound stream")

	// node A opens the stream to send the first message
	err = nodeA.host.send(addrInfosB[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	stream = nodeA.host.getOutboundStream(nodeB.host.id(), nodeB.host.protocolID)
	require.NotNil(t, stream, "node A should have an outbound stream")

	// node A uses the stream to send a second message
	err = nodeA.host.send(addrInfosB[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	stream = nodeA.host.getOutboundStream(nodeB.host.id(), nodeB.host.protocolID)
	require.NotNil(t, stream, "node B should have an outbound stream")

	// node B opens the stream to send the first message
	err = nodeB.host.send(addrInfosA[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerA.messages[nodeB.host.id()], "node A timeout waiting for message from node B")

	stream = nodeB.host.getOutboundStream(nodeA.host.id(), nodeB.host.protocolID)
	require.NotNil(t, stream, "node B should have an outbound stream")

	// node B uses the stream to send a second message
	err = nodeB.host.send(addrInfosA[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerA.messages[nodeB.host.id()], "node A timeout waiting for message from node B")

	stream = nodeB.host.getOutboundStream(nodeA.host.id(), nodeB.host.protocolID)
	require.NotNil(t, stream, "node B should have an outbound stream")
}

func TestStreamCloseMetadataCleanup(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
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
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handlerB := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(blockAnnounceID, handlerB.handleStream)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	testHandshake := &BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     nodeB.blockState.GenesisHash(),
	}

	// node A opens the stream to send the first message
	err = nodeA.host.send(nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID, testHandshake)
	require.NoError(t, err)

	info := nodeA.notificationsProtocols[BlockAnnounceMsgType]

	// Set handshake data to received
	info.handshakeData.Store(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
	})

	// Verify that handshake data exists.
	_, ok := info.getHandshakeData(nodeB.host.id())
	require.True(t, ok)

	time.Sleep(time.Second)
	nodeB.host.close()

	// Wait for cleanup
	time.Sleep(time.Second)

	// Verify that handshake data is cleared.
	_, ok = info.getHandshakeData(nodeB.host.id())
	require.False(t, ok)
}

func createServiceHelper(t *testing.T, num int) []*Service {
	t.Helper()
	var srvcs []*Service
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        uint32(7001 + i),
			RandSeed:    int64(1 + i),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		srvc := createTestService(t, config)
		srvc.noGossip = true
		handler := newTestStreamHandler(testBlockAnnounceMessageDecoder)
		srvc.host.registerStreamHandler("", handler.handleStream)

		srvcs = append(srvcs, srvc)
	}
	return srvcs
}

// nolint
func connectNoSync(t *testing.T, ctx context.Context, a, b *Service) {
	t.Helper()

	idB := b.host.h.ID()
	addrB := b.host.h.Peerstore().Addrs(idB)
	if len(addrB) == 0 {
		t.Fatal("peers setup incorrectly: no local address")
	}

	a.host.h.Peerstore().AddAddrs(idB, addrB, time.Minute)
	pi := peer.AddrInfo{ID: idB}

	err := a.host.h.Connect(ctx, pi)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = a.host.h.Connect(ctx, pi)
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

	nodes := createServiceHelper(t, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := nodes[2].host.dht.FindPeer(ctx, nodes[1].host.id())
	require.Equal(t, err, kbucket.ErrLookupFailure)

	connectNoSync(t, ctx, nodes[1], nodes[0])
	connectNoSync(t, ctx, nodes[2], nodes[0])

	// Can't use `connect` because b and c are only clients.
	wait(t, ctx, nodes[1].host.dht, nodes[0].host.dht)
	wait(t, ctx, nodes[2].host.dht, nodes[0].host.dht)

	_, err = nodes[2].host.dht.FindPeer(ctx, nodes[1].host.id())
	require.NoError(t, err)
}
