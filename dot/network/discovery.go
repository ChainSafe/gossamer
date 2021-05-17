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
	"context"
	"fmt"
	"time"

	badger "github.com/ipfs/go-ds-badger2"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	libp2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
)

// discovery handles discovery of new peers via the kademlia DHT
type discovery struct {
	ctx                context.Context
	dht                *dual.DHT
	h                  libp2phost.Host
	bootnodes          []peer.AddrInfo
	ds                 *badger.Datastore
	pid                protocol.ID
	minPeers, maxPeers int
}

func newDiscovery(ctx context.Context, h libp2phost.Host, bootnodes []peer.AddrInfo, ds *badger.Datastore, pid protocol.ID, min, max int) *discovery {
	return &discovery{
		ctx:       ctx,
		h:         h,
		bootnodes: bootnodes,
		ds:        ds,
		pid:       pid,
		minPeers:  min,
		maxPeers:  max,
	}
}

// start creates the DHT.
func (d *discovery) start() error {
	if len(d.bootnodes) == 0 {
		// get all currently connected peers and use them to bootstrap the DHT
		peers := d.h.Network().Peers()

		for {
			if len(peers) == 0 {
				logger.Info("no peers yet, waiting to start DHT...")
				time.Sleep(time.Second * 10) // wait for peers to connect before starting DHT
			} else {
				break
			}

			peers = d.h.Network().Peers()
		}

		for _, p := range peers {
			d.bootnodes = append(d.bootnodes, d.h.Peerstore().PeerInfo(p))
		}
	}

	logger.Info("starting DHT...", "bootnodes", d.bootnodes)

	dhtOpts := []dual.Option{
		dual.DHTOption(kaddht.Datastore(d.ds)),
		dual.DHTOption(kaddht.BootstrapPeers(d.bootnodes...)),
		dual.DHTOption(kaddht.V1ProtocolOverride(d.pid + "/kad")),
		dual.DHTOption(kaddht.Mode(kaddht.ModeAutoServer)),
		//dual.DHTOption(kaddht.Mode(kaddht.ModeServer)),
	}

	// create DHT service
	dht, err := dual.New(d.ctx, d.h, dhtOpts...)
	if err != nil {
		return err
	}

	d.dht = dht
	return d.discoverAndAdvertise()
}

func (d *discovery) stop() error {
	if d.dht == nil {
		return nil
	}

	return d.dht.Close()
}

func (d *discovery) discoverAndAdvertise() error {
	rd := libp2pdiscovery.NewRoutingDiscovery(d.dht)

	err := d.dht.Bootstrap(d.ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// wait to connect to bootstrap peers
	time.Sleep(time.Second)
	peersToTry := make(map[*peer.AddrInfo]struct{})

	// go func() {
	// 	ttl := time.Second * 30

	// 	for {
	// 		select {
	// 		case <-time.After(ttl):
	// 			logger.Info("advertising ourselves in the DHT...")
	// 			err := d.dht.Bootstrap(d.ctx)
	// 			if err != nil {
	// 				logger.Warn("failed to bootstrap DHT", "error", err)
	// 				continue
	// 			}

	// 			ttl, err = rd.Advertise(d.ctx, string(d.pid))
	// 			if err != nil {
	// 				logger.Warn("failed to advertise in the DHT", "error", err)
	// 				ttl = time.Minute
	// 			}
	// 		case <-d.ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	go func() {
		logger.Info("attempting to find peers...")
		peerCh, err := rd.FindPeers(d.ctx, string(d.pid))
		if err != nil {
			logger.Error("failed to begin finding peers via DHT", "err", err)
			return
		}

		for {
			select {
			case <-d.ctx.Done():
				return
			case <-time.After(time.Minute):
				if len(d.h.Network().Peers()) > d.minPeers {
					continue
				}

				// reconnect to peers if peer count is low
				for p := range peersToTry {
					logger.Info("trying to connect to cached peer", "peer", p.ID)
					err = d.h.Connect(d.ctx, *p)
					if err != nil {
						logger.Info("failed to connect to discovered peer", "peer", p.ID, "err", err)
					}
				}
			case peer := <-peerCh:
				if peer.ID == d.h.ID() || peer.ID == "" {
					continue
				}

				logger.Info("found new peer via DHT", "peer", peer.ID)

				// found a peer, try to connect if we need more peers
				if len(d.h.Network().Peers()) < d.maxPeers {
					err = d.h.Connect(d.ctx, peer)
					if err != nil {
						logger.Info("failed to connect to discovered peer", "peer", peer.ID, "err", err)
					}
				} else {
					d.h.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
					peersToTry[&peer] = struct{}{}
				}
			}
		}
	}()

	logger.Info("DHT discovery started!")
	return nil
}
