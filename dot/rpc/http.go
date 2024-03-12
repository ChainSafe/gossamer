// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/subscription"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/websocket"
)

// HTTPServer gateway for RPC server
type HTTPServer struct {
	logger       *log.Logger
	rpcServer    *rpc.Server // Actual RPC call handler
	serverConfig *HTTPServerConfig
	wsConns      []*subscription.WSConn
}

func (h *HTTPServer) Pause() error {
	//TODO implement me
	panic("implement me")
}

// HTTPServerConfig configures the HTTPServer
type HTTPServerConfig struct {
	LogLvl              log.Level
	BlockAPI            BlockAPI
	StorageAPI          StorageAPI
	NetworkAPI          NetworkAPI
	CoreAPI             CoreAPI
	BlockProducerAPI    BlockProducerAPI
	BlockFinalityAPI    BlockFinalityAPI
	TransactionQueueAPI TransactionStateAPI
	RPCAPI              API
	SystemAPI           SystemAPI
	SyncStateAPI        SyncStateAPI
	SyncAPI             SyncAPI
	NodeStorage         *runtime.NodeStorage
	RPCUnsafe           bool
	RPCExternal         bool
	RPCUnsafeExternal   bool
	Host                string
	RPCPort             uint32
	WSExternal          bool
	WSUnsafeExternal    bool
	WSPort              uint32
	Modules             []string
}

func (h *HTTPServerConfig) rpcUnsafeEnabled() bool {
	return h.RPCUnsafe || h.RPCUnsafeExternal
}

func (h *HTTPServerConfig) wsUnsafeEnabled() bool {
	return h.WSUnsafeExternal
}

func (h *HTTPServerConfig) exposeWS() bool {
	return h.WSExternal || h.WSUnsafeExternal
}

func (h *HTTPServerConfig) exposeRPC() bool {
	return h.RPCExternal || h.RPCUnsafeExternal
}

var logger *log.Logger

// NewHTTPServer creates a new http server and registers an associated rpc server
func NewHTTPServer(cfg *HTTPServerConfig) *HTTPServer {
	logger = log.NewFromGlobal(log.AddContext("pkg", "rpc"))
	logger.Patch(log.SetLevel(cfg.LogLvl))

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
		h.logger.Debug("Enabling rpc module " + mod)
		var srvc interface{}
		switch mod {
		case "system":
			srvc = modules.NewSystemModule(h.serverConfig.NetworkAPI, h.serverConfig.SystemAPI,
				h.serverConfig.CoreAPI, h.serverConfig.StorageAPI, h.serverConfig.TransactionQueueAPI,
				h.serverConfig.BlockAPI, h.serverConfig.SyncAPI)
		case "author":
			srvc = modules.NewAuthorModule(h.logger, h.serverConfig.CoreAPI, h.serverConfig.TransactionQueueAPI)
		case "chain":
			srvc = modules.NewChainModule(h.serverConfig.BlockAPI)
		case "grandpa":
			srvc = modules.NewGrandpaModule(h.serverConfig.BlockAPI, h.serverConfig.BlockFinalityAPI)
		case "state":
			srvc = modules.NewStateModule(h.serverConfig.NetworkAPI, h.serverConfig.StorageAPI,
				h.serverConfig.CoreAPI, h.serverConfig.BlockAPI)
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
			h.logger.Warn("Unrecognised module: " + mod)
			continue
		}

		err := h.rpcServer.RegisterService(srvc, mod)
		if err != nil {
			h.logger.Warnf("Failed to register module %s: %s", mod, err)
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

	h.logger.Infof("Starting HTTP Server on host %s and port %d...", h.serverConfig.Host, h.serverConfig.RPCPort)
	r := mux.NewRouter()
	r.Handle("/", h.rpcServer)

	validate := validator.New()
	// Add custom validator for `common.Hash`
	validate.RegisterCustomTypeFunc(common.HashValidator, common.Hash{})

	h.rpcServer.RegisterValidateRequestFunc(rpcValidator(h.serverConfig, validate))

	go func() {
		server := &http.Server{
			Addr:              fmt.Sprintf(":%d", h.serverConfig.RPCPort),
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           r,
		}

		err := server.ListenAndServe()
		if err != nil {
			h.logger.Errorf("http error: %s", err)
		}
	}()

	if !h.serverConfig.exposeWS() {
		return nil
	}

	h.logger.Infof("Starting WebSocket Server on host %s and port %d...",
		h.serverConfig.Host, h.serverConfig.WSPort)
	ws := mux.NewRouter()
	ws.Handle("/", h)
	go func() {
		wsServer := &http.Server{
			Addr:              fmt.Sprintf(":%d", h.serverConfig.WSPort),
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           ws,
		}

		err := wsServer.ListenAndServe()
		if err != nil {
			h.logger.Errorf("http error: %s", err)
		}
	}()

	return nil
}

// Stop stops the server
func (h *HTTPServer) Stop() error {
	if h.serverConfig.exposeWS() {
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
				h.logger.Errorf("error closing websocket connection: %s", err)
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
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					logger.Errorf("unable to parse remote address %s: %s", ip, err)
					return false
				}

				f := LocalhostFilter()
				if allowed := f.Allowed(ip); allowed {
					return true
				}

				logger.Debug("external websocket request refused")
				return false
			}

			return true
		},
	}

	ws, err := upg.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("websocket upgrade failed: %s", err)
		return
	}
	// create wsConn
	wsc := NewWSConn(ws, h.serverConfig)
	h.wsConns = append(h.wsConns, wsc)

	go wsc.HandleConn()
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
