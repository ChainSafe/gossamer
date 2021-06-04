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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

// helper method to create and start a new network service
func createTestService(t *testing.T, cfg *Config) (srvc *Service) {
	if cfg == nil {
		basePath := utils.NewTestBasePath(t, "node")

		cfg = &Config{
			BasePath:    basePath,
			Port:        7001,
			RandSeed:    1,
			NoBootstrap: true,
			NoMDNS:      true,
			LogLvl:      4,
		}
	}

	if cfg.BlockState == nil {
		cfg.BlockState = newMockBlockState(nil)
	}

	if cfg.TransactionHandler == nil {
		mocktxhandler := &MockTransactionHandler{}
		mocktxhandler.On("HandleTransactionMessage", mock.AnythingOfType("*TransactionMessage")).Return(nil)
		cfg.TransactionHandler = mocktxhandler
	}

	cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.LogLvl == 0 {
		cfg.LogLvl = 4
	}

	if cfg.Syncer == nil {
		cfg.Syncer = newMockSyncer()
	}

	cfg.noPreAllocate = true

	srvc, err := NewService(cfg)
	require.NoError(t, err)

	srvc.noDiscover = true

	err = srvc.Start()
	require.NoError(t, err)
	srvc.syncQueue.stop()

	t.Cleanup(func() {
		srvc.Stop()
		err = os.RemoveAll(cfg.BasePath)
		if err != nil {
			fmt.Printf("failed to remove path %s : %s\n", cfg.BasePath, err)
		}
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
	node := createTestService(t, nil)
	node.Stop()
}

// test broacast messages from core service
func TestBroadcastMessages(t *testing.T) {
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

func TestBroadcastDuplicateMessage(t *testing.T) {
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

func TestService_NodeRoles(t *testing.T) {
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
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	s := createTestService(t, config)

	require.Equal(t, s.Health().IsSyncing, true)
	mockSync := s.syncer.(*mockSyncer)

	mockSync.SetSyncing(false)
	require.Equal(t, s.Health().IsSyncing, false)
}

func TestPersistPeerStore(t *testing.T) {
	nodes := createServiceHelper(t, 2)
	nodeA := nodes[0]
	nodeB := nodes[1]

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
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
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	time.Sleep(time.Second)

	bScore, ok := nodeA.syncQueue.peerScore.Load(nodeB.host.id())
	require.True(t, ok)
	require.Equal(t, 1, bScore)
	aScore, ok := nodeB.syncQueue.peerScore.Load(nodeA.host.id())
	require.True(t, ok)
	require.Equal(t, 1, aScore)
}
