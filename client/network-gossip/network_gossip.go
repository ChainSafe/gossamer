package gossip

import (
	"github.com/ChainSafe/gossamer/client/network"
	"github.com/ChainSafe/gossamer/client/network/service"
	"github.com/ChainSafe/gossamer/client/network/sync"
	"github.com/libp2p/go-libp2p/core"
)

// / Abstraction over a network.
type Network interface {
	// pub trait Network<B: BlockT>: NetworkPeers + NetworkEventStream + NetworkNotification {
	// 	fn add_set_reserved(&self, who: PeerId, protocol: ProtocolName) {
	service.NetworkPeers
	service.NetworkEventStream
	service.NetworkNotification
	AddSetReserved(who core.PeerID, protocol network.ProtocolName)
}

// / Abstraction over the syncing subsystem.
// pub trait Syncing<B: BlockT>: SyncEventStream + NetworkBlock<B::Hash, NumberFor<B>> {}
type Syncing[H, N any] interface {
	sync.SyncEventStream
	service.NetworkBlock[H, N]
}
