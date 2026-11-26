// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package authoritydiscovery

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// ErrServiceClosed is returned when operations are attempted on a closed service
var ErrServiceClosed = errors.New("service is closed")

// AuthorityID represents an authority identifier
type AuthorityID []byte

// ServiceToWorkerMsg represents messages sent from Service to Worker
type ServiceToWorkerMsg interface {
	serviceToWorkerMsg()
}

// GetAddressesByAuthorityID is a message to get addresses for an authority
type GetAddressesByAuthorityID struct {
	AuthorityID AuthorityID
	Response    chan<- *AddressesResponse
}

func (GetAddressesByAuthorityID) serviceToWorkerMsg() {}

// GetAuthorityIDsByPeerID is a message to get authority IDs for a peer
type GetAuthorityIDsByPeerID struct {
	PeerID   peer.ID
	Response chan<- *AuthorityIDsResponse
}

func (GetAuthorityIDsByPeerID) serviceToWorkerMsg() {}

// AddressesResponse represents the response for address queries
type AddressesResponse struct {
	Addresses []ma.Multiaddr
	Err       error
}

// AuthorityIDsResponse represents the response for authority ID queries
type AuthorityIDsResponse struct {
	AuthorityIDs []AuthorityID
	Err          error
}

// Service provides the public API for authority discovery
type Service struct {
	toWorker chan<- ServiceToWorkerMsg
	closed   bool
	mu       sync.RWMutex
}

// NewService creates a new Service instance
func NewService(toWorker chan<- ServiceToWorkerMsg) *Service {
	return &Service{
		toWorker: toWorker,
		closed:   false,
	}
}

// GetAddressesByAuthorityID retrieves addresses associated with an authority ID
func (s *Service) GetAddressesByAuthorityID(ctx context.Context, authority AuthorityID) ([]ma.Multiaddr, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrServiceClosed
	}
	s.mu.RUnlock()

	response := make(chan *AddressesResponse, 1)
	msg := GetAddressesByAuthorityID{
		AuthorityID: authority,
		Response:    response,
	}

	select {
	case s.toWorker <- msg:
		select {
		case resp := <-response:
			if resp.Err != nil {
				return nil, resp.Err
			}
			return resp.Addresses, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetAuthorityIDsByPeerID retrieves authority IDs associated with a peer ID
func (s *Service) GetAuthorityIDsByPeerID(ctx context.Context, peerID peer.ID) ([]AuthorityID, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrServiceClosed
	}
	s.mu.RUnlock()

	response := make(chan *AuthorityIDsResponse, 1)
	msg := GetAuthorityIDsByPeerID{
		PeerID:   peerID,
		Response: response,
	}

	select {
	case s.toWorker <- msg:
		select {
		case resp := <-response:
			if resp.Err != nil {
				return nil, resp.Err
			}
			return resp.AuthorityIDs, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the service
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Note: Closing s.toWorker would typically be done by the owner of the channel
	// or when the worker itself is shut down. The service just stops sending.
	if s.closed {
		return
	}
	s.closed = true
}

// Worker represents the worker that processes messages from the service
type Worker struct {
	fromService <-chan ServiceToWorkerMsg

	// Maps for storing authority-address relationships
	authorityToAddresses map[string][]ma.Multiaddr // Key is string(AuthorityID)
	addressToAuthorities map[string][]AuthorityID  // Key is ma.Multiaddr.String(), Value contains AuthorityID

	// Maps for storing peer-authority relationships
	peerToAuthorities map[peer.ID][]AuthorityID // Key is peer.ID, Value contains AuthorityID
	authorityToPeers  map[string][]peer.ID      // Key is string(AuthorityID), Value contains peer.ID

	mu sync.RWMutex
}

// NewWorker creates a new Worker instance
func NewWorker(fromService <-chan ServiceToWorkerMsg) *Worker {
	return &Worker{
		fromService:          fromService,
		authorityToAddresses: make(map[string][]ma.Multiaddr),
		addressToAuthorities: make(map[string][]AuthorityID),
		peerToAuthorities:    make(map[peer.ID][]AuthorityID),
		authorityToPeers:     make(map[string][]peer.ID),
	}
}

// Run starts the worker's main loop
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case msg, ok := <-w.fromService:
			if !ok { // Channel closed
				return
			}
			w.handleMessage(msg)
		case <-ctx.Done():
			return
		}
	}
}

// handleMessage processes incoming messages from the service
func (w *Worker) handleMessage(msg ServiceToWorkerMsg) {
	switch m := msg.(type) {
	case GetAddressesByAuthorityID:
		w.handleGetAddressesByAuthorityID(m)
	case GetAuthorityIDsByPeerID:
		w.handleGetAuthorityIDsByPeerID(m)
	}
}

// handleGetAddressesByAuthorityID handles requests for addresses by authority ID
func (w *Worker) handleGetAddressesByAuthorityID(msg GetAddressesByAuthorityID) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	addresses, exists := w.authorityToAddresses[string(msg.AuthorityID)]
	if !exists {
		msg.Response <- &AddressesResponse{Addresses: nil, Err: nil}
		return
	}

	// Return a copy of the addresses
	addressesCopy := make([]ma.Multiaddr, len(addresses))
	// ma.Multiaddr is an interface, but typically points to immutable data.
	// A shallow copy of the slice is fine.
	copy(addressesCopy, addresses)

	msg.Response <- &AddressesResponse{Addresses: addressesCopy, Err: nil}
}

// handleGetAuthorityIDsByPeerID handles requests for authority IDs by peer ID
func (w *Worker) handleGetAuthorityIDsByPeerID(msg GetAuthorityIDsByPeerID) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	authorities, exists := w.peerToAuthorities[msg.PeerID]
	if !exists {
		msg.Response <- &AuthorityIDsResponse{AuthorityIDs: nil, Err: nil}
		return
	}

	// Return a copy of the authorities
	authoritiesCopy := make([]AuthorityID, len(authorities))
	for i, auth := range authorities {
		// AuthorityID is []byte, so a new slice must be created for a true copy
		authCopy := make(AuthorityID, len(auth))
		copy(authCopy, auth)
		authoritiesCopy[i] = authCopy
	}

	msg.Response <- &AuthorityIDsResponse{AuthorityIDs: authoritiesCopy, Err: nil}
}

// AddAuthorityAddress adds an address for an authority (for testing or internal worker updates)
func (w *Worker) AddAuthorityAddress(authority AuthorityID, addr ma.Multiaddr) {
	w.mu.Lock()
	defer w.mu.Unlock()

	authKey := string(authority)
	addrKey := addr.String()

	// Add to authority -> addresses mapping
	currentAddresses := w.authorityToAddresses[authKey]
	w.authorityToAddresses[authKey] = appendUniqueMultiaddr(currentAddresses, addr)

	// Add to address -> authorities mapping
	currentAuthoritiesForAddr := w.addressToAuthorities[addrKey]
	w.addressToAuthorities[addrKey] = appendUniqueAuthorityID(currentAuthoritiesForAddr, authority)
}

// AddPeerAuthority adds an authority for a peer (for testing or internal worker updates)
func (w *Worker) AddPeerAuthority(peerID peer.ID, authority AuthorityID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	authKey := string(authority)

	// Add to peer -> authorities mapping
	currentAuthoritiesForPeer := w.peerToAuthorities[peerID]
	w.peerToAuthorities[peerID] = appendUniqueAuthorityID(currentAuthoritiesForPeer, authority)

	// Add to authority -> peers mapping
	currentPeersForAuth := w.authorityToPeers[authKey]
	w.authorityToPeers[authKey] = appendUniquePeerID(currentPeersForAuth, peerID)
}

// Helper functions to append unique values to slices

func appendUniqueMultiaddr(slice []ma.Multiaddr, addr ma.Multiaddr) []ma.Multiaddr {
	for _, existing := range slice {
		if existing.Equal(addr) {
			return slice
		}
	}
	return append(slice, addr)
}

func appendUniqueAuthorityID(slice []AuthorityID, auth AuthorityID) []AuthorityID {
	authStr := string(auth) // Compare as strings to simplify comparison of []byte content
	for _, existing := range slice {
		if string(existing) == authStr {
			return slice
		}
	}
	// Create a copy of auth before appending if auth might be a shared slice
	newAuth := make(AuthorityID, len(auth))
	copy(newAuth, auth)
	return append(slice, newAuth)
}

func appendUniquePeerID(slice []peer.ID, peerID peer.ID) []peer.ID {
	if slices.Contains(slice, peerID) {
		return slice
	}
	return append(slice, peerID)
}
