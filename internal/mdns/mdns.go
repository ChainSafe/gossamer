// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mdns

import (
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/whyrusleeping/mdns"
)

// Notifee is notified when a new peer is found.
type Notifee interface {
	HandlePeerFound(peer.AddrInfo)
}

// Service implements a mDNS service.
type Service struct {
	server     *mdns.Server
	p2pHost    IDNetworker
	serviceTag string
	notifee    Notifee
	logger     Logger
	started    bool
	stop       chan struct{}
	done       chan struct{}
}

// NewService creates and returns a new mDNS service.
func NewService(p2pHost IDNetworker, serviceTag string,
	logger Logger, notifee Notifee) (service *Service) {
	if serviceTag == "" {
		serviceTag = "_ipfs-discovery._udp"
	}

	return &Service{
		p2pHost:    p2pHost,
		serviceTag: serviceTag,
		notifee:    notifee,
		logger:     logger,
	}
}

// Start starts the mDNS service.
func (s *Service) Start() (err error) {
	ips, port := getMDNSIPsAndPort(s.p2pHost)

	hostID := s.p2pHost.ID()

	hostIDPretty := hostID.Pretty()
	txt := []string{hostIDPretty}
	mdnsService, err := mdns.NewMDNSService(hostIDPretty, s.serviceTag, "", "", int(port), ips, txt)
	if err != nil {
		return fmt.Errorf("creating mDNS service: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: mdnsService})
	if err != nil {
		return fmt.Errorf("creating mDNS server: %w", err)
	}
	s.server = server

	s.started = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})

	go s.run()

	return nil
}

// Stop stops the mDNS service and server.
func (s *Service) Stop() (err error) {
	if !s.started {
		return nil
	}

	defer func() {
		s.started = false
	}()
	close(s.stop)
	<-s.done
	return s.server.Shutdown()
}

func (s *Service) run() {
	defer close(s.done)

	const pollPeriod = time.Minute
	ticker := time.NewTicker(pollPeriod)
	defer ticker.Stop()

	for {
		entriesCh := make(chan *mdns.ServiceEntry, 16)
		go func() {
			for entry := range entriesCh {
				s.handleEntry(entry)
			}
		}()

		const queryTimeout = 5 * time.Second
		params := &mdns.QueryParam{
			Domain:  "local",
			Entries: entriesCh,
			Service: s.serviceTag,
			Timeout: queryTimeout,
		}
		err := mdns.Query(params)
		if err != nil {
			s.logger.Warnf("mdns query failed: %s", err)
		}
		close(entriesCh)

		select {
		case <-ticker.C:
		case <-s.stop:
			return
		}
	}
}

func (s *Service) handleEntry(entry *mdns.ServiceEntry) {
	receivedPeerID, err := peer.Decode(entry.Info)
	if err != nil {
		s.logger.Warnf("error parsing peer ID from mdns entry: %s", err)
		return
	}

	if receivedPeerID == s.p2pHost.ID() {
		return
	}

	var ip net.IP
	switch {
	case entry.AddrV4 != nil:
		ip = entry.AddrV4
	case entry.AddrV6 != nil:
		ip = entry.AddrV6
	default:
		s.logger.Warnf("mdns entry from peer id %s has no IP address", receivedPeerID)
		return
	}

	tcpAddress := &net.TCPAddr{
		IP:   ip,
		Port: entry.Port,
	}

	multiAddress, err := manet.FromNetAddr(tcpAddress)
	if err != nil {
		s.logger.Warnf("failed converting tcp address from peer id %s to multiaddress: %s",
			receivedPeerID, err)
		return
	}

	addressInfo := peer.AddrInfo{
		ID:    receivedPeerID,
		Addrs: []multiaddr.Multiaddr{multiAddress},
	}

	s.logger.Debugf("Peer %s has addresses %s", receivedPeerID, addressInfo.Addrs)
	go s.notifee.HandlePeerFound(addressInfo)
}
