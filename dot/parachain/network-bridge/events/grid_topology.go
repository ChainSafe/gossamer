package events

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Topology representation for a session.
type SessionGridTopology struct {
	// An array mapping validator indices to their indices in the
	// shuffling itself. This has the same size as the number of validators
	// 	in the session.
	ShuffledIndices []uint8
	// The canonical shuffling of validators for the session.
	CanonicalShuffling []TopologyPeerInfo
	// The list of peer-ids in an efficient way to search.
	PeerIDs []peer.ID
}

// TopologyPeerInfo contains information about a peer in the gossip topology for a session.
type TopologyPeerInfo struct {
	// The validator's known peer IDs.
	PeerID []peer.ID
	// The index of the validator in the discovery keys of the corresponding
	// `SessionInfo`. This can extend _beyond_ the set of active parachain validators.
	ValidatorIndex parachaintypes.ValidatorIndex
	// The authority discovery public key of the validator in the corresponding
	// `SessionInfo`.
	DiscoveryID parachaintypes.AuthorityDiscoveryID
}
