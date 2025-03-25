// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossip

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/config"
	"github.com/ChainSafe/gossamer/internal/client/network/event"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	netSync "github.com/ChainSafe/gossamer/internal/client/network/sync"
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

type TestNetwork struct{}

func (TestNetwork) SetAuthorizedPeers(peers map[peerid.PeerID]struct{})                   { panic("unimpl") }
func (TestNetwork) SetAuthorizedOnly(reservedOnly bool)                                   { panic("unimpl") }
func (TestNetwork) AddKnownAddress(peerID peerid.PeerID, addr multiaddr.Multiaddr)        { panic("unimpl") }
func (TestNetwork) ReportPeer(peerID peerid.PeerID, costBenefit network.ReputationChange) {}
func (TestNetwork) DisconnectPeer(who peerid.PeerID, protocol network.ProtocolName)       { panic("unimpl") }
func (TestNetwork) AcceptUnreservedPeers()                                                { panic("unimpl") }
func (TestNetwork) DenyUnreservedPeers()                                                  { panic("unimpl") }
func (TestNetwork) AddReservedPeer(peer config.MultiaddrPeerId) error                     { panic("unimpl") }
func (TestNetwork) RemoveReservedPeer(peerID peerid.PeerID)                               { panic("unimpl") }
func (TestNetwork) SetReservedPeers(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (TestNetwork) AddPeersToReservedSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (TestNetwork) RemovePeersFromReservedSet(protocol network.ProtocolName, peers []peerid.PeerID) {
	panic("unimpl")
}
func (TestNetwork) AddToPeersSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (TestNetwork) RemoveFromPeersSet(protocol network.ProtocolName, peers []peerid.PeerID) {
	panic("unimpl")
}
func (TestNetwork) SyncNumConnected() uint { panic("unimpl") }
func (TestNetwork) PeerRole(peerID peerid.PeerID, handshake []byte) *role.ObservedRole {
	var roles role.Roles
	err := scale.Unmarshal(handshake, &roles)
	if err != nil {
		return nil
	}
	role := roles.ObservedRole()
	return &role
}
func (TestNetwork) ReservedPeers() <-chan struct {
	Peers []peerid.PeerID
	Error error
} {
	panic("unimpl")
}
func (TestNetwork) EventStream(name string) chan event.Event                        { panic("unimpl") }
func (TestNetwork) AnnounceBlock(hash hash.H256, data []byte)                       { panic("unimpl") }
func (TestNetwork) NewBestBlockImported(hash hash.H256, number uint64)              { panic("unimpl") }
func (TestNetwork) AddSetReserved(who peerid.PeerID, protocol network.ProtocolName) { panic("unimpl") }
func (TestNetwork) RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName) {
	panic("unimpl")
}

var _ Network = TestNetwork{}

type TestSync struct {
	eventSenders []chan netSync.SyncEvent
	sync.Mutex
}

func (ts *TestSync) EventStream(name string) chan netSync.SyncEvent {
	ts.Lock()
	defer ts.Unlock()
	ch := make(chan netSync.SyncEvent)
	ts.eventSenders = append(ts.eventSenders, ch)
	return ch
}

func (*TestSync) AnnounceBlock(hash hash.H256, data []byte)          { panic("unimpl") }
func (*TestSync) NewBestBlockImported(hash hash.H256, number uint64) { panic("unimpl") }

type TestNotificationService struct {
	ch chan service.NotificationEvent
}

func (TestNotificationService) OpenSubstream(peer peerid.PeerID) <-chan error  { panic("unimpl") }
func (TestNotificationService) CloseSubstream(peer peerid.PeerID) <-chan error { panic("unimpl") }
func (TestNotificationService) SendSyncNotification(peer peerid.PeerID, notification []byte) {
	panic("unimpl")
}
func (TestNotificationService) SendAsyncNotification(peer peerid.PeerID, notification []byte) <-chan error {
	panic("unimpl")
}
func (TestNotificationService) SetHandshake(handshake []byte) <-chan error         { panic("unimpl") }
func (TestNotificationService) TrySetHandshake(handshake []byte) error             { panic("unimpl") }
func (tns TestNotificationService) NextEvent() <-chan service.NotificationEvent    { return tns.ch }
func (TestNotificationService) Protocol() network.ProtocolName                     { panic("unimpl") }
func (TestNotificationService) MessageSink(peer peerid.PeerID) service.MessageSink { panic("unimpl") }

var _ service.NotificationService = TestNotificationService{}

type ChannelLengthTopic struct {
	Length uint
	Topic  hash.H256
}

func (ChannelLengthTopic) Generate(rand *rand.Rand, size int) reflect.Value {
	possibleLength := rand.Intn(100)
	possibleTopics := uint64(rand.Intn(10))
	topicHash := hash.NewH256FromLowUint64BigEndian(possibleTopics).Bytes()
	_ = topicHash
	clt := ChannelLengthTopic{
		Length: uint(possibleLength),
		Topic:  hash.NewH256FromLowUint64BigEndian(possibleTopics),
	}
	return reflect.ValueOf(clt)
}

type Message struct {
	Topic hash.H256
}

func (Message) Generate(rand *rand.Rand, size int) reflect.Value {
	possibleTopics := uint64(rand.Intn(10))
	return reflect.ValueOf(Message{
		Topic: hash.NewH256FromLowUint64BigEndian(possibleTopics),
	})
}

type TestValidator struct{}

func (ao TestValidator) NewPeer(context ValidatorContext[hash.H256], who peerid.PeerID, role role.ObservedRole) {
}
func (ao TestValidator) PeerDisconnected(context ValidatorContext[hash.H256], who peerid.PeerID) {
}
func (ao TestValidator) Validate(
	context ValidatorContext[hash.H256], sender peerid.PeerID, data []byte,
) ValidationResult {
	return ValidationResultProcessAndKeep[hash.H256]{
		Hash: hash.H256(data[0:32]),
	}
}
func (ao TestValidator) MessageExpired() func(topic hash.H256, message []byte) bool {
	return func(topic hash.H256, message []byte) bool {
		return false
	}
}
func (ao TestValidator) MessageAllowed() func(
	who peerid.PeerID, intent MessageIntent, topic hash.H256, data []byte,
) bool {
	return func(who peerid.PeerID, intent MessageIntent, topic hash.H256, data []byte) bool {
		return true
	}
}

func TestGossipEngine(t *testing.T) {
	t.Run("keeps_multiple_subscribers_per_topic_updated_with_both_old_and_new_messages", func(t *testing.T) {
		topic := hash.NewH256()
		protocol := network.ProtocolName("/my_protocol")
		remotePeer := peerid.NewRandomPeerID()
		network := TestNetwork{}
		sync := TestSync{}
		ch := make(chan service.NotificationEvent, 3)
		notificationService := TestNotificationService{ch: ch}

		gossipEngine := newGossipEngine[hash.H256, uint64, runtime.BlakeTwo256](
			network,
			&sync,
			notificationService,
			protocol,
			AllowAll{},
		)

		// Register the remote peer.
		ch <- service.NotificationEventNotificationStreamOpened{
			Peer:               remotePeer,
			Direction:          service.DirectionInbound,
			NegotiatedFallback: nil,
			Handshake:          scale.MustMarshal(role.RolesFull),
		}

		messages := [][]byte{{1}, {2}}

		// Send first event before subscribing.
		ch <- service.NotificationEventNotificationReceived{
			Peer:         remotePeer,
			Notification: messages[0],
		}

		subscribers := make([]chan TopicNotification, 0)
		for i := 0; i < 2; i++ {
			subscribers = append(subscribers, gossipEngine.MessagesFor(topic))
		}

		// Send second event after subscribing.
		ch <- service.NotificationEventNotificationReceived{
			Peer:         remotePeer,
			Notification: messages[1],
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			gossipEngine.poll()
		}()

		for _, message := range messages {
			for _, subscriber := range subscribers {
				topicNotification := <-subscriber
				expected := TopicNotification{
					Message: message,
					Sender:  &remotePeer,
				}
				require.Equal(t, expected, topicNotification)
			}
		}

		close(gossipEngine.stopChan)
		<-done
	})

	t.Run("forwarding_to_different_size_and_topic_channels", func(t *testing.T) {
		var prop = func(t *testing.T, channels []ChannelLengthTopic, notifications [][]Message) {
			protocol := network.ProtocolName("/my_protocol")
			remotePeer := peerid.NewRandomPeerID()
			network := TestNetwork{}
			sync := TestSync{}

			// for NotificationStreamOpened
			chanLength := 1
			// then all the messages
			for _, notification := range notifications {
				chanLength += len(notification)
			}
			ch := make(chan service.NotificationEvent, chanLength)
			notificationService := TestNotificationService{ch: ch}

			numChannelsPerTopic := make(map[hash.H256]uint)
			for _, channel := range channels {
				_, ok := numChannelsPerTopic[channel.Topic]
				if !ok {
					numChannelsPerTopic[channel.Topic] = 0
				}
				numChannelsPerTopic[channel.Topic]++
			}

			expectedTotalMsgsAllChan := uint(0)
			expectedMsgsPerTopicAllChan := make(map[hash.H256]uint)
			acc := make(map[hash.H256]uint)
			for _, messages := range notifications {
				for _, message := range messages {
					_, ok := acc[message.Topic]
					if !ok {
						acc[message.Topic] = 0
					}
					acc[message.Topic]++
				}
			}
			for topic, num := range acc {
				numChannels := numChannelsPerTopic[topic]
				expectedMsgsPerTopicAllChan[topic] = numChannels * num
				expectedTotalMsgsAllChan += numChannels * num
			}

			gossipEngine := newGossipEngine[hash.H256, uint64, runtime.BlakeTwo256](
				network,
				&sync,
				notificationService,
				protocol,
				TestValidator{},
			)

			type topicChan struct {
				Topic hash.H256
				Chan  chan TopicNotification
			}
			// Create channels.
			topicChans := make([]topicChan, 0)
			for _, channel := range channels {
				topicChans = append(topicChans, topicChan{
					Topic: channel.Topic,
					Chan:  make(chan TopicNotification, channel.Length),
				})
			}

			// Insert channels into gossipEngine.
			for _, topicChan := range topicChans {
				_, ok := gossipEngine.messageSinks[topicChan.Topic]
				if !ok {
					gossipEngine.messageSinks[topicChan.Topic] = make([]chan TopicNotification, 0)
				}
				gossipEngine.messageSinks[topicChan.Topic] = append(
					gossipEngine.messageSinks[topicChan.Topic], topicChan.Chan)
			}

			// Register the remote peer.
			ch <- service.NotificationEventNotificationStreamOpened{
				Peer:               remotePeer,
				Direction:          service.DirectionInbound,
				NegotiatedFallback: nil,
				Handshake:          scale.MustMarshal(role.RolesFull),
			}

			// Send messages into the network event stream.
			for iNotification, messages := range notifications {
				var msgs [][]byte
				for iMessage, message := range messages {
					// Embed the topic in the first 256 bytes of the message to be extracted by
					// the TestValidator later on.
					msg := message.Topic.Bytes()

					// Make sure the message is unique via iNotification and iMessage to
					// ensure consensusGossip does not deduplicate it.
					msg = append(msg, byte(iNotification))
					msg = append(msg, byte(iMessage))

					msgs = append(msgs, msg)
				}

				for _, msg := range msgs {
					// Send first event before subscribing.
					ch <- service.NotificationEventNotificationReceived{
						Peer:         remotePeer,
						Notification: msg,
					}
				}
			}

			receivedMsgsPerTopicAllChan := make(map[hash.H256]uint)

			// Poll both gossip engine and each receiver and track the amount of received messages.
			done := make(chan struct{})
			go func() {
				defer close(done)
				gossipEngine.poll()
			}()

			timer := time.NewTimer(1 * time.Second)
			defer timer.Stop()
			delay := timer.C

			msgCount := uint(0)
			var expected <-chan time.Time
		outer:
			for {
				for _, topicChan := range topicChans {
					select {
					case <-topicChan.Chan:
						_, ok := receivedMsgsPerTopicAllChan[topicChan.Topic]
						if !ok {
							receivedMsgsPerTopicAllChan[topicChan.Topic] = 0
						}
						receivedMsgsPerTopicAllChan[topicChan.Topic]++
						msgCount++
					default:
					}
				}

				if msgCount == expectedTotalMsgsAllChan {
					// Set a 1ms timeout, just to ensure we're not receiving more msgs.
					if expected == nil {
						timeout := time.NewTimer(1 * time.Millisecond)
						defer timeout.Stop()
						expected = timeout.C
					}
				}

				select {
				case <-delay:
					break outer
				case <-expected:
					break outer
				default:
				}
			}
			close(gossipEngine.stopChan)
			<-done

			// Compare amount of expected messages with amount of received messages.
			for expectedTopic, expectedNum := range expectedMsgsPerTopicAllChan {
				require.Equal(t, expectedNum, receivedMsgsPerTopicAllChan[expectedTopic])
			}

			for _, topicChan := range topicChans {
				close(topicChan.Chan)
			}
		}

		prop(t, nil, [][]Message{{Message{Topic: hash.NewH256()}}})
		prop(t,
			[]ChannelLengthTopic{{Topic: hash.NewH256(), Length: 71}},
			[][]Message{{{Topic: hash.NewH256()}}},
		)

		f := func(channels []ChannelLengthTopic, notifications [][]Message) bool {
			prop(t, channels, notifications)
			return true
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}
