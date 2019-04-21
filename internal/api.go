package internal

import "github.com/ChainSafe/gossamer/p2p"

// PublicRPC offers network related RPC methods
type PublicRPC struct {
	net            *p2p.Service
	networkVersion uint64
}

type Uint uint

// PeerCount returns the number of connected peers
func (s *PublicRPC) PeerCount() Uint {
	return Uint(s.net.PeerCount())
}