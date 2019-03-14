package p2p

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	//"io"
	//"os"

	host "github.com/libp2p/go-libp2p-host"
	ps "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	swarm "github.com/libp2p/go-libp2p-swarm"
	net "github.com/libp2p/go-libp2p-net"
	tcp "github.com/libp2p/go-tcp-transport"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	secio "github.com/libp2p/go-libp2p-secio"
	yamux "github.com/whyrusleeping/go-smux-yamux"
	csms "github.com/libp2p/go-conn-security-multistream"
	msmux "github.com/whyrusleeping/go-smux-multistream"
)

func stringToPeerInfo(peer string) (ps.PeerInfo, error) {
	maddr := ma.StringCast(peer)
	p, err := ps.InfoFromP2pAddr(maddr)
	return *p, err
}

func stringsToPeerInfos(peers []string) ([]ps.PeerInfo, error) {
	pinfos := make([]ps.PeerInfo, len(peers))
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

// This code is borrowed from the go-ipfs bootstrap process
func (s *Service) bootstrapConnect(ctx context.Context, ph host.Host, peers []ps.PeerInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	// create new swarm which will be used to handle the network of peers
	swarm := swarm.NewSwarm(s.ctx, s.host.ID(), s.host.Peerstore(), nil)
	swarm.SetStreamHandler(func(s net.Stream) {
		defer s.Close()
		fmt.Println("Got a stream from: ", s.Conn().RemotePeer(), s)
		fmt.Fprintln(s, "Hello Friend!")
	})

	err := swarm.AddTransport(tcp.NewTCPTransport(GenUpgrader(swarm)))
	if err != nil {
		return err
	}

	// begin bootstrapping
	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.
		// Also, performed asynchronously for dial speed.

		wg.Add(1)
		go func(p ps.PeerInfo) {
			defer wg.Done()
			defer log.Println(ctx, "bootstrapDial", ph.ID(), p.ID)
			log.Printf("%s bootstrapping to %s", ph.ID(), p.ID)

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, ps.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Println(ctx, "bootstrapDialFailed", p.ID)
				log.Printf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			log.Println(ctx, "bootstrapDialSuccess", p.ID)
			log.Printf("bootstrapped with %v", p.ID)

			// open new stream with each peer
			s, err := swarm.NewStream(s.ctx, p.ID)
			if err != nil {
				panic(err)
			}
			defer s.Close()
			//io.Copy(os.Stdout, s) // pipe the stream to stdout
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
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
	return nil
}
