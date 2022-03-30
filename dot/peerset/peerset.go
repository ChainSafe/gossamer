// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "peerset"))
)

const (
	// disconnectReputationChange Reputation change value for a node when we get disconnected from it.
	disconnectReputationChange Reputation = -256
	// forgetAfterTime amount of time between the moment we disconnect
	// from a node and the moment we remove it from the list.
	forgetAfterTime = time.Second * 3600 // one hour

	msgChanSize = 100
)

// ActionReceiver represents the enum value for action to be performed on peerSet
type ActionReceiver uint8

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

func (a ActionReceiver) String() string {
	switch a {
	case addReservedPeer:
		return "addReservedPeer"
	case removeReservedPeer:
		return "removeReservedPeer"
	case setReservedPeers:
		return "setReservedPeers"
	case setReservedOnly:
		return "setReservedOnly"
	case reportPeer:
		return "reportPeer"
	case addToPeerSet:
		return "addToPeerSet"
	case removeFromPeerSet:
		return "removeFromPeerSet"
	case incoming:
		return "incoming"
	case sortedPeers:
		return "sortedPeers"
	case disconnect:
		return "disconnect"
	default:
		return "invalid action"
	}
}

// action struct stores the action type and required parameters to perform action
type action struct {
	actionCall    ActionReceiver
	setID         int
	reputation    ReputationChange
	peers         peer.IDSlice
	resultPeersCh chan peer.IDSlice
}

func (a action) String() string {
	peersStrings := make([]string, len(a.peers))
	for i := range a.peers {
		peersStrings[i] = a.peers[i].String()
	}
	return fmt.Sprintf("{call=%s, set-id=%d, reputation change %v, peers=[%s]",
		a.actionCall.String(), a.setID, a.reputation, strings.Join(peersStrings, ", "))
}

// Status represents the enum value for Message
type Status uint8

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
	// Status of the peer in current set.
	Status Status
	setID  uint64
	// PeerID peer in message.
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

func (r *ReputationChange) String() string {
	return fmt.Sprintf("value: %d, reason: %s", r.Value, r.Reason)
}

func newReputationChange(value Reputation, reason string) ReputationChange {
	return ReputationChange{value, reason}
}

// MessageProcessor interface allows the network layer to receive and
// process messages from the peerstate layer
type MessageProcessor interface {
	Process(Message)
}

// PeerSet is a container for all the components of a peerSet.
type PeerSet struct {
	sync.Mutex
	peerState *PeersState

	reservedLock sync.RWMutex
	reservedNode map[peer.ID]struct{}
	// TODO: this will be useful for reserved only mode
	// this is for future purpose if reserved-only flag is enabled (#1888).
	isReservedOnly bool
	resultMsgCh    chan Message
	// time when the PeerSet was created.
	created time.Time
	// last time when we updated the reputations of connected nodes.
	latestTimeUpdate time.Time
	// next time to do a periodic call to allocSlots with all Set. This is done once two
	// second, to match the period of the Reputation updates.
	nextPeriodicAllocSlots time.Duration
	actionQueue            <-chan action
}

// config is configuration of a single set.
type config struct {
	// maximum number of slot occupying nodes for incoming connections.
	maxInPeers uint32
	// maximum number of slot occupying nodes for outgoing connections.
	maxOutPeers uint32

	// TODO Use in future for reserved only peers
	// if true, we only accept reservedNodes (#1888).
	reservedOnly bool

	// time duration for a peerSet to periodically call allocSlots.
	periodicAllocTime time.Duration
}

// ConfigSet set of peerSet config.
type ConfigSet struct {
	Set []*config
}

// NewConfigSet creates a new config set for the peerSet
func NewConfigSet(maxInPeers, maxOutPeers uint32, reservedOnly bool, allocTime time.Duration) *ConfigSet {
	set := &config{
		maxInPeers:        maxInPeers,
		maxOutPeers:       maxOutPeers,
		reservedOnly:      reservedOnly,
		periodicAllocTime: allocTime,
	}

	return &ConfigSet{
		// Why are we using an array of config in the set, when we are
		// using just one config
		Set: []*config{set},
	}
}

func newPeerSet(cfg *ConfigSet) (*PeerSet, error) {
	if len(cfg.Set) == 0 {
		return nil, ErrConfigSetIsEmpty
	}

	peerState, err := NewPeerState(cfg.Set)
	if err != nil {
		return nil, err
	}

	// TODO: currently we only have one set, change this once we have more (#1886).
	cfgSet := cfg.Set[0]
	now := time.Now()

	ps := &PeerSet{
		peerState:              peerState,
		reservedNode:           make(map[peer.ID]struct{}),
		isReservedOnly:         cfgSet.reservedOnly,
		created:                now,
		latestTimeUpdate:       now,
		nextPeriodicAllocSlots: cfgSet.periodicAllocTime,
	}

	return ps, nil
}

// If we multiply each second the reputation by `k` (where `k` is between 0 and 1), it
// takes `ln(0.5) / ln(k)` seconds to reduce the reputation by half. Use this formula to
// empirically determine a value of `k` that looks correct.
// we use `k = 0.98`, so we divide by `50`. With that value, it takes 34.3 seconds
// to reduce the reputation by half.
func reputationTick(reput Reputation) Reputation {
	diff := reput / 50
	if diff == 0 && reput < 0 {
		diff = -1
	} else if diff == 0 && reput > 0 {
		diff = 1
	}
	return reput.sub(diff)
}

// updateTime updates the value of latestTimeUpdate and performs all the updates that
// happen over time, such as Reputation increases for staying connected.
func (ps *PeerSet) updateTime() error {
	ps.Lock()
	defer ps.Unlock()

	currTime := time.Now()
	// identify the time difference between current time and last update time for peer reputation in seconds.
	// update the latestTimeUpdate to currTime.
	elapsedLatest := ps.latestTimeUpdate.Sub(ps.created)
	elapsedNow := currTime.Sub(ps.created)
	ps.latestTimeUpdate = currTime
	secDiff := int64(elapsedNow.Seconds() - elapsedLatest.Seconds())

	// This will give for how many seconds decaying is required for each peer.
	// For each elapsed second, move the node reputation towards zero.
	for i := int64(0); i < secDiff; i++ {
		for _, peerID := range ps.peerState.peers() {
			after, err := ps.peerState.updateReputationByTick(peerID)
			if err != nil {
				return fmt.Errorf("cannot update reputation by tick: %w", err)
			}

			if after != 0 {
				continue
			}

			// if the peer reaches reputation 0, and there is no connection to it, forget it.
			length := ps.peerState.getSetLength()
			for set := 0; set < length; set++ {
				if ps.peerState.peerStatus(set, peerID) != notConnectedPeer {
					continue
				}

				lastDiscoveredTime, err := ps.peerState.lastConnectedAndDiscovered(set, peerID)
				if err != nil {
					return fmt.Errorf("cannot get last connected peer: %w", err)
				}

				if lastDiscoveredTime.Add(forgetAfterTime).Second() >= currTime.Second() {
					continue
				}

				// forget peer removes the peer from the list of members of the set.
				err = ps.peerState.forgetPeer(set, peerID)
				if err != nil {
					return fmt.Errorf("cannot forget peer %s from set %d: %w", peerID, set, err)
				}
			}
		}
	}

	return nil
}

// reportPeer on report ReputationChange of the peer based on its behaviour,
// if the updated Reputation is below BannedThresholdValue then, this node need to
// be disconnected and a drop message for the peer is sent in order to disconnect.
func (ps *PeerSet) reportPeer(change ReputationChange, peers ...peer.ID) error {
	// we want reputations to be up-to-date before adjusting them.
	err := ps.updateTime()
	if err != nil {
		return fmt.Errorf("cannot update time: %w", err)
	}

	for _, pid := range peers {
		rep, err := ps.peerState.addReputation(pid, change)
		if err != nil {
			return fmt.Errorf("cannot add reputation (%s) to peer %s: %w", pid, change.String(), err)
		}

		if rep >= BannedThresholdValue {
			return nil
		}

		setLen := ps.peerState.getSetLength()
		for i := 0; i < setLen; i++ {
			if ps.peerState.peerStatus(i, pid) != connectedPeer {
				continue
			}

			// disconnect peer
			err = ps.peerState.disconnect(i, pid)
			if err != nil {
				return fmt.Errorf("cannot disconnect peer %s at set %d: %w", pid, i, err)
			}

			dropMessage := Message{
				Status: Drop,
				setID:  uint64(i),
				PeerID: pid,
			}

			ps.resultMsgCh <- dropMessage

			if err = ps.allocSlots(i); err != nil {
				return fmt.Errorf("could not allocate slots: %w", err)
			}
		}
	}
	return nil
}

// allocSlots tries to fill available outgoing slots of nodes for the given set.
func (ps *PeerSet) allocSlots(setIdx int) error {
	err := ps.updateTime()
	if err != nil {
		return fmt.Errorf("cannot update time: %w", err)
	}

	peerState := ps.peerState
	for reservePeer := range ps.reservedNode {
		status := peerState.peerStatus(setIdx, reservePeer)
		switch status {
		case connectedPeer:
			continue
		case unknownPeer:
			peerState.discover(setIdx, reservePeer)
		}

		node, err := ps.peerState.getNode(reservePeer)
		if err != nil {
			return fmt.Errorf("cannot get node using peer id %s: %w", reservePeer, err)
		}

		if node.reputation < BannedThresholdValue {
			logger.Warnf("reputation is lower than banned threshold value, reputation: %d, banned threshold value: %d",
				node.reputation, BannedThresholdValue)
			break
		}

		if err = peerState.tryOutgoing(setIdx, reservePeer); err != nil {
			return fmt.Errorf("cannot set peer %s from set %d as outgoing: %w", reservePeer, setIdx, err)
		}

		connectMessage := Message{
			Status: Connect,
			setID:  uint64(setIdx),
			PeerID: reservePeer,
		}

		ps.resultMsgCh <- connectMessage
	}

	// nothing more to do if we're in reserved mode.
	if ps.isReservedOnly {
		return nil
	}

	for peerState.hasFreeOutgoingSlot(setIdx) {
		peerID := peerState.highestNotConnectedPeer(setIdx)
		if peerID == "" {
			break
		}

		n := peerState.nodes[peerID]
		if n.reputation < BannedThresholdValue {
			logger.Critical("highest rated peer is below bannedThresholdValue")
			break
		}

		if err = peerState.tryOutgoing(setIdx, peerID); err != nil {
			logger.Errorf("could not set peer %s as outgoing connection: %s", peerID.Pretty(), err)
			break
		}

		connectMessage := Message{
			Status: Connect,
			setID:  uint64(setIdx),
			PeerID: peerID,
		}

		ps.resultMsgCh <- connectMessage

		logger.Debugf("Sent connect message to peer %s", peerID)
	}
	return nil
}

func (ps *PeerSet) addReservedPeers(setID int, peers ...peer.ID) error {
	ps.reservedLock.Lock()
	defer ps.reservedLock.Unlock()

	for _, peerID := range peers {
		if _, ok := ps.reservedNode[peerID]; ok {
			logger.Debugf("peer %s already exists in peerSet", peerID)
			return nil
		}

		ps.peerState.discover(setID, peerID)

		ps.reservedNode[peerID] = struct{}{}
		if err := ps.peerState.addNoSlotNode(setID, peerID); err != nil {
			return fmt.Errorf("could not add to list of no-slot nodes: %w", err)
		}
		if err := ps.allocSlots(setID); err != nil {
			return fmt.Errorf("could not allocate slots: %w", err)
		}
	}
	return nil
}

func (ps *PeerSet) removeReservedPeers(setID int, peers ...peer.ID) error {
	ps.reservedLock.Lock()
	defer ps.reservedLock.Unlock()

	for _, peerID := range peers {
		if _, ok := ps.reservedNode[peerID]; !ok {
			logger.Debugf("peer %s doesn't exist in the peerSet", peerID)
			return nil
		}

		delete(ps.reservedNode, peerID)
		if err := ps.peerState.removeNoSlotNode(setID, peerID); err != nil {
			return fmt.Errorf("could not remove from the list of no-slot nodes: %w", err)
		}

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
				return fmt.Errorf("cannot disconnect peer %s at set %d: %w", peerID, setID, err)
			}

			dropMessage := Message{
				Status: Drop,
				setID:  uint64(setID),
				PeerID: peerID,
			}

			ps.resultMsgCh <- dropMessage
		}
	}

	return nil
}

func (ps *PeerSet) setReservedPeer(setID int, peers ...peer.ID) error {
	toInsert := make([]peer.ID, 0, len(peers))
	toRemove := make([]peer.ID, 0, len(peers))

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

	err := ps.addReservedPeers(setID, toInsert...)
	if err != nil {
		return fmt.Errorf("cannot add reserved peers: %w", err)
	}

	err = ps.removeReservedPeers(setID, toRemove...)
	if err != nil {
		return fmt.Errorf("cannot remove reserved peers: %w", err)
	}

	return nil
}

func (ps *PeerSet) addPeer(setID int, peers peer.IDSlice) error {
	for _, pid := range peers {
		if ps.peerState.peerStatus(setID, pid) != unknownPeer {
			return nil
		}

		ps.peerState.discover(setID, pid)
		if err := ps.allocSlots(setID); err != nil {
			return fmt.Errorf("could not allocate slots: %w", err)
		}
	}
	return nil
}

func (ps *PeerSet) removePeer(setID int, peers ...peer.ID) error {
	for _, pid := range peers {
		if _, ok := ps.reservedNode[pid]; ok {
			logger.Debugf("peer %s is reserved and cannot be removed", pid)
			return nil
		}

		if status := ps.peerState.peerStatus(setID, pid); status == connectedPeer {
			dropMessage := Message{
				Status: Drop,
				setID:  uint64(setID),
				PeerID: pid,
			}

			ps.resultMsgCh <- dropMessage

			// disconnect and forget
			err := ps.peerState.disconnect(setID, pid)
			if err != nil {
				return fmt.Errorf("cannot disconnect peer %s at set %d: %w", pid, setID, err)
			}

			if err = ps.peerState.forgetPeer(setID, pid); err != nil {
				return fmt.Errorf("cannot forget peer %s from set %d: %w", pid, setID, err)
			}
		} else if status == notConnectedPeer {
			if err := ps.peerState.forgetPeer(setID, pid); err != nil {
				return fmt.Errorf("cannot forget peer %s from set %d: %w", pid, setID, err)
			}
		}
	}
	return nil
}

// incoming indicates that we have received an incoming connection. Must be answered
// either with a corresponding `Accept` or `Reject`, except if we were already
// connected to this peer.
func (ps *PeerSet) incoming(setID int, peers ...peer.ID) error {
	err := ps.updateTime()
	if err != nil {
		return fmt.Errorf("cannot update time: %w", err)
	}

	for _, pid := range peers {
		if ps.isReservedOnly {
			_, has := ps.reservedNode[pid]
			if !has {
				message := Message{
					Status: Reject,
					setID:  uint64(setID),
					PeerID: pid,
				}

				ps.resultMsgCh <- message
				continue
			}
		}

		status := ps.peerState.peerStatus(setID, pid)
		switch status {
		case connectedPeer:
			continue
		case notConnectedPeer:
			ps.peerState.nodes[pid].lastConnected[setID] = time.Now()
		case unknownPeer:
			ps.peerState.discover(setID, pid)
		}

		state := ps.peerState

		var nodeReputation Reputation

		state.RLock()
		node, has := state.nodes[pid]
		if has {
			nodeReputation = node.reputation
		}
		state.RUnlock()

		message := Message{
			setID:  uint64(setID),
			PeerID: pid,
		}

		switch {
		case nodeReputation < BannedThresholdValue:
			message.Status = Reject
		default:
			err := state.tryAcceptIncoming(setID, pid)
			if err != nil {
				logger.Errorf("cannot accept incomming peer %pid: %w", pid, err)
				message.Status = Reject
			} else {
				logger.Debugf("incoming connection accepted from peer %s", pid)
				message.Status = Accept
			}
		}

		ps.resultMsgCh <- message
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
		return fmt.Errorf("cannot update time: %w", err)
	}

	state := ps.peerState
	for _, pid := range peers {
		connectionStatus := state.peerStatus(setIdx, pid)
		if connectionStatus != connectedPeer {
			return ErrDisconnectReceivedForNonConnectedPeer
		}

		n := state.nodes[pid]
		n.addReputation(disconnectReputationChange)
		state.nodes[pid] = n

		if err = state.disconnect(setIdx, pid); err != nil {
			return fmt.Errorf("cannot disconnect peer %s at set %d: %w", pid, setIdx, err)
		}

		dropMessage := Message{
			Status: Drop,
			setID:  uint64(setIdx),
			PeerID: pid,
		}

		ps.resultMsgCh <- dropMessage

		// TODO: figure out the condition of connection refuse.
		if reason == RefusedDrop {
			if err = ps.removePeer(setIdx, pid); err != nil {
				return fmt.Errorf("cannot remove peer %s at set %d: %w", pid, setIdx, err)
			}
		}
	}

	return ps.allocSlots(setIdx)
}

// start handles all the action for the peerSet.
func (ps *PeerSet) start(ctx context.Context, actionQueue chan action) {
	ps.actionQueue = actionQueue
	ps.resultMsgCh = make(chan Message, msgChanSize)

	go ps.listenAction(ctx)
	go ps.periodicallyAllocateSlots(ctx)
}

func (ps *PeerSet) listenAction(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case act, ok := <-ps.actionQueue:

			if !ok {
				return
			}

			var err error
			switch act.actionCall {
			case addReservedPeer:
				err = ps.addReservedPeers(act.setID, act.peers...)
			case removeReservedPeer:
				err = ps.removeReservedPeers(act.setID, act.peers...)
			case setReservedPeers:
				// TODO: this is not used yet, might required to implement RPC Call for this.
				err = ps.setReservedPeer(act.setID, act.peers...)
			case setReservedOnly:
				// TODO: not yet implemented (#1888)
				err = fmt.Errorf("not implemented yet")
			case reportPeer:
				err = ps.reportPeer(act.reputation, act.peers...)
			case addToPeerSet:
				err = ps.addPeer(act.setID, act.peers)
			case removeFromPeerSet:
				err = ps.removePeer(act.setID, act.peers...)
			case incoming:
				err = ps.incoming(act.setID, act.peers...)
			case sortedPeers:
				act.resultPeersCh <- ps.peerState.sortedPeers(act.setID)
			case disconnect:
				err = ps.disconnect(act.setID, UnknownDrop, act.peers...)
			}

			if err != nil {
				logger.Errorf("failed to do action %s on peerSet: %s", act, err)
			}
		}
	}
}

func (ps *PeerSet) periodicallyAllocateSlots(ctx context.Context) {
	ticker := time.NewTicker(ps.nextPeriodicAllocSlots)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// TODO: log context error?
			return
		case <-ticker.C:
			setLen := ps.peerState.getSetLength()
			for i := 0; i < setLen; i++ {
				if err := ps.allocSlots(i); err != nil {
					logger.Warnf("failed to do action on peerSet: %s", err)
				}
			}
		}
	}
}
