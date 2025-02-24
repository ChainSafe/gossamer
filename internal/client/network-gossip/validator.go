package gossip

import (
	peerid "github.com/ChainSafe/gossamer/internal/client/network-types/peer-id"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"golang.org/x/exp/constraints"
)

// Validates consensus messages.
type Validator[H constraints.Ordered] interface {
	// New peer is connected.
	NewPeer(context ValidatorContext[H], who peerid.PeerID, role role.ObservedRole)
	// New connection is dropped.
	PeerDisconnected(context ValidatorContext[H], who peerid.PeerID)
	// Validate consensus message.
	Validate(context ValidatorContext[H], sender peerid.PeerID, data []byte) ValidationResult
	// Produce a closure for validating messages on a given topic.
	MessageExpired() func(topic H, message []byte) bool
	// Produce a closure for filtering egress messages.
	MessageAllowed() func(who peerid.PeerID, intent MessageIntent, topic H, data []byte) bool
}

// Validation context. Allows reacting to incoming messages by sending out further messages.
type ValidatorContext[H constraints.Ordered] interface {
	// 	Broadcast all messages with given topic to peers that do not have it yet.
	BroadcastTopic(topic H, force bool)
	// Broadcast a message to all peers that have not received it previously.
	BroadcastMessage(topic H, message []byte, force bool)
	// Send addressed message to a peer.
	SendMessage(who peerid.PeerID, message []byte)
	// Send all messages with given topic to a peer.
	SendTopic(who peerid.PeerID, topic H, force bool)
}

// The reason for sending out the message.
type MessageIntent uint

const (
	// Requested broadcast.
	MessageIntentBroadcast = iota + 1
	// Requested broadcast to all peers.
	MessageIntentForcedBroadcast
	// Periodic rebroadcast of all messages to all peers.
	MessageIntentPeriodicReboradcast
)

// Message should be stored and propagated under given topic.
type ValidationResultProcessAndKeep[H constraints.Ordered] struct {
	Hash H
}

func (ValidationResultProcessAndKeep[H]) isValidationResult() {}

// Message should be processed, but not propagated.
type ValidationResultProcessAndDiscard[H constraints.Ordered] struct {
	Hash H
}

func (ValidationResultProcessAndDiscard[H]) isValidationResult() {}

// Message should be ignored.
type ValidationResultDiscard struct{}

func (ValidationResultDiscard) isValidationResult() {}

// Message validation result.
type ValidationResult interface {
	isValidationResult()
}
