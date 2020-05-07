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
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)
// consts to represent subscription type
const (
	SUB_NEW_HEAD = iota
	SUB_FINALIZED_HEAD
	SUB_STORAGE
)

type SubscriptionBaseResponseJSON struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  interface{} `json:"params"`
	Subscription uint32 `json:"subscription"`
}

func newSubcriptionBaseResponseJSON(sub uint32) SubscriptionBaseResponseJSON {
	return SubscriptionBaseResponseJSON{
		Jsonrpc:      "2.0",
		Subscription: sub,
	}
}

// SubscriptionResponseJSON for json subscription responses
type SubscriptionResponseJSON struct {
	Jsonrpc string   `json:"jsonrpc"`
	Result  uint32   `json:"result"`
	ID      float64 `json:"id"`
}

func newSubscriptionResponseJSON() SubscriptionResponseJSON {
	return SubscriptionResponseJSON{
		Jsonrpc: "2.0",
		Result:  0,
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
			return true
		},
	}
	ws, err := upg.Upgrade(w, r, nil)
	if err != nil {
		log.Error("[rpc] websocket upgrade failed", "error", err)
		return
	}
	for {
		rpcHost := fmt.Sprintf("http://%s:%d/", h.serverConfig.Host, h.serverConfig.RPCPort)
		for {
			_, mbytes, err := ws.ReadMessage()
			if err != nil {
				log.Error("[rpc] websocket failed to read message", "error", err)
				return
			}
			log.Trace("[rpc] websocket received", "message", fmt.Sprintf("%s", mbytes))

			// determine if request is for subscribe method type
			var msg map[string]interface{}
			err = json.Unmarshal(mbytes, &msg)
			if err != nil {
				log.Error("[rpc] websocket failed to unmarshal request message", "error", err)
				res := &ErrorResponseJSON{
					Jsonrpc: "2.0",
					Error: &ErrorMessageJSON{
						Code:    big.NewInt(-32600),
						Message: "Invalid request",
					},
					ID: nil,
				}
				err = ws.WriteJSON(res)
				if err != nil {
					log.Error("[rpc] websocket failed write message", "error", err)
				}
				return
			}
			method := msg["method"]
			// if method contains subscribe, then register subscription
			if strings.Contains(fmt.Sprintf("%s", method), "subscribe") {
				mid := msg["id"].(float64)
				var subType int
				if method == "chain_subscribeNewHeads" ||
					method == "chain_subscribeNewHead" {
					subType = SUB_NEW_HEAD
				} else if method == "chain_subscribeStorage" {
					subType = SUB_STORAGE
				} else if method == "chain_subscribeFinalizedHeads" {
					subType = SUB_FINALIZED_HEAD
				}
				var e1 error
				_, e1 = h.registerSubscription(ws, mid, subType)
				if e1 != nil {
					log.Error("[rpc] failed to register subscription", "error", err)
				}
				continue
			}

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
	}
}

func (h *HTTPServer) registerSubscription(conn *websocket.Conn, reqID float64, subscriptionType int) (uint32, error) {
	wssub := h.serverConfig.WSSubscriptions
	if wssub == nil {
		wssub = make(map[uint32]*WebSocketSubscription)
	}
	sub := uint32(len(wssub)) + 1
	wss := &WebSocketSubscription{
		WSConnection:     conn,
		SubscriptionType: subscriptionType,
	}
	wssub[sub] = wss
	h.serverConfig.WSSubscriptions = wssub
	initRes := newSubscriptionResponseJSON()
	initRes.Result = sub
	initRes.ID = reqID

	return sub, conn.WriteJSON(initRes)
}

func (h *HTTPServer) blockReceivedListener() {
	blkRec := h.serverConfig.CoreAPI.GetBlockReceivedChannel()
	for {
		// receive block from BABE session
		block, ok := <-blkRec
		if ok {
			for i, sub := range h.serverConfig.WSSubscriptions {
				if sub.SubscriptionType == SUB_NEW_HEAD {
					head := modules.HeaderToJSON(*block.Header)
					headM := make(map[string]interface{})
					headM["result"] = head
					res := newSubcriptionBaseResponseJSON(i)
					res.Method = "chain_newHead"
					res.Params = headM
					sub.WSConnection.WriteJSON(res)
				}
				
			}
		} else {
			// if not ok, connection was closed, so re-find channel
			blkRec = h.serverConfig.CoreAPI.GetBlockReceivedChannel()
		}
	}
}