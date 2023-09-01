// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"time"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
	badger "github.com/ipfs/go-ds-badger2"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

const (
	checkPeerCountMetrics = "gossamer/network/peer_count"
	peersStoreMetrics     = "gossamer/network/peerstore_count"
)

var (
	startDHTTimeout             = time.Second * 10
	initialAdvertisementTimeout = time.Millisecond
	tryAdvertiseTimeout         = time.Second * 30
	connectToPeersTimeout       = time.Minute
	findPeersTimeout            = time.Minute
)

// discovery handles discovery of new peers via the kademlia DHT
type discovery struct {
	ctx                context.Context
	dht                *dual.DHT
	rd                 *routing.RoutingDiscovery
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

	currentPeers := d.h.Network().Peers()

	t := time.NewTicker(startDHTTimeout)
	defer t.Stop()

	for len(currentPeers) == 0 {
		select {
		case <-t.C:
			logger.Debug("no peers yet, waiting to start DHT...")
			// wait for peers to connect before starting DHT, otherwise DHT bootstrap nodes
			// will be empty and we will fail to fill the routing table
		case <-d.ctx.Done():
			return nil, d.ctx.Err()
		}

		currentPeers = d.h.Network().Peers()
	}

	peers = make([]peer.AddrInfo, len(currentPeers))
	for idx, peer := range currentPeers {
		peers[idx] = d.h.Peerstore().PeerInfo(peer)
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
	logger.Debugf("V1ProtocolOverride %v...", d.pid+"/kad")

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
	d.rd = routing.NewRoutingDiscovery(d.dht)

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
		timer := time.NewTimer(ttl)

		select {
		case <-d.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			logger.Debug("advertising ourselves in the DHT...")
			err := d.dht.Bootstrap(d.ctx)
			if err != nil {
				logger.Warnf("failed to bootstrap DHT: %s", err)
				continue
			}

			ttl, err = d.rd.Advertise(d.ctx, string(d.pid))
			if err != nil {
				logger.Warnf("failed to advertise in the DHT: %s", err)
				ttl = tryAdvertiseTimeout
			}
		}
	}
}

func (d *discovery) checkPeerCount() {
	ticker := time.NewTicker(connectToPeersTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			if len(d.h.Network().Peers()) >= d.maxPeers {
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
		}
	}
}

func (d *discovery) findPeer(peerID peer.ID) (peer.AddrInfo, error) {
	return d.dht.FindPeer(d.ctx, peerID)
}
