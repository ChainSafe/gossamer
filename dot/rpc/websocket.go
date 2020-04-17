package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/websocket"
)

// HTTPServerWS gateway for WebSocket RPC server
type HTTPServerWS struct {
	rpcServer    *rpc.Server // Actual RPC call handler
	serverConfig *HTTPServerConfigWS
}

// HTTPServerConfigWS configures the WebSocket HTTPServer
type HTTPServerConfigWS struct {
	HTTPServerConfig *HTTPServerConfig
	RPCPort          uint32
}

// NewHTTPServerWS creates a new http server and registers an associated rpc server
func NewHTTPServerWS(cfg *HTTPServerConfigWS) *HTTPServerWS {
	server := &HTTPServerWS{
		rpcServer:    rpc.NewServer(),
		serverConfig: cfg,
	}

	server.RegisterModules(cfg.HTTPServerConfig.Modules)
	return server
}

// RegisterModules registers the RPC services associated with the given API modules
func (h *HTTPServerWS) RegisterModules(mods []string) {

	for _, mod := range mods {
		log.Debug("[rpc] Enabling rpc module", "module", mod)
		var srvc interface{}
		switch mod {
		case "system":
			srvc = modules.NewSystemModule(h.serverConfig.HTTPServerConfig.NetworkAPI)
		case "author":
			srvc = modules.NewAuthorModule(h.serverConfig.HTTPServerConfig.CoreAPI, h.serverConfig.HTTPServerConfig.TransactionQueueAPI)
		case "chain":
			srvc = modules.NewChainModule(h.serverConfig.HTTPServerConfig.BlockAPI)
		default:
			log.Warn("[rpc] Unrecognized module", "module", mod)
			continue
		}

		err := h.rpcServer.RegisterService(srvc, mod)

		if err != nil {
			log.Warn("[rpc] Failed to register module", "mod", mod, "error", err)
		}
	}
}

// Start registers the rpc handler function and starts the server listening on `h.port`
func (h *HTTPServerWS) Start() error {
	// use our DotUpCodec which will capture methods passed in json as _x that is
	//  underscore followed by lower case letter, instead of default RPC calls which
	//  use . followed by Upper case letter
	h.rpcServer.RegisterCodec(NewDotUpCodec(), "application/json")
	r := mux.NewRouter()
	r.Handle("/", h.rpcServer)

	r.Use(h.websocketHandler)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", h.serverConfig.HTTPServerConfig.Port), r)
		if err != nil {
			log.Error("[rpc] http error", "error", err)
		}
	}()

	return nil
}

// Stop stops the server
func (h *HTTPServerWS) Stop() error {
	return nil
}

func (h *HTTPServerWS) websocketHandler(next http.Handler) http.Handler {
	var upg = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upg.Upgrade(w, r, nil)
		if err != nil {
			log.Error("[rpc] websocket upgrade failed", "error", err)
			return
		}

		rpcHost := fmt.Sprintf("http://%s:%d/", h.serverConfig.HTTPServerConfig.Host, h.serverConfig.RPCPort)
		for {
			_, mbytes, err := ws.ReadMessage()
			if err != nil {
				log.Error("[rpc] websocket failed to read message", "error", err)
				return
			}
			log.Trace("[rpc] websocket received", "message", fmt.Sprintf("%s", mbytes))
			client := &http.Client{}
			buf := &bytes.Buffer{}
			_, err = buf.Write(mbytes)
			if err != nil {
				log.Error("[rpc] failed to write message to buffer", "error", err)
				return
			}

			req, err := http.NewRequest("POST", rpcHost, buf)
			if err != nil {
				log.Error("[rpc] failed request to rpc service", "error", err)
				return
			}

			req.Header.Set("Content-Type", "application/json;")

			res, err := client.Do(req)
			if err != nil {
				log.Error("[rpc] websocket error calling rpc", "error", err)
				return
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Error("[rpc] error reading response body", "error", err)
				return
			}

			err = res.Body.Close()
			if err != nil {
				log.Error("[rpc] error closing response body", "error", err)
				return
			}
			var wsSend interface{}
			err = json.Unmarshal(body, &wsSend)
			if err != nil {
				log.Error("[rpc] error unmarshal rpc response", "error", err)
				return
			}

			err = ws.WriteJSON(wsSend)
			if err != nil {
				log.Error("[rpc] error writing json response", "error", err)
				return
			}
		}

	})
}
