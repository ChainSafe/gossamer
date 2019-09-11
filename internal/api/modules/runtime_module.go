package api

import (
	log "github.com/ChainSafe/log15"
)

type rtModule struct {
	rt RuntimeApi
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