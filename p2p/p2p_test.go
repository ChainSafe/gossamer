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
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	//ma "github.com/multiformats/go-multiaddr"
	//peer "github.com/libp2p/go-libp2p-core/peer"
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


func TestSendDirect(t *testing.T) {
    testServiceConfigB := &Config{
        //NoBootstrap: true,
        BootstrapNodes: []string{
            "/ip4/104.211.54.233/tcp/30363/p2p/16Uiu2HAmFWPUx45xYYeCpAryQbvU3dY8PWGdMwS2tLm1dB1CsmCj",
            "/ip4/104.211.48.51/tcp/30363/p2p/16Uiu2HAmJqVCtF5oMvu1rbJvqWubMMRuWiKJtpoM8KSQ3JNnL5Ec",
            "/ip4/104.211.48.247/tcp/30363/p2p/16Uiu2HAkyhNWHTPcA2dVKzMnLpFebXqsDQMpkuGnS9SqjJyDyULi",
            "/ip4/40.117.153.33/tcp/30363/p2p/16Uiu2HAmKXzRnzgyVtSyyp6ozAk5aT9H7PEi2ozkHSzzg7vmX7LV",
        },
        Port: 30304, 
    }

    sb, err := NewService(testServiceConfigB)
    if err != nil {
        t.Fatalf("NewService error: %s", err)
    }

    // peerid, err := peer.IDB58Decode("16Uiu2HAkyhNWHTPcA2dVKzMnLpFebXqsDQMpkuGnS9SqjJyDyULi")
    // if err != nil {
    // 	t.Fatal(err)
    // }
    // protocols, err := sb.Host().Peerstore().GetProtocols(peerid)
    // if err != nil {
    // 	t.Fatal(err)
    // }
    // t.Log(protocols)

    deadline, ok := sb.Ctx().Deadline()
    if !ok {
    	t.Log("ctx has no deadline")
    }
    t.Log(deadline)
   	go func(s *Service) {
    	for {
    		t.Logf("PeerCount %d", sb.PeerCount())
    		time.Sleep(time.Second * 5)
    	}
    }(sb)

    e := sb.Start()
    err = <-e
    if err != nil {
        t.Errorf("Start error: %s", err)
    }


    t.Log(sb.Host().Addrs())
    t.Log(sb.Host().Mux().Protocols())
    	//for {
    		t.Logf("PeerCount %d", sb.PeerCount())
    	// 	time.Sleep(time.Second * 5)
    	// }

   	select{}
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
