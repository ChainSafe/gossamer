// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossip

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	netSync "github.com/ChainSafe/gossamer/internal/client/network/sync"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "client/network-gossip"))

// GossipEngine receives messages from the network via the Network and forwards them to upper layers via messageSinks.
// In the scenario where messages have been received from the network but a subscribed message sink is not ready to
// receiver, we delay 10 ms and will remove the channel from the sinks if the message is not consumed by the end of
// the delay. To model this process a gossip engine can be in two forwarding states: idle, and busy.
//
// GossipEngine utilises and implementation of [Network] and provides gossiping capabilities on
// top of it.
type GossipEngine[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	stateMachine                *consensusGossip[H, Hasher]
	network                     Network
	sync                        Syncing[H, N]
	periodicMaintenanceInterval <-chan time.Time
	protocol                    network.ProtocolName

	// Incoming events from the syncing service.
	syncEventStream chan netSync.SyncEvent
	// Handle for polling notification-related events.
	notificationService service.NotificationService
	// Outgoing events to the consumer.
	messageSinks map[H][]chan TopicNotification
	// Buffered messages (see [`ForwardingState`]).
	forwardingState forwardingState

	isTerminated bool
	stopChan     chan struct{}
	errChan      chan error
}

type forwardingState interface {
	isForwardingState()
}

// The gossip engine is currently not forwarding any messages and will poll the network for more messages to forward.
type forwardingStateIdle struct{}

func (forwardingStateIdle) isForwardingState() {}

// The gossip engine is in the progress of forwarding messages and thus will not poll the network for more messages
// until it has send all current messages into the subscribed message sinks.
type forwardingStateBusy[H runtime.Hash] []hashTopicNotification[H]

func (forwardingStateBusy[H]) isForwardingState() {}

func NewGossipEngine[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	network Network,
	sync Syncing[H, N],
	notificationService service.NotificationService,
	protocol network.ProtocolName,
	validator Validator[H],
) GossipEngine[H, N, Hasher] {
	ge := newGossipEngine[H, N, Hasher](network, sync, notificationService, protocol, validator)
	go func() {
		defer close(ge.errChan)
		ge.errChan <- ge.poll()
	}()
	return ge
}

func newGossipEngine[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	network Network,
	sync Syncing[H, N],
	notificationService service.NotificationService,
	protocol network.ProtocolName,
	validator Validator[H],
) GossipEngine[H, N, Hasher] {
	syncEventStream := sync.EventStream("network-gossip")
	_ = syncEventStream

	ge := GossipEngine[H, N, Hasher]{
		stateMachine:                newConsensusGossip[H, Hasher](validator, protocol),
		network:                     network,
		sync:                        sync,
		notificationService:         notificationService,
		periodicMaintenanceInterval: time.NewTimer(periodicMaintenanceInterval).C,
		protocol:                    protocol,
		syncEventStream:             syncEventStream,
		messageSinks:                make(map[H][]chan TopicNotification),
		forwardingState:             forwardingStateIdle{},
		isTerminated:                false,
		stopChan:                    make(chan struct{}),
		errChan:                     make(chan error, 1),
	}
	return ge
}

func (ge *GossipEngine[H, N, Hasher]) Report(who peerid.PeerID, reputation network.ReputationChange) {
	ge.network.ReportPeer(who, reputation)
}

// RegisterGossipMessage registers a message without propagating it to any peers. The message becomes available to new
// peers or when the service is asked to gossip the message's topic. No validation is performed on the message, if the
// message is already expired it should be dropped on the next garbage collection.
func (ge *GossipEngine[H, N, Hasher]) RegisterGossipMessage(topic H, message []byte) {
	ge.stateMachine.RegisterMessage(topic, message)
}

// BroadcastTopic will broadcast all messages with given topic.
func (ge *GossipEngine[H, N, Hasher]) BroadcastTopic(topic H, force bool) {
	ge.stateMachine.BroadcastTopic(ge.notificationService, topic, force)
}

// MessagesFor retrieves data of valid, incoming messages for a topic (but might have expired).
func (ge *GossipEngine[H, N, Hasher]) MessagesFor(topic H) chan TopicNotification {
	pastMessages := ge.stateMachine.MessagesFor(topic)
	// The channel length is not critical for correctness.
	size := 10
	if len(pastMessages) > size {
		size = len(pastMessages)
	}
	ch := make(chan TopicNotification, size)

	for _, notification := range pastMessages {
		select {
		case ch <- notification:
		default:
			panic("receiver known to be live, and buffer size known to suffice")
		}
	}

	_, ok := ge.messageSinks[topic]
	if !ok {
		ge.messageSinks[topic] = make([]chan TopicNotification, 0)
	}
	ge.messageSinks[topic] = append(ge.messageSinks[topic], ch)
	return ch
}

// SendTopic will send all messages with given topic to a peer.
func (ge *GossipEngine[H, N, Hasher]) SendTopic(who peerid.PeerID, topic H, force bool) {
	ge.stateMachine.SendTopic(ge.notificationService, who, topic, force)
}

// GossipMessage will multicast a message to all peers.
func (ge *GossipEngine[H, N, Hasher]) GossipMessage(topic H, message []byte, force bool) {
	ge.stateMachine.Multicast(ge.notificationService, topic, message, force)
}

// SendMessage will send addressed message to the given peers. The message is not kept or multicast
// later on.
func (ge *GossipEngine[H, N, Hasher]) SendMessage(who []peerid.PeerID, data []byte) {
	for _, who := range who {
		ge.stateMachine.SendMessage(ge.notificationService, who, data)
	}
}

// Announce will notify everyone we're connected to that we have the given block.
//
// Note: this method isn't strictly related to gossiping and should eventually be moved
// somewhere else.
func (ge *GossipEngine[H, N, Hasher]) Announce(block H, associatedData []byte) {
	ge.sync.AnnounceBlock(block, associatedData)
}

func (ge *GossipEngine[H, N, Hasher]) poll() error { //nolint: gocyclo
	var nextNotificationEvent <-chan service.NotificationEvent
	// outer:
	for {
		switch forwardingState := ge.forwardingState.(type) {
		case forwardingStateIdle:
			if nextNotificationEvent == nil {
				nextNotificationEvent = ge.notificationService.NextEvent()
			}
			syncEventStream := ge.syncEventStream

			select {
			case event := <-nextNotificationEvent:
				// if ok {
				switch event := event.(type) {
				case service.NotificationEventValidateInboundSubstream:
					// only accept peers whose role can be determined
					var result service.ValidationResult = service.ValidationResultReject
					role := ge.network.PeerRole(event.Peer, event.Handshake)
					if role != nil {
						result = service.ValidationResultAccept
					}
					event.ResultChan <- result
					close(event.ResultChan)
				case service.NotificationEventNotificationStreamOpened:
					role := ge.network.PeerRole(event.Peer, event.Handshake)
					if role != nil {
						ge.stateMachine.NewPeer(ge.notificationService, event.Peer, *role)
					} else {
						logger.Debugf("role for %s couldn't be determined", event.Peer)
					}
				case service.NotificationEventNotificationStreamClosed:
					ge.stateMachine.PeerDisconnected(ge.notificationService, event.Peer)
				case service.NotificationEventNotificationReceived:
					toForward := ge.stateMachine.OnIncoming(
						ge.network, ge.notificationService, event.Peer, [][]byte{event.Notification})
					ge.forwardingState = forwardingStateBusy[H](toForward)
				default:
					panic("unreachable")
				}
				nextNotificationEvent = nil
			case syncEvent, ok := <-syncEventStream:
				if !ok {
					// The sync event stream closed
					ge.isTerminated = true
					return fmt.Errorf("syncEventStream was terminated unexpectedly")
				}
				if ok {
					switch remote := syncEvent.(type) {
					case netSync.SyncEventPeerConnected:
						ge.network.AddSetReserved(peerid.PeerID(remote), ge.protocol)
					case netSync.SyncEventPeerDisconnected:
						ge.network.RemoveSetReserved(peerid.PeerID(remote), ge.protocol)
					}
				}
			case <-ge.periodicMaintenanceInterval:
				ge.periodicMaintenanceInterval = time.NewTimer(periodicMaintenanceInterval).C
				ge.stateMachine.Tick(ge.notificationService)

				for topic, sinks := range ge.messageSinks {
					retained := make([]chan TopicNotification, 0)
					for _, sink := range sinks {
						if sink != nil {
							retained = append(retained, sink)
						}
					}
					if len(retained) > 0 {
						ge.messageSinks[topic] = retained
					} else {
						delete(ge.messageSinks, topic)
					}
				}
			case <-ge.stopChan:
				return nil
			}

		case forwardingStateBusy[H]:
			var (
				topic        H
				notification TopicNotification
			)
			if len(forwardingState) > 0 {
				htn := forwardingState[0]
				topic = htn.Hash
				notification = htn.TopicNotification
				ge.forwardingState = forwardingState[1:]
			} else {
				ge.forwardingState = forwardingStateIdle{}
				continue
			}

			sinks, ok := ge.messageSinks[topic]
			if !ok {
				continue
			}

			// Filter out all closed sinks.
			retained := make([]chan TopicNotification, 0)
			for _, sink := range sinks {
				if sink != nil {
					retained = append(retained, sink)
				}
			}
			ge.messageSinks[topic] = retained
			sinks = ge.messageSinks[topic]

			if len(sinks) == 0 {
				delete(ge.messageSinks, topic)
				continue
			}

			logger.Tracef("Pushing consensus message to sinks for %s", topic)

			// Send the notification on each sink.
			var wg sync.WaitGroup
			for i, sink := range sinks {
				wg.Add(1)
				go func(sink chan TopicNotification) {
					defer wg.Done()
					timeout := time.NewTimer(10 * time.Millisecond)
					defer timeout.Stop()
					select {
					case sink <- notification:
					// TODO: retry logic?
					case <-timeout.C:
						// Receiver not responding. Will be removed in next iteration
						close(sink)
						ge.messageSinks[topic][i] = nil
					}
				}(sink)
			}
			wg.Wait()

		default:
			panic("unreachable")
		}
	}
}
