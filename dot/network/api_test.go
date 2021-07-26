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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

func TestSendMessage(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	defer nodeA.Stop()
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
	defer nodeB.Stop()
	nodeB.noGossip = true
	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(blockAnnounceID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	// simulate message sent from core service
	nodeA.GossipMessage(testBlockAnnounceMessage)
	time.Sleep(time.Second * 2)
	require.NotNil(t, handler.messages[nodeA.host.id()])
}

func TestSendMessage_DuplicateMessage(t *testing.T) {
	msgCacheTTL = 2 * time.Second

	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	defer nodeA.Stop()
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
	defer nodeB.Stop()
	nodeB.noGossip = true

	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(blockAnnounceID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := nodeA.host.h.NewStream(context.Background(), nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID)
	require.NoError(t, err)
	require.NotNil(t, stream)

	protocol := nodeA.notificationsProtocols[BlockAnnounceMsgType]
	protocol.outboundHandshakeData.Store(nodeB.host.id(), handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	// Only one message will be sent.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(testBlockAnnounceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	time.Sleep(time.Millisecond * 200)
	require.Equal(t, 1, len(handler.messages[nodeA.host.id()]))

	nodeA.host.messageCache = nil

	// All 5 message will be sent since cache is disabled.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(testBlockAnnounceMessage)
		time.Sleep(time.Millisecond * 10)
	}
	require.Equal(t, 6, len(handler.messages[nodeA.host.id()]))
}
