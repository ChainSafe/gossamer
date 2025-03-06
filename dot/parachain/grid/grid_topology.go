package grid

import (
	"fmt"
	"math"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TopologyPeerInfo is an information about the peer in the gossip topology
type TopologyPeerInfo struct {
	// Peers is a list of peers this peer knows about
	Peers peer.IDSlice
	// ValidatorIndex is the index of the validator in the discovery keys of the corresponding `SessionInfo`.
	// This can extend _beyond_ the set of active parachain validators.
	ValidatorIndex parachaintypes.ValidatorIndex
	// DiscoveryID is the authority discovery public key of the validator in the corresponding `SessionInfo`.
	DiscoveryID types.AuthorityID
}

// SessionGridTopology is topology representation for session
type SessionGridTopology struct {
	// Peers List of all peers ids in session. Represented as "hashset" for fast lookup.
	Peers map[peer.ID]struct{}
	// canonicalShuffling is the canonical shuffling of validators for the session.
	CanonicalShuffling []TopologyPeerInfo
	// shuffledIndices is an array mapping validator indices to their indices in the  shuffling itself.
	// This has the same size as the number of validators in the session.
	ShuffledIndices []uint
}

// NewSessionGridTopology creates a new SessionGridTopology
// Peers populated with peer ids from canonicalShuffling argument.
// canonicalShuffling set of authorities for this session. within one session it can have only one key.
func NewSessionGridTopology(shuffledIndices []uint, canonicalShuffling []TopologyPeerInfo) *SessionGridTopology {
	peers := make(map[peer.ID]struct{})
	for _, peerInfo := range canonicalShuffling {
		for _, p := range peerInfo.Peers {
			peers[p] = struct{}{}
		}
	}
	return &SessionGridTopology{
		Peers:              peers,
		CanonicalShuffling: canonicalShuffling,
		ShuffledIndices:    shuffledIndices,
	}
}

// UpdateAuthoritiesIDs Updates the known peer ids in SessionGridTopology for the passed authorities ids.
// Between sessions validators can update their authorityID because of key rotation, peerID might be changed as well.
// hence there could be multiple AuthorityID associated with a peerID.
// Updates Peers hashset with new peer id if the peer is in the grid topology.
// Returns true if the peer is in the grid topology.
func (gt *SessionGridTopology) UpdateAuthoritiesIDs(peer peer.ID, discoveryIDs map[types.AuthorityID]struct{}) bool {
	updated := false
	if _, ok := gt.Peers[peer]; !ok {
		for i := range gt.CanonicalShuffling {
			p := &gt.CanonicalShuffling[i]
			if _, ok := discoveryIDs[p.DiscoveryID]; ok {
				gt.Peers[peer] = struct{}{}
				p.Peers = append(p.Peers, peer)
				updated = true
			}
		}
	}
	return updated
}

// IsValidator returns true is given peerID is in the session
func (gt *SessionGridTopology) IsValidator(peer peer.ID) bool {
	_, ok := gt.Peers[peer]
	return ok
}

func (gt *SessionGridTopology) ComputeGridNeighboursFor(vi parachaintypes.ValidatorIndex) (*GridNeighbours, error) {
	if len(gt.ShuffledIndices) != len(gt.CanonicalShuffling) {
		return nil, fmt.Errorf("grid topology malformed: "+
			"shuffledIndices length %d is not equal to canonicalShuffling length %d",
			len(gt.ShuffledIndices),
			len(gt.CanonicalShuffling),
		)
	}

	shuffledIndex := gt.ShuffledIndices[vi]

	neighbours, err := calculateMatrixNeighbours(shuffledIndex, uint(len(gt.ShuffledIndices)))
	if err != nil {
		return nil, err
	}

	gridSubset := NewEmptyGridNeighbours()

	for _, rN := range neighbours.RowNeighbours {
		n := &gt.CanonicalShuffling[rN]
		gridSubset.ValidatorIndicesRow[n.ValidatorIndex] = struct{}{}
		for _, p := range n.Peers {
			gridSubset.PeersRow[p] = struct{}{}
		}
	}

	for _, cN := range neighbours.ColumnNeighbours {
		n := &gt.CanonicalShuffling[cN]
		gridSubset.ValidatorIndicesCol[n.ValidatorIndex] = struct{}{}
		for _, p := range n.Peers {
			gridSubset.PeersCol[p] = struct{}{}
		}
	}

	return gridSubset, nil
}

type GridNeighbours struct {
	PeersRow map[peer.ID]struct{}
	PeersCol map[peer.ID]struct{}

	ValidatorIndicesRow map[parachaintypes.ValidatorIndex]struct{}
	ValidatorIndicesCol map[parachaintypes.ValidatorIndex]struct{}
}

func NewEmptyGridNeighbours() *GridNeighbours {
	return &GridNeighbours{
		PeersRow:            make(map[peer.ID]struct{}),
		PeersCol:            make(map[peer.ID]struct{}),
		ValidatorIndicesRow: make(map[parachaintypes.ValidatorIndex]struct{}),
		ValidatorIndicesCol: make(map[parachaintypes.ValidatorIndex]struct{}),
	}
}

// RequiredRoutingByIndex Given the originator of a message as a validator index, indicates the part of the topology
func (gn *GridNeighbours) RequiredRoutingByIndex(origin parachaintypes.ValidatorIndex, local bool) RequiredRouting {
	if local {
		return RequiredRoutingGridXY
	}
	_, x := gn.ValidatorIndicesRow[origin]
	_, y := gn.ValidatorIndicesCol[origin]

	if x && y {
		// If all works correctly origin peer can't be in both rows and columns. But we leave it anyways
		return RequiredRoutingGridXY
	}
	if !(x || y) {
		return RequiredRoutingNone
	}
	if x && !y {
		return RequiredRoutingGridY
	}
	return RequiredRoutingGridX
}

// ShouldRouteToPeer indicates does peer should receive a message based on GridTopology and Routing strategy
func (gn *GridNeighbours) ShouldRouteToPeer(routing RequiredRouting, peer peer.ID) bool {
	switch routing {
	case RequiredRoutingAll:
		return true
	case RequiredRoutingNone, PendingTopology:
		return false
	case RequiredRoutingGridXY:
		_, x := gn.PeersRow[peer]
		_, y := gn.PeersCol[peer]
		return x || y
	case RequiredRoutingGridX:
		_, x := gn.PeersRow[peer]
		return x
	case RequiredRoutingGridY:
		_, y := gn.PeersCol[peer]
		return y
	default:
		// No way we get here
		panic(fmt.Sprintf("unknown routing strategy for grid topology: %d", routing))
	}
}

// PeersDiff returns a differents between two GridNeighbours
func (gn *GridNeighbours) PeersDiff(other *GridNeighbours) []peer.ID {
	diff := make([]peer.ID, 0)
	for p := range gn.PeersRow {
		_, inRows := other.PeersRow[p]
		_, inCols := other.PeersCol[p]
		if !inRows && !inCols {
			diff = append(diff, p)
		}
	}
	return diff
}

func (gn *GridNeighbours) Len() int {
	return len(gn.PeersRow) + len(gn.PeersCol)
}

type RequiredRouting uint

const (
	PendingTopology    RequiredRouting = iota
	RequiredRoutingAll                 // All peers in the grid
	RequiredRoutingGridXY
	RequiredRoutingGridX
	RequiredRoutingGridY
	RequiredRoutingNone
)

// MatrixNeighbours holds the row and column neighbours of a given index in a matrix.
type MatrixNeighbours struct {
	RowNeighbours    []uint
	ColumnNeighbours []uint
}

// calculateMatrixNeighbours computes the row and column neighbours of valIndex in a matrix of given length.
// e.g. for size 11 the matrix would be
//
// 0  1  2
// 3  4  5
// 6  7  8
// 9 10
//
// and for index 10, the neighbours would be 1, 4, 7, 9
func calculateMatrixNeighbours(valIndex, length uint) (*MatrixNeighbours, error) {
	if valIndex >= length {
		return nil, fmt.Errorf("grid topology malformed: valIndex %d is greater than length %d",
			valIndex,
			length,
		)
	}

	sqrt := uint(math.Sqrt(float64(length)))
	ourRow := valIndex / sqrt
	ourColumn := valIndex % sqrt

	rowStart := ourRow * sqrt
	rowEnd := uint(math.Min(float64(rowStart+sqrt), float64(length)))

	rowNeighbours := make([]uint, 0)
	for i := rowStart; i < rowEnd; i++ {
		if i != valIndex {
			rowNeighbours = append(rowNeighbours, i)
		}
	}

	columnNeighbours := make([]uint, 0)
	for i := ourColumn; i < length; i += sqrt {
		if i != valIndex {
			columnNeighbours = append(columnNeighbours, i)
		}
	}

	return &MatrixNeighbours{
		RowNeighbours:    rowNeighbours,
		ColumnNeighbours: columnNeighbours,
	}, nil
}

// SessionGridTopologyEntry is a single entry in the SessionGridTopologyStorage. Contain SessionTopology itself,
// calculated neighbours for this topology according to the validator index.
type SessionGridTopologyEntry struct {
	Topology        *SessionGridTopology
	LocalNeighbours *GridNeighbours
	LocalIndex      parachaintypes.ValidatorIndex
	SessionIndex    parachaintypes.SessionIndex
}

func (s *SessionGridTopologyEntry) PeersToRoute(routing RequiredRouting) []peer.ID {
	switch routing {
	case RequiredRoutingAll:
		peers := make([]peer.ID, 0)
		for p := range s.Topology.Peers {
			peers = append(peers, p)
		}
		return peers
	case RequiredRoutingGridY:
		peers := make([]peer.ID, 0)
		for p := range s.LocalNeighbours.PeersCol {
			peers = append(peers, p)
		}
		return peers
	case RequiredRoutingGridX:
		peers := make([]peer.ID, 0)
		for p := range s.LocalNeighbours.PeersRow {
			peers = append(peers, p)
		}
		return peers
	case RequiredRoutingGridXY:
		peers := make([]peer.ID, 0)
		for p := range s.LocalNeighbours.PeersCol {
			peers = append(peers, p)
		}
		for p := range s.LocalNeighbours.PeersRow {
			peers = append(peers, p)
		}
		return peers
	}
	return make([]peer.ID, 0)
}

func (s *SessionGridTopologyEntry) UpdateAuthoritiesIDs(
	peer peer.ID,
	discoveryIDs map[types.AuthorityID]struct{}) (bool, error) {
	if s.Topology.UpdateAuthoritiesIDs(peer, discoveryIDs) {
		// If authorities update, recompile neighbours
		new_neighbours, err := s.Topology.ComputeGridNeighboursFor(s.LocalIndex)
		if err != nil {
			return false, err
		}
		s.LocalNeighbours = new_neighbours
		return true, nil
	}
	return false, nil
}

type SessionGridTopologyStorage struct {
	CurrentTopology *SessionGridTopologyEntry
	PrevTopology    *SessionGridTopologyEntry
}

func (s *SessionGridTopologyStorage) GetTopologyBySessionIndex(
	idx parachaintypes.SessionIndex) *SessionGridTopologyEntry {
	if s.CurrentTopology.SessionIndex == idx {
		return s.CurrentTopology
	}
	if s.PrevTopology.SessionIndex == idx {
		return s.PrevTopology
	}
	return nil
}

func (s *SessionGridTopologyStorage) GetTopologyOrFallback(
	idx parachaintypes.SessionIndex) *SessionGridTopologyEntry {
	toplogy := s.GetTopologyBySessionIndex(idx)
	if toplogy != nil {
		return toplogy
	} else {
		return s.CurrentTopology
	}
}

func (s *SessionGridTopologyStorage) UpdateCurrentTopology(idx parachaintypes.SessionIndex,
	topology *SessionGridTopology,
	localIndex parachaintypes.ValidatorIndex) error {
	localNeighbours, err := topology.ComputeGridNeighboursFor(localIndex)
	if err != nil {
		return err
	}
	s.PrevTopology = s.CurrentTopology
	s.CurrentTopology = &SessionGridTopologyEntry{
		Topology:        topology,
		LocalNeighbours: localNeighbours,
		LocalIndex:      localIndex,
		SessionIndex:    idx,
	}
	return nil
}
