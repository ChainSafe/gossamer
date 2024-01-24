package gossip

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/event"
	"github.com/ChainSafe/gossamer/internal/client/network/sync"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"golang.org/x/exp/constraints"
)

// / Wraps around an implementation of the [`Network`] trait and provides gossiping capabilities on
// / top of it.
type GossipEngine[H constraints.Ordered, N runtime.Number] struct {
	stateMachine                ConsensusGossip[H]
	network                     Network
	sync                        Syncing[H, N]
	periodicMaintenanceInterval time.Timer
	protocol                    network.ProtocolName

	/// Incoming events from the network.
	networkEventStream chan event.Event
	/// Incoming events from the syncing service.
	syncEventStream chan sync.SyncEvent
	/// Outgoing events to the consumer.
	messageSinks map[H]chan TopicNotification
	/// Buffered messages (see [`ForwardingState`]).
	forwardingState forwardingState

	isTerminated bool
}

// / The gossip engine is currently not forwarding any messages and will poll the network for
// / more messages to forward.
type idle struct{}

// / The gossip engine is in the progress of forwarding messages and thus will not poll the
// / network for more messages until it has send all current messages into the subscribed
// / message sinks.
type busy[H constraints.Ordered] []struct {
	Hash H
	TopicNotification
}

// / A gossip engine receives messages from the network via the `network_event_stream` and forwards
// / them to upper layers via the `message_sinks`. In the scenario where messages have been received
// / from the network but a subscribed message sink is not yet ready to receive the messages, the
// / messages are buffered. To model this process a gossip engine can be in two states.
type forwardingStates[H constraints.Ordered] interface {
	idle | busy[H]
}
type forwardingState any
