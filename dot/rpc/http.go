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
	"net/http"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"

	log "github.com/ChainSafe/log15"
)

// HTTPServer gateway for RPC server
type HTTPServer struct {
	rpcServer    *rpc.Server // Actual RPC call handler
	serverConfig *HTTPServerConfig
	rpcMethods   []string // list of method names offered by rpc
}

// HTTPServerConfig configures the HTTPServer
type HTTPServerConfig struct {
	BlockAPI            modules.BlockAPI
	StorageAPI          modules.StorageAPI
	NetworkAPI          modules.NetworkAPI
	CoreAPI             modules.CoreAPI
	TransactionQueueAPI modules.TransactionQueueAPI
	Host                string
	RPCPort             uint32
	WSPort              uint32
	Modules             []string
}

// NewHTTPServer creates a new http server and registers an associated rpc server
func NewHTTPServer(cfg *HTTPServerConfig) *HTTPServer {
	server := &HTTPServer{
		rpcServer:    rpc.NewServer(),
		serverConfig: cfg,
	}

	server.RegisterModules(cfg.Modules)
	return server
}

// RegisterModules registers the RPC services associated with the given API modules
func (h *HTTPServer) RegisterModules(mods []string) {

	for _, mod := range mods {
		log.Debug("[rpc] Enabling rpc module", "module", mod)
		var srvc interface{}
		switch mod {
		case "system":
			srvc = modules.NewSystemModule(h.serverConfig.NetworkAPI)
		case "author":
			srvc = modules.NewAuthorModule(h.serverConfig.CoreAPI, h.serverConfig.TransactionQueueAPI)
		case "chain":
			srvc = modules.NewChainModule(h.serverConfig.BlockAPI)
		case "state":
			srvc = modules.NewStateModule(h.serverConfig.NetworkAPI, h.serverConfig.StorageAPI, h.serverConfig.CoreAPI)
		case "rpc":
			srvc = modules.NewRPCModule(h)
		default:
			log.Warn("[rpc] Unrecognized module", "module", mod)
			continue
		}

		err := h.rpcServer.RegisterService(srvc, mod)

		if err != nil {
			log.Warn("[rpc] Failed to register module", "mod", mod, "err", err)
		}

		h.buildMethodNames(srvc, mod)
	}
}

// Start registers the rpc handler function and starts the rpc http and websocket server
func (h *HTTPServer) Start() error {
	// use our DotUpCodec which will capture methods passed in json as _x that is
	//  underscore followed by lower case letter, instead of default RPC calls which
	//  use . followed by Upper case letter
	h.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json")
	h.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json;charset=UTF-8")

	log.Info("[rpc] Starting HTTP Server...", "host", h.serverConfig.Host, "port", h.serverConfig.RPCPort)
	r := mux.NewRouter()
	r.Handle("/", h.rpcServer)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", h.serverConfig.RPCPort), r)
		if err != nil {
			log.Error("[rpc] http error", "err", err)
		}
	}()

	log.Info("[rpc] Starting WebSocket Server...", "host", h.serverConfig.Host, "port", h.serverConfig.WSPort)
	ws := mux.NewRouter()
	ws.Handle("/", h)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", h.serverConfig.WSPort), ws)
		if err != nil {
			log.Error("[rpc] http error", "err", err)
		}
	}()

	return nil
}

// Stop stops the server
func (h *HTTPServer) Stop() error {
	return nil
}

// Methods returns list of methods available via RPC call
func (h *HTTPServer) Methods() []string {
	return h.rpcMethods
}

var (
	// Precompute the reflect.Type of error and http.Request
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfRequest = reflect.TypeOf((*http.Request)(nil)).Elem()
)

// this takes receiver interface and populates rpcMethods array with available
//  method names
func (h *HTTPServer) buildMethodNames(rcvr interface{}, name string) {
	rcvrType := reflect.TypeOf(rcvr)
	for i := 0; i < rcvrType.NumMethod(); i++ {
		method := rcvrType.Method(i)
		mtype := method.Type
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs four ins: receiver, *http.Request, *args, *reply.
		if mtype.NumIn() != 4 {
			continue
		}
		// First argument must be a pointer and must be http.Request.
		reqType := mtype.In(1)
		if reqType.Kind() != reflect.Ptr || reqType.Elem() != typeOfRequest {
			continue
		}
		// Second argument must be a pointer and must be exported.
		args := mtype.In(2)
		if args.Kind() != reflect.Ptr || !isExportedOrBuiltin(args) {
			continue
		}
		// Third argument must be a pointer and must be exported.
		reply := mtype.In(3)
		if reply.Kind() != reflect.Ptr || !isExportedOrBuiltin(reply) {
			continue
		}
		// Method needs one out: error.
		if mtype.NumOut() != 1 {
			continue
		}
		if returnType := mtype.Out(0); returnType != typeOfError {
			continue
		}

		h.rpcMethods = append(h.rpcMethods, name+"_"+strings.ToLower(string(method.Name[0]))+method.Name[1:])
	}
}

// isExported returns true of a string is an exported (upper case) name.
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// isExportedOrBuiltin returns true if a type is exported or a builtin.
func isExportedOrBuiltin(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}
