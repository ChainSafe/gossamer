package p2p

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	host "github.com/libp2p/go-libp2p-host"
	ps "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
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

// This code is borrowed from the go-ipfs bootstrap process
func bootstrapConnect(ctx context.Context, ph host.Host, peers []ps.PeerInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

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
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
	close(errs)
	count := 0
	var err error
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
