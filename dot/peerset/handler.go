package peerset

import "github.com/libp2p/go-libp2p-core/peer"

// Handler manages peerSet.
type Handler struct {
	actionQueue chan<- Action
	peerSet     *PeerSet
}

// NewPeerSetHandler initiates peerSet.
func NewPeerSetHandler(cfg *ConfigSet) (*Handler, error) {
	actionCh := make(chan Action, 256)
	ps, err := newPeerSet(cfg, actionCh)
	if err != nil {
		return nil, err
	}

	h := &Handler{
		actionQueue: actionCh,
		peerSet:     ps,
	}

	return h, nil
}

// AddReservedPeer adds reserved peerStatus into peerSet.
func (h *Handler) AddReservedPeer(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: addReservedPeer,
		setID:      setID,
		peerID:     peerID,
	}
}

// RemoveReservedPeer remove reserved peerStatus from peerSet.
func (h *Handler) RemoveReservedPeer(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: removeReservedPeer,
		setID:      setID,
		peerID:     peerID,
	}
}

// SetReservedPeer set the reserve peerStatus into peerSet
func (h *Handler) SetReservedPeer(setID int, peerIDs map[peer.ID]struct{}) {
	h.actionQueue <- Action{
		actionCall: setReservedPeers,
		setID:      setID,
		peerIds:    peerIDs,
	}
}

// AddToPeerSet adds peerStatus to peerSet.
func (h *Handler) AddToPeerSet(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: addToPeerSet,
		setID:      setID,
		peerID:     peerID,
	}
}

// RemoveFromPeerSet removes peerStatus from peerSet.
func (h *Handler) RemoveFromPeerSet(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: removeFromPeerSet,
		setID:      setID,
		peerID:     peerID,
	}
}

// ReportPeer reports peerStatus for the reputation change.
func (h *Handler) ReportPeer(peerID peer.ID, rep ReputationChange) {
	h.actionQueue <- Action{
		actionCall: reportPeer,
		reputation: rep,
		peerID:     peerID,
	}
}

// Incoming calls for an incoming connection from peerStatus.
func (h *Handler) Incoming(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: incoming,
		peerID:     peerID,
		setID:      setID,
	}
}

// GetMessageChan return message chan.
func (h *Handler) GetMessageChan() chan interface{} {
	return h.peerSet.resultMsgCh
}

// DisconnectPeer calls to disconnect a connection from peer.
func (h *Handler) DisconnectPeer(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: disconnect,
		setID:      setID,
		peerID:     peerID,
	}
}

// GetReputation returns the reputation of the peer.ID.
func (h *Handler) GetReputation(peerID peer.ID) (int32, error) {
	node, err := h.peerSet.peerState.getNode(peerID)
	if err != nil {
		return 0, err
	}
	return node.getReputation(), nil
}

// Start starts peerSet processing
func (h *Handler) Start() {
	h.peerSet.start()
}

// GetSortedPeers return chan for sorted connected peer in the peer set.
func (h *Handler) GetSortedPeers() chan interface{} {
	resultPeersCh := make(chan interface{}, 1)
	h.actionQueue <- Action{
		actionCall:    sortedPeers,
		resultPeersCh: resultPeersCh,
	}

	return resultPeersCh
}
