// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"fmt"
	api "github.com/ChainSafe/gossamer/internal"
	log "github.com/inconshreveable/log15"
	"net/http"
)

type Config struct {
	Port uint32
}

type HTTPServer struct {
	cfg			*Config
	rpcServer	*Server
	modules		[]string
}

func NewHttpServer(api *api.Service, modules []string, cfg *Config) *HTTPServer {
	server := &HTTPServer{
		cfg: cfg,
		rpcServer: NewServer(modules, api),
	}

	return server
}

func (h *HTTPServer) Setup() {
	// TODO: Maybe move logic from NewHttpServer here
	// TODO: Select which modules to enable and verify they are valid
	err := h.rpcServer.RegisterService(NewCoreModule(h.rpcServer.api), "core")
	if err != nil {
		log.Error("Error on HTTP server setup.", "err", err)
	}
}

// Start registers the rpc handler function and start the server listening on h.port
func (h *HTTPServer) Start () {
	log.Debug("[rpc] Starting HTTP Server...", "port", h.cfg.Port)
	go func() {
		http.HandleFunc("/", h.rpcServer.ServeHTTP)
		err := http.ListenAndServe(fmt.Sprintf(":%d", h.cfg.Port), nil)
		if err != nil {
			log.Error("[rpc] http error", "err", err)
		}
	}()
}
