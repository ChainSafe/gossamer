package api

import (
	log "github.com/ChainSafe/log15"
)

type coreModule struct {
	p2p p2pApi
	runtime runtimeApi
}

func (m *coreModule) Version() string {
	log.Debug("[rpc] Executing Core.Version", "params", nil)
	// TODO: Stubbed. Return m.runtime.CoreVersion() (pending PR)
	return m.runtime.Version()
}

// TODO: Move to 'p2p' module
// TODO: Why are these all returning strings?
func (m *coreModule) PeerCount() int {
	log.Debug("[rpc] Executing Core.PeerCount", "params", nil)
	return m.p2p.PeerCount()
}