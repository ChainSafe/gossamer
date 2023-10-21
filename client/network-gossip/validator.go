package gossip

import (
	"github.com/ChainSafe/gossamer/client/network/role"
	"github.com/ChainSafe/gossamer/primitives/runtime"
	"github.com/libp2p/go-libp2p/core"
)

// / Validates consensus messages.
type Validator[H runtime.Hash] interface {
	/// New peer is connected.
	NewPeer(context ValidatorContext[H], who core.PeerID, role role.ObservedRole)
	/// New connection is dropped.
	PeerDisconnected(context ValidatorContext[H], who core.PeerID)
	/// Validate consensus message.
	Validate(context ValidatorContext[H], sender core.PeerID, data []byte) ValidationResult
	/// Produce a closure for validating messages on a given topic.
	MessageExpired() func(topic H, message []byte) bool
	/// Produce a closure for filtering egress messages.
	MessageAllowed() func(who core.PeerID, intent MessageIntent, topic H, data []byte) bool
}

// / Validation context. Allows reacting to incoming messages by sending out further messages.
type ValidatorContext[H runtime.Hash] interface {
	// 	/// Broadcast all messages with given topic to peers that do not have it yet.
	BroadcastTopic(topic H, force bool)
	// /// Broadcast a message to all peers that have not received it previously.
	BroadcastMessage(topic H, message []byte, force bool)
	// /// Send addressed message to a peer.
	SendMessage(who core.PeerID, message []byte)
	// /// Send all messages with given topic to a peer.
	SendTopic(who core.PeerID, topic H, force bool)
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
type ValidationResultProcessAndKeep[H runtime.Hash] struct {
	Hash H
}

// / Message should be processed, but not propagated.
type ValidationResultProcessAndDiscard[H runtime.Hash] struct {
	Hash H
}

// / Message should be ignored.
type ValidationResultDiscard struct{}

// / Message validation result.
type ValidationResults[H runtime.Hash] interface {
	ValidationResultProcessAndKeep[H] | ValidationResultProcessAndDiscard[H] | ValidationResultDiscard
}
type ValidationResult any
