// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/utils"
)

var TestProtocolID = "/gossamer/test/0"

// maximum wait time for non-status message to be handled
var TestMessageTimeout = time.Second

// time between connection retries (BackoffBase default 5 seconds)
var TestBackoffTimeout = 5 * time.Second

// failedToDial returns true if "failed to dial" error, otherwise false
func failedToDial(err error) bool {
	return err != nil && strings.Contains(err.Error(), "failed to dial")
}

func createServiceHelper(t *testing.T, num int) []*Service {
	t.Helper()

	var srvcs []*Service
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    utils.NewTestBasePath(t, fmt.Sprintf("node%d", i)),
			Port:        uint16(availablePorts.get()),
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

// helper method to create and start a new network service
func createTestService(t *testing.T, cfg *Config) (srvc *Service) {
	t.Helper()

	if cfg == nil {
		basePath := utils.NewTestBasePath(t, "node")

		cfg = &Config{
			BasePath:    basePath,
			Port:        uint16(availablePorts.get()),
			NoBootstrap: true,
			NoMDNS:      true,
			LogLvl:      4,
		}
	}

	if cfg.BlockState == nil {
		cfg.BlockState = NewMockBlockState(nil)
	}

	if cfg.TransactionHandler == nil {
		mocktxhandler := &MockTransactionHandler{}
		mocktxhandler.On("HandleTransactionMessage",
			mock.AnythingOfType("peer.ID"),
			mock.AnythingOfType("*network.TransactionMessage")).
			Return(true, nil)
		mocktxhandler.On("TransactionsCount").Return(0)
		cfg.TransactionHandler = mocktxhandler
	}

	cfg.SlotDuration = time.Second

	cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.LogLvl == 0 {
		cfg.LogLvl = 4
	}

	if cfg.Syncer == nil {
		cfg.Syncer = NewMockSyncer()
	}

	cfg.noPreAllocate = true

	srvc, err := NewService(cfg)
	require.NoError(t, err)

	srvc.noDiscover = true

	err = srvc.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		srvc.Stop()
		availablePorts.put(int(cfg.Port))
	})
	return srvc
}

func TestMain(m *testing.M) {
	// Start all tests
	code := m.Run()

	// Cleanup test path.
	err := os.RemoveAll(utils.TestDir)
	if err != nil {
		fmt.Printf("failed to remove path %s : %s\n", utils.TestDir, err)
	}
	os.Exit(code)
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

	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	anounceMessage := &BlockAnnounceMessage{
		Number: big.NewInt(128 * 7),
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

	basePathA := utils.NewTestBasePath(t, "nodeA")
	configA := &Config{
		BasePath:    basePathA,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	handler := newTestStreamHandler(testBlockAnnounceHandshakeDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID+blockAnnounceID, handler.handleStream)

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
	protocol.outboundHandshakeData.Store(nodeB.host.id(), &handshakeData{
		received:  true,
		validated: true,
		stream:    stream,
	})

	announceMessage := &BlockAnnounceMessage{
		Number: big.NewInt(128 * 7),
		Digest: types.NewDigest(),
	}

	// Only one message will be sent.
	for i := 0; i < 5; i++ {
		nodeA.GossipMessage(announceMessage)
		time.Sleep(time.Millisecond * 10)
	}

	time.Sleep(time.Millisecond * 200)
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

	basePath := utils.NewTestBasePath(t, "node")
	cfg := &Config{
		BasePath: basePath,
		Roles:    1,
	}
	svc := createTestService(t, cfg)

	role := svc.NodeRoles()
	require.Equal(t, cfg.Roles, role)
}

func TestService_Health(t *testing.T) {
	t.Parallel()

	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	mocksyncer := &MockSyncer{}
	mocksyncer.On("SetSyncing", mock.AnythingOfType("bool"))

	s := createTestService(t, config)
	s.syncer = mocksyncer

	mocksyncer.On("IsSynced").Return(false).Once()
	h := s.Health()
	require.Equal(t, true, h.IsSyncing)

	mocksyncer.On("IsSynced").Return(true).Once()
	h = s.Health()
	require.Equal(t, false, h.IsSyncing)
}

func TestPersistPeerStore(t *testing.T) {
	t.Parallel()

	nodes := createServiceHelper(t, 2)
	nodeA := nodes[0]
	nodeB := nodes[1]

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	require.NotEmpty(t, nodeA.host.h.Peerstore().PeerInfo(nodeB.host.id()).Addrs)

	// Stop a node and reinitialise a new node with same base path.
	err = nodeA.Stop()
	require.NoError(t, err)

	// Since nodeAA uses the persistent peerstore of nodeA, it should be have nodeB in it's peerstore.
	nodeAA := createTestService(t, nodeA.cfg)
	require.NotEmpty(t, nodeAA.host.h.Peerstore().PeerInfo(nodeB.host.id()).Addrs)
}

func TestHandleConn(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)
}

func TestSerivceIsMajorSyncMetrics(t *testing.T) {
	t.Parallel()

	mocksyncer := new(MockSyncer)

	node := &Service{
		syncer: mocksyncer,
	}

	mocksyncer.On("IsSynced").Return(false).Once()
	m := node.CollectGauge()

	require.Equal(t, int64(1), m[gssmrIsMajorSyncMetric])

	mocksyncer.On("IsSynced").Return(true).Once()
	m = node.CollectGauge()

	require.Equal(t, int64(0), m[gssmrIsMajorSyncMetric])
}
