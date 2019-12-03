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

package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/common"

	log "github.com/ChainSafe/log15"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"

	"github.com/libp2p/go-libp2p"

	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	ma "github.com/multiformats/go-multiaddr"
)

const DefaultProtocolId = protocol.ID("/gossamer/dot/0")

const mdnsPeriod = time.Minute

// host wraps libp2p host with host services and information
type host struct {
	ctx         context.Context
	h           libp2phost.Host
	dht         *kaddht.IpfsDHT
	mdns        discovery.Service
	bootnodes   []peer.AddrInfo
	noBootstrap bool
	noGossip    bool
	noMdns      bool
	address     ma.Multiaddr
	protocolId  protocol.ID
	peerStatus  map[peer.ID]bool // TODO: investigate peer status storage options
}

// newHost creates a host wrapper with a new libp2p host instance
func newHost(ctx context.Context, cfg *Config) (*host, error) {

	opts, err := cfg.buildOpts()
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// use default protocol if none provided
	protocolId := protocol.ID(cfg.ProtocolId)
	if protocolId == "" {
		protocolId = DefaultProtocolId
	}

	// create new datastore and DHT
	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	dht := kaddht.NewDHT(ctx, h, dstore)

	// wrap host and DHT with routed host so that we can look up peers in DHT
	h = rhost.Wrap(h, dht)

	// use "p2p" for multiaddress format
	ma.SwapToP2pMultiaddrs()

	// create host multiaddress that includes host "p2p" id
	address, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID()))
	if err != nil {
		return nil, err
	}

	// format bootstrap nodes list
	bootstrapNodes, err := stringsToPeerInfos(cfg.BootstrapNodes)
	if err != nil {
		return nil, err
	}

	// TODO: investigate peer status storage options
	peerStatus := make(map[peer.ID]bool)

	return &host{
		ctx:         ctx,
		h:           h,
		dht:         dht,
		bootnodes:   bootstrapNodes,
		noBootstrap: cfg.NoBootstrap,
		noGossip:    cfg.NoGossip,
		noMdns:      cfg.NoMdns,
		address:     address,
		protocolId:  protocolId,
		peerStatus:  peerStatus, // TODO: investigate peer status storage options
	}, nil

}

// close shuts down the host
func (h *host) close() error {

	// shut down host
	err := h.h.Close()
	if err != nil {
		return err
	}

	// close DHT process
	err = h.dht.Close()
	if err != nil {
		return err
	}

	return nil
}

// bootstrap connects the host to the configured bootnodes
func (h *host) bootstrap() {
	if len(h.bootnodes) == 0 && !h.noBootstrap {
		log.Error(
			"bootstrap",
			"error", "no bootnodes defined and bootstrap enabled",
		)
	}
	// loop through bootnode peers
	for _, peerInfo := range h.bootnodes {
		log.Trace(
			"bootstrap",
			"host", h.id(),
			"peer", peerInfo.ID,
		)
		// connect to each peer
		err := h.connect(peerInfo)
		if err != nil {
			log.Error("connect", "error", err)
		}
	}
}

// startMdns starts a new MDNS discovery service
func (h *host) startMdns() {
	if !h.noMdns {

		// create new MDNS service
		mdns, err := discovery.NewMdnsService(
			h.ctx,
			h.h,
			mdnsPeriod,
			string(h.protocolId),
		)
		if err != nil {
			log.Error("start mdns", "error", err)
		}

		log.Debug(
			"start mdns",
			"host", h.id(),
			"period", mdnsPeriod,
			"protocol", h.protocolId,
		)

		// register notifee on MDNS service
		mdns.RegisterNotifee(Notifee{ctx: h.ctx, host: h.h})

		h.mdns = mdns
	}
}

// printHostAddresses prints the multiaddresses of the host
func (h *host) printHostAddresses() {
	fmt.Println("Listening on the following addresses...")
	for _, addr := range h.h.Addrs() {
		fmt.Println(addr.Encapsulate(h.address).String())
	}
}

// registerConnHandler registers the connection handler (see handleConn)
func (h *host) registerConnHandler(handler func(network.Conn)) {
	h.h.Network().SetConnHandler(handler)
}

// registerStreamHandler registers the stream handler (see handleStream)
func (h *host) registerStreamHandler(handler func(network.Stream)) {
	h.h.SetStreamHandler(h.protocolId, handler)
}

// connect connects the host to a specific peer address
func (h *host) connect(addrInfo peer.AddrInfo) (err error) {
	err = h.h.Connect(h.ctx, addrInfo)
	return err
}

// newStream opens a new stream with a specific peer using the host protocol
func (h *host) newStream(p peer.ID) (network.Stream, error) {

	// create new stream with host protocol id
	stream, err := h.h.NewStream(h.ctx, p, h.protocolId)
	if err != nil {
		return nil, err
	}

	log.Trace(
		"opened stream",
		"host", stream.Conn().LocalPeer(),
		"peer", stream.Conn().RemotePeer(),
		"protocol", stream.Protocol(),
	)

	return stream, nil
}

// send sends a non-status message to a specific peer
func (h *host) send(p peer.ID, msg Message) (err error) {

	stream, err := h.newStream(p)
	if err != nil {
		log.Error("new stream", "error", err)
		return err
	}

	encMsg, err := msg.Encode()
	if err != nil {
		log.Error("encode message", "error", err)
		return err
	}

	_, err = stream.Write(common.Uint16ToBytes(uint16(len(encMsg)))[0:1])
	if err != nil {
		log.Error("write message", "error", err)
		return err
	}

	_, err = stream.Write(encMsg)
	if err != nil {
		log.Error("write message", "error", err)
		return err
	}

	log.Trace(
		"message sent",
		"host", h.id(),
		"peer", p,
		"type", msg.GetType(),
	)

	return nil
}

// broadcast sends a message to each connected peer
func (h *host) broadcast(msg Message) {
	log.Trace(
		"broadcast message",
		"host", h.id(),
		"type", msg.GetType(),
	)
	// loop through connected peers
	for _, peer := range h.h.Network().Peers() {
		// send message to each connect peer
		err := h.send(peer, msg)
		if err != nil {
			log.Error("send message", "error", err)
		}
	}
}

// ping pings a peer using DHT
func (h *host) ping(peer peer.ID) error {
	return h.dht.Ping(h.ctx, peer)
}

// id returns the host id
func (h *host) id() string {
	return h.h.ID().String()
}

// peerCount returns the number of connected peers
func (h *host) peerCount() int {
	peers := h.h.Network().Peers()
	return len(peers)
}

// fullAddrs returns the full multiaddresses of the host
func (h *host) fullAddrs() (maddrs []ma.Multiaddr) {
	addrs := h.h.Addrs()
	for _, a := range addrs {
		maddr, err := ma.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", a, h.h.ID()))
		if err != nil {
			continue
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs
}
