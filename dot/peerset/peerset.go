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
	// disconnectReputationChange Reputation change value for a node when we get disconnected from it.
	disconnectReputationChange Reputation = -256
	// forgetAfterTime amount of time between the moment we disconnect from a node and the moment we remove it from the list.
	forgetAfterTime = time.Second * 3600 // seconds
	// default channel size for peerSet.
	msgChanSize = 100
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
	// incoming is for inbound request
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
	peers         peer.IDSlice
	IncomingIndex uint64
	resultPeersCh chan interface{}
}

// Status represents the enum value for Message
type Status int

const (
	// Connect is request to open a connection to the given peer.
	Connect Status = iota
	// Drop the connection to the given peer, or cancel the connection attempt after a Connect.
	Drop
	// Accept incoming connect request.
	Accept
	// Reject incoming connect request.
	Reject
)

// Message that will be sent by the peerSet.
type Message struct {
	// Status ...
	Status Status
	setID  uint64
	// PeerID ...
	PeerID peer.ID
}

// Reputation represents reputation value of the node
type Reputation int32

// add handles overflow and underflow condition while adding two Reputation values.
func (r Reputation) add(num Reputation) Reputation {
	if num > 0 {
		if r > math.MaxInt32-num {
			return math.MaxInt32
		}
	} else if r < math.MinInt32-num {
		return math.MinInt32
	}
	return r + num
}

// sub handles underflow condition while subtracting two Reputation values.
func (r Reputation) sub(num Reputation) Reputation {
	if num < 0 {
		if r > math.MaxInt32+num {
			return math.MaxInt32
		}
	} else if r < math.MinInt32+num {
		return math.MinInt32
	}
	return r - num
}

// ReputationChange is description of a reputation adjustment for a node
type ReputationChange struct {
	// PeerReputation value
	Value Reputation
	// Reason for reputation change
	Reason string
}

func newReputationChange(value Reputation, reason string) ReputationChange {
	return ReputationChange{value, reason}
}

// PeerSet is a container for all the components of a peerSet.
type PeerSet struct {
	peerState *PeersState

	reservedNode map[peer.ID]struct{}
	// TODO: this will be useful for reserved only mode
	// this is for future purpose if reserved-only flag is enabled.
	isReservedOnly bool
	resultMsgCh    chan interface{}
	// time when the PeerSet was created.
	created time.Time
	// last time when we updated the reputations of connected nodes.
	latestTimeUpdate time.Time
	// next time to do a periodic call to allocSlots with all Set. This is done once two
	// second, to match the period of the Reputation updates.
	nextPeriodicAllocSlots time.Duration
	// chan for receiving action request.
	actionQueue <-chan Action
}

// config is configuration of a single set.
type config struct {
	// maximum number of slot occupying nodes for incoming connections.
	inPeers uint32
	// maximum number of slot occupying nodes for outgoing connections.
	outPeers uint32

	// list of bootstrap nodes to initialise the set.
	// keep in mind that the networking has to know an address for these nodes,
	// otherwise it will not be able to connect to them.
	bootNodes []peer.ID

	// lists of nodes we should always be connected to.
	// keep in mind that the networking has to know an address for these nodes,
	// otherwise it will not be able to connect to them.
	reservedNodes []peer.ID

	// TODO Use in future for reserved only peers
	// if true, we only accept reservedNodes.
	reservedOnly bool

	// time duration for a peerSet to periodically call allocSlots.
	periodicAllocTime time.Duration
}

// ConfigSet set of peerSet config.
type ConfigSet struct {
	Set []*config
}

// NewConfigSet creates a new config set for the peerSet
func NewConfigSet(in, out uint32, bootNodes, reservedNodes []peer.ID, reservedOnly bool, allocTime time.Duration) *ConfigSet {
	set := &config{
		inPeers:           in,
		outPeers:          out,
		bootNodes:         bootNodes,
		reservedNodes:     reservedNodes,
		reservedOnly:      reservedOnly,
		periodicAllocTime: allocTime,
	}

	return &ConfigSet{[]*config{set}}
}

func newPeerSet(cfg *ConfigSet, actionCh <-chan Action) (*PeerSet, error) {
	now := time.Now()

	if len(cfg.Set) == 0 {
		return nil, errors.New("config set is empty")
	}

	peerState, err := NewPeerState(cfg.Set)
	if err != nil {
		return nil, err
	}

	// TODO: currently we only have one set, change this once we have more.
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
		resultMsgCh:            make(chan interface{}, msgChanSize),
		created:                now,
		latestTimeUpdate:       now,
		nextPeriodicAllocSlots: cfgSet.periodicAllocTime,
		actionQueue:            actionCh,
	}

	for _, n := range cfgSet.bootNodes {
		if ps.peerState.peerStatus(0, n) == unknownPeer {
			peerState.discover(0, n)
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
// over time, such as Reputation increases for staying connected.
func (ps *PeerSet) updateTime() error {
	currTime := time.Now()
	// identify the time difference between current time and last update time for peerScoring/reputation in seconds.
	// update the latestTimeUpdate to currTime.
	elapsedLatest := ps.latestTimeUpdate.Sub(ps.created)
	elapsedNow := currTime.Sub(ps.created)
	ps.latestTimeUpdate = currTime
	secDiff := int64(elapsedNow.Seconds() - elapsedLatest.Seconds())

	// this will give for how many seconds decaying is required for each peer
	// for each elapsed second, move the node reputation towards zero.
	// If we multiply each second the reputation by `k` (where `k` is between 0 and 1), it
	// takes `ln(0.5) / ln(k)` seconds to reduce the reputation by half. Use this formula to
	// empirically determine a value of `k` that looks correct.
	for i := int64(0); i < secDiff; i++ {
		// we use `k = 0.98`, so we divide by `50`. With that value, it takes 34.3 seconds
		// to reduce the reputation by half.
		for _, peerID := range ps.peerState.peers() {
			reputTick := func(reput Reputation) Reputation {
				diff := reput / 50
				if diff == 0 && reput < 0 {
					diff = -1
				} else if diff == 0 && reput > 0 {
					diff = 1
				}

				reput = reput.sub(diff)
				return reput
			}

			n, err := ps.peerState.getNode(peerID)
			if err != nil {
				return err
			}

			before := n.getReputation()
			after := reputTick(before)
			n.setReputation(after)
			ps.peerState.nodes[peerID] = n

			if after != 0 {
				continue
			}

			// if the peer reaches reputation 0, and there is no connection to it, forget it.
			length := ps.peerState.getSetLength()
			for set := 0; set < length; set++ {
				if ps.peerState.peerStatus(set, peerID) != notConnectedPeer {
					continue
				}

				lastDiscoveredTime := ps.peerState.lastConnectedAndDiscovered(set, peerID)
				if lastDiscoveredTime.Add(forgetAfterTime).Second() >= currTime.Second() {
					continue
				}

				// forget peer removes the peer from the list of members of the set.
				err = ps.peerState.forgetPeer(set, peerID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// reportPeer on report ReputationChange of the peer based on its behaviour,
// if the updated Reputation is below BannedThresholdValue then, this node need to be disconnected
// and a drop message for the peer is sent in order to disconnect.
func (ps *PeerSet) reportPeer(change ReputationChange, peers ...peer.ID) error {
	// we want reputations to be up-to-date before adjusting them.
	if err := ps.updateTime(); err != nil {
		return err
	}

	for _, pid := range peers {
		n, err := ps.peerState.getNode(pid)
		if err != nil {
			return err
		}

		rep := n.addReputation(change.Value)
		ps.peerState.nodes[pid] = n
		if rep >= BannedThresholdValue {
			return nil
		}

		setLen := ps.peerState.getSetLength()
		for i := 0; i < setLen; i++ {
			if ps.peerState.peerStatus(i, pid) == connectedPeer {
				// disconnect peer
				err = ps.peerState.disconnect(i, pid)
				if err != nil {
					return err
				}

				ps.resultMsgCh <- Message{
					Status: Drop,
					setID:  uint64(i),
					PeerID: pid,
				}
				if err = ps.allocSlots(i); err != nil {
					return err
				}
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
		switch status {
		case connectedPeer:
			continue
		case unknownPeer:
			peerState.discover(setIDX, reservePeer)
		}

		var n *node
		n, err = ps.peerState.getNode(reservePeer)
		if err != nil {
			return err
		}

		if n.getReputation() < BannedThresholdValue {
			break
		}

		if err = peerState.tryOutgoing(setIDX, reservePeer); err != nil {
			return err
		}

		ps.resultMsgCh <- Message{
			Status: Connect,
			setID:  uint64(setIDX),
			PeerID: reservePeer,
		}
	}
	// nothing more to do if we're in reserved mode.
	if ps.isReservedOnly {
		return nil
	}

	for peerState.hasFreeOutgoingSlot(setIDX) {
		peerID := peerState.highestNotConnectedPeer(setIDX)
		if peerID == "" {
			break
		}

		n := peerState.nodes[peerID]
		if n.getReputation() < BannedThresholdValue {
			break
		}

		if err = peerState.tryOutgoing(setIDX, peerID); err != nil {
			break
		}

		ps.resultMsgCh <- Message{
			Status: Connect,
			setID:  uint64(setIDX),
			PeerID: peerID,
		}
	}
	return nil
}

func (ps *PeerSet) addReservedPeers(setID int, peers ...peer.ID) error {
	for _, peerID := range peers {
		if _, ok := ps.reservedNode[peerID]; ok {
			logger.Debug("peer already exists in peerSet", "peer", peerID)
			return nil
		}

		ps.reservedNode[peerID] = struct{}{}
		ps.peerState.addNoSlotNode(setID, peerID)
		if err := ps.allocSlots(setID); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PeerSet) removeReservedPeers(setID int, peers ...peer.ID) error {
	for _, peerID := range peers {
		if _, ok := ps.reservedNode[peerID]; !ok {
			logger.Debug("peer doesn't exists in the peerSet", "peerID", peerID)
			return nil
		}

		delete(ps.reservedNode, peerID)
		ps.peerState.removeNoSlotNode(setID, peerID)

		// nothing more to do if not in reservedOnly mode.
		if !ps.isReservedOnly {
			return nil
		}

		// reservedOnly mode is not yet implemented for future this code will help.
		// If however the peerSet is in reserved-only mode, then non-reserved node peers needs to be
		// disconnected.
		if ps.peerState.peerStatus(setID, peerID) == connectedPeer {
			err := ps.peerState.disconnect(setID, peerID)
			if err != nil {
				return err
			}

			ps.resultMsgCh <- Message{
				Status: Drop,
				setID:  uint64(setID),
				PeerID: peerID,
			}
		}
	}

	return nil
}

func (ps *PeerSet) setReservedPeer(setID int, peers ...peer.ID) error {
	toInsert, toRemove := make([]peer.ID, 0, len(peers)), make([]peer.ID, 0, len(peers))
	peerIDMap := make(map[peer.ID]struct{}, len(peers))
	for _, pid := range peers {
		peerIDMap[pid] = struct{}{}
		if _, ok := ps.reservedNode[pid]; ok {
			continue
		}
		toInsert = append(toInsert, pid)
	}

	for pid := range ps.reservedNode {
		if _, ok := peerIDMap[pid]; ok {
			continue
		}
		toRemove = append(toRemove, pid)
	}

	if err := ps.addReservedPeers(setID, toInsert...); err != nil {
		return err
	}

	return ps.removeReservedPeers(setID, toRemove...)
}

func (ps *PeerSet) addPeer(setID int, peers peer.IDSlice) error {
	for _, pid := range peers {
		if ps.peerState.peerStatus(setID, pid) != unknownPeer {
			return nil
		}

		ps.peerState.discover(setID, pid)
		if err := ps.allocSlots(setID); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PeerSet) removePeer(setID int, peers ...peer.ID) error {
	for _, pid := range peers {
		// don't do anything if node is reserved.
		if _, ok := ps.reservedNode[pid]; ok {
			logger.Debug("peer is reserved cannot be removed", "peer: ", pid)
			return nil
		}

		if status := ps.peerState.peerStatus(setID, pid); status == connectedPeer {
			ps.resultMsgCh <- Message{
				Status: Drop,
				setID:  uint64(setID),
				PeerID: pid,
			}

			// disconnect and forget
			err := ps.peerState.disconnect(setID, pid)
			if err != nil {
				return err
			}

			if err = ps.peerState.forgetPeer(setID, pid); err != nil {
				return err
			}
		} else if status == notConnectedPeer {
			if err := ps.peerState.forgetPeer(setID, pid); err != nil {
				return err
			}
		}
	}
	return nil
}

// incoming indicates that we have received an incoming connection. Must be answered either with
// a corresponding `Accept` or `Reject`, except if we were already connected to this peer.
func (ps *PeerSet) incoming(setID int, peers ...peer.ID) error {
	if err := ps.updateTime(); err != nil {
		return err
	}

	// This is for reserved only mode.
	for _, pid := range peers {
		if ps.isReservedOnly {
			if _, ok := ps.reservedNode[pid]; !ok {
				ps.resultMsgCh <- Message{Status: Reject}
				return nil
			}
		}

		status := ps.peerState.peerStatus(setID, pid)
		switch status {
		case connectedPeer:
			return nil
		case notConnectedPeer:
			ps.peerState.nodes[pid].lastConnected[setID] = time.Now()
		case unknownPeer:
			ps.peerState.discover(setID, pid)
		}

		state := ps.peerState
		p := state.nodes[pid]
		switch {
		case p.getReputation() < BannedThresholdValue:
			ps.resultMsgCh <- Message{Status: Reject}
		case state.tryAcceptIncoming(setID, pid) != nil:
			ps.resultMsgCh <- Message{Status: Reject}
		default:
			ps.resultMsgCh <- Message{Status: Accept}
		}
	}

	return nil
}

// DropReason represents reason for disconnection of the peer
type DropReason int

const (
	// UnknownDrop is used when substream or connection has been closed for an unknown reason
	UnknownDrop DropReason = iota
	// RefusedDrop is used when sub-stream or connection has been explicitly refused by the target.
	// In other words, the peer doesn't actually belong to this set.
	RefusedDrop
)

// disconnect indicate that we disconnect an active connection with a peer, or that we failed to connect.
// Must only be called after the peerSet has either generated a Connect message with this
// peer, or accepted an incoming connection with this peer.
func (ps *PeerSet) disconnect(setIdx int, reason DropReason, peers ...peer.ID) error {
	err := ps.updateTime()
	if err != nil {
		return err
	}

	state := ps.peerState
	for _, pid := range peers {
		connectionStatus := state.peerStatus(setIdx, pid)
		if connectionStatus != connectedPeer {
			return errors.New("received disconnect for non-connected node")
		}

		n := state.nodes[pid]
		n.addReputation(disconnectReputationChange)
		state.nodes[pid] = n

		if err = state.disconnect(setIdx, pid); err != nil {
			return err
		}
		ps.resultMsgCh <- Message{
			Status: Drop,
			PeerID: pid,
		}

		// TODO: figure out the condition of connection refuse.
		if reason == RefusedDrop {
			if err = ps.removePeer(setIdx, pid); err != nil {
				return err
			}
		}
	}

	return ps.allocSlots(setIdx)
}

// start handles all the action for the peerSet.
func (ps *PeerSet) start() {
	ticker := time.NewTicker(ps.nextPeriodicAllocSlots)
	defer ticker.Stop()
	var err error

actionLoop:
	for {
		select {
		case <-ticker.C:
			{
				l := ps.peerState.getSetLength()
				for i := 0; i < l; i++ {
					if err = ps.allocSlots(i); err != nil {
						logger.Debug("failed to do action on peerSet ", "error", err)
					}
				}
			}
		case act, ok := <-ps.actionQueue:
			if !ok {
				break actionLoop
			}

			switch act.actionCall {
			case addReservedPeer:
				err = ps.addReservedPeers(act.setID, act.peers...)
			case removeReservedPeer:
				err = ps.removeReservedPeers(act.setID, act.peers...)
			case setReservedPeers:
				// TODO: this is not used yet, might required to implement RPC Call for this.
				err = ps.setReservedPeer(act.setID, act.peers...)
			case setReservedOnly:
				// TODO: not yet implemented
			case reportPeer:
				err = ps.reportPeer(act.reputation, act.peers...)
			case addToPeerSet:
				err = ps.addPeer(act.setID, act.peers)
			case removeFromPeerSet:
				err = ps.removePeer(act.setID, act.peers...)
			case incoming:
				err = ps.incoming(act.setID, act.peers...)
			case sortedPeers:
				ps.peerState.sortedPeers(0, act.resultPeersCh)
			case disconnect:
				err = ps.disconnect(0, UnknownDrop, act.peers...)
			}

			if err != nil {
				logger.Error("failed to do action on peerSet", "action", act, "error", err)
			}
		}
	}
}
