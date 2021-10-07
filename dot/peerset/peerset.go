package peerset

import (
	"errors"
	"math"
	"time"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/peer"
)

var logger log.Logger = log.New("pkg", "peerset")

const (
	// BannedThreshold We don't accept nodes whose reputation is under this value.
	BannedThreshold int32 = 82 * (math.MinInt32 / 100)
	// disconnectReputationChange Reputation change for a node when we get disconnected from it.
	disconnectReputationChange int32 = -256
	// forgetAfterTime amount of time between the moment we disconnect from a node and the moment we remove it from the list.
	forgetAfterTime = time.Second * 3600 // seconds
)

// ActionReceiver represents the enum value for action to be performed on peerSet
type ActionReceiver int

const (
	// addReservedPeer is for adding reserved peers
	addReservedPeer ActionReceiver = iota
	// removeReservedPeer is for removing reserved peers
	removeReservedPeer
	// setReservedPeers is for setting peerList in peerSet reserved peers
	setReservedPeers
	// setReservedOnly is for setting peerList in peerSet reserved peers only
	setReservedOnly
	// reportPeer is for reporting peers if it misbehaves
	reportPeer
	// addToPeerSet is for adding peer in the peerSet
	addToPeerSet
	// removeFromPeerSet is for removing peer in the peerSet
	removeFromPeerSet
	// Incoming is for inbound request
	incoming
	// sortedPeers is for sorted connected peers
	sortedPeers
	// disconnect peer
	disconnect
)

// Action struct stores the action type and required parameters to perform action
type Action struct {
	actionCall    ActionReceiver
	setID         int
	reputation    ReputationChange
	peerID        peer.ID
	peerIds       map[peer.ID]struct{}
	IncomingIndex uint64
	resultPeersCh chan interface{}
}

// MessageStatus represents the enum value for Message
type MessageStatus int

const (
	// Connect is request to open a connection to the given peer. From the point of view of the PSM, we are
	// immediately connected.
	Connect MessageStatus = iota
	// Drop the connection to the given peer, or cancel the connection attempt after a Connect
	Drop
	// Accept incoming connect request
	Accept
	// Reject incoming connect request
	Reject
)

// Message that can be sent by the peer set manager (PSM).
type Message struct {
	messageStatus MessageStatus
	setID         uint64
	peerID        peer.ID
}

// GetStatus returns the messageStatus.
func (m *Message) GetStatus() MessageStatus {
	return m.messageStatus
}

// GetPeerID returns the messageStatus.
func (m *Message) GetPeerID() peer.ID {
	return m.peerID
}

// ReputationChange is description of a reputation adjustment for a node
type ReputationChange struct {
	// Reputation delta
	Value int32
	// Reason for reputation change
	Reason string
}

func newReputationChange(value int32, reason string) ReputationChange {
	return ReputationChange{value, reason}
}

// PeerSet Side of the peerStatus set manager owned by the network. In other words, the "receiving" side.
type PeerSet struct {
	// Underlying peerState structure for the nodes states.
	peerState *PeersState

	reservedNode map[peer.ID]struct{}
	// TODO: this will be useful for reserved only mode
	// This is for future purpose if reserved-only flag is enabled.
	isReservedOnly bool
	resultMsgCh    chan interface{}
	// When the PeerSet was created.
	created time.Time
	// Last time when we updated the reputations of connected nodes.
	latestTimeUpdate time.Time
	// Next time to do a periodic call to allocSlots with all Set. This is done once per
	// second, to match the period of the reputation updates.
	nextPeriodicAllocSlots time.Duration
	actionQueue            <-chan Action
}

// config is configuration of a single set.
type config struct {
	// Maximum allowed number of slot occupying nodes for ingoing connections.
	inPeers uint32
	// Maximum allowed number of slot occupying nodes for outgoing connections.
	outPeers uint32

	// List of bootstrap nodes to initialise the set with.
	// keep ingoing mind that the networking has to know an address for these nodes,
	// otherwise it will not be able to connect to them.
	bootNodes []peer.ID

	// Lists of nodes we should always be connected to.
	// Keep ingoing mind that the networking has to know an address for these nodes,
	// otherwise it will not be able to connect to them.
	reservedNodes []peer.ID

	// TODO Use in future for reserved only peers
	// If true, we only accept nodes ingoing reservedNodes.
	reservedOnly bool
}

// ConfigSet set of peerSet config.
type ConfigSet struct {
	Set []*config
}

// NewConfigSet creates a new config set for the peerSet
func NewConfigSet(in, out uint32, bootNodes, reservedNodes []peer.ID, reservedOnly bool) *ConfigSet {
	set := &config{
		inPeers:       in,
		outPeers:      out,
		bootNodes:     bootNodes,
		reservedNodes: reservedNodes,
		reservedOnly:  reservedOnly,
	}

	return &ConfigSet{[]*config{set}}
}

func newPeerSet(cfg *ConfigSet, actionCh <-chan Action) (*PeerSet, error) {
	now := time.Now()

	peerState, err := NewPeerState(cfg.Set)
	if err != nil {
		return nil, err
	}

	if len(cfg.Set) == 0 {
		return nil, errors.New("config set is empty")
	}

	cfgSet := cfg.Set[0]
	reservedNodes := make(map[peer.ID]struct{}, len(cfgSet.reservedNodes))
	for _, peerID := range cfgSet.reservedNodes {
		reservedNodes[peerID] = struct{}{}
		peerState.addNoSlotNode(0, peerID)
	}

	ps := &PeerSet{
		peerState:              peerState,
		reservedNode:           reservedNodes,
		isReservedOnly:         cfgSet.reservedOnly,
		resultMsgCh:            make(chan interface{}, 100), // TODO: fix a size
		created:                now,
		latestTimeUpdate:       now,
		nextPeriodicAllocSlots: time.Second * 2,
		actionQueue:            actionCh,
	}

	for _, node := range cfgSet.bootNodes {
		if ps.peerState.peerStatus(0, node) == unknownPeer {
			peerState.discover(0, node)
		}
	}

	for i := 0; i < len(ps.peerState.sets); i++ {
		if err = ps.allocSlots(i); err != nil {
			return nil, err
		}
	}

	return ps, nil
}

// updateTime updates the value of latestTimeUpdate and performs all the updates that happen
// over time, such as reputation increases for staying connected.
func (ps *PeerSet) updateTime() error {
	// Take now time
	now := time.Now()
	// identify the time difference between now time and last update time for peerScoring/reputation ingoing seconds.
	// update the latestTimeUpdate to now.
	elapsedLatest := ps.latestTimeUpdate.Sub(ps.created)
	elapsedNow := now.Sub(ps.created)
	ps.latestTimeUpdate = now
	secDiff := int64(elapsedNow.Seconds() - elapsedLatest.Seconds())

	// this will give for how many seconds decaying is required for each peer
	// For each elapsed second, move the node reputation towards zero.
	// If we multiply each second the reputation by `k` (where `k` is between 0 and 1), it
	// takes `ln(0.5) / ln(k)` seconds to reduce the reputation by half. Use this formula to
	// empirically determine a value of `k` that looks correct.
	for i := int64(0); i < secDiff; i++ {
		// We use `k = 0.98`, so we divide by `50`. With that value, it takes 34.3 seconds
		// to reduce the reputation by half.
		for _, peerID := range ps.peerState.peers() {
			reputTick := func(reput int32) int32 {
				diff := reput / 50
				if diff == 0 && reput < 0 {
					diff = -1
				} else if diff == 0 && reput > 0 {
					diff = 1
				}

				reput = subInt32(reput, diff)
				return reput
			}

			node, err := ps.peerState.getNode(peerID)
			if err != nil {
				return err
			}

			before := node.getReputation()
			after := reputTick(before)
			node.setReputation(after)
			ps.peerState.nodes[peerID] = node

			if after != 0 {
				continue
			}

			// If the peerStatus reaches a reputation of 0, and there is no connection to it, forget it.
			length := ps.peerState.getSetLength()
			for set := 0; set < length; set++ {
				if ps.peerState.peerStatus(set, peerID) != notConnectedPeer {
					continue
				}

				lastDiscoveredTime := ps.peerState.lastConnectedAndDiscovered(set, peerID)
				if lastDiscoveredTime.Add(forgetAfterTime).Second() >= now.Second() {
					continue
				}

				// forget peerStatus: Removes the peerStatus from the list of members of the set.
				err = ps.peerState.forgetPeer(set, peerID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// reportPeer on report peerStatus the peerStatus reputation need to be upgraded,
// If the updated reputation is below BannedThreshold then, this node need to be disconnected
// and a drop message for the peerStatus sent to the network package ingoing order to disconnect.
func (ps *PeerSet) reportPeer(peerID peer.ID, change ReputationChange) error {
	// We want reputations to be up-to-date before adjusting them.
	if err := ps.updateTime(); err != nil {
		return err
	}
	node, err := ps.peerState.getNode(peerID)
	if err != nil {
		return err
	}

	reputation := node.addReputation(change.Value)
	ps.peerState.nodes[peerID] = node
	if reputation >= BannedThreshold {
		return nil
	}

	setLen := ps.peerState.getSetLength()
	for i := 0; i < setLen; i++ {
		if ps.peerState.peerStatus(i, peerID) == connectedPeer {
			// disconnect peer
			err = ps.peerState.disconnect(i, peerID)
			if err != nil {
				return err
			}

			ps.resultMsgCh <- Message{
				messageStatus: Drop,
				setID:         uint64(i),
				peerID:        peerID,
			}

			if err = ps.allocSlots(i); err != nil {
				return err
			}
		}
	}

	return nil
}

// allocSlots tries to fill available outgoing slots of nodes for the given set.
func (ps *PeerSet) allocSlots(setIDX int) error {
	err := ps.updateTime()
	if err != nil {
		return err
	}

	peerState := ps.peerState
	for reservePeer := range ps.reservedNode {
		status := peerState.peerStatus(setIDX, reservePeer)
		if status == connectedPeer {
			continue
		} else if status == unknownPeer {
			peerState.discover(setIDX, reservePeer)
		}

		node, err := ps.peerState.getNode(reservePeer)
		if err != nil {
			return err
		}

		if node.getReputation() < BannedThreshold {
			break
		}

		err = peerState.tryOutgoing(setIDX, reservePeer)
		if err != nil {
			return err
		}

		ps.resultMsgCh <- Message{
			messageStatus: Connect,
			setID:         uint64(setIDX),
			peerID:        reservePeer,
		}
	}
	// Nothing more to do if we're ingoing reserved mode.
	if ps.isReservedOnly {
		return nil
	}

	for peerState.hasFreeOutgoingSlot(setIDX) {
		peerID := peerState.highestNotConnectedPeer(setIDX)
		if peerID == "" {
			break
		}

		node := peerState.nodes[peerID]
		if node.getReputation() < BannedThreshold {
			break
		}

		err := peerState.tryOutgoing(setIDX, peerID)
		if err != nil {
			break
		}

		ps.resultMsgCh <- Message{
			messageStatus: Connect,
			setID:         uint64(setIDX),
			peerID:        peerID,
		}
	}
	return nil
}

func (ps *PeerSet) addReservedPeer(setID int, peerID peer.ID) error {
	if _, ok := ps.reservedNode[peerID]; ok {
		logger.Info("peerStatus already exists ingoing peerStatus set", "peerID", peerID)
		return nil
	}

	ps.reservedNode[peerID] = struct{}{}
	ps.peerState.addNoSlotNode(setID, peerID)

	err := ps.allocSlots(setID)
	return err
}

func (ps *PeerSet) removeReservedPeer(setID int, peerID peer.ID) error {
	if _, ok := ps.reservedNode[peerID]; !ok {
		logger.Info("peerStatus doesn't exists ingoing the peerStatus set", "peerID:", peerID)
		return nil
	}

	delete(ps.reservedNode, peerID)
	ps.peerState.removeNoSlotNode(setID, peerID)

	// Nothing more to do if not ingoing reservedOnly mode.
	if !ps.isReservedOnly {
		return nil
	}

	// reservedOnly mode is not yet implemented for future this code will help.
	// If, however, the ps is ingoing reserved-only mode, then the removed node needs to be
	// disconnected.
	if ps.peerState.peerStatus(setID, peerID) == connectedPeer {
		err := ps.peerState.disconnect(setID, peerID)
		if err != nil {
			return err
		}

		ps.resultMsgCh <- Message{
			messageStatus: Drop,
			setID:         uint64(setID),
			peerID:        peerID,
		}
	}
	return nil
}

// TODO: not yet used.
func (ps *PeerSet) setReservedPeer(setID int, peerIds map[peer.ID]struct{}) error {
	toInsert, toRemove := make([]peer.ID, 0, len(peerIds)), make([]peer.ID, 0, len(peerIds))
	for pid := range peerIds {
		if _, ok := ps.reservedNode[pid]; ok {
			continue
		}
		toInsert = append(toInsert, pid)
	}

	for pid := range ps.reservedNode {
		if _, ok := peerIds[pid]; ok {
			continue
		}
		toRemove = append(toRemove, pid)
	}

	for _, p := range toInsert {
		if err := ps.addReservedPeer(setID, p); err != nil {
			return err
		}
	}

	for _, p := range toRemove {
		if err := ps.removeReservedPeer(setID, p); err != nil {
			return err
		}
	}

	return nil
}

// TODO: Not used yet, require to implement RPC call to get all the reserved peers.
// reservedPeers returns the list of reserved peers.
func (ps *PeerSet) reservedPeers() []peer.ID { // nolint
	reservedPeerList := make([]peer.ID, 0, len(ps.reservedNode))
	for node := range ps.reservedNode {
		reservedPeerList = append(reservedPeerList, node)
	}
	return reservedPeerList
}

func (ps *PeerSet) addToPeerSet(setID int, peerID peer.ID) error {
	if ps.peerState.peerStatus(setID, peerID) != unknownPeer {
		return nil
	}

	ps.peerState.discover(setID, peerID)
	return ps.allocSlots(setID)
}

func (ps *PeerSet) removeFromPeerSet(setID int, peerID peer.ID) error {
	// Don't do anything if node is reserved.
	if _, ok := ps.reservedNode[peerID]; ok {
		logger.Info("peerStatus is reserved", "peerID: ", peerID)
		return nil
	}

	peerConnectionStatus := ps.peerState.peerStatus(setID, peerID)
	if peerConnectionStatus == connectedPeer {
		ps.resultMsgCh <- Message{
			messageStatus: Drop,
			setID:         uint64(setID),
			peerID:        peerID,
		}

		// disconnect and forget
		err := ps.peerState.disconnect(setID, peerID)
		if err != nil {
			return err
		}

		if err = ps.peerState.forgetPeer(setID, peerID); err != nil {
			return err
		}
	} else if peerConnectionStatus == notConnectedPeer {
		if err := ps.peerState.forgetPeer(setID, peerID); err != nil {
			return err
		}
	}
	return nil
}

// incoming indicate that we received an incoming connection. Must be answered either with
// a corresponding `Accept` or `Reject`, except if we were already connected to this peerStatus.
// Note that this mechanism is orthogonal to Connect/Drop. Accepting an incoming
// connection implicitly means `Connect`, but incoming connections aren't cancelled by
// disconnect
func (ps *PeerSet) incoming(setID int, peerID peer.ID) error {
	if err := ps.updateTime(); err != nil {
		return err
	}

	// This is for reserved only mode.
	if ps.isReservedOnly {
		if _, ok := ps.reservedNode[peerID]; !ok {
			ps.resultMsgCh <- Message{messageStatus: Reject}
			return nil
		}
	}

	switch ps.peerState.peerStatus(setID, peerID) {
	case connectedPeer:
		return nil
	case notConnectedPeer:
		ps.peerState.nodes[peerID].lastConnected[setID] = time.Now()
	case unknownPeer:
		ps.peerState.discover(setID, peerID)
	}

	state := ps.peerState
	p := state.nodes[peerID]
	if p.getReputation() < BannedThreshold {
		ps.resultMsgCh <- Message{messageStatus: Reject}
	} else if err := state.tryAcceptIncoming(setID, peerID); err != nil {
		ps.resultMsgCh <- Message{messageStatus: Reject}
	} else {
		ps.resultMsgCh <- Message{messageStatus: Accept}
	}

	return nil
}

// DropReason represents reason for disconnection of the peer
type DropReason int

const (
	// UnknownDrop is used when substream or connection has been closed for an unknown reason
	UnknownDrop DropReason = iota
	// RefusedDrop is used when substream or connection has been explicitly refused by the target.
	// In other words, the peer doesn't actually belong to this set.
	RefusedDrop
)

// disconnect indicate that we disconnect an active connection with a peerStatus, or that we failed to connect.
// Must only be called after the peer set has either generated a Connect message with this
// peerId, or accepted an incoming connection with this PeerId.
func (ps *PeerSet) disconnect(setIDX int, peerID peer.ID, reason DropReason) error {
	err := ps.updateTime()
	if err != nil {
		return err
	}

	state := ps.peerState
	connectionStatus := state.peerStatus(setIDX, peerID)
	if connectionStatus != connectedPeer {
		return errors.New("received disconnect for non-connected node")
	}

	n := state.nodes[peerID]
	n.addReputation(disconnectReputationChange)
	state.nodes[peerID] = n

	if err = state.disconnect(setIDX, peerID); err != nil {
		return err
	}
	ps.resultMsgCh <- Message{
		messageStatus: Drop,
		peerID:        peerID,
	}

	// TODO: figure out the condition of connection refuse.
	if reason == RefusedDrop {
		if err = ps.removeFromPeerSet(setIDX, peerID); err != nil {
			return err
		}
	}

	err = ps.allocSlots(setIDX)
	return err
}

// start manager to handle Action request
func (ps *PeerSet) start() {
	ticker := time.NewTicker(ps.nextPeriodicAllocSlots)
	defer ticker.Stop()
	var err error
	for {
		select {
		case <-ticker.C:
			{
				l := ps.peerState.getSetLength()
				for i := 0; i < l; i++ {
					if err = ps.allocSlots(i); err != nil {
						logger.Error("failed to do action on peerSet ", "error", err)
					}
				}
			}
		case act := <-ps.actionQueue:
			switch act.actionCall {
			case addReservedPeer:
				err = ps.addReservedPeer(act.setID, act.peerID)
			case removeReservedPeer:
				err = ps.removeReservedPeer(act.setID, act.peerID)
			case setReservedPeers:
				// TODO: TBD This is not used yet, I might required to implement RPC Call for this.
				err = ps.setReservedPeer(act.setID, act.peerIds)
			case setReservedOnly:
				// TODO TBD if this is useful
			case reportPeer:
				err = ps.reportPeer(act.peerID, act.reputation)
			case addToPeerSet:
				err = ps.addToPeerSet(act.setID, act.peerID)
			case removeFromPeerSet:
				err = ps.removeFromPeerSet(act.setID, act.peerID)
			case incoming:
				err = ps.incoming(act.setID, act.peerID)
			case sortedPeers:
				ps.peerState.sortedConnectedPeers(0, act.resultPeersCh)
			case disconnect:
				err = ps.disconnect(0, act.peerID, UnknownDrop)
			}

			if err != nil {
				logger.Error("failed to do action on peerset", "action", act, "error", err)
			}
		}
	}
}
