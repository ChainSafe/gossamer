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
	"net"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
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

	_, err = nodeA.host.send(addrInfosB[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
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

	// node A opens the stream to send the first message
	stream, err := nodeA.host.send(addrInfosB[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	// node A uses the stream to send a second message
	err = nodeA.host.writeToStream(stream, testBlockRequestMessage)
	require.NoError(t, err)
	require.NotNil(t, handlerB.messages[nodeA.host.id()], "node B timeout waiting for message from node A")

	// node B opens the stream to send the first message
	stream, err = nodeB.host.send(addrInfosA[0].ID, nodeB.host.protocolID, testBlockRequestMessage)
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
	_, err = nodeA.host.send(nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID, testHandshake)
	require.NoError(t, err)

	info := nodeA.notificationsProtocols[BlockAnnounceMsgType]

	// Set handshake data to received
	info.inboundHandshakeData.Store(nodeB.host.id(), handshakeData{
		received:  true,
		validated: true,
	})

	// Verify that handshake data exists.
	_, ok := info.getHandshakeData(nodeB.host.id(), true)
	require.True(t, ok)

	time.Sleep(time.Second)
	nodeB.host.close()

	// Wait for cleanup
	time.Sleep(time.Second)

	// Verify that handshake data is cleared.
	_, ok = info.getHandshakeData(nodeB.host.id(), true)
	require.False(t, ok)
}
