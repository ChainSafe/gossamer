package modules

import (
	"net/http"
)

// DevModule is an RPC module that provides developer endpoints
type DevModule struct {
	networkAPI       NetworkAPI
	blockProducerAPI BlockProducerAPI
}

// NewDevModule creates a new Dev module.
func NewDevModule(bp BlockProducerAPI, net NetworkAPI) *DevModule {
	return &DevModule{
		networkAPI:       net,
		blockProducerAPI: bp,
	}
}

// Control to send start and stop messages to services
func (m *DevModule) Control(r *http.Request, req *[]string, res *string) error {
	reqA := *req
	var err error
	switch reqA[0] {
	case "babe":
		switch reqA[1] {
		case "stop":
			err = m.blockProducerAPI.Pause()
			*res = "babe service stopped"
		case "start":
			err = m.blockProducerAPI.Resume()
			*res = "babe service started"
		}
	case "network":
		switch reqA[1] {
		case "stop":
			err = m.networkAPI.Stop()
			*res = "network service stopped"
		case "start":
			err = m.networkAPI.Start()
			*res = "network service started"
		}
	}
	return err
}
