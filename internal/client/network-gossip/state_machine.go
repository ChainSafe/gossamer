// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossip

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/dolthub/maphash"
	"github.com/elastic/go-freelru"
)

// NOTE: The current value is adjusted based on largest production network deployment (Kusama) and the current main
// gossip user (GRANDPA). Currently there are ~800 validators on Kusama, as such,each GRANDPA round should generate
// ~1600 messages, and we currently keep track of the last 2completed rounds and the current live one. That makes it so
// that at any point we will be holding ~4800 live messages.
//
// Assuming that each known message is tracked with a 32 byte hash, then this cache should take about 256 KB of memory.
const knownMessageCacheSize uint32 = 8192

const rebroadcastInterval time.Duration = 750 * time.Millisecond

const periodicMaintenanceInterval time.Duration = 1100 * time.Millisecond

var (
	// Reputation change when a peer sends us a gossip message that we didn't know about.
	gossipSuccess = network.NewReputationChange(1<<4, "Successful gossip")
	// Reputation change when a peer sends us a gossip message that we already knew about.
	duplicateGossip = network.NewReputationChange(-(1 << 2), "Duplicate gossip")
)

type peerConsensus[H comparable] struct {
	knownMessages map[H]any
}

// Topic stream message with sender.
type TopicNotification struct {
	// Message data.
	Message []byte
	// Sender if available.
	Sender *peerid.PeerID
}

type messageEntry[H runtime.Hash] struct {
	messageHash H
	topic       H
	message     []byte
	sender      *peerid.PeerID
}

// Local implementation of [ValidatorContext].
type newtorkContext[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	gossip              *consensusGossip[H, Hasher]
	notificationService service.NotificationService
}

// Broadcast all messages with given topic to peers that do not have it yet.
func (nc newtorkContext[H, Hasher]) BroadcastTopic(topic H, force bool) {
	nc.gossip.BroadcastTopic(nc.notificationService, topic, force)
}

// Broadcast a message to all peers that have not received it previously.
func (nc newtorkContext[H, Hasher]) BroadcastMessage(topic H, message []byte, force bool) {
	nc.gossip.Multicast(nc.notificationService, topic, message, force)
}

// Send addressed message to a peer.
func (nc newtorkContext[H, Hasher]) SendMessage(who peerid.PeerID, message []byte) {
	nc.notificationService.SendSyncNotification(who, message)
}

// Send all messages with given topic to a peer.
func (nc newtorkContext[H, Hasher]) SendTopic(who peerid.PeerID, topic H, force bool) {
	nc.gossip.SendTopic(nc.notificationService, who, topic, force)
}

func propagate[H runtime.Hash](
	notificationService service.NotificationService,
	_ network.ProtocolName,
	messages []messageEntry[H],
	intent MessageIntent,
	peers map[peerid.PeerID]peerConsensus[H],
	validator Validator[H],
) {
	messageAllowed := validator.MessageAllowed()

	for id, peer := range peers {
		for _, message := range messages {
			switch intent {
			case MessageIntentBroadcast:
				if _, ok := peer.knownMessages[message.messageHash]; ok {
					continue
				}
				intent = MessageIntentBroadcast
			case MessageIntentPeriodicReboradcast:
				if _, ok := peer.knownMessages[message.messageHash]; ok {
					intent = MessageIntentPeriodicReboradcast
				} else {
					// peer doesn't know message, so the logic should treat it as an
					// initial broadcast.
					intent = MessageIntentBroadcast
				}
			default:
			}

			if !messageAllowed(id, intent, message.topic, message.message) {
				continue
			}

			peer.knownMessages[message.messageHash] = nil
			peers[id] = peer

			notificationService.SendSyncNotification(id, message.message)
		}
	}

}

// Consensus network protocol handler. Manages statements and candidate requests.
type consensusGossip[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	peers         map[peerid.PeerID]peerConsensus[H]
	messages      []messageEntry[H]
	knownMessages freelru.LRU[H, any]
	protocol      network.ProtocolName
	validator     Validator[H]
	nextBroadcast time.Time
}

type hasher[K comparable] struct {
	maphash.Hasher[K]
}

func (h hasher[K]) Hash(key K) uint32 {
	return uint32(h.Hasher.Hash(key))
}

// Create a new instance using the given validator.
func newConsensusGossip[H runtime.Hash, Hasher runtime.Hasher[H]](
	validator Validator[H],
	protocol network.ProtocolName,
) consensusGossip[H, Hasher] {
	h := hasher[H]{maphash.NewHasher[H]()}
	knownMessages, err := freelru.New[H, any](knownMessageCacheSize, h.Hash)
	if err != nil {
		panic(err)
	}
	return consensusGossip[H, Hasher]{
		peers:         make(map[peerid.PeerID]peerConsensus[H]),
		messages:      make([]messageEntry[H], 0),
		knownMessages: *knownMessages,
		protocol:      protocol,
		validator:     validator,
		nextBroadcast: time.Now().Add(rebroadcastInterval),
	}
}

// Handle new connected peer.
func (cg *consensusGossip[H, Hasher]) NewPeer(
	notificationService service.NotificationService,
	who peerid.PeerID,
	role role.ObservedRole,
) {
	cg.peers[who] = peerConsensus[H]{knownMessages: make(map[H]any)}

	validator := cg.validator
	context := newtorkContext[H, Hasher]{gossip: cg, notificationService: notificationService}
	validator.NewPeer(context, who, role)
}

func (cg *consensusGossip[H, Hasher]) registerMessageHashed(
	messageHash H, topic H, message []byte, sender *peerid.PeerID,
) {
	cg.knownMessages.Add(messageHash, nil)
	cg.messages = append(cg.messages, messageEntry[H]{
		messageHash: messageHash,
		topic:       topic,
		message:     message,
		sender:      sender,
	})
	//TODO: registered meessages metrics
}

// Registers a message without propagating it to any peers. The message becomes available to new peers or when the
// service is asked to gossip the message's topic. No validation is performed on the message, if the message is already
// expired it should be dropped on the next garbage collection.
func (cg *consensusGossip[H, Hasher]) RegisterMessage(topic H, message []byte) {
	messageHash := (*new(Hasher)).Hash(message)
	cg.registerMessageHashed(messageHash, topic, message, nil)
}

// Call when a peer has been disconnected to stop tracking gossip status.
func (cg *consensusGossip[H, Hasher]) PeerDisconnected(
	notificationService service.NotificationService, who peerid.PeerID,
) {
	validator := cg.validator
	context := newtorkContext[H, Hasher]{gossip: cg, notificationService: notificationService}
	validator.PeerDisconnected(context, who)
	delete(cg.peers, who)
}

// Perform periodic maintenance
func (cg *consensusGossip[H, Hasher]) Tick(notificationService service.NotificationService) {
	cg.CollectGarbage()
	now := time.Now()
	if now.After(cg.nextBroadcast) || now.Equal(cg.nextBroadcast) {
		cg.rebroadcast(notificationService)
		cg.nextBroadcast = time.Now().Add(rebroadcastInterval)
	}
}

// Rebroadcast all messages to all peers.
func (cg *consensusGossip[H, Hasher]) rebroadcast(notificationService service.NotificationService) {
	propagate(notificationService, cg.protocol, cg.messages, MessageIntentPeriodicReboradcast, cg.peers, cg.validator)
}

// Broadcast all messages with given topic.
func (cg *consensusGossip[H, Hasher]) BroadcastTopic(
	notificationService service.NotificationService, topic H, force bool,
) {
	var messages []messageEntry[H]
	for _, entry := range cg.messages {
		if entry.topic == topic {
			messages = append(messages, entry)
		}
	}

	var intent MessageIntent = MessageIntentBroadcast
	if force {
		intent = MessageIntentForcedBroadcast
	}
	propagate(notificationService, cg.protocol, messages, intent, cg.peers, cg.validator)
}

// Prune old or no longer relevant consensus messages. Provide a predicate for pruning, which returns false when the
// items with a given topic should be pruned.
func (cg *consensusGossip[H, Hasher]) CollectGarbage() {
	knownMessages := cg.knownMessages
	before := len(cg.messages)

	messageExpired := cg.validator.MessageExpired()
	tempMessages := make([]messageEntry[H], 0)
	for _, msg := range cg.messages {
		if !messageExpired(msg.topic, msg.message) {
			tempMessages = append(tempMessages, msg)
		}
	}
	cg.messages = tempMessages

	expiredMessages := before - len(cg.messages)
	_ = expiredMessages
	// TODO: expired messages metric

	for id, peer := range cg.peers {
		for h := range peer.knownMessages {
			if _, ok := knownMessages.Get(h); !ok {
				delete(peer.knownMessages, h)
			}
		}
		cg.peers[id] = peer
	}
}

// Get valid messages received in the past for a topic (might have expired meanwhile).
func (cg *consensusGossip[H, Hasher]) MessagesFor(topic H) []TopicNotification {
	notifications := make([]TopicNotification, 0)
	for _, entry := range cg.messages {
		if entry.topic == topic {
			notifications = append(notifications, TopicNotification{Message: entry.message, Sender: entry.sender})
		}
	}
	return notifications
}

type hashTopicNotification[H runtime.Hash] struct {
	Hash H
	TopicNotification
}

// Register incoming messages and return the ones that are new and valid (according to a [Validator]) and should thus
// be forwarded to the upper layers.
func (cg *consensusGossip[H, Hasher]) OnIncoming(
	network Network,
	notificationService service.NotificationService,
	who peerid.PeerID,
	messages [][]byte,
) []hashTopicNotification[H] {
	toForward := make([]hashTopicNotification[H], 0)

	for _, message := range messages {
		messageHash := (*new(Hasher)).Hash(message)

		if _, ok := cg.knownMessages.Get(messageHash); ok {
			// If the peer already send us the message once, let's report them.
			peer, ok := cg.peers[who]
			if ok {
				_, ok := peer.knownMessages[messageHash]
				if ok {
					network.ReportPeer(who, duplicateGossip)
				}
				peer.knownMessages[messageHash] = nil
				cg.peers[who] = peer
			}
			continue
		}

		// validate the message
		validator := cg.validator
		context := newtorkContext[H, Hasher]{gossip: cg, notificationService: notificationService}
		validation := validator.Validate(&context, who, message)

		var (
			topic H
			keep  bool
		)
		switch validation := validation.(type) {
		case ValidationResultProcessAndKeep[H]:
			topic = validation.Hash
			keep = true
		case ValidationResultProcessAndDiscard[H]:
			topic = validation.Hash
			keep = false
		case ValidationResultDiscard:
			continue
		default:
			panic("unreachable")
		}

		peer, ok := cg.peers[who]
		if !ok {
			continue
		}

		network.ReportPeer(who, gossipSuccess)
		peer.knownMessages[messageHash] = nil
		toForward = append(toForward, hashTopicNotification[H]{
			Hash:              topic,
			TopicNotification: TopicNotification{Message: message, Sender: &who},
		})

		if keep {
			cg.registerMessageHashed(messageHash, topic, message, &who)
		}
	}

	return toForward
}

// Send all messages with given topic to a peer.
func (cg *consensusGossip[H, Hasher]) SendTopic(
	notificationService service.NotificationService, who peerid.PeerID, topic H, force bool,
) {
	messageAllowed := cg.validator.MessageAllowed()

	if peer, ok := cg.peers[who]; ok {
		for _, entry := range cg.messages {
			var intent MessageIntent = MessageIntentBroadcast
			if force {
				intent = MessageIntentForcedBroadcast
			}

			_, ok := peer.knownMessages[entry.messageHash]
			if !force && ok {
				continue
			}

			if !messageAllowed(who, intent, entry.topic, entry.message) {
				continue
			}

			peer.knownMessages[entry.messageHash] = nil
			cg.peers[who] = peer

			notificationService.SendSyncNotification(who, entry.message)
		}
	}
}

// Multicast a message to all peers.
func (cg *consensusGossip[H, Hasher]) Multicast(
	notificationService service.NotificationService, topic H, message []byte, force bool,
) {
	messageHash := (*new(Hasher)).Hash(message)
	cg.registerMessageHashed(messageHash, topic, message, nil)
	var intent MessageIntent = MessageIntentBroadcast
	if force {
		intent = MessageIntentForcedBroadcast
	}

	propagate(
		notificationService,
		cg.protocol,
		[]messageEntry[H]{{messageHash: messageHash, topic: topic, message: message}},
		intent,
		cg.peers,
		cg.validator,
	)
}

// Send addressed message to a peer. The message is not kept or multicast later on.
func (cg *consensusGossip[H, Hasher]) SendMessage(
	notificationService service.NotificationService, who peerid.PeerID, message []byte,
) {
	peer, ok := cg.peers[who]
	if !ok {
		return
	}

	messageHash := (*new(Hasher)).Hash(message)
	peer.knownMessages[messageHash] = nil
	cg.peers[who] = peer
	notificationService.SendSyncNotification(who, message)
}
