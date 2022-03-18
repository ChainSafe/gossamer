// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
)

const logStringPattern = "call=%s, set-id=%d, reputation change %s, peers=[%s]"

// Handler manages peerSet.
type Handler struct {
	peerSet *PeerSet
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

// SetReservedOnlyPeer not yet implemented
func (h *Handler) SetReservedOnlyPeer(setID int, peers ...peer.ID) {
	// TODO: not yet implemented (#1888)
	logger.Errorf("failed to do action %s on peerSet: not implemented yet", setReservedOnly)
}

// AddReservedPeer adds reserved peer into peerSet.
func (h *Handler) AddReservedPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.addReservedPeers(setID, peers...)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, addReservedPeer, setID, "", stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// RemoveReservedPeer removes reserved peer from peerSet.
func (h *Handler) RemoveReservedPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.removeReservedPeers(setID, peers...)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, removeReservedPeer, setID, "", stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// SetReservedPeer sets the reserve peer into peerSet
func (h *Handler) SetReservedPeer(setID int, peers ...peer.ID) {
	// TODO: this is not used yet, it might be required to implement an RPC Call for this.
	err := h.peerSet.setReservedPeer(setID, peers...)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, setReservedPeers, setID, "", stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// AddPeer adds peer to peerSet.
func (h *Handler) AddPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.addPeer(setID, peers)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, addToPeerSet, setID, "", stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// RemovePeer removes peer from peerSet.
func (h *Handler) RemovePeer(setID int, peers ...peer.ID) {
	err := h.peerSet.removePeer(setID, peers...)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, removeFromPeerSet, setID, "", stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// ReportPeer reports ReputationChange according to the peer behaviour.
func (h *Handler) ReportPeer(rep ReputationChange, peers ...peer.ID) {
	err := h.peerSet.reportPeer(rep, peers...)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, reportPeer, 0, rep.String(), stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// Incoming calls when we have an incoming connection from peer.
func (h *Handler) Incoming(setID int, peers ...peer.ID) (status []Status, err error) {
	peersStatus, err := h.peerSet.incoming(setID, peers...)
	return peersStatus, err
}

// DisconnectPeer calls for disconnecting a connection from peer.
func (h *Handler) DisconnectPeer(setID int, peers ...peer.ID) {
	err := h.peerSet.disconnect(setID, UnknownDrop, peers...)
	if err != nil {
		msg := fmt.Sprintf(logStringPattern, disconnect, setID, "", stringfyPeers(peers))
		logger.Errorf("failed to do action %s on peerSet: %s", msg, err)
	}
}

// PeerReputation returns the reputation of the peer.
func (h *Handler) PeerReputation(peerID peer.ID) (Reputation, error) {
	n, err := h.peerSet.peerState.getNode(peerID)
	if err != nil {
		return 0, err
	}
	return n.reputation, nil
}

// SetMessageProcessor sets the peerset processor of the handler
// to process peerset messages.
func (h *Handler) SetMessageProcessor(processor MessageProcessor) {
	h.peerSet.processor = processor
}

// Start starts peerSet processing
func (h *Handler) Start(ctx context.Context) {
	go h.peerSet.periodicallyAllocateSlots(ctx)
}

// SortedPeers returns a sorted peer ID slice for connected peers in the peerSet.
func (h *Handler) SortedPeers(setIdx int) peer.IDSlice {
	return h.peerSet.peerState.sortedPeers(setIdx)
}

func stringfyPeers(peers peer.IDSlice) string {
	peersStrings := make([]string, len(peers))
	for i := range peers {
		peersStrings[i] = peers[i].String()
	}

	return strings.Join(peersStrings, ", ")
}
