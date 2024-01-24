package gossip

import (
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	libp2p "github.com/libp2p/go-libp2p/core"
	"golang.org/x/exp/constraints"
)

// / Validates consensus messages.
type Validator[H constraints.Ordered] interface {
	/// New peer is connected.
	NewPeer(context ValidatorContext[H], who libp2p.PeerID, role role.ObservedRole)
	/// New connection is dropped.
	PeerDisconnected(context ValidatorContext[H], who libp2p.PeerID)
	/// Validate consensus message.
	Validate(context ValidatorContext[H], sender libp2p.PeerID, data []byte) ValidationResult
	/// Produce a closure for validating messages on a given topic.
	MessageExpired() func(topic H, message []byte) bool
	/// Produce a closure for filtering egress messages.
	MessageAllowed() func(who libp2p.PeerID, intent MessageIntent, topic H, data []byte) bool
}

// / Validation context. Allows reacting to incoming messages by sending out further messages.
type ValidatorContext[H constraints.Ordered] interface {
	// 	/// Broadcast all messages with given topic to peers that do not have it yet.
	BroadcastTopic(topic H, force bool)
	// /// Broadcast a message to all peers that have not received it previously.
	BroadcastMessage(topic H, message []byte, force bool)
	// /// Send addressed message to a peer.
	SendMessage(who libp2p.PeerID, message []byte)
	// /// Send all messages with given topic to a peer.
	SendTopic(who libp2p.PeerID, topic H, force bool)
}

// / Requested broadcast.
type MessageIntentBroadcast struct{}

// / Requested broadcast to all peers.
type MessageIntentForcedBroadcast struct{}

// / Periodic rebroadcast of all messages to all peers.
type MessageIntentPeriodicReboradcast struct{}

// / The reason for sending out the message.
type MessageIntents interface {
	MessageIntentBroadcast | MessageIntentForcedBroadcast | MessageIntentPeriodicReboradcast
}
type MessageIntent any

// / Message should be stored and propagated under given topic.
type ValidationResultProcessAndKeep[H constraints.Ordered] struct {
	Hash H
}

// / Message should be processed, but not propagated.
type ValidationResultProcessAndDiscard[H constraints.Ordered] struct {
	Hash H
}

// / Message should be ignored.
type ValidationResultDiscard struct{}

// / Message validation result.
type ValidationResults[H constraints.Ordered] interface {
	ValidationResultProcessAndKeep[H] | ValidationResultProcessAndDiscard[H] | ValidationResultDiscard
}
type ValidationResult any
