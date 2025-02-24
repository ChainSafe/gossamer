package gossip

import (
	"github.com/ChainSafe/gossamer/internal/client/network"
	peerid "github.com/ChainSafe/gossamer/internal/client/network-types/peer-id"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	"github.com/ChainSafe/gossamer/internal/client/network/sync"
)

// Abstraction over a network.
type Network interface {
	service.NetworkPeers
	service.NetworkEventStream
	AddSetReserved(who peerid.PeerID, protocol network.ProtocolName)
	RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName)
}

// Abstraction over the syncing subsystem.
type Syncing[H, N any] interface {
	sync.SyncEventStream
	service.NetworkBlock[H, N]
}
