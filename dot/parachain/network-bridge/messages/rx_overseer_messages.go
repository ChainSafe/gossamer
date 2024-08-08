package messages

import (
	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Inform the distribution subsystems about the new
// gossip network topology formed.
//
// The only reason to have this here, is the availability of the
// authority discovery service, otherwise, the `GossipSupport`
// subsystem would make more sense.
type NewGossipTopology struct {
	// The session info this gossip topology is concerned with.
	Session parachaintypes.SessionIndex //nolint
	// Our validator index in the session, if any.
	LocalIndex *parachaintypes.ValidatorIndex //nolint
	//  The canonical shuffling of validators for the session.
	CanonicalShuffling []events.CanonicalShuffling //nolint
	// The reverse mapping of `canonical_shuffling`: from validator index
	// to the index in `canonical_shuffling`
	ShuffledIndices []uint8 //nolint
}

// UpdateAuthorityIDs is used to inform the distribution subsystems about `AuthorityDiscoveryId` key rotations.
type UpdateAuthorityIDs struct {
	// The `PeerId` of the peer that updated its `AuthorityDiscoveryId`s.
	PeerID peer.ID //nolint
	// The updated authority discovery keys of the peer.
	AuthorityDiscoveryIDs []parachaintypes.AuthorityDiscoveryID //nolint
}
