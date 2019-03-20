package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	csms "github.com/libp2p/go-conn-security-multistream"
	ps "github.com/libp2p/go-libp2p-peerstore"
	secio "github.com/libp2p/go-libp2p-secio"
	swarm "github.com/libp2p/go-libp2p-swarm"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	tcp "github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

var LOCAL_PEER_ENDPOINT = "http://localhost:5001/api/v0/id"

// Borrowed from ipfs code to parse the results of the command `ipfs id`
type IdOutput struct {
	ID              string
	PublicKey       string
	Addresses       []string
	AgentVersion    string
	ProtocolVersion string
}

// get the local ipfs daemon's address for bootstrapping
func GetLocalPeerInfo() string {
	resp, err := http.Get(LOCAL_PEER_ENDPOINT)
	if err != nil {
		log.Fatalln(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var js IdOutput
	err = json.Unmarshal(body, &js)
	if err != nil {
		log.Fatalln(err)
	}
	for _, addr := range js.Addresses {
		// For some reason, possibly NAT traversal, we need to grab the loopback ip address
		if addr[0:8] == "/ip4/127" {
			return addr
		}
	}
	log.Fatalln(err)
	return ""
}

func stringToPeerInfo(peer string) (*ps.PeerInfo, error) {
	maddr := ma.StringCast(peer)
	p, err := ps.InfoFromP2pAddr(maddr)
	return p, err
}

func stringsToPeerInfos(peers []string) ([]*ps.PeerInfo, error) {
	pinfos := make([]*ps.PeerInfo, len(peers))
	for i, peer := range peers {
		p, err := stringToPeerInfo(peer)
		if err != nil {
			return nil, err
		}
		pinfos[i] = p
	}
	return pinfos, nil
}

// GenUpgrader creates a new connection upgrader for use with this swarm.
func GenUpgrader(n *swarm.Swarm) *tptu.Upgrader {
	id := n.LocalPeer()
	pk := n.Peerstore().PrivKey(id)
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(secio.ID, &secio.Transport{
		LocalID:    id,
		PrivateKey: pk,
	})

	stMuxer := msmux.NewBlankTransport()
	stMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	return &tptu.Upgrader{
		Secure:  secMuxer,
		Muxer:   stMuxer,
		Filters: n.Filters,
	}
}

// NewSwarm creates a new swarm which will be used to handle the network of peers
func (s *Service) NewSwarm() (*swarm.Swarm, error) {
	swarm := swarm.NewSwarm(s.ctx, s.host.ID(), s.host.Peerstore(), nil)
	swarm.SetStreamHandler(handleStream)
	err := swarm.AddTransport(tcp.NewTCPTransport(GenUpgrader(swarm)))
	return swarm, err
}

// this code is borrowed from the go-ipfs bootstrap process
func (s *Service) bootstrapConnect() error {
	peers := s.bootstrapNodes
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	// begin bootstrapping
	errs := make(chan error, len(peers))
	var wg sync.WaitGroup

	var err error
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.

		wg.Add(1)
		go func(p *ps.PeerInfo) {
			defer wg.Done()
			defer log.Println(s.ctx, "bootstrapDial", s.host.ID(), p.ID)
			log.Printf("%s bootstrapping to %s", s.host.ID(), p.ID)

			s.host.Peerstore().AddAddrs(p.ID, p.Addrs, ps.PermanentAddrTTL)
			if err = s.host.Connect(s.ctx, *p); err != nil {
				log.Println(s.ctx, "bootstrapDialFailed", p.ID)
				log.Printf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			log.Println(s.ctx, "bootstrapDialSuccess", p.ID)
			log.Printf("bootstrapped with %v", p.ID)
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// drain the errs channel, counting the results.
	close(errs)
	count := 0
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return err
}
