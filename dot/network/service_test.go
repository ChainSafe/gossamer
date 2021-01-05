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
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
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

// helper method to create and start a new network service
func createTestService(t *testing.T, cfg *Config) (srvc *Service) {
	if cfg.BlockState == nil {
		cfg.BlockState = newMockBlockState(nil)
	}

	if cfg.TransactionHandler == nil {
		cfg.TransactionHandler = newMockTransactionHandler()
	}

	if cfg.TransactionHandler == nil {
		cfg.TransactionHandler = newMockTransactionHandler()
	}

	cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.LogLvl == 0 {
		cfg.LogLvl = 3
	}

	if cfg.Syncer == nil {
		cfg.Syncer = newMockSyncer()
	}

	srvc, err := NewService(cfg)
	require.NoError(t, err)

	err = srvc.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		utils.RemoveTestDir(t)
		srvc.Stop()
	})
	return srvc
}

// test network service starts
func TestStartService(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "node")

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	node := createTestService(t, config)
	node.Stop()
}

// test broacast messages from core service
func TestBroadcastMessages(t *testing.T) {
	basePathA := utils.NewTestBasePath(t, "nodeA")

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

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
	handler := newTestStreamHandler(testBlockAnnounceMessageDecoder)
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
	time.Sleep(time.Second)
	require.NotNil(t, nodeB.syncing[nodeA.host.id()])
}

func TestHandleSyncMessage_BlockResponse(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	defer utils.RemoveTestDir(t)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	peerID := peer.ID("noot")
	msg := &BlockResponseMessage{}
	s.syncing[peerID] = struct{}{}

	s.handleSyncMessage(peerID, msg)
	_, isSyncing := s.syncing[peerID]
	require.False(t, isSyncing)
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
	defer utils.RemoveTestDir(t)

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

	mockSync.setSyncedState(true)
	require.Equal(t, s.Health().IsSyncing, false)
}

func TestHandleLightMessage_Response(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	defer utils.RemoveTestDir(t)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	s := createTestService(t, config)

	peerID := peer.ID("noot")

	// Testing empty request
	msg := &LightRequest{}
	err := s.handleLightSyncMsg(peerID, msg)
	require.NoError(t, err)

	expectedErr := "failed to find any peer in table"

	// Testing remoteCallResp()
	msg = &LightRequest{
		RmtCallRequest: &RemoteCallRequest{},
	}
	err = s.handleLightSyncMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteHeaderResp()
	msg = &LightRequest{
		RmtHeaderRequest: &RemoteHeaderRequest{},
	}
	err = s.handleLightSyncMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteChangeResp()
	msg = &LightRequest{
		RmtChangesRequest: &RemoteChangesRequest{},
	}
	err = s.handleLightSyncMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadResp()
	msg = &LightRequest{
		RmtReadRequest: &RemoteReadRequest{},
	}
	err = s.handleLightSyncMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadChildResp()
	msg = &LightRequest{
		RmtReadChildRequest: &RemoteReadChildRequest{},
	}
	err = s.handleLightSyncMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())
}

func TestDecodeSyncMessage(t *testing.T) {
	s := &Service{
		syncing: make(map[peer.ID]struct{}),
	}

	testPeer := peer.ID("noot")

	testBlockResponseMessage := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	reqEnc, err := testBlockRequestMessage.Encode()
	require.NoError(t, err)

	msg, err := s.decodeSyncMessage(reqEnc, testPeer)
	require.NoError(t, err)

	req, ok := msg.(*BlockRequestMessage)
	require.True(t, ok)
	require.Equal(t, testBlockRequestMessage, req)

	s.syncing[testPeer] = struct{}{}

	respEnc, err := testBlockResponseMessage.Encode()
	require.NoError(t, err)

	msg, err = s.decodeSyncMessage(respEnc, testPeer)
	require.NoError(t, err)
	resp, ok := msg.(*BlockResponseMessage)
	require.True(t, ok)
	require.Equal(t, testBlockResponseMessage, resp)
}

func TestDecodeLightMessage(t *testing.T) {
	s := &Service{
		lightRequest: make(map[peer.ID]struct{}),
	}

	testPeer := peer.ID("noot")

	testLightRequest := NewLightRequest()
	testLightResponse := NewLightResponse()

	reqEnc, err := testLightRequest.Encode()
	require.NoError(t, err)

	msg, err := s.decodeLightMessage(reqEnc, testPeer)
	require.NoError(t, err)

	req, ok := msg.(*LightRequest)
	require.True(t, ok)
	resEnc, err := req.Encode()
	require.NoError(t, err)
	require.Equal(t, reqEnc, resEnc)

	s.lightRequest[testPeer] = struct{}{}

	respEnc, err := testLightResponse.Encode()
	require.NoError(t, err)

	msg, err = s.decodeLightMessage(respEnc, testPeer)
	require.NoError(t, err)
	resp, ok := msg.(*LightResponse)
	require.True(t, ok)
	resEnc, err = resp.Encode()
	require.NoError(t, err)
	require.Equal(t, respEnc, resEnc)
}
