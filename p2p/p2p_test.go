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
	"fmt"
	"log"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

func TestBuildOpts(t *testing.T) {
	testServiceConfig := &Config{
		BootstrapNodes: []string{},
		Port:           7001,
	}

	_, err := testServiceConfig.buildOpts()
	if err != nil {
		t.Fatalf("TestBuildOpts error: %s", err)
	}
}

func TestGenerateKey(t *testing.T) {
	privA, err := generateKey(33)
	if err != nil {
		t.Fatalf("GenerateKey error: %s", err)
	}

	privC, err := generateKey(0)
	if err != nil {
		t.Fatalf("GenerateKey error: %s", err)
	}

	if crypto.KeyEqual(privA, privC) {
		t.Fatal("GenerateKey error: created same key for different seed")
	}
}

func TestStart(t *testing.T) {
	ipfsNode, err := StartIpfsNode()
	if err != nil {
		t.Fatalf("Could not start IPFS node: %s", err)
	}

	defer ipfsNode.Close()

	ipfsAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/4001/ipfs/%s", ipfsNode.Identity.String())

	t.Log("ipfsAddr:", ipfsAddr)

	testServiceConfig := &Config{
		BootstrapNodes: []string{
			ipfsAddr,
		},
		Port: 7001,
	}

	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := s.Start()
	err = <-e
	if err != nil {
		t.Errorf("Start error: %s", err)
	}
}

func TestService_PeerCount(t *testing.T) {
	ipfsNode, err := StartIpfsNode()
	if err != nil {
		t.Fatalf("Could not start IPFS node: %s", err)
	}

	defer ipfsNode.Close()

	ipfsAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/4001/ipfs/%s", ipfsNode.Identity.String())

	testServiceConfig := &Config{
		BootstrapNodes: []string{
			ipfsAddr,
		},
		Port: 7001,
	}

	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := s.Start()
	err = <-e
	if err != nil {
		t.Errorf("Start error: %s", err)
	}

	count := s.PeerCount()
	if count != 1 {
		t.Fatalf("incorrect peerCount expected %d got %d", 1, count)
	}
}

func TestSend(t *testing.T) {
	sim, err := NewSimulator(2)
	if err != nil {
		log.Fatal(err)
	}

	defer sim.IpfsNode.Close()

	for _, node := range sim.Nodes {
		e := node.Start()
		if <-e != nil {
			log.Println("start err: ", err)
		}
	}

	sa := sim.Nodes[0]
	sb := sim.Nodes[1]
	peer, err := sa.dht.FindPeer(sa.ctx, sb.host.ID())
	if err != nil {
		t.Fatalf("could not find peer: %s", err)
	}

	msg := []byte("hello there\n")
	err = sa.Send(peer, msg)
	if err != nil {
		t.Errorf("Send error: %s", err)
	}
}

func TestNoBootstrap(t *testing.T) {
	testServiceConfigA := &Config{
		NoBootstrap: true,
		Port:        7001,
	}

	sa, err := NewService(testServiceConfigA)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := sa.Start()
	err = <-e
	if err != nil {
		t.Errorf("Start error: %s", err)
	}
}

func TestGossipSub(t *testing.T) {

	//Start node A
	nodeConfigA := &Config{
		Port: 7000,
	}

	nodeA, err := NewService(nodeConfigA)
	if err != nil {
		t.Fatalf("Could not start p2p service: %s", err)
	}

	defer nodeA.Stop()

	nodeA_Addr := nodeA.hostAddr.String()

	//Start node B
	nodeConfigB := &Config{
		BootstrapNodes: []string{
			nodeA_Addr,
		},
		Port: 7001,
	}

	nodeB, err := NewService(nodeConfigB)
	if err != nil {
		t.Fatalf("Could not start p2p service: %s", err)
	}

	defer nodeB.Stop()

	nodeB_Addr := nodeA.hostAddr.String()

	//Connect node A & node B
	err = nodeB.bootstrapConnect()
	if err != nil {
		t.Errorf("Start error :%s", err)
	}

	//Start node C
	nodeConfigC := &Config{
		BootstrapNodes: []string{
			nodeB_Addr,
		},
		Port: 7002,
	}

	nodeC, err := NewService(nodeConfigC)
	if err != nil {
		t.Fatalf("Could not start p2p service: %s", err)
	}

	defer nodeC.Stop()

	//Connect node B & node C
	err = nodeC.bootstrapConnect()
	if err != nil {
		t.Errorf("Start error :%s", err)
	}

	peer, _ := nodeB.dht.FindPeer(nodeB.ctx, nodeA.dht.PeerID())
	fmt.Printf("%s peer's: %s\n", nodeB.hostAddr.String(), peer)

	peer, _ = nodeA.dht.FindPeer(nodeA.ctx, nodeB.dht.PeerID())
	fmt.Printf("%s peer's: %s\n", nodeA.hostAddr.String(), peer)

	peer, _ = nodeC.dht.FindPeer(nodeC.ctx, nodeB.dht.PeerID())
	fmt.Printf("%s peer's: %s\n", nodeC.hostAddr.String(), peer)

}

// PING is not implemented in the kad-dht.
// see https://github.com/libp2p/specs/pull/108
// func TestPing(t *testing.T) {
// 	sim, err := NewSimulator(2)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	defer sim.IpfsNode.Close()

// 	for _, node := range sim.Nodes {
// 		e := node.Start()
// 		if <-e != nil {
// 			log.Println("start err: ", err)
// 		}
// 	}

// 	sa := sim.Nodes[0]
// 	sb := sim.Nodes[1]
// 	err = sa.Ping(sb.host.ID())
// 	if err != nil {
// 		t.Errorf("Ping error: %s", err)
// 	}
// }
