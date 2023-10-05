package communication

import (
	gossip "github.com/ChainSafe/gossamer/client/network-gossip"
	"github.com/ChainSafe/gossamer/client/network/service"
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
type Syncing interface {
	service.NetworkSyncForkRequest
	service.NetworkBlock
	service.SyncEventStream
}

// / Bridge between the underlying network service, gossiping consensus messages and Grandpa
type NetworkBridge struct {
	service Network
}
