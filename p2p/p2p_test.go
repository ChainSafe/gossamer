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

package p2p

import (
	"math/big"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/common/optional"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func startNewService(t *testing.T, cfg *Config, sendChan chan []byte, recChan chan BlockAnnounceMessage) *Service {
	node, err := NewService(cfg, sendChan, recChan)
	if err != nil {
		t.Fatal(err)
	}

	err = node.Start()
	if err != nil {
		t.Fatal(err)
	}

	return node
}

func TestBuildOptions(t *testing.T) {
	configA := &Config{
		DataDir: path.Join(os.TempDir(), "p2p-test"),
	}

	_, err := configA.buildOpts()
	if err != nil {
		t.Fatal(err)
	}

	if configA.privateKey == nil {
		t.Error("Private key was not set.")
	}

	configB := &Config{
		DataDir: path.Join(os.TempDir(), "p2p-test"),
	}

	_, err = configB.buildOpts()
	if err != nil {
		t.Fatal(err)
	}

	if configA.privateKey == configB.privateKey {
		t.Error("Private keys should not match.")
	}
}

func TestStartService(t *testing.T) {
	config := &Config{
		RandSeed: 1,
	}

	node := startNewService(t, config, nil, nil)
	node.Stop()
}

func TestBootstrapNode(t *testing.T) {
	configA := &Config{
		RandSeed: 1,
	}

	nodeA := startNewService(t, configA, nil, nil)
	defer nodeA.Stop()

	addrA := nodeA.host.fullAddrs()[0]

	configB := &Config{
		BootstrapNodes: []string{addrA.String()},
		RandSeed:       2,
	}

	nodeB := startNewService(t, configB, nil, nil)
	defer nodeB.Stop()

	peerCountA := nodeA.host.peerCount()

	if peerCountA != 1 {
		t.Errorf("Expected peer count: 1. Got peer count: %d.", peerCountA)
	}
}

func TestConnectNode(t *testing.T) {
	configA := &Config{
		RandSeed: 1,
	}

	nodeA := startNewService(t, configA, nil, nil)
	defer nodeA.Stop()

	configB := &Config{
		RandSeed: 2,
	}

	nodeB := startNewService(t, configB, nil, nil)
	defer nodeB.Stop()

	addrA := nodeA.host.fullAddrs()[0]
	addrInfoA, err := peer.AddrInfoFromP2pAddr(addrA)
	if err != nil {
		t.Fatal(err)
	}

	err = nodeB.host.connect(*addrInfoA)
	if err != nil {
		t.Fatal(err)
	}

	peerCountB := nodeB.host.peerCount()

	if peerCountB != 1 {
		t.Errorf("Expected peer count: 1. Got peer count: %d.", peerCountB)
	}
}

func TestPingNode(t *testing.T) {
	configA := &Config{
		RandSeed: 1,
	}

	nodeA := startNewService(t, configA, nil, nil)
	defer nodeA.Stop()

	configB := &Config{
		RandSeed: 2,
	}

	sendChanB := make(chan []byte)

	nodeB := startNewService(t, configB, sendChanB, nil)
	defer nodeB.Stop()

	addrA := nodeA.host.fullAddrs()[0]
	addrInfoA, err := peer.AddrInfoFromP2pAddr(addrA)
	if err != nil {
		t.Fatal(err)
	}

	err = nodeB.host.connect(*addrInfoA)
	if err != nil {
		t.Fatal(err)
	}

	err = nodeB.host.ping(addrInfoA.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendRequest(t *testing.T) {
	configA := &Config{
		RandSeed: 1,
	}

	nodeA := startNewService(t, configA, nil, nil)
	defer nodeA.Stop()

	configB := &Config{
		RandSeed: 2,
	}

	sendChanB := make(chan []byte)

	nodeB := startNewService(t, configB, sendChanB, nil)
	defer nodeB.Stop()

	addrA := nodeA.host.fullAddrs()[0]

	addrInfoA, err := peer.AddrInfoFromP2pAddr(addrA)
	if err != nil {
		t.Fatal(err)
	}

	err = nodeB.host.connect(*addrInfoA)
	if err != nil {
		t.Fatal(err)
	}

	addrB := nodeB.host.fullAddrs()[0]

	addrInfoB, err := peer.AddrInfoFromP2pAddr(addrB)
	if err != nil {
		t.Fatal(err)
	}

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	if err != nil {
		t.Fatal(err)
	}

	blockRequest := &BlockRequestMessage{
		ID:            1,
		RequestedData: 1,
		StartingBlock: []byte{1, 1},
		EndBlockHash:  optional.NewHash(true, endBlock),
		Direction:     1,
		Max:           optional.NewUint32(true, 1),
	}

	encBlockRequest, err := blockRequest.Encode()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.send(*addrInfoB, encBlockRequest)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-sendChanB:
		if !reflect.DeepEqual(message, encBlockRequest) {
			t.Error("Did not receive the correct message.")
		}
	case <-time.After(30 * time.Second):
		t.Errorf("Did not receive message from %s.", nodeA.host.hostAddr)
	}
}

func TestGossip(t *testing.T) {
	configA := &Config{
		RandSeed: 1,
	}

	nodeA := startNewService(t, configA, nil, nil)
	defer nodeA.Stop()

	addrA := nodeA.host.fullAddrs()[0]

	configB := &Config{
		BootstrapNodes: []string{
			addrA.String(),
		},
		RandSeed: 2,
	}

	sendChanB := make(chan []byte)

	nodeB := startNewService(t, configB, sendChanB, nil)
	defer nodeB.Stop()

	nodeBAddr := nodeB.host.fullAddrs()[0]

	configC := &Config{
		BootstrapNodes: []string{
			nodeBAddr.String(),
		},
		RandSeed: 3,
	}

	sendChanC := make(chan []byte)

	nodeC := startNewService(t, configC, sendChanC, nil)
	defer nodeC.Stop()

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	if err != nil {
		t.Fatal(err)
	}

	blockRequest := &BlockRequestMessage{
		ID:            1,
		RequestedData: 1,
		StartingBlock: []byte{1, 1},
		EndBlockHash:  optional.NewHash(true, endBlock),
		Direction:     1,
		Max:           optional.NewUint32(true, 1),
	}

	err = nodeA.Broadcast(blockRequest)
	if err != nil {
		t.Fatal(err)
	}

	encBlockRequest, err := blockRequest.Encode()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-sendChanB:
		if !reflect.DeepEqual(message, encBlockRequest) {
			t.Error("Did not receive the correct message.")
		}
	case <-time.After(30 * time.Second):
		t.Errorf("Did not receive message from %s.", nodeA.host.hostAddr)
	}

	select {
	case message := <-sendChanC:
		if !reflect.DeepEqual(encBlockRequest, message) {
			t.Error("Did not receive the correct message.")
		}
	case <-time.After(30 * time.Second):
		t.Errorf("Did not receive message from %s.", nodeB.host.hostAddr)
	}
}

func TestReceiveChannel(t *testing.T) {
	configA := &Config{
		RandSeed: 1,
	}

	recChanA := make(chan BlockAnnounceMessage)

	nodeA := startNewService(t, configA, nil, recChanA)
	defer nodeA.Stop()

	addrA := nodeA.host.fullAddrs()[0]

	configB := &Config{
		BootstrapNodes: []string{
			addrA.String(),
		},
		RandSeed: 2,
	}

	sendChanB := make(chan []byte)

	nodeB := startNewService(t, configB, sendChanB, nil)
	defer nodeB.Stop()

	blockAnnounce := BlockAnnounceMessage{
		Number: big.NewInt(10),
	}

	recChanA <- blockAnnounce

	encBlockAnnounce, err := blockAnnounce.Encode()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-sendChanB:
		if !reflect.DeepEqual(message, encBlockAnnounce) {
			t.Error("Did not receive the correct message.")
		}
	case <-time.After(30 * time.Second):
		t.Errorf("Did not receive message from %s.", nodeB.host.hostAddr)
	}
}
