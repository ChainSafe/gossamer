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

// import (
// 	"net/http"
// 	"reflect"
// 	"strings"

// 	"github.com/ChainSafe/gossamer/internal/api"
// 	log "github.com/ChainSafe/log15"
// 	peer "github.com/libp2p/go-libp2p-core/peer"
// )

// //System is an RPC system
// type System struct {
// 	node Node
// 	api  *api.Api
// }

// type Node struct {
// 	chain   string
// 	Health  string
// 	name    string
// 	peers   []peer.AddrInfo
// 	version string
// }

// func (s *System) chain() string {
// 	return s.chain()
// }

// func (s *System) name() string {
// 	return s.name()
// }

// func (s *System) version() string {
// 	return s.version()
// }

// func (s *System) peers() string {
// 	return s.peers()
// }

// // ServeSystem handles http requests to the RPC server.
// func (s *System) ServeSystem(w http.ResponseWriter, r *http.Request) {
// 	log.Debug("[rpc] Serving System request...")
// 	if r.Method != "POST" {
// 		WriteError(w, http.StatusMethodNotAllowed, "rpc: Only accepts POST requests, got: "+r.Method)
// 	}
// 	contentType := r.Header.Get("Content-Type")
// 	idx := strings.Index(contentType, ";")
// 	if idx != -1 {
// 		contentType = contentType[:idx]
// 	}
// 	if contentType != "application/json" {
// 		WriteError(w, http.StatusUnsupportedMediaType, "rpc: Only application/json content allowed, got: "+r.Header.Get("Content-Type"))
// 	}
// 	log.Debug("[rpc] Got application/json request, proceeding...")
// 	codecReq := s.codec.NewRequest(r)
// 	method, errMethod := codecReq.Method()
// 	if errMethod != nil {
// 		codecReq.WriteError(w, http.StatusBadRequest, errMethod)
// 	}
// 	serviceSpec, methodSpec, errGet := s.services.get(method)
// 	if errGet != nil {
// 		codecReq.WriteError(w, http.StatusBadRequest, errGet)
// 		return
// 	}

// 	args := reflect.New(methodSpec.argsType)
// 	if errRead := codecReq.ReadRequest(args.Interface()); errRead != nil {
// 		codecReq.WriteError(w, http.StatusBadRequest, errRead)
// 	}

// 	reply := reflect.New(methodSpec.replyType)
// 	errValue := methodSpec.method.Func.Call([]reflect.Value{
// 		serviceSpec.rcvr,
// 		reflect.ValueOf(r),
// 		args,
// 		reply,
// 	})

// 	var errResult error
// 	statusCode := http.StatusOK
// 	errInter := errValue[0].Interface()
// 	if errInter != nil {
// 		statusCode = http.StatusBadRequest
// 		errResult = errInter.(error)
// 	}

// 	// Encode the response.
// 	if errResult == nil {
// 		codecReq.WriteResponse(w, reply.Interface())
// 	} else {
// 		codecReq.WriteError(w, statusCode, errResult)
// 	}
// }

// // NewSystem creates a new System.
// func NewSystem(mods []api.Module, api *api.Api) *System {
// 	s := &System{
// 		services: new(serviceMap),
// 	}

// 	s.RegisterModules(mods)

// 	return s
// }

// // // NewSystemServer creates a new system server and registers an associated rpc server
// // func NewSystemServer(api *api.Api, codec Codec, cfg *Config) *HttpServer {
// // 	server := &SystemServer{
// // 		cfg:       cfg,
// // 		rpcServer: NewSystemServer(cfg.Modules, api),
// // 	}

// // 	server.rpcServer.RegisterCodec(codec)

// // 	return server
// // }
