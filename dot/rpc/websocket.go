// Copyright 2020 ChainSafe Systems (ON) Corp.
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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
)

// SubscriptionBaseResponseJSON for base json response
type SubscriptionBaseResponseJSON struct {
	Jsonrpc      string      `json:"jsonrpc"`
	Method       string      `json:"method"`
	Params       interface{} `json:"params"`
	Subscription int        `json:"subscription"`
}

func newSubcriptionBaseResponseJSON(subID int) SubscriptionBaseResponseJSON {
	return SubscriptionBaseResponseJSON{
		Jsonrpc:      "2.0",
		Subscription: subID,
	}
}

// SubscriptionResponseJSON for json subscription responses
type SubscriptionResponseJSON struct {
	Jsonrpc string  `json:"jsonrpc"`
	Result  int    `json:"result"`
	ID      float64 `json:"id"`
}

func newSubscriptionResponseJSON(subID int, reqID float64) SubscriptionResponseJSON {
	return SubscriptionResponseJSON{
		Jsonrpc: "2.0",
		Result:  subID,
		ID:      reqID,
	}
}

// ErrorResponseJSON json for error responses
type ErrorResponseJSON struct {
	Jsonrpc string            `json:"jsonrpc"`
	Error   *ErrorMessageJSON `json:"error"`
	ID      *big.Int          `json:"id"`
}

// ErrorMessageJSON json for error messages
type ErrorMessageJSON struct {
	Code    *big.Int `json:"code"`
	Message string   `json:"message"`
}

// ServeHTTP implemented to handle WebSocket connections
func (h *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upg = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // todo determine how this should check orgigin
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

	go wsc.handleComm()
}

// NewWSConn to create new WebSocket Connection struct
func NewWSConn(conn *websocket.Conn, cfg *HTTPServerConfig) *WSConn {
	logger := log.New("pkg", "rpc")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))
	c := &WSConn{
		wsconn:       conn,
		serverConfig: cfg,
		logger:       logger,
		subscriptions: make(map[int]Listener),
		blockSubChannels: make(map[int]byte),
		storageSubChannels: make(map[int]byte),
	}
	return c
}

func (c *WSConn) safeSend(msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.wsconn.WriteJSON(msg)
}

func (c *WSConn) handleComm() {
	for {
		_, mbytes, err := c.wsconn.ReadMessage()
		if err != nil {
			c.logger.Error("websocket failed to read message", "error", err)
			return
		}
		c.logger.Debug("websocket received", "message", fmt.Sprintf("%s", mbytes))

		// determine if request is for subscribe method type
		var msg map[string]interface{}
		err = json.Unmarshal(mbytes, &msg)
		if err != nil {
			c.logger.Error("websocket failed to unmarshal request message", "error", err)
			res := &ErrorResponseJSON{
				Jsonrpc: "2.0",
				Error: &ErrorMessageJSON{
					Code:    big.NewInt(-32600),
					Message: "Invalid request",
				},
				ID: nil,
			}
			err = c.safeSend(res)
			if err != nil {
				c.logger.Error("websocket failed write message", "error", err)
			}
			continue
		}
		method := msg["method"]
		// if method contains subscribe, then register subscription
		if strings.Contains(fmt.Sprintf("%s", method), "subscribe") {
			reqid := msg["id"].(float64)
			params := msg["params"]
			switch method {
			case "chain_subscribeNewHeads", "chain_subscribeNewHead":
				bl, err1 := c.initBlockListener(reqid)
				if err1 != nil {
					c.logger.Error("failed to create block listener", "error", err)
				}
				c.startListener(bl)
			case "state_subscribeStorage":
				scl, err2 := c.initStorageChangeListener(reqid, params)
				if err2 != nil {
					c.logger.Error("failed to create state change listener", "error", err)
				}
				c.startListener(scl)
			case "chain_subscribeFinalizedHeads":
			}
			continue
		}

		// handle non-subscribe calls
		client := &http.Client{}
		buf := &bytes.Buffer{}
		_, err = buf.Write(mbytes)
		if err != nil {
			c.logger.Error("failed to write message to buffer", "error", err)
			return
		}

		rpcHost := fmt.Sprintf("http://%s:%d/", c.serverConfig.Host, c.serverConfig.RPCPort)
		req, err := http.NewRequest("POST", rpcHost, buf)
		if err != nil {
			c.logger.Error("failed request to rpc service", "error", err)
			return
		}

		req.Header.Set("Content-Type", "application/json;")

		res, err := client.Do(req)
		if err != nil {
			c.logger.Error("websocket error calling rpc", "error", err)
			return
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			c.logger.Error("error reading response body", "error", err)
			return
		}

		err = res.Body.Close()
		if err != nil {
			c.logger.Error("error closing response body", "error", err)
			return
		}
		var wsSend interface{}
		err = json.Unmarshal(body, &wsSend)
		if err != nil {
			c.logger.Error("error unmarshal rpc response", "error", err)
			return
		}

		err = c.safeSend(wsSend)
		if err != nil {
			c.logger.Error("error writing json response", "error", err)
			return
		}
	}
}
func (c *WSConn)startListener(lid int) {
	go c.subscriptions[lid].Listen()
}
type Listener interface {
	Listen()
}

// StorageChangeListener for listening to state change channels
type StorageChangeListener struct {
	channel chan *state.KeyValue
	filter  map[string]bool
	wsconn  *WSConn
	chanID  byte
	subID int
}

func (c *WSConn) initStorageChangeListener(reqID float64, params interface{}) (int, error) {
	scl := &StorageChangeListener{
		channel: make(chan *state.KeyValue),
		filter:  make(map[string]bool),
		wsconn:  c,
	}
	pA := params.([]interface{})
	for _, param := range pA {
		scl.filter[param.(string)] = true
	}

	chanID, err := c.serverConfig.StorageAPI.RegisterStorageChangeChannel(scl.channel)
	if err != nil {
		return 0, err
	}
	scl.chanID = chanID

	c.qtyListeners++
	scl.subID = c.qtyListeners
	c.subscriptions[scl.subID] = scl
	c.storageSubChannels[scl.subID] = chanID

	initRes := newSubscriptionResponseJSON(scl.subID, reqID)
	err = c.safeSend(initRes)
	if err != nil {
		return 0, err
	}
	return scl.subID, nil
}

// make this two parts, one in init and return chan ID, second to lister/respond
func (l *StorageChangeListener) Listen() {
	for change := range l.channel {
		if change != nil {
			//check if change key is in subscription filter
			cKey := common.BytesToHex(change.Key)
			if len(l.filter) > 0 && !l.filter[cKey] {
				continue
			}

			changeM := make(map[string]interface{})
			changeM["result"] = []string{cKey, common.BytesToHex(change.Value)}
			res := newSubcriptionBaseResponseJSON(l.subID)
			res.Method = "state_storage"
			res.Params = changeM
			err := l.wsconn.safeSend(res)
			if err != nil {
				l.wsconn.logger.Error("error sending websocket message", "error", err)
			}
		}
	}
}

// BlockListener to handle listening for blocks channel
type BlockListener struct {
	channel chan *types.Block
	wsconn  *WSConn
	chanID  byte
	subID int
}

func (c *WSConn) initBlockListener(reqID float64) (int, error) {
	bl := &BlockListener{
		channel: make(chan *types.Block),
		wsconn:  c,
	}

	chanID, err := c.serverConfig.BlockAPI.RegisterImportedChannel(bl.channel)
	if err != nil {
		return 0, err
	}
	bl.chanID = chanID
	c.qtyListeners++
	bl.subID = c.qtyListeners
	c.subscriptions[bl.subID] = bl
	initRes := newSubscriptionResponseJSON(bl.subID, reqID)
	err = c.safeSend(initRes)
	if err != nil {
		return 0, err
	}
	return bl.subID, nil
}

func (l *BlockListener) Listen() {
	for block := range l.channel {
		if block != nil {
			head := modules.HeaderToJSON(*block.Header)
			headM := make(map[string]interface{})
			headM["result"] = head
			res := newSubcriptionBaseResponseJSON(l.subID)
			res.Method = "chain_newHead"
			res.Params = headM
			err := l.wsconn.safeSend(res)
			if err != nil {
				l.wsconn.logger.Error("error sending websocket message", "error", err)
			}
		}
	}
}
