// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
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
	"github.com/multiformats/go-multiaddr"
)

const (
	checkPeerCountMetrics = "gossamer/network/peer_count"
	peersStoreMetrics     = "gossamer/network/peerstore_count"
)

var (
	startDHTTimeout             = time.Second * 10
	initialAdvertisementTimeout = time.Millisecond
	retryAdvertiseTimeout       = time.Second * 10
	connectToPeersTimeout       = time.Minute
	findPeersTimeout            = time.Minute
)

// discovery handles discovery of new peers via the kademlia DHT
type discovery struct {
	ctx       context.Context
	dht       *dual.DHT
	rd        *routing.RoutingDiscovery
	h         libp2phost.Host
	bootnodes []peer.AddrInfo
	ds        *badger.Datastore
	pid       protocol.ID
	maxPeers  int
	handler   PeerSetHandler
}

func newDiscovery(ctx context.Context, h libp2phost.Host,
	bootnodes []peer.AddrInfo, ds *badger.Datastore,
	pid protocol.ID, max int, handler PeerSetHandler) *discovery {
	return &discovery{
		ctx:       ctx,
		h:         h,
		bootnodes: bootnodes,
		ds:        ds,
		pid:       pid,
		maxPeers:  max,
		handler:   handler,
	}
}

// start initiates DHT structure with bootnodes if they are provided. Starts nodes discovery with bootstrap process
func (d *discovery) start() error {
	//if len(d.bootnodes) == 0 {
	//	peers, err := d.waitForPeers()
	//	if err != nil {
	//		return fmt.Errorf("failed while waiting for peers: %w", err)
	//	}
	//
	//	d.bootnodes = peers
	//}
	logger.Infof("starting DHT with bootnodes %v...", d.bootnodes)
	logger.Infof("V1ProtocolOverride %v...", d.pid+"/kad")

	dhtOpts := []dual.Option{
		dual.DHTOption(kaddht.Datastore(d.ds)),
		dual.DHTOption(kaddht.BootstrapPeers(d.bootnodes...)),
		dual.DHTOption(kaddht.V1ProtocolOverride(d.pid + "/kad")),
		dual.DHTOption(kaddht.Mode(kaddht.ModeAutoServer)),
		dual.DHTOption(kaddht.AddressFilter(func(as []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			var addrs []multiaddr.Multiaddr
			for _, addr := range as {
				if !privateIPs.AddrBlocked(addr) {
					addrs = append(addrs, addr)
				}
			}

			return append(addrs, d.h.Addrs()...)
		})),
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

//
//func (d *discovery) discoverAndAdvertise() error {
//	d.rd = routing.NewRoutingDiscovery(d.dht)
//
//	err := d.dht.Bootstrap(d.ctx)
//	if err != nil {
//		return fmt.Errorf("failed to bootstrap DHT: %w", err)
//	}
//
//	// wait to connect to bootstrap peers
//	time.Sleep(time.Second)
//	go d.advertise()
//	go d.checkPeerCount()
//
//	return nil
//}

// discoverAndAdvertise
func (d *discovery) discoverAndAdvertise() error {
	d.rd = routing.NewRoutingDiscovery(d.dht)
	bootstrapTimer := time.NewTimer(initialAdvertisementTimeout)
	advertismentTimer := time.NewTimer(initialAdvertisementTimeout)

	go d.checkPeerCount()

	for {
		select {
		case <-d.ctx.Done():
			bootstrapTimer.Stop()
			advertismentTimer.Stop()
			return nil
		case <-bootstrapTimer.C:
			err := d.dht.Bootstrap(d.ctx)
			bootstrapTimer = time.NewTimer(retryAdvertiseTimeout)
			bootstrapTimer = time.NewTimer(time.Second * 5)
			if err != nil {
				logger.Warnf("failed to bootstrap DHT: %s", err)
				continue
			}
		case <-advertismentTimer.C:
			advTTL, err := d.rd.Advertise(d.ctx, string(d.pid))
			if err != nil {
				logger.Warnf("failed to advertise in the DHT: %s", err)
				advertismentTimer = time.NewTimer(retryAdvertiseTimeout)
				continue
			}
			advertismentTimer = time.NewTimer(advTTL)
			advertismentTimer = time.NewTimer(time.Second * 15)
		}
	}
}

// checkPeerCount find peers if amount of connected peers is less then maximum amount allowed by configuration
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
			logger.Infof("Found peer %v", peer)
			logger.Tracef("found new peer %s via DHT", peer.ID)
			d.h.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
			d.handler.AddPeer(0, peer.ID)
		}
	}
}

// waitForPeers passively checks connected peers.
func (d *discovery) waitForPeers() (peers []peer.AddrInfo, err error) {
	// get all currently connected peers and use them to bootstrap the DHT
	currentPeers := d.h.Network().Peers()

	t := time.NewTicker(startDHTTimeout)
	defer t.Stop()

	for len(currentPeers) == 0 {
		select {
		case <-t.C:
			logger.Info("no peers yet, waiting to start DHT...")
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
