package api

import (
	"github.com/ChainSafe/gossamer/p2p"
)

type coreModule struct {
	p2p *p2p.Service
}

func (m *coreModule) Version() string{
	// TODO: Stubbed. Return runtime.CoreVersion()
	return "1.2.3"
}