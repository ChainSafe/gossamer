package api

import (
	log "github.com/ChainSafe/log15"
)

type systemModule struct {
	p2p     P2pApi
	runtime RuntimeApi
}

func NewSystemModule(p2p P2pApi, rt RuntimeApi) *systemModule {
	log.Debug("API | Instatiating new system module...")
	return &systemModule{
		p2p,
		rt,
	}
}

func (m *systemModule) Version() string {
	log.Debug("API | [System.Version]", "version", m.runtime.Version())
	return m.runtime.Version()
}

// TODO: Move to 'p2p' module
func (m *systemModule) PeerCount() int {
	log.Debug("API | [System.PeerCount]", "peerCount", m.p2p.PeerCount())
	return m.p2p.PeerCount()
}
