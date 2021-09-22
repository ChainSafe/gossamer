package peerset

import (
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	ConnectedPeer    = "ConnectedPeer"
	NotConnectedPeer = "NotConnectedPeer"
	UnknownPeer      = "UnknownPeer"
)

type MembershipState int

const (
	// NotMember Node isn't part of that set.
	NotMember MembershipState = iota
	// In node is connected through an ingoing connection.
	In
	// Out node is connected through an outgoing connection.
	Out
	// NotConnected node is part of that set, but we are not connected to it.
	NotConnected
)

func isConnected(state MembershipState) bool {
	if state == In || state == Out {
		return true
	}
	return false
}

// Info is state of a single set.
type Info struct {
	// Number of slot occupying nodes for which the MembershipState is In.
	numIn uint32

	// Number of slot occupying nodes for which the MembershipState is In.
	numOut uint32

	// Maximum allowed number of slot occupying nodes for which the MembershipState is In.
	maxIn uint32

	// Maximum allowed number of slot occupying nodes for which the MembershipState is Out.
	maxOut uint32

	// List of node identities (discovered or not) that don't occupy slots.
	// Note for future readers: this module is purely dedicated to managing slots. If you are
	// considering adding more features, please consider doing so outside this module rather
	// than inside.
	noSlotNodes map[peer.ID]interface{}
}

type Node struct {
	/// List of sets the node belongs to.
	/// Always has a fixed size equal to the one of PeersState sets. The various possible sets
	/// are indices into this sets.
	// TODO check why we are using slice for sets, how multiple sets need to be managed?

	sets []MembershipState
	// When we were last connected to the node, or if we were never connected when we
	// discovered it.
	lastConnected []time.Time

	// Reputation value of the node, between int32 MIN (we hate that node) and
	// int32 MAX (we love that node).
	reputation int32
}

// newNode method to create a node with 0 reputation at starting.
func newNode(len int) *Node {
	now := time.Now()
	sets := make([]MembershipState, len)
	lastConnected := make([]time.Time, len)
	for i := 0; i < len; i++ {
		sets[i] = NotMember
		lastConnected[i] = now
	}

	return &Node{
		sets:          sets,
		lastConnected: lastConnected,
		reputation:    0,
	}
}

func (n *Node) getReputation() int32 {
	return n.reputation
}

func (n *Node) addReputation(modifier int32) int32 {
	n.reputation = addInt32(n.reputation, modifier)
	return n.reputation
}

// addInt32 handles overflow and underflow condition while adding two int32 values.
func addInt32(left, right int32) int32 {
	if right > 0 {
		if left > MAX-right {
			return MAX
		}
	} else {
		if left < MIN-right {
			return MIN
		}
	}
	return left + right
}

// subInt32 handles underflow condition while subtracting two int32 values.
func subInt32(left, right int32) int32 {
	if right < 0 {
		if left > MAX+right {
			return MAX
		}
	} else {
		if left < MIN+right {
			return MIN
		}
	}
	return left - right
}

func (n *Node) setReputation(modifier int32) {
	n.reputation = modifier
}

type PeersState struct {
	// List of nodes that we know about.
	//
	// This list should really be ordered by decreasing reputation, so that we can
	// easily select the best node to connect to. As a first draft, however, we don't
	// sort, to make the logic easier.
	nodes map[peer.ID]*Node
	// Configuration of each set. The size of this Info is never modified.
	// TODO Why we have slice for Info, How we are managing multiple Info?
	// Since, single Info can also manage the flow.
	sets []Info
}

func (ps *PeersState) getNode(p peer.ID) *Node {
	if n, ok := ps.nodes[p]; ok {
		return n
	}

	return nil
}

func (ps *PeersState) setNode(p peer.ID, n *Node) {
	ps.nodes[p] = n
}

// NewPeerState builds a new empty PeersState
func NewPeerState(sets []*SetConfig) (*PeersState, error) {
	infoSet := make([]Info, 0, len(sets))
	for _, cfg := range sets {
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
		nodes: make(map[peer.ID]*Node, 0),
		sets:  infoSet,
	}

	return peerState, nil
}

func (ps *PeersState) getSetLength() int {
	return len(ps.sets)
}

// peer returns an object that grants access to the state of a peer in the context of a specific set.
func (ps *PeersState) peer(set int, peerId peer.ID) string {
	// This checks whether the peer status is connected, notConnected or disconnect mode.
	if node := ps.getNode(peerId); node != nil {
		switch ps.nodes[peerId].sets[set] {
		case In, Out:
			return ConnectedPeer
		case NotConnected:
			return NotConnectedPeer
		}
	}

	return UnknownPeer
}

// peers return the list of all the peers we know of.
func (ps *PeersState) peers() []peer.ID {
	var peerIDs []peer.ID
	for k := range ps.nodes {
		peerIDs = append(peerIDs, k)
	}
	return peerIDs
}

// connectedPeers returns the list of peers we are connected to in the context of a specific set.
func (ps *PeersState) connectedPeers(set int) []peer.ID {
	var peerIDs []peer.ID
	if len(ps.sets) < set {
		return nil
	}

	for k := range ps.nodes {
		state := ps.nodes[k].sets[set]
		if isConnected(state) {
			peerIDs = append(peerIDs, k)
		}
	}

	return peerIDs
}

// highestNotConnectedPeer returns the peer with the highest reputation and that we are not connected to.
func (ps *PeersState) highestNotConnectedPeer(set int) peer.ID {
	var maxRep = MIN
	var peerID peer.ID
	for id := range ps.nodes {
		node := ps.nodes[id]
		if node.sets[set] == NotConnected {
			if node.reputation >= maxRep {
				maxRep = node.reputation
				peerID = id
			}
		}
	}

	return peerID
}

func (ps *PeersState) hasFreeOutgoingSlot(set int) bool {
	return ps.sets[set].numOut < ps.sets[set].maxOut
}

func (ps *PeersState) hasFreeIncomingSlot(set int) bool {
	return ps.sets[set].numIn >= ps.sets[set].maxIn
}

// addNoSlotNode Adds a node to the list of nodes that don't occupy slots.
// has no effect if the node was already in the group.
func (ps *PeersState) addNoSlotNode(idx int, peerId peer.ID) {
	if _, ok := ps.sets[idx].noSlotNodes[peerId]; ok {
		return
	}

	//Insert peer
	ps.sets[idx].noSlotNodes[peerId] = nil
	node := ps.getNode(peerId)
	if node != nil {
		if node.sets[idx] == In {
			ps.sets[idx].numIn -= 1
		} else if node.sets[idx] == Out {
			ps.sets[idx].numOut -= 1
		}

		ps.nodes[peerId] = node
	}
}

func (ps *PeersState) removeNoSlotNode(idx int, peerId peer.ID) {
	if _, ok := ps.sets[idx].noSlotNodes[peerId]; ok {
		delete(ps.sets[idx].noSlotNodes, peerId)
	} else {
		// TODO: add log
		return
	}

	if _, ok := ps.nodes[peerId]; ok {
		if ps.nodes[peerId].sets[idx] == In {
			ps.sets[idx].numIn += 1
		} else if ps.nodes[peerId].sets[idx] == Out {
			ps.sets[idx].numOut += 1
		}
	}
}

// Implement the methods for Unknown,Connected and NotConnected peers
// create a method
// disconnect method updates the node status to NotConnected state it should be called only when the node is in connected state.
func (ps *PeersState) disconnect(idx int, peerId peer.ID) error {
	// check for if it's isNoSlotOccupy is true or false.
	if node := ps.getNode(peerId); node != nil {
		info := ps.sets[idx]
		if _, ok := info.noSlotNodes[peerId]; !ok {
			if node.sets[idx] == In {
				if info.numIn > 0 {
					info.numIn -= 1
				}
			} else if node.sets[idx] == Out {
				if info.numOut > 0 {
					info.numOut -= 1
				}
			} else if node.sets[idx] == NotMember || node.sets[idx] == NotConnected {
				return errors.New("state inconsistency: disconnecting disconnected node")
			}
		}
		//Make node state to disconnected node state
		node.sets[idx] = NotConnected
		node.lastConnected[idx] = time.Now()

		ps.sets[idx] = info
		ps.setNode(peerId, node)
	} else {
		return errors.New("State inconsistency: disconnecting disconnected node")
	}
	return nil
}

// discover: This take input for set id and create a node and insert in the list.
// the initial reputation of the peer will be 0 and in NotMember state.
func (ps *PeersState) discover(set int, peerId peer.ID) {
	numSet := len(ps.sets)
	if _, ok := ps.nodes[peerId]; !ok {
		n := newNode(numSet)
		n.sets[set] = NotConnected
		ps.nodes[peerId] = n
	}
}

// TODO Implement forgetPeer, lastConnectedAndDiscovered
func (ps *PeersState) lastConnectedAndDiscovered(set int, peerId peer.ID) time.Time {
	node := ps.getNode(peerId)
	if node != nil && node.sets[set] == NotConnected {
		return node.lastConnected[set]
	}

	// else return now time
	return time.Now()
}

// Removes the peer from the list of members of the set.
func (ps *PeersState) forgetPeer(set int, peerId peer.ID) {

	node := ps.getNode(peerId)
	if node == nil {
		return
	}

	if node.sets[set] != NotMember {
		node.sets[set] = NotMember
	}

	if node.getReputation() == 0 {
		// Remove the peer from ps nodes entirely if it isn't a member of any set.
		remove := true
		for _, state := range node.sets {
			if state != NotMember {
				remove = false
				break
			}
		}

		if remove {
			delete(ps.nodes, peerId)
		}
	}
}

// Tries to set the peer as connected as an outgoing connection.
// if there are enough slots available, switches the node to Connected and returns nil error. If
// the slots are full, the node stays "not connected" and we return error.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryOutgoing(setId int, peerId peer.ID) error {
	var isNoSlotOccupied bool
	if _, ok := ps.sets[setId].noSlotNodes[peerId]; ok {
		isNoSlotOccupied = true
	}

	if !ps.hasFreeOutgoingSlot(setId) && !isNoSlotOccupied {
		return errors.New("not enough outgoing slots")
	}

	if _, ok := ps.nodes[peerId]; ok {
		ps.nodes[peerId].sets[setId] = Out
		if !isNoSlotOccupied {
			ps.sets[setId].numOut += 1
		}
	} else {
		return errors.New("state inconsistency: tryOutgoing on an unknown node")
	}

	return nil
}

// tryAcceptIncoming tries to accept the peer as an incoming connection.
// if there are enough slots available, switches the node to Connected and returns nil. If
// the slots are full, the node stays "not connected" and we return Err.
// non slot occupying nodes don't count towards the number of slots.
func (ps *PeersState) tryAcceptIncoming(setId int, peerId peer.ID) error {

	var isNoSlotOccupied bool
	if _, ok := ps.sets[setId].noSlotNodes[peerId]; ok {
		isNoSlotOccupied = true
	}

	// Note that it is possible for num_in to be strictly superior to the max, in case we were
	// connected to reserved node then marked them as not reserved.
	// If slot is not available and the node is not a reserved node then error
	if ps.hasFreeIncomingSlot(setId) && !isNoSlotOccupied {
		return errors.New("not enough incoming slots")
	}

	// If slot is avalible or the node is reserved then accept else reject
	if _, ok := ps.nodes[peerId]; ok {
		ps.nodes[peerId].sets[setId] = In
		if !isNoSlotOccupied {
			// This need to be added as incoming connection allocate slot.
			ps.sets[setId].numIn += 1

		}
	} else {
		return errors.New("state inconsistency: tryOutgoing on an unknown node")
	}

	return nil
}
