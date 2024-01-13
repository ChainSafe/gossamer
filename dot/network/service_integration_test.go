//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ChainSafe/gossamer/dot/types"
)

func createServiceHelper(t *testing.T, num int) []*Service {
	t.Helper()

	var srvcs []*Service
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		srvc := createTestService(t, config)
		srvc.noGossip = true
		handler := newTestStreamHandler(testBlockAnnounceMessageDecoder)
		srvc.host.registerStreamHandler(srvc.host.protocolID, handler.handleStream)

		srvcs = append(srvcs, srvc)
	}
	return srvcs
}

// test network service starts
func TestStartService(t *testing.T) {
	t.Parallel()

	node := createTestService(t, nil)
	require.NoError(t, node.Stop())
}

// test broacast messages from core service
func TestBroadcastMessages(t *testing.T) {
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
	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	anounceMessage := &BlockAnnounceMessage{
		Number: 128 * 7,
		Digest: types.NewDigest(),
	}

	// simulate message sent from core service
	nodeA.GossipMessage(anounceMessage)
	time.Sleep(time.Second * 2)
	require.NotNil(t, handler.messages[nodeA.host.id()])
}

func TestBroadcastDuplicateMessage(t *testing.T) {
	t.Parallel()

	msgCacheTTL = 2 * time.Second

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

	// TODO: create a decoder that handles both handshakes and messages
	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := nodeA.host.p2pHost.NewStream(context.Background(),
		nodeB.host.id(), nodeB.host.protocolID+blockAnnounceID)
	require.NoError(t, err)
	require.NotNil(t, stream)

	protocol := nodeA.notificationsProtocols[blockAnnounceMsgType]
	protocol.peersData.setOutboundHandshakeData(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	announceMessage := &BlockAnnounceMessage{
		Number: 128 * 7,
		Digest: types.NewDigest(),
	}

	delete(handler.messages, nodeA.host.id())

	// Only one message will be sent.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(announceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	time.Sleep(time.Millisecond * 500)
	require.Equal(t, 1, len(handler.messages[nodeA.host.id()]))

	nodeA.host.messageCache = nil

	// All 5 message will be sent since cache is disabled.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(announceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	require.Equal(t, 6, len(handler.messages[nodeA.host.id()]))
}

func TestService_NodeRoles(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		BasePath: t.TempDir(),
		Roles:    1,
		Port:     availablePort(t),
	}
	svc := createTestService(t, cfg)

	role := svc.NodeRoles()
	require.Equal(t, cfg.Roles, role)
}

func TestService_Health(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	syncer := NewMockSyncer(ctrl)

	s := createTestService(t, config)
	s.syncer = syncer

	syncer.EXPECT().IsSynced().Return(false)
	h := s.Health()
	require.Equal(t, true, h.IsSyncing)

	syncer.EXPECT().IsSynced().Return(true)
	h = s.Health()
	require.Equal(t, false, h.IsSyncing)
}

func TestInMemoryPeerStore(t *testing.T) {
	t.Parallel()

	nodes := createServiceHelper(t, 2)
	nodeA := nodes[0]
	nodeB := nodes[1]

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	require.NotEmpty(t, nodeA.host.p2pHost.Peerstore().PeerInfo(nodeB.host.id()).Addrs)

	// Stop a node and reinitialise a new node with same base path.
	err = nodeA.Stop()
	require.NoError(t, err)

	// Should be empty since peerstore is kept in memory
	nodeAA := createTestService(t, nodeA.cfg)
	require.Empty(t, nodeAA.host.p2pHost.Peerstore().PeerInfo(nodeB.host.id()).Addrs)
}

func TestHandleConn(t *testing.T) {
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

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)
}
