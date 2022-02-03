// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
)

// Handler manages peerSet.
type Handler struct {
	actionQueue chan<- action
	peerSet     *PeerSet
	closeCh     chan struct{}

	cancelCtx context.CancelFunc
}

// NewPeerSetHandler creates a new *peerset.Handler.
func NewPeerSetHandler(cfg *ConfigSet) (*Handler, error) {
	ps, err := newPeerSet(cfg)
	if err != nil {
		return nil, err
	}

	return &Handler{
		peerSet: ps,
	}, nil
}

// AddReservedPeer adds reserved peer into peerSet.
func (h *Handler) SetReservedOnlyPeer(setID int, peers ...peer.ID) {
	// TODO: not yet implemented (#1888)
	logger.Errorf("failed to do action %s on peerSet: not implemented yet", setReservedOnly)
}

// AddReservedPeer adds reserved peer into peerSet.
func (h *Handler) AddReservedPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.addReservedPeers(setID, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", addReservedPeer, err)
	}
}

// RemoveReservedPeer remove reserved peer from peerSet.
func (h *Handler) RemoveReservedPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.removeReservedPeers(setID, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", removeReservedPeer, err)
	}
}

// SetReservedPeer set the reserve peer into peerSet
func (h *Handler) SetReservedPeer(setID int, peers ...peer.ID) {
	// TODO: this is not used yet, might required to implement RPC Call for this.
	err := h.peerSet.setReservedPeer(setID, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", setReservedPeers, err)
	}
}

// AddPeer adds peer to peerSet.
func (h *Handler) AddPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.addPeer(setID, peers)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", addToPeerSet, err)
	}
}

// RemovePeer removes peer from peerSet.
func (h *Handler) RemovePeer(setID int, peers ...peer.ID) {
	err := h.peerSet.removePeer(setID, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", removeFromPeerSet, err)
	}
}

// ReportPeer reports ReputationChange according to the peer behaviour.
func (h *Handler) ReportPeer(rep ReputationChange, peers ...peer.ID) {
	err := h.peerSet.reportPeer(rep, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", reportPeer, err)
	}
}

// Incoming calls when we have an incoming connection from peer.
func (h *Handler) Incoming(setID int, peers ...peer.ID) {
	err := h.peerSet.incoming(setID, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", incoming, err)
	}
}

// DisconnectPeer calls for disconnecting a connection from peer.
func (h *Handler) DisconnectPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.disconnect(setID, UnknownDrop, peers...)
	if err != nil {
		logger.Errorf("failed to do action %s on peerSet: %s", disconnect, err)
	}
}

// PeerReputation returns the reputation of the peer.
func (h *Handler) PeerReputation(peerID peer.ID) (Reputation, error) {
	n, err := h.peerSet.peerState.getNode(peerID)
	if err != nil {
		return 0, err
	}
	return n.rep, nil
}

// Start starts peerSet processing
func (h *Handler) Start(ctx context.Context, processMessageFn func(Message)) {
	ctx, cancel := context.WithCancel(ctx)
	h.cancelCtx = cancel
	h.peerSet.start(ctx, processMessageFn)
}

// SortedPeers return chan for sorted connected peer in the peerSet.
func (h *Handler) SortedPeers(setIdx int) peer.IDSlice {
	return h.peerSet.peerState.sortedPeers(setIdx)
}

// Stop closes the actionQueue and result message chan.
func (h *Handler) Stop() {
	h.cancelCtx()
}
