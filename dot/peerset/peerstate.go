// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import (
	"math"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
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

	// number of slot occupying nodes for which the MembershipState is ingoing.
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
	// list of Set the node belongs to.
	// always has a fixed size equal to the one of PeersState Set. The various possible Set
	// are indices into this Set.
	state []MembershipState

	// when we were last connected to the node, or if we were never connected when we
	// discovered it.
	lastConnected []time.Time

	// Reputation of the node, between int32 MIN and int32 MAX.
	rep Reputation
}

// newNode method to create a node with 0 Reputation at starting.
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

func (n *node) getReputation() Reputation {
	return n.rep
}

func (n *node) addReputation(modifier Reputation) Reputation {
	n.rep = n.rep.add(modifier)
	return n.rep
}

func (n *node) setReputation(modifier Reputation) {
	n.rep = modifier
}

// PeersState struct contains a list of nodes, where each node
// has a reputation and is either connected to us or not
type PeersState struct {
	// list of nodes that we know about.
	nodes map[peer.ID]*node
	// configuration of each set. The size of this Info is never modified.
	// since, single Info can also manage the flow.
	sets []Info
}

func (ps *PeersState) getNode(p peer.ID) (*node, error) {
	if n, ok := ps.nodes[p]; ok {
		return n, nil
	}

	return nil, ErrPeerDoesNotExist
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
			maxIn:       cfg.inPeers,
			maxOut:      cfg.outPeers,
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
	n, err := ps.getNode(peerID)
	if err != nil {
		return unknownPeer
	}

	switch n.state[set] {
	case ingoing, outgoing:
		return connectedPeer
	case notConnected:
		return notConnectedPeer
	}

	return unknownPeer
}

// peers return the list of all the peers we know of.
func (ps *PeersState) peers() []peer.ID {
	peerIDs := make([]peer.ID, 0, len(ps.nodes))
	for k := range ps.nodes {
		peerIDs = append(peerIDs, k)
	}
	return peerIDs
}

// sortedPeers returns the list of peers we are connected to of a specific set.
func (ps *PeersState) sortedPeers(idx int) peer.IDSlice {
	if len(ps.sets) < idx {
		logger.Debug("peer state doesn't have info for the provided index")
		return nil
	}

	type kv struct {
		peerID peer.ID
		Node   *node
	}

	var ss []kv
	for k, v := range ps.nodes {
		state := v.state[idx]
		if isPeerConnected(state) {
			ss = append(ss, kv{k, v})
		}
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Node.rep > ss[j].Node.rep
	})

	peerIDs := make(peer.IDSlice, len(ss))
	for i, kv := range ss {
		peerIDs[i] = kv.peerID
	}

	return peerIDs
}

// highestNotConnectedPeer returns the peer with the highest Reputation and that we are not connected to.
func (ps *PeersState) highestNotConnectedPeer(set int) peer.ID {
	var maxRep = math.MinInt32
	var peerID peer.ID
	for id, n := range ps.nodes {
		if n.state[set] != notConnected {
			continue
		}

		val := int(n.rep)
		if val >= maxRep {
			maxRep = val
			peerID = id
		}
	}

	return peerID
}

func (ps *PeersState) hasFreeOutgoingSlot(set int) bool {
	return ps.sets[set].numOut < ps.sets[set].maxOut
}

// Note: that it is possible for numIn to be strictly superior to the max, in case we were
// connected to reserved node then marked them as not reserved.
func (ps *PeersState) hasFreeIncomingSlot(set int) bool {
	return ps.sets[set].numIn >= ps.sets[set].maxIn
}

// addNoSlotNode adds a node to the list of nodes that don't occupy slots.
// has no effect if the node was already in the group.
func (ps *PeersState) addNoSlotNode(idx int, peerID peer.ID) {
	if _, ok := ps.sets[idx].noSlotNodes[peerID]; ok {
		logger.Debugf("peer %s already exists in no slot node", peerID)
		return
	}

	// Insert peerStatus
	ps.sets[idx].noSlotNodes[peerID] = struct{}{}
	n, err := ps.getNode(peerID)
	if err != nil {
		return
	}

	switch n.state[idx] {
	case ingoing:
		ps.sets[idx].numIn--
	case outgoing:
		ps.sets[idx].numOut--
	}

	ps.nodes[peerID] = n
}

func (ps *PeersState) removeNoSlotNode(idx int, peerID peer.ID) {
	if _, ok := ps.sets[idx].noSlotNodes[peerID]; !ok {
		return
	}

	delete(ps.sets[idx].noSlotNodes, peerID)
	n, err := ps.getNode(peerID)
	if err != nil {
		return
	}

	switch n.state[idx] {
	case ingoing:
		ps.sets[idx].numIn++
	case outgoing:
		ps.sets[idx].numOut++
	}
}

// disconnect updates the node status to the notConnected state.
// It should be called only when the node is in connected state.
func (ps *PeersState) disconnect(idx int, peerID peer.ID) error {
	info := ps.sets[idx]
	n, err := ps.getNode(peerID)
	if err != nil {
		return err
	}

	if _, ok := info.noSlotNodes[peerID]; !ok {
		switch n.state[idx] {
		case ingoing:
			info.numIn--
		case outgoing:
			info.numOut--
		default:
			return ErrPeerDisconnected
		}
	}

	// set node state to notConnected.
	n.state[idx] = notConnected
	n.lastConnected[idx] = time.Now()
	ps.sets[idx] = info
	return nil
}

// discover takes input for set id and create a node and insert in the list.
// the initial Reputation of the peer will be 0 and ingoing notMember state.
func (ps *PeersState) discover(set int, peerID peer.ID) {
	numSet := len(ps.sets)
	if _, err := ps.getNode(peerID); err != nil {
		n := newNode(numSet)
		n.state[set] = notConnected
		ps.nodes[peerID] = n
	}
}

func (ps *PeersState) lastConnectedAndDiscovered(set int, peerID peer.ID) time.Time {
	node, err := ps.getNode(peerID)
	if err != nil && node.state[set] == notConnected {
		return node.lastConnected[set]
	}
	return time.Now()
}

// forgetPeer removes the peer with reputation 0 from the peerSet.
func (ps *PeersState) forgetPeer(set int, peerID peer.ID) error {
	n, err := ps.getNode(peerID)
	if err != nil {
		return err
	}

	if n.state[set] != notMember {
		n.state[set] = notMember
	}

	if n.getReputation() != 0 {
		return nil
	}
	// remove the peer from peerSet nodes entirely if it isn't a member of any set.
	remove := true
	for _, state := range n.state {
		if state != notMember {
			remove = false
			break
		}
	}

	if remove {
		delete(ps.nodes, peerID)
	}

	return nil
}

// tryOutgoing tries to set the peer as connected as an outgoing connection.
// If there are enough slots available, switches the node to Connected and returns nil error. If
// the slots are full, the node stays "not connected" and we return error.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryOutgoing(setID int, peerID peer.ID) error {
	var isNoSlotOccupied bool
	if _, ok := ps.sets[setID].noSlotNodes[peerID]; ok {
		isNoSlotOccupied = true
	}

	if !ps.hasFreeOutgoingSlot(setID) && !isNoSlotOccupied {
		return ErrOutgoingSlotsUnavailable
	}

	n, err := ps.getNode(peerID)
	if err != nil {
		return err
	}

	n.state[setID] = outgoing
	if !isNoSlotOccupied {
		ps.sets[setID].numOut++
	}

	return nil
}

// tryAcceptIncoming tries to accept the peer as an incoming connection.
// if there are enough slots available, switches the node to Connected and returns nil.
// If the slots are full, the node stays "not connected" and we return Err.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryAcceptIncoming(setID int, peerID peer.ID) error {
	var isNoSlotOccupied bool
	if _, ok := ps.sets[setID].noSlotNodes[peerID]; ok {
		isNoSlotOccupied = true
	}

	// if slot is not available and the node is not a reserved node then error
	if ps.hasFreeIncomingSlot(setID) && !isNoSlotOccupied {
		return ErrIncomingSlotsUnavailable
	}

	n, err := ps.getNode(peerID)
	if err != nil {
		// state inconsistency tryOutgoing on an unknown node
		return err
	}

	n.state[setID] = ingoing
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
