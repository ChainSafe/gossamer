package api

import (
	log "github.com/ChainSafe/log15"
)

type p2pModule struct {
	p2p P2pApi
}
type rtModule struct {
	rt RuntimeApi
}

func NewP2PModule(p2papi P2pApi) *p2pModule {
	return &p2pModule{p2papi}
}

func NewRTModule(RTapi RuntimeApi) *rtModule {
	return &rtModule{RTapi}
}

// Release version
func (r *rtModule) Version() string {
	log.Debug("[rpc] Executing System.Version", "params", nil)
	//TODO: Replace with dynamic version
	return "0.0.1"
}

func (r *rtModule) Name() string {
	log.Debug("[rpc] Executing System.Name", "params", nil)
	//TODO: Replace with dynamic name
	return "Gossamer"
}

func (p *p2pModule) PeerCount() int {
	log.Debug("[rpc] Executing System.PeerCount", "params", nil)
	return len(p.Peers())
}

// Peers of the node
func (p *p2pModule) Peers() []string {
	log.Debug("[rpc] Executing System.Peers", "params", nil)
	return p.p2p.Peers()
}

func (p *p2pModule) ShouldHavePeers() bool {
	return p.p2p.ShouldHavePeers()
}

func (p *p2pModule) ID() string {
	log.Debug("[rpc] Executing System.networkState", "params", nil)
	return p.p2p.ID()
}

func (p *p2pModule) IsSyncing() bool {
	return false
}
