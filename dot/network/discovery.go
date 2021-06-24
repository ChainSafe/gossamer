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

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
	badger "github.com/ipfs/go-ds-badger2"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	libp2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
)

const (
	checkPeerCountMetrics = "gossamer/network/peer_count"
	peersStoreMetrics     = "gossamer/network/peerstore_count"
)

var (
	startDHTTimeout             = time.Second * 10
	initialAdvertisementTimeout = time.Millisecond
	tryAdvertiseTimeout         = time.Second * 30
	connectToPeersTimeout       = time.Minute * 5
	findPeersTimeout            = time.Minute
)

// discovery handles discovery of new peers via the kademlia DHT
type discovery struct {
	ctx                context.Context
	dht                *dual.DHT
	rd                 *libp2pdiscovery.RoutingDiscovery
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
			if len(peers) > 0 {
				break
			}

			select {
			case <-time.After(startDHTTimeout):
				logger.Debug("no peers yet, waiting to start DHT...")
				// wait for peers to connect before starting DHT, otherwise DHT bootstrap nodes
				// will be empty and we will fail to fill the routing table
			case <-d.ctx.Done():
				return nil
			}

			peers = d.h.Network().Peers()
		}

		for _, p := range peers {
			d.bootnodes = append(d.bootnodes, d.h.Peerstore().PeerInfo(p))
		}
	}

	logger.Debug("starting DHT...", "bootnodes", d.bootnodes)

	dhtOpts := []dual.Option{
		dual.DHTOption(kaddht.Datastore(d.ds)),
		dual.DHTOption(kaddht.BootstrapPeers(d.bootnodes...)),
		dual.DHTOption(kaddht.V1ProtocolOverride(d.pid + "/kad")),
		dual.DHTOption(kaddht.Mode(kaddht.ModeAutoServer)),
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

	ethmetrics.Unregister(checkPeerCountMetrics)
	ethmetrics.Unregister(peersStoreMetrics)

	return d.dht.Close()
}

func (d *discovery) discoverAndAdvertise() error {
	d.rd = libp2pdiscovery.NewRoutingDiscovery(d.dht)

	err := d.dht.Bootstrap(d.ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// wait to connect to bootstrap peers
	time.Sleep(time.Second)
	go d.advertise()
	go d.checkPeerCount()

	logger.Debug("DHT discovery started!")
	return nil
}

func (d *discovery) advertise() {
	ttl := initialAdvertisementTimeout

	for {
		select {
		case <-time.After(ttl):
			logger.Debug("advertising ourselves in the DHT...")
			err := d.dht.Bootstrap(d.ctx)
			if err != nil {
				logger.Warn("failed to bootstrap DHT", "error", err)
				continue
			}

			ttl, err = d.rd.Advertise(d.ctx, string(d.pid))
			if err != nil {
				logger.Debug("failed to advertise in the DHT", "error", err)
				ttl = tryAdvertiseTimeout
			}
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *discovery) checkPeerCount() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-time.After(connectToPeersTimeout):
			if len(d.h.Network().Peers()) > d.minPeers {
				continue
			}

			ctx, cancel := context.WithTimeout(d.ctx, findPeersTimeout)
			defer cancel()
			d.findPeers(ctx)
		}
	}
}

func (d *discovery) findPeers(ctx context.Context) {
	logger.Debug("attempting to find DHT peers...")
	peerCh, err := d.rd.FindPeers(d.ctx, string(d.pid))
	if err != nil {
		logger.Warn("failed to begin finding peers via DHT", "err", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case peer := <-peerCh:
			if peer.ID == d.h.ID() || peer.ID == "" {
				continue
			}

			logger.Trace("found new peer via DHT", "peer", peer.ID)

			// found a peer, try to connect if we need more peers
			if len(d.h.Network().Peers()) < d.maxPeers {
				err = d.h.Connect(d.ctx, peer)
				if err != nil {
					logger.Trace("failed to connect to discovered peer", "peer", peer.ID, "err", err)
				}
			} else {
				d.h.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
				return
			}
		}
	}
}
