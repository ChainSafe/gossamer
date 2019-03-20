package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	ps "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

const LOCAL_PEER_ENDPOINT = "http://localhost:5001/api/v0/id"

// IDOutput is borrowed from ipfs code to parse the results of the command `ipfs id`
type IDOutput struct {
	ID              string
	PublicKey       string
	Addresses       []string
	AgentVersion    string
	ProtocolVersion string
}

// GetLocalPeerInfo gets the local ipfs daemon's address for bootstrapping
func GetLocalPeerInfo() (string, error) {
	resp, err := http.Get(LOCAL_PEER_ENDPOINT)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var js IDOutput
	err = json.Unmarshal(body, &js)
	if err != nil {
		return "", err
	}

	for _, addr := range js.Addresses {
		if addr[0:8] == "/ip4/127" {
			return addr, nil
		}
	}

	return "", err
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
