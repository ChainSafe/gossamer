// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/internal/pubip"
	"github.com/dgraph-io/ristretto"
	badger "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	mempstore "github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	rm "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
)

func newPrivateIPFilters() (privateIPs *ma.Filters, err error) {
	privateCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"100.64.0.0/10",
		"198.18.0.0/15",
		"192.168.0.0/16",
		"169.254.0.0/16",
	}
	privateIPs = ma.NewFilters()
	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return privateIPs, err
		}
		privateIPs.AddFilter(*ipnet, ma.ActionDeny)
	}
	return
}

var (
	privateIPs *ma.Filters
)

func init() {
	var err error
	privateIPs, err = newPrivateIPFilters()
	if err != nil {
		log.Panic(err)
	}
}

const (
	peerSetSlotAllocTime = time.Second * 2
	connectTimeout       = time.Second * 5
)

// host wraps libp2p host with network host configuration and services
type host struct {
	ctx             context.Context
	p2pHost         libp2phost.Host
	discovery       *discovery
	bootnodes       []peer.AddrInfo
	persistentPeers []peer.AddrInfo
	protocolID      protocol.ID
	cm              *ConnManager
	ds              *badger.Datastore
	messageCache    *messageCache
	bwc             *metrics.BandwidthCounter
	closeSync       sync.Once
	externalAddr    ma.Multiaddr
}

func newHost(ctx context.Context, cfg *Config) (*host, error) {
	// create multiaddress (without p2p identity)
	listenAddress := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port)
	if cfg.ListenAddress != "" {
		listenAddress = cfg.ListenAddress
	}
	addr, err := ma.NewMultiaddr(listenAddress)
	if err != nil {
		return nil, err
	}

	portString, err := addr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return nil, err
	}

	port, err := strconv.ParseUint(portString, 10, 64)
	if err != nil {
		return nil, err
	}
	var externalAddr ma.Multiaddr

	switch {
	case strings.TrimSpace(cfg.PublicIP) != "":
		ip := net.ParseIP(cfg.PublicIP)
		if ip == nil {
			return nil, fmt.Errorf("invalid public ip: %s", cfg.PublicIP)
		}
		logger.Debugf("using config PublicIP: %s", ip)
		externalAddr, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
		if err != nil {
			return nil, err
		}
	case strings.TrimSpace(cfg.PublicDNS) != "":
		logger.Debugf("using config PublicDNS: %s", cfg.PublicDNS)
		externalAddr, err = ma.NewMultiaddr(fmt.Sprintf("/dns/%s/tcp/%d", cfg.PublicDNS, port))
		if err != nil {
			return nil, err
		}
	default:
		ip, err := pubip.Get()
		if err != nil {
			logger.Errorf("failed to get public IP error: %v", err)
		} else {
			logger.Debugf("got public IP address %s", ip)
			externalAddr, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
			if err != nil {
				return nil, err
			}
		}
	}

	// format bootnodes
	bns, err := stringsToAddrInfos(cfg.Bootnodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bootnodes: %w", err)
	}

	// format persistent peers
	pps, err := stringsToAddrInfos(cfg.PersistentPeers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse persistent peers: %w", err)
	}

	// We have tried to set maxInPeers and maxOutPeers such that number of peer
	// connections remain between min peers and max peers
	const reservedOnly = false
	peerCfgSet := peerset.NewConfigSet(
		//TODO: there is no any understanding of maxOutPeers and maxInPirs calculations.
		// This needs to be explicitly mentioned

		// maxInPeers is later used in peerstate only and defines available Incoming connection slots
		uint32(cfg.MaxPeers-cfg.MinPeers),
		// maxOutPeers is later used in peerstate only and defines available Outgoing connection slots
		uint32(cfg.MaxPeers/2),
		reservedOnly,
		peerSetSlotAllocTime,
	)

	// create connection manager
	cm, err := newConnManager(cfg.MaxPeers, peerCfgSet)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	for _, pp := range pps {
		cm.persistentPeers.Store(pp.ID, struct{}{})
	}

	// format protocol id
	pid := protocol.ID(cfg.ProtocolID)

	ds, err := badger.NewDatastore(path.Join(cfg.BasePath, "libp2p-datastore"), &badger.DefaultOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p datastore: %w", err)
	}

	ps, err := mempstore.NewPeerstore()
	if err != nil {
		return nil, fmt.Errorf("failed to create peerstore: %w", err)
	}

	limiter := rm.NewFixedLimiter(rm.DefaultLimits.AutoScale())
	var managerOptions []rm.Option

	if cfg.Metrics.Publish {
		rm.MustRegisterWith(prometheus.DefaultRegisterer)
		reporter, err := rm.NewStatsTraceReporter()
		if err != nil {
			return nil, fmt.Errorf("while creating resource manager stats trace reporter: %w", err)
		}

		managerOptions = append(managerOptions, rm.WithTraceReporter(reporter))
	}

	manager, err := rm.NewResourceManager(limiter, managerOptions...)
	if err != nil {
		return nil, fmt.Errorf("while creating the resource manager: %w", err)
	}

	// set libp2p host options
	opts := []libp2p.Option{
		libp2p.ResourceManager(manager),
		libp2p.ListenAddrs(addr),
		libp2p.DisableRelay(),
		libp2p.Identity(cfg.privateKey),
		libp2p.NATPortMap(),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(cm),
		libp2p.AddrsFactory(func(as []ma.Multiaddr) []ma.Multiaddr {
			var addrs []ma.Multiaddr
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
	h, err := libp2p.New(opts...)
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
	discovery := newDiscovery(ctx, h, bns, ds, pid, cfg.MaxPeers, cm.peerSetHandler)

	host := &host{
		ctx:             ctx,
		p2pHost:         h,
		discovery:       discovery,
		bootnodes:       bns,
		protocolID:      pid,
		cm:              cm,
		ds:              ds,
		persistentPeers: pps,
		messageCache:    msgCache,
		bwc:             bwc,
		externalAddr:    externalAddr,
	}

	cm.host = host
	return host, nil
}

// close closes host services and the libp2p host (host services first)
func (h *host) close() error {
	// close DHT service
	err := h.discovery.stop()
	if err != nil {
		logger.Errorf("Failed to close DHT service: %s", err)
		return err
	}

	// close libp2p host
	err = h.p2pHost.Close()
	if err != nil {
		logger.Errorf("Failed to close libp2p host: %s", err)
		return err
	}

	h.closeSync.Do(func() {
		err = h.p2pHost.Peerstore().Close()
		if err != nil {
			logger.Errorf("Failed to close libp2p peerstore: %s", err)
			return
		}

		err = h.ds.Close()
		if err != nil {
			logger.Errorf("Failed to close libp2p host datastore: %s", err)
			return
		}
	})
	return nil
}

// registerStreamHandler registers the stream handler for the given protocol id.
func (h *host) registerStreamHandler(pid protocol.ID, handler func(network.Stream)) {
	h.p2pHost.SetStreamHandler(pid, handler)
}

// connect connects the host to a specific peer address
func (h *host) connect(p peer.AddrInfo) (err error) {
	h.p2pHost.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
	ctx, cancel := context.WithTimeout(h.ctx, connectTimeout)
	defer cancel()
	err = h.p2pHost.Connect(ctx, p)
	return err
}

// bootstrap connects the host to the configured bootnodes
func (h *host) bootstrap() {
	for _, info := range h.persistentPeers {
		h.p2pHost.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		h.cm.peerSetHandler.AddReservedPeer(0, info.ID)
	}

	for _, addrInfo := range h.bootnodes {
		logger.Debugf("bootstrapping to peer %s", addrInfo.ID)
		h.p2pHost.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)
		h.cm.peerSetHandler.AddPeer(0, addrInfo.ID)
		ctx, cancel := context.WithTimeout(h.ctx, connectTimeout)
		err := h.p2pHost.Connect(ctx, addrInfo)
		if err != nil {
			logger.Errorf("Failed to connect to peer %s: %s", addrInfo.ID, err)
		}
		cancel()
	}
}

// send creates a new outbound stream with the given peer and writes the message. It also returns
// the newly created stream.
func (h *host) send(p peer.ID, pid protocol.ID, msg Message) (network.Stream, error) {
	// open outbound stream with host protocol id
	stream, err := h.p2pHost.NewStream(h.ctx, p, pid)
	if err != nil {
		logger.Tracef("failed to open new stream with peer %s using protocol %s: %s", p, pid, err)
		return nil, err
	}

	logger.Tracef(
		"Opened stream with host %s, peer %s and protocol %s",
		h.id(), p, pid)

	err = h.writeToStream(stream, msg)
	if err != nil {
		return nil, err
	}

	logger.Tracef(
		"Sent message %s to peer %s using protocol %s and host %s",
		msg, p, pid, h.id())

	return stream, nil
}

func (h *host) writeToStream(s network.Stream, msg Message) error {
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

// id returns the host id
func (h *host) id() peer.ID {
	return h.p2pHost.ID()
}

// Peers returns connected peers
func (h *host) peers() []peer.ID {
	return h.p2pHost.Network().Peers()
}

// addReservedPeers adds the peers `addrs` to the protected peers list and connects to them
func (h *host) addReservedPeers(addrs ...string) error {
	for _, addr := range addrs {
		mAddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return err
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(mAddr)
		if err != nil {
			return err
		}
		h.p2pHost.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)
		h.cm.peerSetHandler.AddReservedPeer(0, addrInfo.ID)
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
		h.cm.peerSetHandler.RemoveReservedPeer(0, peerID)
		h.p2pHost.ConnManager().Unprotect(peerID, "")
	}

	return nil
}

// supportsProtocol checks if the protocol is supported by peerID
// returns an error if could not get peer protocols
func (h *host) supportsProtocol(peerID peer.ID, protocol protocol.ID) (bool, error) {
	peerProtocols, err := h.p2pHost.Peerstore().SupportsProtocols(peerID, protocol)
	if err != nil {
		return false, err
	}

	return len(peerProtocols) > 0, nil
}

// peerCount returns the number of connected peers
func (h *host) peerCount() int {
	peers := h.p2pHost.Network().Peers()
	return len(peers)
}

// multiaddrs returns the multiaddresses of the host
func (h *host) multiaddrs() (multiaddrs []ma.Multiaddr) {
	addrs := h.p2pHost.Addrs()
	for _, addr := range addrs {
		multiaddr, err := ma.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", addr, h.id()))
		if err != nil {
			continue
		}
		multiaddrs = append(multiaddrs, multiaddr)
	}
	return multiaddrs
}

// protocols returns all protocols currently supported by the node as strings.
func (h *host) protocols() []string {
	protocolIDs := h.p2pHost.Mux().Protocols()
	protocols := make([]string, len(protocolIDs))
	for i := range protocolIDs {
		protocols[i] = string(protocolIDs[i])
	}
	return protocols
}

// closePeer closes connection with peer.
func (h *host) closePeer(peer peer.ID) error {
	return h.p2pHost.Network().ClosePeer(peer)
}

func (h *host) closeProtocolStream(pID protocol.ID, p peer.ID) {
	connToPeer := h.p2pHost.Network().ConnsToPeer(p)
	for _, c := range connToPeer {
		for _, st := range c.GetStreams() {
			if st.Protocol() != pID {
				continue
			}
			err := st.Close()
			if err != nil {
				logger.Tracef("Failed to close stream for protocol %s: %s", pID, err)
			}
		}
	}
}
