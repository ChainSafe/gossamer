package communication

import (
	gossip "github.com/ChainSafe/gossamer/client/network-gossip"
	"github.com/ChainSafe/gossamer/client/network/service"
	"github.com/ChainSafe/gossamer/client/network/sync"
	"github.com/ChainSafe/gossamer/primitives/runtime"
)

// / A handle to the network.
// /
// / Something that provides the capabilities needed for the `gossip_network::Network` trait.
type Network interface {
	gossip.Network
}

// / A handle to syncing-related services.
// /
// / Something that provides the ability to set a fork sync request for a particular block.
type Syncing[H, N any] interface {
	service.NetworkSyncForkRequest[H, N]
	service.NetworkBlock[H, N]
	sync.SyncEventStream
}

// / Bridge between the underlying network service, gossiping consensus messages and Grandpa
type NetworkBridge[H runtime.Hash, N runtime.Number] struct {
	service      Network
	sync         Syncing[H, N]
	gossipEngine gossip.GossipEngine[H, N]

	/// Sender side of the neighbor packet channel.
	///
	/// Packets sent into this channel are processed by the `NeighborPacketWorker` and passed on to
	/// the underlying `GossipEngine`.
	// neighbor_sender: periodic::NeighborPacketSender<B>,
	neighborSender

	/// `NeighborPacketWorker` processing packets sent through the `NeighborPacketSender`.
	// `NetworkBridge` is required to be cloneable, thus one needs to be able to clone its
	// children, thus one has to wrap `neighbor_packet_worker` with an `Arc` `Mutex`.
	// neighbor_packet_worker: Arc<Mutex<periodic::NeighborPacketWorker<B>>>,

	/// Receiver side of the peer report stream populated by the gossip validator, forwarded to the
	/// gossip engine.
	// `NetworkBridge` is required to be cloneable, thus one needs to be able to clone its
	// children, thus one has to wrap gossip_validator_report_stream with an `Arc` `Mutex`. Given
	// that it is just an `UnboundedReceiver`, one could also switch to a
	// multi-producer-*multi*-consumer channel implementation.
	// gossip_validator_report_stream: Arc<Mutex<TracingUnboundedReceiver<PeerReport>>>,

	// telemetry: Option<TelemetryHandle>,
}
