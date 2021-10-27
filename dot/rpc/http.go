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
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/subscription"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	log "github.com/ChainSafe/log15"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/websocket"
)

// HTTPServer gateway for RPC server
type HTTPServer struct {
	logger       log.Logger
	rpcServer    *rpc.Server // Actual RPC call handler
	serverConfig *HTTPServerConfig
	wsConns      []*subscription.WSConn
}

// HTTPServerConfig configures the HTTPServer
type HTTPServerConfig struct {
	LogLvl              log.Lvl
	BlockAPI            modules.BlockAPI
	StorageAPI          modules.StorageAPI
	NetworkAPI          modules.NetworkAPI
	CoreAPI             modules.CoreAPI
	BlockProducerAPI    modules.BlockProducerAPI
	BlockFinalityAPI    modules.BlockFinalityAPI
	TransactionQueueAPI modules.TransactionStateAPI
	RPCAPI              modules.RPCAPI
	SystemAPI           modules.SystemAPI
	SyncStateAPI        modules.SyncStateAPI
	NodeStorage         *runtime.NodeStorage
	RPC                 bool
	RPCExternal         bool
	RPCUnsafe           bool
	RPCUnsafeExternal   bool
	Host                string
	RPCPort             uint32
	WS                  bool
	WSExternal          bool
	WSUnsafe            bool
	WSUnsafeExternal    bool
	WSPort              uint32
	Modules             []string
}

func (h *HTTPServerConfig) rpcUnsafeEnabled() bool {
	return h.RPCUnsafe || h.RPCUnsafeExternal
}

func (h *HTTPServerConfig) wsUnsafeEnabled() bool {
	return h.WSUnsafe || h.WSUnsafeExternal
}

func (h *HTTPServerConfig) exposeWS() bool {
	return h.WSExternal || h.WSUnsafeExternal
}

func (h *HTTPServerConfig) exposeRPC() bool {
	return h.RPCExternal || h.RPCUnsafeExternal
}

var logger log.Logger

// NewHTTPServer creates a new http server and registers an associated rpc server
func NewHTTPServer(cfg *HTTPServerConfig) *HTTPServer {
	logger = log.New("pkg", "rpc")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	h = log.CallerFileHandler(h)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))

	server := &HTTPServer{
		logger:       logger,
		rpcServer:    rpc.NewServer(),
		serverConfig: cfg,
	}

	server.RegisterModules(cfg.Modules)
	return server
}

// RegisterModules registers the RPC services associated with the given API modules
func (h *HTTPServer) RegisterModules(mods []string) {
	for _, mod := range mods {
		h.logger.Debug("Enabling rpc module", "module", mod)
		var srvc interface{}
		switch mod {
		case "system":
			srvc = modules.NewSystemModule(h.serverConfig.NetworkAPI, h.serverConfig.SystemAPI,
				h.serverConfig.CoreAPI, h.serverConfig.StorageAPI, h.serverConfig.TransactionQueueAPI, h.serverConfig.BlockAPI)
		case "author":
			srvc = modules.NewAuthorModule(h.logger, h.serverConfig.CoreAPI, h.serverConfig.TransactionQueueAPI)
		case "chain":
			srvc = modules.NewChainModule(h.serverConfig.BlockAPI)
		case "grandpa":
			srvc = modules.NewGrandpaModule(h.serverConfig.BlockAPI, h.serverConfig.BlockFinalityAPI)
		case "state":
			srvc = modules.NewStateModule(h.serverConfig.NetworkAPI, h.serverConfig.StorageAPI, h.serverConfig.CoreAPI)
		case "rpc":
			srvc = modules.NewRPCModule(h.serverConfig.RPCAPI)
		case "dev":
			srvc = modules.NewDevModule(h.serverConfig.BlockProducerAPI, h.serverConfig.NetworkAPI)
		case "offchain":
			srvc = modules.NewOffchainModule(h.serverConfig.NodeStorage)
		case "childstate":
			srvc = modules.NewChildStateModule(h.serverConfig.StorageAPI, h.serverConfig.BlockAPI)
		case "syncstate":
			srvc = modules.NewSyncStateModule(h.serverConfig.SyncStateAPI)
		case "payment":
			srvc = modules.NewPaymentModule(h.serverConfig.BlockAPI)
		default:
			h.logger.Warn("Unrecognised module", "module", mod)
			continue
		}

		err := h.rpcServer.RegisterService(srvc, mod)
		if err != nil {
			h.logger.Warn("Failed to register module", "mod", mod, "err", err)
		}

		h.serverConfig.RPCAPI.BuildMethodNames(srvc, mod)
	}
}

// Start registers the rpc handler function and starts the rpc http and websocket server
func (h *HTTPServer) Start() error {
	// use our DotUpCodec which will capture methods passed in json as _x that is
	//  underscore followed by lower case letter, instead of default RPC calls which
	//  use . followed by Upper case letter
	h.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json")
	h.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json;charset=UTF-8")

	h.logger.Info("Starting HTTP Server...", "host", h.serverConfig.Host, "port", h.serverConfig.RPCPort)
	r := mux.NewRouter()
	r.Handle("/", h.rpcServer)

	validate := validator.New()
	// Add custom validator for `common.Hash`
	validate.RegisterCustomTypeFunc(common.HashValidator, common.Hash{})

	h.rpcServer.RegisterValidateRequestFunc(rpcValidator(h.serverConfig, validate))

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", h.serverConfig.RPCPort), r)
		if err != nil {
			h.logger.Error("http error", "err", err)
		}
	}()

	if !h.serverConfig.WS {
		return nil
	}

	h.logger.Info("Starting WebSocket Server...", "host", h.serverConfig.Host, "port", h.serverConfig.WSPort)
	ws := mux.NewRouter()
	ws.Handle("/", h)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", h.serverConfig.WSPort), ws)
		if err != nil {
			h.logger.Error("http error", "err", err)
		}
	}()

	return nil
}

// Stop stops the server
func (h *HTTPServer) Stop() error {
	if h.serverConfig.WS {
		// close all channels and websocket connections
		for _, conn := range h.wsConns {
			for _, sub := range conn.Subscriptions {
				switch v := sub.(type) {
				case *subscription.StorageObserver:
					h.serverConfig.StorageAPI.UnregisterStorageObserver(v)
				case *subscription.BlockListener:
					h.serverConfig.BlockAPI.FreeImportedBlockNotifierChannel(v.Channel)
				}
			}

			err := conn.Wsconn.Close()
			if err != nil {
				h.logger.Error("error closing websocket connection", "error", err)
			}
		}
	}
	return nil
}

// ServeHTTP implemented to handle WebSocket connections
func (h *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upg = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if !h.serverConfig.exposeWS() {
				ip, _, error := net.SplitHostPort(r.RemoteAddr)
				if error != nil {
					logger.Error("unable to parse IP", "error")
					return false
				}

				f := LocalhostFilter()
				if allowed := f.Allowed(ip); allowed {
					return true
				}

				logger.Debug("external websocket request refused", "error")
				return false
			}

			return true
		},
	}

	ws, err := upg.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}
	// create wsConn
	wsc := NewWSConn(ws, h.serverConfig)
	h.wsConns = append(h.wsConns, wsc)

	go wsc.HandleComm()
}

// NewWSConn to create new WebSocket Connection struct
func NewWSConn(conn *websocket.Conn, cfg *HTTPServerConfig) *subscription.WSConn {
	c := &subscription.WSConn{
		UnsafeEnabled: cfg.wsUnsafeEnabled(),
		Wsconn:        conn,
		Subscriptions: make(map[uint32]subscription.Listener),
		StorageAPI:    cfg.StorageAPI,
		BlockAPI:      cfg.BlockAPI,
		CoreAPI:       cfg.CoreAPI,
		TxStateAPI:    cfg.TransactionQueueAPI,
		RPCHost:       fmt.Sprintf("http://%s:%d/", cfg.Host, cfg.RPCPort),
		HTTP: &http.Client{
			Timeout: time.Second * 30,
		},
	}
	return c
}
