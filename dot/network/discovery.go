// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"time"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
	badger "github.com/ipfs/go-ds-badger2"
	libp2pdiscovery "github.com/libp2p/go-libp2p-core/discovery"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	libp2prouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	checkPeerCountMetrics = "gossamer/network/peer_count"
	peersStoreMetrics     = "gossamer/network/peerstore_count"
)

var (
	startDHTTimeout             = time.Second * 10
	initialAdvertisementTimeout = time.Millisecond
	tryAdvertiseTimeout         = time.Second * 30
	connectToPeersTimeout       = time.Second * 5
	findPeersTimeout            = time.Minute
)

// discovery handles discovery of new peers via the kademlia DHT
type discovery struct {
	ctx                context.Context
	dht                *dual.DHT
	rd                 *libp2prouting.RoutingDiscovery
	h                  libp2phost.Host
	bootnodes          []peer.AddrInfo
	ds                 *badger.Datastore
	pid                protocol.ID
	minPeers, maxPeers int
	handler            PeerSetHandler
}

func newDiscovery(ctx context.Context, h libp2phost.Host,
	bootnodes []peer.AddrInfo, ds *badger.Datastore,
	pid protocol.ID, min, max int, handler PeerSetHandler) *discovery {
	return &discovery{
		ctx:       ctx,
		h:         h,
		bootnodes: bootnodes,
		ds:        ds,
		pid:       pid,
		minPeers:  min,
		maxPeers:  max,
		handler:   handler,
	}
}

func (d *discovery) waitForPeers() (peers []peer.AddrInfo, err error) {
	// get all currently connected peers and use them to bootstrap the DHT
	connPeers := d.h.Network().Conns()

	t := time.NewTicker(startDHTTimeout)
	defer t.Stop()

	for len(connPeers) == 0 {
		select {
		case <-t.C:
			logger.Debug("no peers yet, waiting to start DHT...")
			// wait for peers to connect before starting DHT, otherwise DHT bootstrap nodes
			// will be empty and we will fail to fill the routing table
		case <-d.ctx.Done():
			return nil, d.ctx.Err()
		}

		connPeers = d.h.Network().Conns()
	}

	peers = make([]peer.AddrInfo, len(connPeers))
	for idx, conn := range connPeers {
		peers[idx] = peer.AddrInfo{
			ID:    conn.RemotePeer(),
			Addrs: []ma.Multiaddr{conn.RemoteMultiaddr()},
		}
	}

	return peers, nil
}

// start creates the DHT.
func (d *discovery) start() error {
	if len(d.bootnodes) == 0 {
		peers, err := d.waitForPeers()
		if err != nil {
			return fmt.Errorf("failed while waiting for peers: %w", err)
		}

		d.bootnodes = peers
	}

	logger.Debugf("starting DHT with bootnodes %v...", d.bootnodes)

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
	d.rd = libp2prouting.NewRoutingDiscovery(d.dht)

	err := d.dht.Bootstrap(d.ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// wait to connect to bootstrap peers
	time.Sleep(3 * time.Second)
	go d.advertise()
	go d.checkPeerCount()

	logger.Debug("DHT discovery started!")
	return nil
}

func (d *discovery) advertise() {
	ttl := initialAdvertisementTimeout

	for {
		timer := time.NewTimer(ttl)

		select {
		case <-d.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			logger.Debug("advertising ourselves in the DHT...")
			var err error
			ttl, err = d.rd.Advertise(d.ctx, string(d.pid), libp2pdiscovery.TTL(ttl))
			if err != nil {
				logger.Warnf("failed to advertise in the DHT: %s", err)
				ttl = tryAdvertiseTimeout
			}
		}
	}
}

func (d *discovery) checkPeerCount() {
	timer := time.NewTicker(connectToPeersTimeout)
	defer timer.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-timer.C:
			if len(d.h.Network().Conns()) > d.minPeers {
				continue
			}

			d.findPeers()
		}
	}
}

func (d *discovery) findPeers() {
	logger.Debug("attempting to find DHT peers...")
	peerCh, err := d.rd.FindPeers(d.ctx, string(d.pid))
	if err != nil {
		logger.Warnf("failed to begin finding peers via DHT: %s", err)
		return
	}

	timer := time.NewTimer(findPeersTimeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return
		case peer := <-peerCh:
			if peer.ID == d.h.ID() || peer.ID == "" {
				continue
			}

			logger.Tracef("found new peer %s via DHT", peer.ID)
			d.h.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
			d.handler.AddPeer(0, peer.ID)

			if !timer.Stop() {
				<-timer.C
			}
		}
	}
}

func (d *discovery) findPeer(peerID peer.ID) (peer.AddrInfo, error) {
	return d.dht.FindPeer(d.ctx, peerID)
}
