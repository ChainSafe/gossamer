package grid

import (
	"fmt"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"math"
)

// TopologyPeerInfo is an information about the peer in the gossip topology
type TopologyPeerInfo struct {
	// Peers is a list pf peers this peer knows about
	Peers peer.IDSlice
	/// ValidatorIndex is the index of the validator in the discovery keys of the corresponding
	/// `SessionInfo`. This can extend _beyond_ the set of active parachain validators.
	ValidatorIndex parachaintypes.ValidatorIndex
	/// DiscoveryID is the authority discovery public key of the validator in the corresponding
	/// `SessionInfo`.
	DiscoveryID types.AuthorityID
}

// SessionGridTopology is topology representation for session
type SessionGridTopology struct {
	// Peers List of all peers ids in session. Represented as "hashset" for fast lookup.
	Peers map[peer.ID]struct{}
	/// canonicalShuffling is the canonical shuffling of validators for the session.
	CanonicalShuffling []TopologyPeerInfo
	/// shuffledIndices is an array mapping validator indices to their indices in the
	/// shuffling itself. This has the same size as the number of validators
	/// in the session.
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

func (gt *SessionGridTopology) ComputeGridNeighborsFor(vi parachaintypes.ValidatorIndex) (*GridNeighbors, error) {
	if len(gt.ShuffledIndices) != len(gt.CanonicalShuffling) {
		return nil, fmt.Errorf("grid topology malformed: "+
			"shuffledIndices length %d is not equal to canonicalShuffling length %d",
			len(gt.ShuffledIndices),
			len(gt.CanonicalShuffling),
		)
	}

	shuffledIndex := gt.ShuffledIndices[vi]

	neighbors, err := CalculateMatrixNeighbors(shuffledIndex, uint(len(gt.ShuffledIndices)))
	if err != nil {
		return nil, err
	}

	gridSubset := NewEmptyGridNeighbors()

	for _, rN := range neighbors.RowNeighbors {
		n := &gt.CanonicalShuffling[rN]
		gridSubset.ValidatorIndicesRow[n.ValidatorIndex] = struct{}{}
		for _, p := range n.Peers {
			gridSubset.PeersRow[p] = struct{}{}
		}
	}

	for _, cN := range neighbors.ColumnNeighbors {
		n := &gt.CanonicalShuffling[cN]
		gridSubset.ValidatorIndicesCol[n.ValidatorIndex] = struct{}{}
		for _, p := range n.Peers {
			gridSubset.PeersCol[p] = struct{}{}
		}
	}

	return gridSubset, nil
}

type GridNeighbors struct {
	PeersRow map[peer.ID]struct{}
	PeersCol map[peer.ID]struct{}

	ValidatorIndicesRow map[parachaintypes.ValidatorIndex]struct{}
	ValidatorIndicesCol map[parachaintypes.ValidatorIndex]struct{}
}

func NewEmptyGridNeighbors() *GridNeighbors {
	return &GridNeighbors{
		PeersRow:            make(map[peer.ID]struct{}),
		PeersCol:            make(map[peer.ID]struct{}),
		ValidatorIndicesRow: make(map[parachaintypes.ValidatorIndex]struct{}),
		ValidatorIndicesCol: make(map[parachaintypes.ValidatorIndex]struct{}),
	}
}

func (gn *GridNeighbors) RequiredRoutingByIndex(origin parachaintypes.ValidatorIndex, local bool) RequiredRouting {
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

// RequiredRoutingByPeer Given the originator of a message as a peer index, indicates the part of the topology
// we're meant to send the message to.
// TODO: This method is actually not used in the codebase.
func (gn *GridNeighbors) RequiredRoutingByPeer(origin peer.ID, local bool) RequiredRouting {
	if local {
		return RequiredRoutingGridXY
	}
	_, x := gn.PeersRow[origin]
	_, y := gn.PeersCol[origin]

	if x && y {
		// If all works correctly origin peer can't be in both rows and columns. But we leave it anyways.
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
func (gn *GridNeighbors) ShouldRouteToPeer(routing RequiredRouting, peer peer.ID) bool {
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
		return false
	}
}

// PeersDiff returns a differents between two GridNeighbors
func (gn *GridNeighbors) PeersDiff(other *GridNeighbors) []peer.ID {
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

func (gn *GridNeighbors) Len() int {
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

// MatrixNeighbors holds the row and column neighbors of a given index in a matrix.
type MatrixNeighbors struct {
	RowNeighbors    []uint
	ColumnNeighbors []uint
}

// CalculateMatrixNeighbors computes the row and column neighbors of valIndex in a matrix of given length.
// e.g. for size 11 the matrix would be
//
// 0  1  2
// 3  4  5
// 6  7  8
// 9 10
//
// and for index 10, the neighbors would be 1, 4, 7, 9
func CalculateMatrixNeighbors(valIndex, length uint) (*MatrixNeighbors, error) {
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

	rowNeighbors := make([]uint, 0)
	for i := rowStart; i < rowEnd; i++ {
		if i != valIndex {
			rowNeighbors = append(rowNeighbors, i)
		}
	}

	columnNeighbors := make([]uint, 0)
	for i := ourColumn; i < length; i += sqrt {
		if i != valIndex {
			columnNeighbors = append(columnNeighbors, i)
		}
	}

	return &MatrixNeighbors{
		RowNeighbors:    rowNeighbors,
		ColumnNeighbors: columnNeighbors,
	}, nil
}
