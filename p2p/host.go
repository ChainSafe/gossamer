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
	"github.com/libp2p/go-libp2p/p2p/discovery"

	libp2phost "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	ma "github.com/multiformats/go-multiaddr"
)

const DefaultProtocolId = protocol.ID("/gossamer/dot/0")
const mdnsPeriod = time.Minute

// host describes a wrapper for a libp2p host with dht and mdns
type host struct {
	ctx         context.Context
	h           libp2phost.Host
	hostAddr    ma.Multiaddr
	dht         *kaddht.IpfsDHT
	bootnodes   []peer.AddrInfo
	noBootstrap bool
	noMdns      bool
	mdns        discovery.Service
	protocolId  protocol.ID
	// TODO: store status in peer metadata
	peerStatus map[peer.ID]bool
}

// newHost creates a host wrapper with an attached libp2p host instance
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

	// create datastore and dht service
	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	dht := kaddht.NewDHT(ctx, h, dstore)

	// wrap the host with routed host so we can look up peers in dht
	h = rhost.Wrap(h, dht)

	// use "p2p" for multiaddress format
	ma.SwapToP2pMultiaddrs()

	// create host multiaddress including host "p2p" id
	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID()))
	if err != nil {
		return nil, err
	}

	// format bootstrap nodes
	bootstrapNodes, err := stringsToPeerInfos(cfg.BootstrapNodes)
	if err != nil {
		return nil, err
	}

	// TODO: store status in peer metadata
	peerStatus := make(map[peer.ID]bool)

	return &host{
		ctx:         ctx,
		h:           h,
		hostAddr:    hostAddr,
		dht:         dht,
		bootnodes:   bootstrapNodes,
		protocolId:  protocolId,
		noBootstrap: cfg.NoBootstrap,
		noMdns:      cfg.NoMdns,
		peerStatus:  peerStatus, // TODO: store status in peer metadata
	}, nil

}

// bootstrap connects the host to the configured bootnodes
func (h *host) bootstrap() {
	if len(h.bootnodes) == 0 && !h.noBootstrap {
		log.Error(
			"bootstrap",
			"error", "NoBootrap must be true if no bootnodes are defined",
		)
	}
	// loop through bootnodes
	for _, peerInfo := range h.bootnodes {
		log.Debug(
			"bootstrap",
			"host", h.h.ID(),
			"peer", peerInfo.ID,
		)
		// connect to each bootnode
		err := h.h.Connect(h.ctx, peerInfo)
		if err != nil {
			log.Error("bootstrap", "error", err)
		}
	}
}

// startMdns starts a new discovery mdns service
func (h *host) startMdns() {
	if !h.noMdns {

		// create new mdns service
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
			"host", h.h.ID(),
			"period", mdnsPeriod,
			"protocol", h.protocolId,
		)

		// register notifee on mdns service
		mdns.RegisterNotifee(Notifee{ctx: h.ctx, host: h.h})

		h.mdns = mdns
	}
}

// printHostAddresses prints the multiaddresses of the host
func (h *host) printHostAddresses() {
	fmt.Println("Listening on the following addresses...")
	for _, addr := range h.h.Addrs() {
		fmt.Println(addr.Encapsulate(h.hostAddr).String())
	}
}

// registerStreamHandler registers the stream handler (see handleStream)
func (h *host) registerStreamHandler(handler func(net.Stream)) {
	h.h.SetStreamHandler(h.protocolId, handler)
}

// connect connects the host to a specific peer address
func (h *host) connect(addrInfo peer.AddrInfo) (err error) {
	err = h.h.Connect(h.ctx, addrInfo)
	return err
}

// getExistingStream attempts to get an existing stream
func (h *host) getExistingStream(p peer.ID) (stream net.Stream, err error) {
	for _, conn := range h.h.Network().ConnsToPeer(p) {
		for _, stream := range conn.GetStreams() {
			if stream.Protocol() == h.protocolId {
				return stream, nil
			}
		}
	}
	return nil, nil
}

// openStream opens a new stream
func (h *host) openStream(p peer.ID) (stream net.Stream, err error) {
	stream, err = h.h.NewStream(h.ctx, p, h.protocolId)
	if err != nil {
		log.Error("new stream", "error", err)
		return nil, err
	}
	log.Debug(
		"opened stream",
		"host", stream.Conn().LocalPeer(),
		"peer", stream.Conn().RemotePeer(),
		"protocol", stream.Protocol(),
	)
	return stream, nil
}

// send sends a non-status message to a specific peer
func (h *host) send(pid peer.ID, msg Message) (err error) {

	// TODO: investigate existing stream breaking status exchange

	// stream, err := h.getExistingStream(pid)
	// if err != nil {
	// 	log.Error("get stream", "error", err)
	// 	return err
	// }

	stream, err := h.openStream(pid)
	if err != nil {
		log.Error("open stream", "error", err)
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

	return nil
}

// broadcast sends a message to each connected peer
func (h *host) broadcast(msg Message) {
	log.Debug(
		"broadcasting",
		"host", h.id(),
		"message", msg,
	)

	for _, peer := range h.h.Network().Peers() {
		log.Debug(
			"sending message",
			"host", h.id(),
			"peer", peer,
			"message", msg,
		)

		err := h.send(peer, msg)
		if err != nil {
			log.Error("sending message", "error", err)
		}
	}
}

// ping pings a peer using dht
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

// close shuts down the host and all its components
func (h *host) close() error {
	err := h.h.Close()
	if err != nil {
		return err
	}
	err = h.dht.Close()
	if err != nil {
		return err
	}
	return nil
}

// fullAddrs returns the multiaddresses of the host
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
