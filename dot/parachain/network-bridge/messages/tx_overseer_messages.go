// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p/core/peer"
)

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
	// TODO: make this versioned
	ValidationProtocolMessage validationprotocol.ValidationProtocol
}

type PeerSetType int

const (
	ValidationProtocol PeerSetType = iota
	CollationProtocol
)

// ConnectToValidators is a subsystem message to network bridge for connecting to
// peers who represent the given `validator_ids`.
//
// Also ask the network to stay connected to these peers at least
// until a new request is issued.
//
// Because it overrides the previous request, it must be ensured
// that `validator_ids` include all peers the subsystems
// are interested in (per `PeerSet`).
//
// A caller can learn about validator connections by listening to the
// `PeerConnected` events from the network bridge.
type ConnectToValidators struct {
	// IDs of the validators to connect to.
	ValidatorIDs []parachaintypes.AuthorityDiscoveryID
	// The underlying protocol to use for this request.
	PeerSet PeerSetType
	// Sends back the number of `AuthorityDiscoveryId`s which
	// authority discovery has Failed to resolve.
	Failed chan<- uint
}
