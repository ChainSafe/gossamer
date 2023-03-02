// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mdns

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

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
	// Dependencies and configuration injected
	p2pHost    IDNetworker
	serviceTag string
	logger     Logger
	notifee    Notifee

	// Constant fields
	pollPeriod time.Duration

	// Fields set by the Start method.
	server *mdns.Server

	// Internal service management fields.
	// startStopMutex is to prevent concurrent calls to Start and Stop.
	startStopMutex sync.Mutex
	started        bool
	stop           chan struct{}
	done           chan struct{}
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
		pollPeriod: time.Minute,
	}
}

// Start starts the mDNS service.
func (s *Service) Start() (err error) {
	s.startStopMutex.Lock()
	defer s.startStopMutex.Unlock()

	if s.started {
		return nil
	}

	ips, port := getMDNSIPsAndPort(s.p2pHost.Network())

	hostID := s.p2pHost.ID()

	hostIDPretty := hostID.Pretty()
	txt := []string{hostIDPretty}

	mdns.DisableLogging = true
	mdnsService, err := mdns.NewMDNSService(hostIDPretty, s.serviceTag, "", "", int(port), ips, txt)
	if err != nil {
		return fmt.Errorf("creating mDNS service: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: mdnsService})
	if err != nil {
		return fmt.Errorf("creating mDNS server: %w", err)
	}
	s.server = server

	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	ready := make(chan struct{})

	go s.run(ready)
	// It takes a few milliseconds to launch a goroutine
	// so we wait for the run goroutine to be ready.
	<-ready

	s.started = true

	return nil
}

// Stop stops the mDNS service and server.
func (s *Service) Stop() (err error) {
	s.startStopMutex.Lock()
	defer s.startStopMutex.Unlock()

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

func (s *Service) run(ready chan<- struct{}) {
	defer close(s.done)

	ticker := time.NewTicker(s.pollPeriod)
	defer ticker.Stop()

	const queryTimeout = 5 * time.Second
	params := &mdns.QueryParam{
		Domain:  "local",
		Service: s.serviceTag,
		Timeout: queryTimeout,
	}

	close(ready)

	for {
		entriesListeningReady := make(chan struct{})
		entriesListeningDone := make(chan struct{})
		entriesCh := make(chan *mdns.ServiceEntry, 16)
		go func() {
			defer close(entriesListeningDone)
			close(entriesListeningReady)
			for entry := range entriesCh {
				err := s.handleEntry(entry)
				if err != nil {
					s.logger.Warnf("handling mDNS entry: %s", err)
				}
			}
		}()
		<-entriesListeningReady

		params.Entries = entriesCh
		err := mdns.Query(params)
		if err != nil {
			s.logger.Warnf("mdns query failed: %s", err)
		}

		close(entriesCh)
		<-entriesListeningDone

		select {
		case <-ticker.C:
		case <-s.stop:
			return
		}
	}
}

var (
	errEntryHasNoIP = errors.New("MDNS entry has no IP address")
)

func (s *Service) handleEntry(entry *mdns.ServiceEntry) (err error) {
	receivedPeerID, err := peer.Decode(entry.Info)
	if err != nil {
		return fmt.Errorf("parsing peer ID from mdns entry: %w", err)
	}

	if receivedPeerID == s.p2pHost.ID() {
		return nil
	}

	var ip net.IP
	switch {
	case entry.AddrV4 != nil:
		ip = entry.AddrV4
	case entry.AddrV6 != nil:
		ip = entry.AddrV6
	default:
		return fmt.Errorf("%w: from peer id %s", errEntryHasNoIP, receivedPeerID)
	}

	tcpAddress := &net.TCPAddr{
		IP:   ip,
		Port: entry.Port,
	}

	multiAddress, err := manet.FromNetAddr(tcpAddress)
	if err != nil {
		return fmt.Errorf("converting tcp address from peer id %s to multiaddress: %w",
			receivedPeerID, err)
	}

	addressInfo := peer.AddrInfo{
		ID:    receivedPeerID,
		Addrs: []multiaddr.Multiaddr{multiAddress},
	}

	s.logger.Debugf("Peer %s has addresses %s", receivedPeerID, addressInfo.Addrs)
	go s.notifee.HandlePeerFound(addressInfo)
	return nil
}
