package peerset

import (
	"errors"
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

// isConnected returns true if peer is connected else false
func isConnected(state MembershipState) bool { // nolint
	if state == ingoing || state == outgoing {
		return true
	}
	return false
}

// Info is state of a single set.
type Info struct {
	// Number of slot occupying nodes for which the MembershipState is ingoing.
	numIn uint32

	// Number of slot occupying nodes for which the MembershipState is ingoing.
	numOut uint32

	// Maximum allowed number of slot occupying nodes for which the MembershipState is ingoing.
	maxIn uint32

	// Maximum allowed number of slot occupying nodes for which the MembershipState is outgoing.
	maxOut uint32

	// List of node identities (discovered or not) that don't occupy slots.
	// Note for future readers: this module is purely dedicated to managing slots. If you are
	// considering adding more features, please consider doing so outside this module rather
	// than inside.
	noSlotNodes map[peer.ID]interface{}
}

// node represents state of a single node that we know about
type node struct {
	// List of Set the node belongs to.
	// always has a fixed size equal to the one of PeersState Set. The various possible Set
	// are indices into this Set.
	// TODO check why we are using slice for Set, how multiple Set need to be managed?

	state []MembershipState
	// When we were last connected to the node, or if we were never connected when we
	// discovered it.
	lastConnected []time.Time

	// Reputation value of the node, between int32 MIN (we hate that node) and
	// int32 math.MaxInt32 (we love that node).
	reputation int32
}

// newNode method to create a node with 0 reputation at starting.
func newNode(len int) *node {
	now := time.Now()
	sets := make([]MembershipState, len)
	lastConnected := make([]time.Time, len)
	for i := 0; i < len; i++ {
		sets[i] = notMember
		lastConnected[i] = now
	}

	return &node{
		state:         sets,
		lastConnected: lastConnected,
		reputation:    0,
	}
}

func (n *node) getReputation() int32 {
	return n.reputation
}

func (n *node) addReputation(modifier int32) int32 {
	n.reputation = addInt32(n.reputation, modifier)
	return n.reputation
}

// addInt32 handles overflow and underflow condition while adding two int32 values.
func addInt32(left, right int32) int32 {
	if right > 0 {
		if left > math.MaxInt32-right {
			return math.MaxInt32
		}
	} else {
		if left < math.MinInt32-right {
			return math.MinInt32
		}
	}
	return left + right
}

// subInt32 handles underflow condition while subtracting two int32 values.
func subInt32(left, right int32) int32 {
	if right < 0 {
		if left > math.MaxInt32+right {
			return math.MaxInt32
		}
	} else {
		if left < math.MinInt32+right {
			return math.MinInt32
		}
	}
	return left - right
}

func (n *node) setReputation(modifier int32) {
	n.reputation = modifier
}

// PeersState struct is nothing more but a data structure containing a list of nodes, where each node
// has a reputation and is either connected to us or not
type PeersState struct {
	// List of nodes that we know about.
	nodes map[peer.ID]*node
	// Configuration of each set. The size of this Info is never modified.
	// since, single Info can also manage the flow.
	// TODO Why we have slice for Info, How we are managing multiple Info?
	sets []Info
}

func (ps *PeersState) getNode(p peer.ID) (*node, error) {
	if n, ok := ps.nodes[p]; ok {
		return n, nil
	}

	return nil, errors.New("peer doesn't exist")
}

func (ps *PeersState) setNode(p peer.ID, n *node) {
	ps.nodes[p] = n
}

// NewPeerState builds a new PeersState
func NewPeerState(set []*config) (*PeersState, error) {
	if len(set) == 0 {
		return nil, errors.New("config set is empty")
	}
	infoSet := make([]Info, 0, len(set))
	for _, cfg := range set {
		info := Info{
			numIn:       0,
			numOut:      0,
			maxIn:       cfg.inPeers,
			maxOut:      cfg.outPeers,
			noSlotNodes: make(map[peer.ID]interface{}),
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

// peerStatus returns the status of peer based on its connection state, i.e. connectedPeer, notConnectedPeer or unknownPeer.
func (ps *PeersState) peerStatus(set int, peerID peer.ID) string {
	node, err := ps.getNode(peerID)
	if err != nil {
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
	peerIDs := make([]peer.ID, 0, len(ps.nodes))
	for k := range ps.nodes {
		peerIDs = append(peerIDs, k)
	}
	return peerIDs
}

// TODO this will be used once set reserved peers implemented
// sortedConnectedPeers returns the list of peers we are connected to of a specific set.
func (ps *PeersState) sortedConnectedPeers(idx int, peersCh chan interface{}) { // nolint
	var peerIDs peer.IDSlice
	if len(ps.sets) < idx {
		logger.Error("peer state doesn't have info for the provided index")
		return
	}

	type kv struct {
		peerID peer.ID
		Node   *node
	}

	var ss []kv
	for k, v := range ps.nodes {
		state := v.state[idx]
		if isConnected(state) {
			ss = append(ss, kv{k, v})
		}
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Node.reputation > ss[j].Node.reputation
	})

	for _, kv := range ss {
		peerIDs = append(peerIDs, kv.peerID)
	}

	peersCh <- peerIDs
}

// highestNotConnectedPeer returns the peerStatus with the highest reputation and that we are not connected to.
func (ps *PeersState) highestNotConnectedPeer(set int) peer.ID {
	var maxRep = math.MinInt32
	var peerID peer.ID
	for id, node := range ps.nodes {
		if node.state[set] != notConnected {
			continue
		}

		val := int(node.reputation)
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

// Note that it is possible for numIn to be strictly superior to the max, ingoing case we were
// connected to reserved node then marked them as not reserved.
func (ps *PeersState) hasFreeIncomingSlot(set int) bool {
	return ps.sets[set].numIn >= ps.sets[set].maxIn
}

// addNoSlotNode Adds a node to the list of nodes that don't occupy slots.
// has no effect if the node was already ingoing the group.
func (ps *PeersState) addNoSlotNode(idx int, peerID peer.ID) {
	if _, ok := ps.sets[idx].noSlotNodes[peerID]; ok {
		logger.Info("peer is already exists ingoing no slot node", "peerID", peerID)
		return
	}

	// Insert peerStatus
	ps.sets[idx].noSlotNodes[peerID] = struct{}{}
	node, err := ps.getNode(peerID)
	if err != nil {
		return
	}

	logger.Info("addNoSlotNode", "state", node.state[idx])
	if node.state[idx] == ingoing {
		ps.sets[idx].numIn--
	} else if node.state[idx] == outgoing {
		ps.sets[idx].numOut--
	}

	ps.nodes[peerID] = node
}

func (ps *PeersState) removeNoSlotNode(idx int, peerID peer.ID) {
	if _, ok := ps.sets[idx].noSlotNodes[peerID]; !ok {
		return
	}

	delete(ps.sets[idx].noSlotNodes, peerID)
	n, ok := ps.nodes[peerID]
	if !ok {
		return
	}

	switch n.state[idx] {
	case ingoing:
		ps.sets[idx].numIn++
	case outgoing:
		ps.sets[idx].numOut++
	}
}

// disconnect method updates the node status to notConnected state it should be called only when the node is ingoing connected state.
func (ps *PeersState) disconnect(idx int, peerID peer.ID) error {
	info := ps.sets[idx]
	n, err := ps.getNode(peerID)
	if err != nil {
		return err
	}

	// check for if it's isNoSlotOccupy is true or false.
	if _, ok := info.noSlotNodes[peerID]; !ok {
		switch n.state[idx] {
		case ingoing:
			if info.numIn > 0 {
				info.numIn--
			}
		case outgoing:
			if info.numOut > 0 {
				info.numOut--
			}
		case notMember, notConnected:
			return errors.New("state inconsistency: disconnecting disconnected node")
		}
	}

	// set node state to notConnected.
	n.state[idx] = notConnected
	n.lastConnected[idx] = time.Now()

	ps.sets[idx] = info
	return nil
}

// discover takes input for set id and create a node and insert ingoing the list.
// the initial reputation of the peerStatus will be 0 and ingoing notMember state.
func (ps *PeersState) discover(set int, peerID peer.ID) {
	numSet := len(ps.sets)
	if _, ok := ps.nodes[peerID]; !ok {
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
	// else return now time
	return time.Now()
}

// Removes the peerStatus from the list of members of the set.
func (ps *PeersState) forgetPeer(set int, peerID peer.ID) error {
	node, err := ps.getNode(peerID)
	if err != nil {
		return err
	}

	if node.state[set] != notMember {
		node.state[set] = notMember
	}

	if node.getReputation() == 0 {
		// Remove the peerStatus from ps nodes entirely if it isn't a member of any set.
		remove := true
		for _, state := range node.state {
			if state != notMember {
				remove = false
				break
			}
		}

		if remove {
			delete(ps.nodes, peerID)
		}
	}

	return nil
}

// Tries to set the peerStatus as connected as an outgoing connection.
// if there are enough slots available, switches the node to Connected and returns nil error. If
// the slots are full, the node stays "not connected" and we return error.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryOutgoing(setID int, peerID peer.ID) error {
	var isNoSlotOccupied bool
	if _, ok := ps.sets[setID].noSlotNodes[peerID]; ok {
		isNoSlotOccupied = true
	}

	if !ps.hasFreeOutgoingSlot(setID) && !isNoSlotOccupied {
		return errors.New("not enough outgoing slots")
	}

	node, ok := ps.nodes[peerID]
	if !ok {
		return errors.New("state inconsistency: tryOutgoing on an unknown node")
	}

	node.state[setID] = outgoing
	if !isNoSlotOccupied {
		ps.sets[setID].numOut++
	}

	return nil
}

// tryAcceptIncoming tries to accept the peerStatus as an incoming connection.
// if there are enough slots available, switches the node to Connected and returns nil. If
// the slots are full, the node stays "not connected" and we return Err.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryAcceptIncoming(setID int, peerID peer.ID) error {
	var isNoSlotOccupied bool
	if _, ok := ps.sets[setID].noSlotNodes[peerID]; ok {
		isNoSlotOccupied = true
	}

	// If slot is not available and the node is not a reserved node then error
	if ps.hasFreeIncomingSlot(setID) && !isNoSlotOccupied {
		return errors.New("not enough incoming slots")
	}

	// If slot is available or the node is reserved then accept else reject
	node, err := ps.getNode(peerID)
	if err != nil {
		// state inconsistency: tryOutgoing on an unknown node
		return err
	}

	node.state[setID] = ingoing
	if !isNoSlotOccupied {
		// This need to be added as incoming connection allocate slot.
		ps.sets[setID].numIn++
	}

	return nil
}
