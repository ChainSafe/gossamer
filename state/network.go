package state

import "github.com/ChainSafe/gossamer/p2p"

// NetworkState is a wrapper for the the network state
type NetworkState struct {
	p2p *p2p.Service
}

// NewNetworkState will create a new instance of NetworkState
func NewNetworkState() *NetworkState {
	return &NetworkState{
		// TODO: pass p2p service instance to network state
		p2p: &p2p.Service{},
	}
}

// Health return Health() of p2p service
func (ns *NetworkState) Health() p2p.Health {
	// TODO: return Health() of p2p service
	return p2p.Health{}
}

// NetworkState return NetworkState() of p2p service
func (ns *NetworkState) NetworkState() p2p.NetworkState {
	// TODO: return NetworkState() of p2p service
	return p2p.NetworkState{}
}

// Peers return Peers() of p2p service
func (ns *NetworkState) Peers() []p2p.PeerInfo {
	// TODO: return Peers() of p2p service
	return []p2p.PeerInfo{}
}
