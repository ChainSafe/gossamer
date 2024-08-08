package events

import (
	collationprotocol "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerConnected struct {
	PeerID                peer.ID
	OverservedRole        common.NetworkRole
	ProtocolVersion       uint32
	AuthorityDiscoveryIDs *[]parachaintypes.AuthorityDiscoveryID
}

type PeerDisconnected struct {
	PeerID peer.ID
}

// // Inform the distribution subsystems about the new
// // gossip network topology formed.
// //
// // The only reason to have this here, is the availability of the
// // authority discovery service, otherwise, the `GossipSupport`
// // subsystem would make more sense.
// type NewGossipTopology struct {
// 	// The session info this gossip topology is concerned with.
// 	Session parachaintypes.SessionIndex //nolint
// 	// Our validator index in the session, if any.
// 	LocalIndex *parachaintypes.ValidatorIndex //nolint
// 	//  The canonical shuffling of validators for the session.
// 	CanonicalShuffling []CanonicalShuffling //nolint
// 	// The reverse mapping of `canonical_shuffling`: from validator index
// 	// to the index in `canonical_shuffling`
// 	ShuffledIndices uint8 //nolint
// }

type CanonicalShuffling struct { //nolint
	AuthorityDiscoveryID parachaintypes.AuthorityDiscoveryID
	ValidatorIndex       parachaintypes.ValidatorIndex
}

type NewGossipTopology struct {
	Session    parachaintypes.SessionIndex
	Topotogy   SessionGridTopology
	LocalIndex *parachaintypes.ValidatorIndex
}

type PeerViewChange struct {
	PeerID peer.ID
	View   View
}

// View is a succinct representation of a peer's view. This consists of a bounded amount of chain heads
// and the highest known finalized block number.
//
// Up to `N` (5?) chain heads.
type View struct {
	// a bounded amount of chain heads
	heads []common.Hash //nolint
	// the highest known finalized number
	finalizedNumber uint32 //nolint
}

type OurViewChange struct {
	View View
}

type PeerMessage[Message collationprotocol.CollationProtocol | validationprotocol.ValidationProtocol] struct {
	PeerID   peer.ID
	Messaage Message
}

type UpdatedAuthorityIds struct {
	PeerID                peer.ID
	AuthorityDiscoveryIDs []parachaintypes.AuthorityDiscoveryID
}

// pub enum NetworkBridgeEvent<M> {
// 	/// A peer has connected.
// 	PeerConnected(PeerId, ObservedRole, ProtocolVersion, Option<HashSet<AuthorityDiscoveryId>>),

// 	/// A peer has disconnected.
// 	PeerDisconnected(PeerId),

// 	/// Our neighbors in the new gossip topology for the session.
// 	/// We're not necessarily connected to all of them.
// 	///
// 	/// This message is issued only on the validation peer set.
// 	///
// 	/// Note, that the distribution subsystems need to handle the last
// 	/// view update of the newly added gossip peers manually.
// 	NewGossipTopology(NewGossipTopology),

// 	/// Peer has sent a message.
// 	PeerMessage(PeerId, M),

// 	/// Peer's `View` has changed.
// 	PeerViewChange(PeerId, View),

// 	/// Our view has changed.
// 	OurViewChange(OurView),

// 	/// The authority discovery session key has been rotated.
// 	UpdatedAuthorityIds(PeerId, HashSet<AuthorityDiscoveryId>),
// }
