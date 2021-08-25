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
	"net"
	"path"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	badger "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	secio "github.com/libp2p/go-libp2p-secio"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/chyeh/pubip"
)

var privateCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"100.64.0.0/10",
	"198.18.0.0/15",
	"169.254.0.0/16",
}

var connectTimeout = time.Second * 5

// host wraps libp2p host with network host configuration and services
type host struct {
	ctx             context.Context
	h               libp2phost.Host
	discovery       *discovery
	bootnodes       []peer.AddrInfo
	persistentPeers []peer.AddrInfo
	protocolID      protocol.ID
	cm              *ConnManager
	ds              *badger.Datastore
	messageCache    *messageCache
	bwc             *metrics.BandwidthCounter
	closeSync       sync.Once
}

// newHost creates a host wrapper with a new libp2p host instance
func newHost(ctx context.Context, cfg *Config) (*host, error) {
	// create multiaddress (without p2p identity)
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port))
	if err != nil {
		return nil, err
	}

	var externalAddr ma.Multiaddr
	ip, err := pubip.Get()
	if err != nil {
		logger.Error("failed to get public IP", "error", err)
	} else {
		logger.Debug("got public IP", "IP", ip)
		externalAddr, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.Port))
		if err != nil {
			return nil, err
		}
	}

	// create connection manager
	cm := newConnManager(cfg.MinPeers, cfg.MaxPeers)

	// format bootnodes
	bns, err := stringsToAddrInfos(cfg.Bootnodes)
	if err != nil {
		return nil, err
	}

	// format persistent peers
	pps, err := stringsToAddrInfos(cfg.PersistentPeers)
	if err != nil {
		return nil, err
	}

	for _, pp := range pps {
		cm.persistentPeers.Store(pp.ID, struct{}{})
	}

	// format protocol id
	pid := protocol.ID(cfg.ProtocolID)

	ds, err := badger.NewDatastore(path.Join(cfg.BasePath, "libp2p-datastore"), &badger.DefaultOptions)
	if err != nil {
		return nil, err
	}

	privateIPs := ma.NewFilters()
	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr) //nolint
		if err != nil {
			return nil, err
		}

		privateIPs.AddFilter(*ipnet, ma.ActionDeny)
	}

	ps, err := pstoreds.NewPeerstore(ctx, ds, pstoreds.DefaultOpts())
	if err != nil {
		return nil, err
	}

	// set libp2p host options
	opts := []libp2p.Option{
		libp2p.ListenAddrs(addr),
		libp2p.DisableRelay(),
		libp2p.Identity(cfg.privateKey),
		libp2p.NATPortMap(),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(cm),
		libp2p.ChainOptions(libp2p.DefaultSecurity, libp2p.Security(secio.ID, secio.New)), // TODO: deprecate secio?
		libp2p.AddrsFactory(func(as []ma.Multiaddr) []ma.Multiaddr {
			addrs := []ma.Multiaddr{}
			for _, addr := range as {
				if !privateIPs.AddrBlocked(addr) {
					addrs = append(addrs, addr)
				}
			}
			if externalAddr == nil {
				return addrs
			}

			return append(addrs, externalAddr)
		}),
	}

	// create libp2p host instance
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	cacheSize := 64 << 20 // 64 MB
	config := ristretto.Config{
		NumCounters: int64(float64(cacheSize) * 0.05 * 2),
		MaxCost:     int64(float64(cacheSize) * 0.95),
		BufferItems: 64,
		Cost: func(value interface{}) int64 {
			return int64(1)
		},
	}
	msgCache, err := newMessageCache(config, msgCacheTTL)
	if err != nil {
		return nil, err
	}

	bwc := metrics.NewBandwidthCounter()
	discovery := newDiscovery(ctx, h, bns, ds, pid, cfg.MinPeers, cfg.MaxPeers)

	host := &host{
		ctx:             ctx,
		h:               h,
		discovery:       discovery,
		bootnodes:       bns,
		protocolID:      pid,
		cm:              cm,
		ds:              ds,
		persistentPeers: pps,
		messageCache:    msgCache,
		bwc:             bwc,
	}

	cm.host = host
	return host, nil
}

// close closes host services and the libp2p host (host services first)
func (h *host) close() error {
	// close DHT service
	err := h.discovery.stop()
	if err != nil {
		logger.Error("Failed to close DHT service", "error", err)
		return err
	}

	// close libp2p host
	err = h.h.Close()
	if err != nil {
		logger.Error("Failed to close libp2p host", "error", err)
		return err
	}

	h.closeSync.Do(func() {
		err = h.h.Peerstore().Close()
		if err != nil {
			logger.Error("Failed to close libp2p peerstore", "error", err)
			return
		}

		err = h.ds.Close()
		if err != nil {
			logger.Error("Failed to close libp2p host datastore", "error", err)
			return
		}
	})
	return nil
}

// registerStreamHandler registers the stream handler, appending the given sub-protocol to the main protocol ID
func (h *host) registerStreamHandler(sub protocol.ID, handler func(libp2pnetwork.Stream)) {
	h.h.SetStreamHandler(h.protocolID+sub, handler)
}

// registerStreamHandlerWithOverwrite registers the stream handler. if overwrite is true, it uses the passed protocol ID
// for the handler, otherwise it appends the given sub-protocol to the main protocol ID
func (h *host) registerStreamHandlerWithOverwrite(pid protocol.ID, overwrite bool, handler func(libp2pnetwork.Stream)) {
	if overwrite {
		h.h.SetStreamHandler(pid, handler)
	} else {
		h.h.SetStreamHandler(h.protocolID+pid, handler)
	}
}

// connect connects the host to a specific peer address
func (h *host) connect(p peer.AddrInfo) (err error) {
	h.h.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
	ctx, cancel := context.WithTimeout(h.ctx, connectTimeout)
	defer cancel()
	err = h.h.Connect(ctx, p)
	return err
}

// bootstrap connects the host to the configured bootnodes
func (h *host) bootstrap() {
	failed := 0
	all := append(h.bootnodes, h.persistentPeers...)
	for _, addrInfo := range all {
		logger.Debug("bootstrapping to peer", "peer", addrInfo.ID)
		err := h.connect(addrInfo)
		if err != nil {
			logger.Debug("failed to bootstrap to peer", "error", err)
			failed++
		}
	}
	if failed == len(all) && len(all) != 0 {
		logger.Error("failed to bootstrap to any bootnode")
	}
}

// send creates a new outbound stream with the given peer and writes the message. It also returns
// the newly created stream.
func (h *host) send(p peer.ID, pid protocol.ID, msg Message) (libp2pnetwork.Stream, error) {
	// open outbound stream with host protocol id
	stream, err := h.h.NewStream(h.ctx, p, pid)
	if err != nil {
		logger.Trace("failed to open new stream with peer", "peer", p, "protocol", pid, "error", err)
		return nil, err
	}

	logger.Trace(
		"Opened stream",
		"host", h.id(),
		"peer", p,
		"protocol", pid,
	)

	err = h.writeToStream(stream, msg)
	if err != nil {
		return nil, err
	}

	logger.Trace(
		"Sent message to peer",
		"protocol", pid,
		"host", h.id(),
		"peer", p,
		"message", msg.String(),
	)

	return stream, nil
}

func (h *host) writeToStream(s libp2pnetwork.Stream, msg Message) error {
	encMsg, err := msg.Encode()
	if err != nil {
		return err
	}

	msgLen := uint64(len(encMsg))
	lenBytes := uint64ToLEB128(msgLen)
	encMsg = append(lenBytes, encMsg...)

	sent, err := s.Write(encMsg)
	if err != nil {
		return err
	}

	h.bwc.LogSentMessage(int64(sent))

	return nil
}

// getOutboundStream returns the outbound message stream for the given peer or returns
// nil if no outbound message stream exists. For each peer, each host opens an
// outbound message stream and writes to the same stream until closed or reset.
func (h *host) getOutboundStream(p peer.ID, pid protocol.ID) (stream libp2pnetwork.Stream) {
	conns := h.h.Network().ConnsToPeer(p)

	// loop through connections (only one for now)
	for _, conn := range conns {
		streams := conn.GetStreams()

		// loop through connection streams (unassigned streams and ipfs dht streams included)
		for _, stream := range streams {

			// return stream with matching host protocol id and stream direction outbound
			if stream.Protocol() == pid && stream.Stat().Direction == libp2pnetwork.DirOutbound {
				return stream
			}
		}
	}
	return nil
}

// closeStream closes a stream open to the peer with the given sub-protocol, if it exists.
func (h *host) closeStream(p peer.ID, pid protocol.ID) {
	stream := h.getOutboundStream(p, pid)
	if stream != nil {
		_ = stream.Close()
	}
}

// closePeer closes the peer connection
func (h *host) closePeer(peer peer.ID) error { //nolint
	return h.h.Network().ClosePeer(peer)
}

// id returns the host id
func (h *host) id() peer.ID {
	return h.h.ID()
}

// Peers returns connected peers
func (h *host) peers() []peer.ID {
	return h.h.Network().Peers()
}

// addReservedPeers adds the peers `addrs` to the protected peers list and connects to them
func (h *host) addReservedPeers(addrs ...string) error {
	for _, addr := range addrs {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return err
		}

		addinfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return err
		}

		h.h.ConnManager().Protect(addinfo.ID, "")
		if err := h.connect(*addinfo); err != nil {
			return err
		}
	}

	return nil
}

// removeReservedPeers will remove the given peers from the protected peers list
func (h *host) removeReservedPeers(ids ...string) error {
	for _, id := range ids {
		peerID, err := peer.Decode(id)
		if err != nil {
			return err
		}

		h.h.ConnManager().Unprotect(peerID, "")
	}

	return nil
}

// supportsProtocol checks if the protocol is supported by peerID
// returns an error if could not get peer protocols
func (h *host) supportsProtocol(peerID peer.ID, protocol protocol.ID) (bool, error) {
	peerProtocols, err := h.h.Peerstore().SupportsProtocols(peerID, string(protocol))
	if err != nil {
		return false, err
	}

	return len(peerProtocols) > 0, nil
}

// peerCount returns the number of connected peers
func (h *host) peerCount() int {
	peers := h.h.Network().Peers()
	return len(peers)
}

// addrInfo returns the libp2p peer.AddrInfo of the host
func (h *host) addrInfo() peer.AddrInfo {
	return peer.AddrInfo{
		ID:    h.h.ID(),
		Addrs: h.h.Addrs(),
	}
}

// multiaddrs returns the multiaddresses of the host
func (h *host) multiaddrs() (multiaddrs []ma.Multiaddr) {
	addrs := h.h.Addrs()
	for _, addr := range addrs {
		multiaddr, err := ma.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", addr, h.id()))
		if err != nil {
			continue
		}
		multiaddrs = append(multiaddrs, multiaddr)
	}
	return multiaddrs
}

// protocols returns all protocols currently supported by the node
func (h *host) protocols() []string {
	return h.h.Mux().Protocols()
}
