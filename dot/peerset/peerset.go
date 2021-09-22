package peerset

import (
	"math"
	"time"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	MIN int32 = math.MinInt32 // -2147483648
	MAX int32 = math.MaxInt32 // 2147483647
	// bannedThreshold We don't accept nodes whose reputation is under this value.
	bannedThreshold int32 = 82 * (MIN / 100)
	// disconnectReputationChange Reputation change for a node when we get disconnected from it.
	disconnectReputationChange int32 = -256
	// forgetAfter amount of time between the moment we disconnect from a node and the moment we remove it from the list.
	forgetAfter = time.Second * 3600 //Seconds
)

type ActionReceiver int

const (
	AddReservedPeer ActionReceiver = iota
	RemoveReservedPeer
	SetReservedPeers
	SetReservedOnly
	ReportPeer
	AddToPeersSet
	RemoveFromPeersSet
)

type Action struct {
	actionCall ActionReceiver
	setId      int
	reputation ReputationChange
	peerId     peer.ID
	peerIds    []map[peer.ID]bool
}

type MessageStatus int

const (
	Connect MessageStatus = iota
	Drop
	Accept
	Reject
)

type Message struct {
	messageStatus MessageStatus
	// TODO set_id, peer_id and IncomingIndex to be checked.
	setId         uint64
	peerId        peer.ID
	IncomingIndex uint64 // TODO: make IncomingIndex a type of uint64.
}

type ReputationChange struct {
	// Reputation delta
	Value  int32
	Reason string
}

func newReputationChange(value int32, reason string) ReputationChange {
	return ReputationChange{value, reason}
}

func newFatalReputationChange(reason string) ReputationChange {
	return ReputationChange{MIN, reason}
}

// PeerSet Side of the peer set manager owned by the network. In other words, the "receiving" side.
// Implements the `Stream` trait and can be polled for messages. The `Stream` never ends and never
// errors.
type PeerSet struct {
	// Underlying peerState structure for the nodes's states.
	peerState *PeersState
	// For each set, lists of nodes that don't occupy slots and that we should try to always be
	// connected to, and whether only reserved nodes are accepted. Is kept in sync with the list
	// of non-slot-occupying nodes in [`PeerSet::peerState`].
	reservedNode map[peer.ID]bool
	// This is for future purpose if reserved-only flag is enabled.
	isReservedOnly bool

	// Queue of messages to be emitted when the PeerSet is polled.
	messageQueue []Message

	// When the `PeerSet` was created.
	created time.Time
	// Last time when we updated the reputations of connected nodes.
	latestTimeUpdate time.Time
	// Next time to do a periodic call to allocSlots with all sets. This is done once per
	// second, to match the period of the reputation updates.
	// TODO: implement this.
	nextPeriodicAllocSlots time.Duration

	actionQueue chan Action
	//actionLock sync.RWMutex
}

// SetConfig is configuration of a single set.
type SetConfig struct {
	// Maximum allowed number of slot occupying nodes for ingoing connections.
	inPeers uint32
	// Maximum allowed number of slot occupying nodes for outgoing connections.
	outPeers uint32

	// List of bootstrap nodes to initialize the set with.
	// keep in mind that the networking has to know an address for these nodes,
	// otherwise it will not be able to connect to them.
	bootNodes []peer.ID

	// Lists of nodes we should always be connected to.
	// Keep in mind that the networking has to know an address for these nodes,
	// otherwise it will not be able to connect to them.
	reservedNodes []peer.ID

	// If true, we only accept nodes in reservedNodes.
	reservedOnly bool
}

type config struct {
	// List of sets of nodes the peerSet manages.
	sets []*SetConfig
}

// fromConfig initiates peerSet.
func fromConfig(cfg *config) (*PeerSet, error) {
	now := time.Now()

	peerSate, err := NewPeerState(cfg.sets)
	if err != nil {
		return nil, err
	}

	cfgSet := cfg.sets[0]
	reservedNodes := make(map[peer.ID]bool, len(cfgSet.reservedNodes))
	for _, peerID := range cfgSet.reservedNodes {
		reservedNodes[peerID] = cfgSet.reservedOnly
		peerSate.addNoSlotNode(0, peerID)
	}

	ps := &PeerSet{
		peerState:              peerSate,
		reservedNode:           reservedNodes,
		isReservedOnly:         false,
		messageQueue:           make([]Message, 0),
		created:                now,
		latestTimeUpdate:       now,
		nextPeriodicAllocSlots: time.Duration(0),
		actionQueue:            make(chan Action, 10),
	}

	for idx, node := range cfgSet.bootNodes {
		if UnknownPeer == ps.peerState.peer(idx, node) {
			peerSate.discover(idx, node)
		} else {
			log.Info("peerSet", "Duplicate bootNode in config: ", node)
		}
	}

	for i := 0; i < len(ps.peerState.sets); i++ {
		ps.allocSlots(i)
	}

	return ps, nil
}

// updateTime updates the value of latestTimeUpdate and performs all the updates that happen
// over time, such as reputation increases for staying connected.
func (ps *PeerSet) updateTime() {
	// Take now time
	now := time.Now()
	// identify the time difference between now time and last update time for peerScoring/reputation in seconds.
	// update the latestTimeUpdate to now.
	elapsedLatest := ps.latestTimeUpdate.Sub(ps.created)
	elapsedNow := now.Sub(ps.created)
	ps.latestTimeUpdate = now
	secDiff := int64(elapsedNow.Seconds() - elapsedLatest.Seconds())

	// this will give for how many seconds decaying is required for each peers in the list...
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

			node := ps.peerState.getNode(peerID)
			if node == nil {
				return // TODO: print log
			}

			before := node.getReputation()

			after := reputTick(before)
			node.setReputation(after)

			ps.peerState.nodes[peerID] = node

			if after != 0 {
				continue
			}

			// If the peer reaches a reputation of 0, and there is no connection to it, forget it.
			length := ps.peerState.getSetLength()
			for set := 0; set < length; set++ {
				if NotConnectedPeer == ps.peerState.peer(set, peerID) {
					lastDiscoveredTime := ps.peerState.lastConnectedAndDiscovered(set, peerID)
					if lastDiscoveredTime.Add(forgetAfter).Second() < now.Second() {
						// forget peer: Removes the peer from the list of members of the set.
						ps.peerState.forgetPeer(set, peerID)
					}
				}
			}
		}
	}

}

// onReportPeer on report peer the peer reputation need to be upgraded, If the updated reputation is below bannedThreshold then, this node need to be disconnected
// and a drop message for the peer sent to the network package in order to disconnect.
func (ps *PeerSet) onReportPeer(peerId peer.ID, change ReputationChange) {
	// We want reputations to be up-to-date before adjusting them.
	ps.updateTime()
	node := ps.peerState.getNode(peerId)
	reputation := node.addReputation(change.Value)
	ps.peerState.nodes[peerId] = node
	if reputation >= bannedThreshold {
		return
	}

	setLen := ps.peerState.getSetLength()
	for i := 0; i < setLen; i++ {
		if ConnectedPeer == ps.peerState.peer(i, peerId) {
			// disconnect peer
			err := ps.peerState.disconnect(i, peerId)
			if err != nil {
				return
			}
			message := Message{
				messageStatus: Drop,
				setId:         uint64(i),
				peerId:        peerId,
			}

			ps.messageQueue = append(ps.messageQueue, message)
			ps.allocSlots(i)
		}
	}

}

// allocSlots tries to fill available outgoing slots of nodes for the given set.
func (ps *PeerSet) allocSlots(setId int) {
	ps.updateTime()

	peerState := ps.peerState
	for reservePeer := range ps.reservedNode {

		status := peerState.peer(setId, reservePeer)
		if status == ConnectedPeer {
			continue
		} else if status == UnknownPeer {
			peerState.discover(setId, reservePeer)
		}

		node := ps.peerState.getNode(reservePeer)
		if node == nil {
			// TODO: add errors
			return
		}

		if node.getReputation() < bannedThreshold {
			break
		}

		err := peerState.tryOutgoing(setId, reservePeer)
		if err != nil {
			// TODO: add errors
			return
		}

		message := Message{
			messageStatus: Connect,
			setId:         uint64(setId),
			peerId:        reservePeer,
		}

		ps.messageQueue = append(ps.messageQueue, message)
	}

	// Nothing more to do if we're in reserved mode.
	if ps.isReservedOnly {
		return
	}

	for {
		if !peerState.hasFreeOutgoingSlot(setId) {
			break
		}

		peerId := peerState.highestNotConnectedPeer(setId)
		if peerId == "" {
			break
		}

		node := peerState.nodes[peerId]
		if node.getReputation() < bannedThreshold {
			break
		}

		err := peerState.tryOutgoing(setId, peerId)
		if err != nil {
			break
		}

		message := Message{
			messageStatus: Connect,
			setId:         uint64(setId),
			peerId:        peerId,
		}

		ps.messageQueue = append(ps.messageQueue, message)
	}

}

func (ps *PeerSet) onAddReservedPeer(setId int, peerId peer.ID) {
	if _, ok := ps.reservedNode[peerId]; ok {
		return
	}

	ps.reservedNode[peerId] = ps.isReservedOnly
	ps.peerState.addNoSlotNode(setId, peerId)
	ps.allocSlots(setId)
}

func (ps *PeerSet) onRemoveReservedPeer(setId int, peerId peer.ID) {
	if _, ok := ps.reservedNode[peerId]; !ok {
		return
	}

	delete(ps.reservedNode, peerId)
	ps.peerState.removeNoSlotNode(setId, peerId)

	// Nothing more to do if not in reservedOnly mode.
	if !ps.isReservedOnly {
		return
	}

	// reservedOnly mode is not yet implemented for future this code will help.
	// If, however, the ps is in reserved-only mode, then the removed node needs to be
	// disconnected.
	if ps.peerState.peer(setId, peerId) == ConnectedPeer {
		err := ps.peerState.disconnect(setId, peerId)
		if err != nil {
			// TODO: add errors
			return
		}

		message := Message{
			messageStatus: Drop,
			setId:         uint64(setId),
			peerId:        peerId,
		}

		ps.messageQueue = append(ps.messageQueue, message)
	}

}

// TODO: implement this.
func (ps *PeerSet) onSetReservedPeer(setId int, peerId peer.ID) {
	return
}

// reservedPeers returns the list of reserved peers.
func (ps *PeerSet) reservedPeers(setId int) []peer.ID {
	var reservedPeerList []peer.ID
	for node := range ps.reservedNode {
		reservedPeerList = append(reservedPeerList, node)
	}
	return reservedPeerList
}
func (ps *PeerSet) addToPeerSet(setId int, peerId peer.ID) {
	if UnknownPeer == ps.peerState.peer(setId, peerId) {
		ps.peerState.discover(setId, peerId)
		ps.allocSlots(setId)
	}
}

func (ps *PeerSet) onRemoveFromPeerSet(setId int, peerId peer.ID) {
	// Don't do anything if node is reserved.
	if _, ok := ps.reservedNode[peerId]; ok {
		return
	}
	peerConnectionStatus := ps.peerState.peer(setId, peerId)
	if ConnectedPeer == peerConnectionStatus {
		message := Message{
			messageStatus: Drop,
			setId:         uint64(setId),
			peerId:        peerId,
		}

		ps.messageQueue = append(ps.messageQueue, message)

		// disconnect and forget
		err := ps.peerState.disconnect(setId, peerId)
		if err != nil {
			// TODO: add error
			return
		}

		ps.peerState.forgetPeer(setId, peerId)
	} else if NotConnectedPeer == peerConnectionStatus {
		ps.peerState.forgetPeer(setId, peerId)
	}
}

func (ps *PeerSet) numDiscoveredPeers() int {
	return len(ps.peerState.peers())
}

// incoming indicate that we received an incoming connection. Must be answered either with
// a corresponding `Accept` or `Reject`, except if we were already connected to this peer.
// Note that this mechanism is orthogonal to Connect/Drop. Accepting an incoming
// connection implicitly means `Connect`, but incoming connections aren't cancelled by
// dropped
func (ps *PeerSet) incoming(setId int, peerId peer.ID, incomingIndex uint64) {
	ps.updateTime()
	// This is for reserved only mode.
	if ps.isReservedOnly {
		if _, ok := ps.reservedNode[peerId]; !ok {
			message := Message{
				messageStatus: Reject,
				IncomingIndex: incomingIndex,
			}
			ps.messageQueue = append(ps.messageQueue, message)
			return
		}
	}

	peerConnectionStatus := ps.peerState.peer(setId, peerId)
	if ConnectedPeer == peerConnectionStatus {
		return
	} else if NotConnectedPeer == peerConnectionStatus {
		ps.peerState.nodes[peerId].lastConnected[setId] = time.Now()
	} else if UnknownPeer == peerConnectionStatus {
		ps.peerState.discover(setId, peerId)
	}

	state := ps.peerState
	peer := state.nodes[peerId]
	if peer.getReputation() < bannedThreshold {
		message := Message{
			messageStatus: Reject,
			IncomingIndex: incomingIndex,
		}
		ps.messageQueue = append(ps.messageQueue, message)
		return
	}

	var message = Message{
		messageStatus: Accept,
		IncomingIndex: incomingIndex,
	}

	err := state.tryAcceptIncoming(setId, peerId)
	if err != nil {
		message.messageStatus = Reject
	}

	ps.messageQueue = append(ps.messageQueue, message)
}

type DropReason int

const (
	UnknownDrop DropReason = iota
	RefusedDrop
)

// dropped indicate that we dropped an active connection with a peer, or that we failed to connect.
// Must only be called after the PSM has either generated a Connect message with this
// PeerId, or accepted an incoming connection with this PeerId.
func (ps *PeerSet) dropped(setId int, peerId peer.ID, reason DropReason) {
	ps.updateTime()
	state := ps.peerState
	connectionStatus := state.peer(setId, peerId)
	if connectionStatus == ConnectedPeer {
		node := state.nodes[peerId]
		node.addReputation(disconnectReputationChange)
		state.nodes[peerId] = node

		err := state.disconnect(setId, peerId)
		if err != nil {
			// TODO: add error
			return
		}

	} else {
		return
	}

	if reason == RefusedDrop {
		ps.onRemoveFromPeerSet(setId, peerId)
	}
	ps.allocSlots(setId)
}

// Psm manager to handle Action request
func (ps *PeerSet) Psm() {

	for {
		select {
		case act := <-ps.actionQueue:
			{
				switch act.actionCall {
				case AddReservedPeer:
					ps.onAddReservedPeer(act.setId, act.peerId)
				case RemoveReservedPeer:
					ps.onRemoveReservedPeer(act.setId, act.peerId)
				case SetReservedPeers:
					ps.onSetReservedPeer(act.setId, act.peerId)
				case SetReservedOnly:
					// TODO TBD if this is useful
				case ReportPeer:
					ps.onReportPeer(act.peerId, act.reputation)
				case AddToPeersSet:
					ps.addToPeerSet(act.setId, act.peerId)
				case RemoveFromPeersSet:
					ps.onRemoveFromPeerSet(act.setId, act.peerId)
				}

			}
		}
	}
}
