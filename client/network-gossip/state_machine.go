package gossip

import (
	"time"

	"github.com/ChainSafe/gossamer/client/network"
	"github.com/ChainSafe/gossamer/primitives/runtime"
	libp2p "github.com/libp2p/go-libp2p/core"
)

type peerConsensus[H comparable] struct {
	knownMessages map[H]any
}

// / Topic stream message with sender.
type TopicNotification struct {
	/// Message data.
	Message []byte
	/// Sender if available.
	Sender *libp2p.PeerID
}

//	struct MessageEntry<B: BlockT> {
//		message_hash: B::Hash,
//		topic: B::Hash,
//		message: Vec<u8>,
//		sender: Option<PeerId>,
//	}
type MessageEntry[H runtime.Hash] struct {
	messageHash H
	topic       H
	message     []byte
	sender      *libp2p.PeerID
}

// / Consensus network protocol handler. Manages statements and candidate requests.
type ConsensusGossip[H runtime.Hash] struct {
	peers    map[libp2p.PeerID]peerConsensus[H]
	messages []MessageEntry[H]
	// TODO: known_messages: LruCache<B::Hash, ()>,
	knownMessages map[H]any
	protocol      network.ProtocolName
	validator     Validator[H]
	// next_broadcast: Instant,
	nextBroadcast time.Time
	metrics       metrics
}

//	struct Metrics {
//		registered_messages: Counter<U64>,
//		expired_messages: Counter<U64>,
//	}
type metrics struct{}
