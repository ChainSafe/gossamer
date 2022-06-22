// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	libp2pdiscovery "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// Notifee See https://godoc.org/github.com/libp2p/go-libp2p/p2p/discovery#Notifee
type Notifee struct {
	logger log.LeveledLogger
	ctx    context.Context
	host   *host
}

// mdns submodule
type mdns struct {
	logger log.LeveledLogger
	host   *host
	mdns   libp2pdiscovery.Service
}

// newMDNS creates a new mDNS instance from the host
func newMDNS(host *host) *mdns {
	return &mdns{
		logger: log.NewFromGlobal(log.AddContext("module", "mdns")),
		host:   host,
	}
}

// startMDNS starts a new mDNS discovery service
func (m *mdns) start() {
	m.logger.Debugf(
		"Starting mDNS discovery service with host %s and protocol %s...",
		m.host.id(), m.host.protocolID)

	// create and start service
	mdns := libp2pdiscovery.NewMdnsService(
		m.host.p2pHost,
		string(m.host.protocolID),
		Notifee{
			logger: m.logger,
			ctx:    m.host.ctx,
			host:   m.host,
		},
	)

	if err := mdns.Start(); err != nil {
		m.logger.Debugf(
			"failed to start mDNS discovery service: %s",
			m.host.id(), m.host.protocolID, err)
	}

	m.mdns = mdns
}

// close shuts down the mDNS discovery service
func (m *mdns) close() error {
	// check if service is running
	if m.mdns == nil {
		return nil
	}

	// close service
	err := m.mdns.Close()
	if err != nil {
		m.logger.Warnf("Failed to close mDNS discovery service: %s", err)
		return err
	}

	return nil
}

// HandlePeerFound is event handler called when a peer is found
func (n Notifee) HandlePeerFound(p peer.AddrInfo) {
	n.logger.Debugf(
		"Peer %s found using mDNS discovery, with host %s",
		p.ID, n.host.id())

	n.host.p2pHost.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
	// connect to found peer
	n.host.cm.peerSetHandler.AddPeer(0, p.ID)
}
