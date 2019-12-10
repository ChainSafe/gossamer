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
// GNU Lesser General Public License for more detailg.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"context"
	"time"

	log "github.com/ChainSafe/log15"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery"
)

const mdnsPeriod = time.Minute

// See https://godoc.org/github.com/libp2p/go-libp2p/p2p/discovery#Notifee
type Notifee struct {
	ctx  context.Context
	host libp2phost.Host
}

// discovery submodule
type disc struct {
	ctx  context.Context
	host *host
	mdns discovery.Service
}

// newDiscovery creates a new discovery instance from the host
func newDiscovery(ctx context.Context, host *host) (d *disc, err error) {
	d = &disc{
		ctx:  ctx,
		host: host,
	}
	return d, err
}

// startMdns starts a new mDNS discovery service
func (d *disc) startMdns() {

	log.Trace(
		"Starting MDNS...",
		"host", d.host.id(),
		"period", mdnsPeriod,
		"protocol", d.host.protocolId,
	)

	// create and start mDNS discovery service
	mdns, err := discovery.NewMdnsService(
		d.ctx,
		d.host.h,
		mdnsPeriod,
		string(d.host.protocolId),
	)
	if err != nil {
		log.Error("Failed to start mDNS discovery service", "err", err)
	}

	// register Notifee on MDNS service
	mdns.RegisterNotifee(Notifee{
		ctx:  d.ctx,
		host: d.host.h,
	})

	d.mdns = mdns
}

// close closes the mDNS discovery service
func (d *disc) closeMdns() error {

	// check if service is running
	if d.mdns != nil {

		// close mDNS discovery service
		err := d.mdns.Close()
		if err != nil {
			return err
		}

	}

	return nil
}

// HandlePeerFound is invoked when a peer in discovered by the mDNS service
func (n Notifee) HandlePeerFound(p peer.AddrInfo) {
	log.Trace(
		"Peer found using mDNS discovery service",
		"host", n.host.ID(),
		"peer", p.ID,
	)

	// add peer address to peerstore
	n.host.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)

	// connect to found peer
	err := n.host.Connect(n.ctx, p)
	if err != nil {
		log.Error("Failed to connect to peer using mDNS discovery service", "err", err)
	}
}
