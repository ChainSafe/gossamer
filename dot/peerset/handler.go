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

	go ps.start()

	h := &Handler{
		actionQueue: actionCh,
		peerSet:     ps,
	}

	return h, nil
}

// AddReservedPeer adds reserved peerStatus into peerSet.
func (h *Handler) AddReservedPeer(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: AddReservedPeer,
		setID:      setID,
		peerID:     peerID,
	}
}

// RemoveReservedPeer remove reserved peerStatus from peerSet.
func (h *Handler) RemoveReservedPeer(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: RemoveReservedPeer,
		setID:      setID,
		peerID:     peerID,
	}
}

// SetReservedPeer sets the reserve peerStatus into peerSet
func (h *Handler) SetReservedPeer(setID int, peerIDs map[peer.ID]struct{}) {
	h.actionQueue <- Action{
		actionCall: SetReservedPeers,
		setID:      setID,
		peerIds:    peerIDs,
	}
}

// AddToPeerSet adds peerStatus to peerSet.
func (h *Handler) AddToPeerSet(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: AddToPeerSet,
		setID:      setID,
		peerID:     peerID,
	}
}

// RemoveFromPeerSet removes peerStatus from peerSet.
func (h *Handler) RemoveFromPeerSet(setID int, peerID peer.ID) {
	h.actionQueue <- Action{
		actionCall: RemoveFromPeerSet,
		setID:      setID,
		peerID:     peerID,
	}
}

// ReportPeer reports peerStatus for the reputation change.
func (h *Handler) ReportPeer(peerID peer.ID, rep ReputationChange) {
	h.actionQueue <- Action{
		actionCall: ReportPeer,
		reputation: rep,
		peerID:     peerID,
	}
}

// Incoming calls for an incoming connection from peerStatus.
func (h *Handler) Incoming(setID int, peerID peer.ID, incomingIndex uint64) (Message, error) {
	return h.peerSet.incoming(setID, peerID, incomingIndex)
}

// GetMessageQueue returns message of the peerSet
func (h *Handler) GetMessageQueue() []Message {
	return h.peerSet.getMessageQueue()
}

// Dropped calls for dropping a connection.
func (h *Handler) Dropped(setID int, peerID peer.ID, reason DropReason) error {
	return h.peerSet.dropped(setID, peerID, reason)
}

// GetReputation returns the reputation of the peer.ID.
func (h *Handler) GetReputation(peerID peer.ID) (int32, error) {
	node, err := h.peerSet.peerState.getNode(peerID)
	if err != nil {
		return 0, err
	}
	return node.getReputation(), nil
}
