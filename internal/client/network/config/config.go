package config

import (
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
)

// MultiaddrPeerId is the address of a node, including its identity.
// This struct represents a decoded version of a multiaddress that ends with "/p2p/<peerid>".
type MultiaddrPeerId struct {
	multiaddr.Multiaddr
	peerid.PeerID
}
