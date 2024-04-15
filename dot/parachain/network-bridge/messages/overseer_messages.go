package messages

import (
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p/core/peer"
)

// NOTE: This is not same as corresponding rust structure
// TODO: If need be, add ability to report multiple peers in batches
type ReportPeer struct {
	PeerID           peer.ID
	ReputationChange peerset.ReputationChange
}

type DisconnectPeer struct {
	Peer    peer.ID
	PeerSet PeerSetType
}

type SendCollationMessage struct {
	To []peer.ID
	// TODO: make this versioned
	CollationProtocolMessage collatorprotocolmessages.CollationProtocol
}

type SendValidationMessage struct {
	To []peer.ID
	// TODO: move validation protocol to a new package to be able to be used here
	// validationProtocolMessage
}

type PeerSetType int

const (
	ValidationProtocol PeerSetType = iota
	CollationProtocol
)

type ConnectToValidators struct {
	// IDs of the validators to connect to.
	ValidatorIDs []parachaintypes.AuthorityDiscoveryID
	// The underlying protocol to use for this request.
	PeerSet PeerSetType
	// Sends back the number of `AuthorityDiscoveryId`s which
	// authority discovery has Failed to resolve.
	Failed chan<- uint
}
