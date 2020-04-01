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
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
)


// ServerConfig configures the server
type ServerConfig struct {
	BlockAPI            modules.BlockAPI
	StorageAPI          modules.StorageAPI
	NetworkAPI          modules.NetworkAPI
	CoreAPI             modules.CoreAPI
	TransactionQueueAPI modules.TransactionQueueAPI
	Modules             []string
}

// Server is an RPC server.
type Server struct {
	blockAPI            modules.BlockAPI
	storageAPI          modules.StorageAPI
	networkAPI          modules.NetworkAPI
	coreAPI             modules.CoreAPI
	transactionQueueAPI modules.TransactionQueueAPI
	rpcServer *rpc.Server
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{	}
}

// NewStateServer creates a new Server that interfaces with the state service.
func NewStateServer(cfg *ServerConfig) *Server {
	s := &Server{
		//services:            new(serviceMap),
		blockAPI:            cfg.BlockAPI,
		storageAPI:          cfg.StorageAPI,
		networkAPI:          cfg.NetworkAPI,
		coreAPI:             cfg.CoreAPI,
		transactionQueueAPI: cfg.TransactionQueueAPI,
		rpcServer: rpc.NewServer(),
	}
	s.RegisterModules(cfg.Modules)
	s.RegisterCodec()
	return s
}

// RegisterModules registers the RPC services associated with the given API modules
func (s *Server) RegisterModules(mods []string) {

	for _, mod := range mods {
		log.Debug("[rpc] Enabling rpc module", "module", mod)
		var srvc interface{}
		switch mod {
		case "system":
			srvc = modules.NewSystemModule(s.networkAPI)
		case "author":
			srvc = modules.NewAuthorModule(s.coreAPI, s.transactionQueueAPI)
		default:
			log.Warn("[rpc] Unrecognized module", "module", mod)
			continue
		}

		err := s.RegisterService(srvc, mod)

		if err != nil {
			log.Warn("[rpc] Failed to register module", "mod", mod, "err", err)
		}
		r := mux.NewRouter()
		r.Handle("/", s.rpcServer)
	}
}

// RegisterCodec set the codec for the server.
func (s *Server) RegisterCodec() {
	// use our DotUpCodec which will capture methods passed in json as _x that is
	//  underscore followed by lower case letter, instead of default RPC calls which
	//  use . followed by Upper case letter
	s.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json")
	s.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json;charset=UTF-8")
}

// RegisterService adds a service to the servers service map.
func (s *Server) RegisterService(receiver interface{}, name string) error {
	return s.rpcServer.RegisterService(receiver, name)
}
