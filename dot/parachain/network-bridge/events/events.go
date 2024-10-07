package events

import (
	collationprotocol "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

// NOTE: Not adding PeerMessage here to make this a bit simpler.
// TODO: Add Event in overseer
// We will use it someday, since it helps us group all the network events together. For now, let's just
// use them separately.
type Event[Message collationprotocol.CollationProtocol | validationprotocol.ValidationProtocol] struct {
	Inner any
}

type EventValues[Message collationprotocol.CollationProtocol | validationprotocol.ValidationProtocol] interface {
	PeerConnected | PeerDisconnected | NewGossipTopology | PeerViewChange | OurViewChange |
		UpdatedAuthorityIDs | PeerMessage[Message]
}

type PeerConnected struct {
	PeerID                peer.ID
	OverservedRole        common.NetworkRole
	ProtocolVersion       uint32
	AuthorityDiscoveryIDs *[]parachaintypes.AuthorityDiscoveryID
}

type PeerDisconnected struct {
	PeerID peer.ID
}

type CanonicalShuffling struct {
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
	Heads []common.Hash
	// the highest known finalized number
	FinalizedNumber uint32
}

type OurViewChange struct {
	View View
}

type PeerMessage[Message collationprotocol.CollationProtocol | validationprotocol.ValidationProtocol] struct {
	PeerID  peer.ID
	Message Message
}

type UpdatedAuthorityIDs struct {
	PeerID                peer.ID
	AuthorityDiscoveryIDs []parachaintypes.AuthorityDiscoveryID
}
