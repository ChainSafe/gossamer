// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import "github.com/libp2p/go-libp2p-core/peer"

// Handler manages peerSet.
type Handler struct {
	actionQueue chan<- action
	peerSet     *PeerSet
	closeCh     chan struct{}
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
func (h *Handler) AddReservedPeer(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: addReservedPeer,
		setID:      setID,
		peers:      peers,
	}
}

// RemoveReservedPeer remove reserved peer from peerSet.
func (h *Handler) RemoveReservedPeer(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: removeReservedPeer,
		setID:      setID,
		peers:      peers,
	}
}

// SetReservedPeer set the reserve peer into peerSet
func (h *Handler) SetReservedPeer(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: setReservedPeers,
		setID:      setID,
		peers:      peers,
	}
}

// AddPeer adds peer to peerSet.
func (h *Handler) AddPeer(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: addToPeerSet,
		setID:      setID,
		peers:      peers,
	}
}

// RemovePeer removes peer from peerSet.
func (h *Handler) RemovePeer(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: removeFromPeerSet,
		setID:      setID,
		peers:      peers,
	}
}

// ReportPeer reports ReputationChange according to the peer behaviour.
func (h *Handler) ReportPeer(rep ReputationChange, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: reportPeer,
		reputation: rep,
		peers:      peers,
	}
}

// Incoming calls when we have an incoming connection from peer.
func (h *Handler) Incoming(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: incoming,
		peers:      peers,
		setID:      setID,
	}
}

// Messages return result message chan.
func (h *Handler) Messages() chan interface{} {
	return h.peerSet.resultMsgCh
}

// DisconnectPeer calls for disconnecting a connection from peer.
func (h *Handler) DisconnectPeer(setID int, peers ...peer.ID) {
	h.actionQueue <- action{
		actionCall: disconnect,
		setID:      setID,
		peers:      peers,
	}
}

// PeerReputation returns the reputation of the peer.
func (h *Handler) PeerReputation(peerID peer.ID) (Reputation, error) {
	n, err := h.peerSet.peerState.getNode(peerID)
	if err != nil {
		return 0, err
	}
	return n.getReputation(), nil
}

// Start starts peerSet processing
func (h *Handler) Start() {
	actionCh := make(chan action, msgChanSize)
	h.closeCh = make(chan struct{})
	h.actionQueue = actionCh
	h.peerSet.start(actionCh)
}

// SortedPeers return chan for sorted connected peer in the peerSet.
func (h *Handler) SortedPeers(setIdx int) chan peer.IDSlice {
	resultPeersCh := make(chan peer.IDSlice)
	h.actionQueue <- action{
		actionCall:    sortedPeers,
		resultPeersCh: resultPeersCh,
		setID:         setIdx,
	}

	return resultPeersCh
}

// Stop closes the actionQueue and result message chan.
func (h *Handler) Stop() {
	select {
	case <-h.closeCh:
	default:
		close(h.closeCh)
		close(h.actionQueue)
		h.peerSet.stop()
	}
}
