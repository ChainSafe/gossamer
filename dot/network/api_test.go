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

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

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

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	// simulate message sent from core service
	nodeA.SendMessage(testBlockAnnounceMessage)
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

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
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
		nodeA.SendMessage(testBlockAnnounceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	time.Sleep(time.Millisecond * 200)
	require.Equal(t, 1, len(handler.messages[nodeA.host.id()]))

	nodeA.host.messageCache = nil

	// All 5 message will be sent since cache is disabled.
	for i := 0; i < 5; i++ {
		nodeA.SendMessage(testBlockAnnounceMessage)
		time.Sleep(time.Millisecond * 10)
	}
	require.Equal(t, 6, len(handler.messages[nodeA.host.id()]))
}

func TestSendCatchUpRequest(t *testing.T) {
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

	err := nodeA.RegisterNotificationsProtocol(
		testFinalityProtocolID,
		ConsensusMsgType,
		getMockFinalityHandshake,
		decodeMockFinalityHandshake,
		validateMockFinalityHandshake,
		decodeMockConsensusMessage,
		nodeA.handleMockConsensusMessage,
		true,
	)
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
	defer nodeB.Stop()
	nodeB.noGossip = true

	err = nodeB.RegisterNotificationsProtocol(
		testFinalityProtocolID,
		ConsensusMsgType,
		getMockFinalityHandshake,
		decodeMockFinalityHandshake,
		validateMockFinalityHandshake,
		decodeMockConsensusMessage,
		nodeB.handleMockConsensusMessage,
		true,
	)
	require.NoError(t, err)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	stream, err := nodeB.host.h.NewStream(nodeB.host.ctx, nodeA.host.id(), testFinalityProtocolID)
	require.NoError(t, err)

	t.Log("nodeB", stream.ID())

	nodeA.notificationsProtocols[ConsensusMsgType].inboundHandshakeData.Store(nodeB.host.id(), handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	nodeB.notificationsProtocols[ConsensusMsgType].outboundHandshakeData.Store(nodeA.host.id(), handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	req := &ConsensusMessage{
		// catchUpRequestType = 3
		Data: []byte{3},
	}

	resp, err := nodeA.SendCatchUpRequest(nodeB.host.id(), ConsensusMsgType, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

var testFinalityProtocolID protocol.ID = "test-finality"

type mockFinalityHandshake struct{}

func (hs *mockFinalityHandshake) Decode(_ []byte) error {
	return nil
}

func (hs *mockFinalityHandshake) Encode() ([]byte, error) {
	return nil, nil
}

func (hs *mockFinalityHandshake) Hash() common.Hash {
	return common.Hash{}
}

func (hs *mockFinalityHandshake) IsHandshake() bool {
	return true
}

func (hs *mockFinalityHandshake) String() string {
	return ""
}

func (hs *mockFinalityHandshake) SubProtocol() string {
	return string(testFinalityProtocolID)
}

func (hs *mockFinalityHandshake) Type() byte {
	return ConsensusMsgType
}

func getMockFinalityHandshake() (Handshake, error) {
	return &mockFinalityHandshake{}, nil
}

func decodeMockFinalityHandshake(_ []byte) (Handshake, error) {
	return &mockFinalityHandshake{}, nil
}

func validateMockFinalityHandshake(_ peer.ID, _ Handshake) error {
	return nil
}

func decodeMockConsensusMessage(in []byte) (NotificationsMessage, error) {
	return &ConsensusMessage{
		Data: in,
	}, nil
}

func (s *Service) handleMockConsensusMessage(_ peer.ID, _ NotificationsMessage) (bool, error) {
	s.SendMessage(&ConsensusMessage{
		Data: []byte{4},
	})

	return false, nil
}
