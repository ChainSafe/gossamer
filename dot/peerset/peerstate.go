// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

const (
	// connectedPeer peerStatus is ingoing connected state.
	connectedPeer = "connectedPeer"
	// notConnectedPeer peerStatus is ingoing not connected state.
	notConnectedPeer = "notConnectedPeer"
	// unknownPeer peerStatus is unknown
	unknownPeer = "unknownPeer"
)

// MembershipState represent the state of node ingoing the set.
type MembershipState int

const (
	// notMember node isn't part of that set.
	notMember MembershipState = iota
	// ingoing node is connected through an ingoing connection.
	ingoing
	// outgoing node is connected through an outgoing connection.
	outgoing
	// notConnected node is part of that set, but we are not connected to it.
	notConnected
)

// Info is state of a single set.
type Info struct {
	// number of slot occupying nodes for which the MembershipState is ingoing.
	numIn uint32

	// number of slot occupying nodes for which the MembershipState is outgoing.
	numOut uint32

	// maximum allowed number of slot occupying nodes for which the MembershipState is ingoing.
	maxIn uint32

	// maximum allowed number of slot occupying nodes for which the MembershipState is outgoing.
	maxOut uint32

	// list of node identities (discovered or not) that don't occupy slots.
	// Note for future readers: this module is purely dedicated to managing slots.
	// If you are considering adding more features, please consider doing so outside this module rather
	// than inside.
	noSlotNodes map[peer.ID]struct{}
}

// node represents state of a single node that we know about
type node struct {
	// state is a list of sets containing the node.
	// always has a fixed size, equal to the one of PeersState Set. The various possible Set
	// are indices into this Set.
	state []MembershipState

	// when we were last connected to the node, or if we were never connected when we
	// discovered it.
	lastConnected []time.Time

	// Reputation of the node, between int32 MIN and int32 MAX.
	reputation Reputation
}

// newNode creates a node with n number of sets and 0 reputation.
func newNode(n int) *node {
	now := time.Now()
	sets := make([]MembershipState, n)
	lastConnected := make([]time.Time, n)
	for i := 0; i < n; i++ {
		sets[i] = notMember
		lastConnected[i] = now
	}

	return &node{
		state:         sets,
		lastConnected: lastConnected,
	}
}

func (n *node) addReputation(modifier Reputation) Reputation {
	n.reputation = n.reputation.add(modifier)
	return n.reputation
}

// PeersState struct contains a list of nodes, where each node
// has a reputation and is either connected to us or not
type PeersState struct {
	// list of nodes that we know about.
	nodes map[peer.ID]*node
	// configuration of each set. The size of this Info is never modified.
	// since, single Info can also manage the flow.
	sets []Info

	sync.RWMutex
}

func (ps *PeersState) getNode(p peer.ID) (*node, error) {
	ps.RLock()
	defer ps.RUnlock()
	if n, ok := ps.nodes[p]; ok {
		return n, nil
	}

	return nil, fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, p)
}

// NewPeerState initiates a new PeersState
func NewPeerState(cfgs []*config) (*PeersState, error) {
	if len(cfgs) == 0 {
		return nil, ErrConfigSetIsEmpty
	}

	infoSet := make([]Info, 0, len(cfgs))
	for _, cfg := range cfgs {
		info := Info{
			numIn:       0,
			numOut:      0,
			maxIn:       cfg.maxInPeers,
			maxOut:      cfg.maxOutPeers,
			noSlotNodes: make(map[peer.ID]struct{}),
		}

		infoSet = append(infoSet, info)
	}

	peerState := &PeersState{
		nodes: make(map[peer.ID]*node),
		sets:  infoSet,
	}

	return peerState, nil
}

func (ps *PeersState) getSetLength() int {
	return len(ps.sets)
}

// peerStatus returns the status of peer based on its connection state
// i.e. connectedPeer, notConnectedPeer or unknownPeer.
func (ps *PeersState) peerStatus(set int, peerID peer.ID) string {
	ps.RLock()
	defer ps.RUnlock()

	node, has := ps.nodes[peerID]
	if !has {
		return unknownPeer
	}

	switch node.state[set] {
	case ingoing, outgoing:
		return connectedPeer
	case notConnected:
		return notConnectedPeer
	}

	return unknownPeer
}

// peers return the list of all the peers we know of.
func (ps *PeersState) peers() []peer.ID {
	ps.RLock()
	defer ps.RUnlock()
	return maps.Keys(ps.nodes)
}

// sortedPeers returns the list of peers we are connected to of a specific set.
func (ps *PeersState) sortedPeers(idx int) peer.IDSlice {
	ps.RLock()
	defer ps.RUnlock()

	if len(ps.sets) == 0 || len(ps.sets) < idx {
		logger.Debug("peer state doesn't have info for the provided index")
		return nil
	}

	type connectedPeerReputation struct {
		peerID     peer.ID
		reputation Reputation
	}

	connectedPeersReps := make([]connectedPeerReputation, 0, len(ps.nodes))

	for peerID, node := range ps.nodes {
		state := node.state[idx]

		if isPeerConnected(state) {
			connectedPeersReps = append(connectedPeersReps, connectedPeerReputation{
				peerID:     peerID,
				reputation: node.reputation,
			})
		}
	}

	sort.Slice(connectedPeersReps, func(i, j int) bool {
		return connectedPeersReps[i].reputation > connectedPeersReps[j].reputation
	})

	peerIDs := make(peer.IDSlice, len(connectedPeersReps))
	for i, kv := range connectedPeersReps {
		peerIDs[i] = kv.peerID
	}

	return peerIDs
}

func (ps *PeersState) updateReputationByTick(peerID peer.ID) (newReputation Reputation, err error) {
	ps.Lock()
	defer ps.Unlock()

	node, has := ps.nodes[peerID]
	if !has {
		return 0, fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	newReputation = reputationTick(node.reputation)

	node.reputation = newReputation
	ps.nodes[peerID] = node

	return newReputation, nil
}

func (ps *PeersState) addReputation(peerID peer.ID, change ReputationChange) (
	newReputation Reputation, err error) {

	ps.Lock()
	defer ps.Unlock()

	node, has := ps.nodes[peerID]
	if !has {
		return 0, fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	newReputation = node.addReputation(change.Value)
	ps.nodes[peerID] = node

	return newReputation, nil
}

// highestNotConnectedPeer returns the peer with the highest Reputation and that we are not connected to.
func (ps *PeersState) highestNotConnectedPeer(set int) (highestPeerID peer.ID) {
	ps.RLock()
	defer ps.RUnlock()

	maxRep := math.MinInt32
	for peerID, node := range ps.nodes {
		if node.state[set] != notConnected {
			continue
		}

		val := int(node.reputation)
		if val >= maxRep {
			maxRep = val
			highestPeerID = peerID
		}
	}

	return highestPeerID
}

// hasFreeOutgoingSlot check does number of connected out peers is less then max amount allowed connected peers.
// maxOut is defined as config param as.
func (ps *PeersState) hasFreeOutgoingSlot(set int) bool {
	return ps.sets[set].numOut < ps.sets[set].maxOut
}

// Note: that it is possible for numIn to be strictly superior to the max, in case we were
// connected to reserved node then marked them as not reserved.
// maxIn is defined as config param.
func (ps *PeersState) hasFreeIncomingSlot(set int) bool {
	return ps.sets[set].numIn < ps.sets[set].maxIn
}

// addNoSlotNode adds a node to the list of nodes that don't occupy slots.
// has no effect if the node was already in the group.
func (ps *PeersState) addNoSlotNode(idx int, peerID peer.ID) error {
	ps.Lock()
	defer ps.Unlock()

	if _, ok := ps.sets[idx].noSlotNodes[peerID]; ok {
		logger.Debugf("peer %s already exists in no slot node", peerID)
		return nil
	}

	// Insert peerStatus
	ps.sets[idx].noSlotNodes[peerID] = struct{}{}

	node, has := ps.nodes[peerID]
	if !has {
		return fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	switch node.state[idx] {
	case ingoing:
		ps.sets[idx].numIn--
	case outgoing:
		ps.sets[idx].numOut--
	}

	return nil
}

func (ps *PeersState) removeNoSlotNode(idx int, peerID peer.ID) error {
	ps.Lock()
	defer ps.Unlock()

	if _, ok := ps.sets[idx].noSlotNodes[peerID]; !ok {
		logger.Debugf("peer %s is not in no-slot node map", peerID)
		return nil
	}

	delete(ps.sets[idx].noSlotNodes, peerID)

	node, has := ps.nodes[peerID]
	if !has {
		return fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	switch node.state[idx] {
	case ingoing:
		ps.sets[idx].numIn++
	case outgoing:
		ps.sets[idx].numOut++
	}

	return nil
}

// disconnect updates the node status to the notConnected state.
// It should be called only when the node is in connected state.
func (ps *PeersState) disconnect(idx int, peerID peer.ID) error {
	ps.Lock()
	defer ps.Unlock()

	info := ps.sets[idx]
	node, has := ps.nodes[peerID]
	if !has {
		return fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	_, has = info.noSlotNodes[peerID]
	if !has {
		switch node.state[idx] {
		case ingoing:
			info.numIn--
		case outgoing:
			info.numOut--
		default:
			return ErrPeerDisconnected
		}
	}

	// set node state to notConnected.
	node.state[idx] = notConnected
	node.lastConnected[idx] = time.Now()
	ps.sets[idx] = info

	return nil
}

// insertPeer takes input for set id and create a node and insert in the list.
// the initial Reputation of the peer will be 0 and ingoing notMember state.
func (ps *PeersState) insertPeer(set int, peerID peer.ID) {
	ps.Lock()
	defer ps.Unlock()

	_, has := ps.nodes[peerID]
	if !has {
		n := newNode(len(ps.sets))
		n.state[set] = notConnected
		ps.nodes[peerID] = n
	}
}

func (ps *PeersState) lastConnectedAndDiscovered(set int, peerID peer.ID) (time.Time, error) {
	ps.RLock()
	defer ps.RUnlock()

	node, has := ps.nodes[peerID]
	if !has {
		return time.Time{}, fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	if node.state[set] == notConnected {
		return node.lastConnected[set], nil
	}

	return time.Now(), nil
}

// forgetPeer removes the peer with reputation 0 from the peerSet
func (ps *PeersState) forgetPeer(set int, peerID peer.ID) error {
	ps.Lock()
	defer ps.Unlock()

	node, has := ps.nodes[peerID]
	if !has {
		return fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	if node.state[set] != notMember {
		node.state[set] = notMember
	}

	if node.reputation != 0 {
		return nil
	}

	// remove the peer from peerSet nodes entirely if it isn't a member of any set.
	remove := true
	for _, state := range node.state {
		if state != notMember {
			remove = false
			break
		}
	}

	if remove {
		logger.Warnf("removing peer %v from peerset, %v remaining peers", peerID, len(ps.nodes))
		delete(ps.nodes, peerID)
	}

	return nil
}

// tryOutgoing tries to set the peer as connected as an outgoing connection.
// If there are enough slots available, switches the node to Connected and returns nil.
// If the slots are full, the node stays "not connected" and we return the error ErrOutgoingSlotsUnavailable.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryOutgoing(setID int, peerID peer.ID) error {
	ps.Lock()
	defer ps.Unlock()

	_, isNoSlotNode := ps.sets[setID].noSlotNodes[peerID]

	if !ps.hasFreeOutgoingSlot(setID) && !isNoSlotNode {
		return ErrOutgoingSlotsUnavailable
	}

	node, has := ps.nodes[peerID]
	if !has {
		return fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	node.state[setID] = outgoing

	if !isNoSlotNode {
		ps.sets[setID].numOut++
	}

	return nil
}

// tryAcceptIncoming tries to accept the peer as an incoming connection.
// if there are enough slots available, switches the node to Connected and returns nil.
// If the slots are full, the node stays "not connected" and we return Err.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryAcceptIncoming(setID int, peerID peer.ID) error {
	ps.Lock()
	defer ps.Unlock()

	_, isNoSlotOccupied := ps.sets[setID].noSlotNodes[peerID]

	// if slot is not available and the node is not a reserved node then error
	if !ps.hasFreeIncomingSlot(setID) && !isNoSlotOccupied {
		return ErrIncomingSlotsUnavailable
	}

	node, has := ps.nodes[peerID]
	if !has {
		// state inconsistency tryOutgoing on an unknown node
		return fmt.Errorf("%w: for peer id %s", ErrPeerDoesNotExist, peerID)
	}

	node.state[setID] = ingoing
	if !isNoSlotOccupied {
		// this need to be added as incoming connection allocate slot.
		ps.sets[setID].numIn++
	}

	return nil
}

// isPeerConnected returns true if peer is connected else false
func isPeerConnected(state MembershipState) bool {
	return state == ingoing || state == outgoing
}
