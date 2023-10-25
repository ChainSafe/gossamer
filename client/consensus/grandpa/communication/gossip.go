package communication

import (
	"github.com/ChainSafe/gossamer/client/peerset"
	libp2p "github.com/libp2p/go-libp2p/core"
)

// / V1 neighbor packet. Neighbor packets are sent from nodes to their peers
// / and are not repropagated. These contain information about the node's state.
type neighborPacket[N any] struct {
	/// The round the node is currently at.
	round Round
	/// The set ID the node is currently at.
	setID SetID
	/// The highest finalizing commit observed.
	commitFinalizedHeight N
}

// / Report specifying a reputation change for a given peer.
type peerReport struct {
	who         libp2p.PeerID
	costBenefit peerset.ReputationChange
}
