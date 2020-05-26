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
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

var TestProtocolID = "/gossamer/test/0"

// arbitrary block request message
var TestMessage = &BlockRequestMessage{
	ID:            1,
	RequestedData: 1,
	// TODO: investigate starting block mismatch with different slice length
	StartingBlock: variadic.NewUint64OrHashFromBytes([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1}),
	EndBlockHash:  optional.NewHash(true, common.Hash{}),
	Direction:     1,
	Max:           optional.NewUint32(true, 1),
}

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
		cfg.BlockState = &MockBlockState{}
	}

	if cfg.NetworkState == nil {
		cfg.NetworkState = &MockNetworkState{}
	}

	cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.MsgRec == nil {
		cfg.MsgRec = make(chan Message, 10)
	}

	if cfg.MsgSend == nil {
		cfg.MsgSend = make(chan Message, 10)
	}

	if cfg.SyncChan == nil {
		cfg.SyncChan = make(chan *big.Int, 10)
	}

	srvc, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = srvc.Start()
	if err != nil {
		t.Fatal(err)
	}

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

	msgRecA := make(chan Message)

	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
		MsgRec:      msgRecA,
	}

	nodeA := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	msgSendB := make(chan Message)

	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
		MsgSend:     msgSendB,
	}

	nodeB := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	if err != nil {
		t.Fatal(err)
	}

	// simulate message sent from core service
	msgRecA <- TestMessage

	select {
	case msg := <-msgSendB:
		if !reflect.DeepEqual(msg, TestMessage) {
			t.Error(
				"node B received unexpected message from node A",
				"\nexpected:", TestMessage,
				"\nreceived:", msg,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("node B timeout waiting for message")
	}
}

func TestHandleMessage_BlockResponse(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

	msgSend := make(chan Message, 4)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
		NoStatus:    true,
		MsgSend:     msgSend,
	}

	s := createTestService(t, config)

	peerID := peer.ID("noot")
	msgID := uint64(17)
	msg := &BlockResponseMessage{
		ID: msgID,
	}

	s.syncer.addRequestedBlockID(msgID)

	s.handleMessage(peerID, msg)
	if s.syncer.hasRequestedBlockID(msgID) {
		t.Fatal("Fail: should have removed ID")
	}

	select {
	case recv := <-msgSend:
		if !reflect.DeepEqual(recv, msg) {
			t.Error(
				"node B received unexpected message",
				"\nexpected:", msg,
				"\nreceived:", recv,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}

	msg = &BlockResponseMessage{
		ID: 77,
	}

	s.handleMessage(peerID, msg)

	select {
	case <-msgSend:
		t.Fatal("Fail: should not have sent msg")
	case <-time.After(TestMessageTimeout):
		// expected
	}

	reqMsg := &BlockRequestMessage{}
	s.handleMessage(peerID, reqMsg)

	select {
	case recv := <-msgSend:
		if !reflect.DeepEqual(recv, reqMsg) {
			t.Error(
				"node B received unexpected message",
				"\nexpected:", reqMsg,
				"\nreceived:", recv,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
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
