package main

import (
	"fmt"
	"log"
	"os"

	//golog "github.com/ipfs/go-log"
	//gologging "github.com/whyrusleeping/go-logging"
	peer "github.com/libp2p/go-libp2p-peer"
	//iaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/ChainSafeSystems/gossamer/p2p"
)

var LOCAL_PEER_ENDPOINT = "http://localhost:5001/api/v0/id"

type Simulator struct {
	nodes []*p2p.Service
}

func NewSimulator(num int) (sim *Simulator, err error) {
	sim = new(Simulator)
	sim.nodes = make([]*p2p.Service, num)

	conf := &p2p.ServiceConfig{
		BootstrapNodes: []string{
			"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		},
		Port: 4000,
	}

	for i := 0; i < num; i++ {
		conf.Port = conf.Port + i
		sim.nodes[i] = new(p2p.Service)
		sim.nodes[i], err = p2p.NewService(conf)
		if err != nil {
			return nil, err
		}

	}

	return sim, nil
}

func main() {
	//golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// num := 5

	// sim, err := NewSimulator(num)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for i := 0; i < num; i++ {
	// 	sim.nodes[i].Start()
	// }

	// var conf *p2p.ServiceConfig

	// if len(os.Args) < 2 {
	// 	conf = &p2p.ServiceConfig{
	// 		BootstrapNodes: []string{
	// 			getLocalPeerInfo(),
	// 		},
	// 		Port: 4000,
	// 	}
	// } else {
	// 	conf = &p2p.ServiceConfig{
	// 		BootstrapNodes: []string{
	// 			os.Args[1],
	// 		},
	// 		Port: 4000,
	// 	}
	// }

	conf := &p2p.ServiceConfig{
		BootstrapNodes: []string{
			p2p.GetLocalPeerInfo(),
		},
		Port: 4002,
	}

	s, err := p2p.NewService(conf)
	if err != nil {
		log.Fatalf("NewService error: %s", err)
	}

	e := s.Start()
	if <-e != nil {
		log.Fatalf("Start error: %s", err)
	}

	if len(os.Args) < 2 {
		select{}
	}
	
	peerStr := os.Args[1]
	peerid, err := peer.IDB58Decode(peerStr[len(peerStr)-46:])
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("peers: ", s.Host().Peerstore().Peers())

	// open new stream with each peer
	ps, err := s.DHT().FindPeer(s.Ctx(), peerid)
	if err != nil {
		log.Fatal(err)
	}

	stream, err := s.Host().NewStream(s.Ctx(), ps.ID, "/polkadot/0.0.0") 
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("sending message...")
	_, err = stream.Write([]byte("Hello, world!\n"))
	if err != nil {
		log.Fatalln(err)
	}

	select{}
}