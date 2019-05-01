package api

import (
	"net/http"

	"github.com/ChainSafe/gossamer/p2p"
)

// PublicRPC offers network related RPC methods
type PublicRPC struct {
	Net *p2p.Service
}

type Uint uint

type PublicRPCResponse struct {
	Count Uint
}

type PublicRPCRequest struct{}

// NewPublicNetAPI creates a new net API instance.
func NewPublicRPC(net *p2p.Service) *PublicRPC {
	return &PublicRPC{
		Net: net,
	}
}

// PeerCount returns the number of connected peers
func (s *PublicRPC) PeerCount(r *http.Request, args *PublicRPCRequest, res *PublicRPCResponse) error {
	res.Count = Uint(s.Net.PeerCount())
	return nil
}
