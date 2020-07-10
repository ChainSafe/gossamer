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
	"github.com/ChainSafe/gossamer/dot/state"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

// consts to represent subscription type
const (
	SUB_NEW_HEAD = iota
	SUB_FINALIZED_HEAD
	SUB_STORAGE
)

// SubscriptionBaseResponseJSON for base json response
type SubscriptionBaseResponseJSON struct {
	Jsonrpc      string      `json:"jsonrpc"`
	Method       string      `json:"method"`
	Params       interface{} `json:"params"`
	Subscription uint32      `json:"subscription"`
}

func newSubcriptionBaseResponseJSON(sub uint32) SubscriptionBaseResponseJSON {
	return SubscriptionBaseResponseJSON{
		Jsonrpc:      "2.0",
		Subscription: sub,
	}
}

// SubscriptionResponseJSON for json subscription responses
type SubscriptionResponseJSON struct {
	Jsonrpc string  `json:"jsonrpc"`
	Result  uint32  `json:"result"`
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
			continue
		}
		method := msg["method"]
		// if method contains subscribe, then register subscription
		if strings.Contains(fmt.Sprintf("%s", method), "subscribe") {
			reqid := msg["id"].(float64)
			//var subType int
			switch method {
			case "chain_subscribeNewHeads", "chain_subscribeNewHead":
				//subType = SUB_NEW_HEAD
			case "state_subscribeStorage":
				//subType = SUB_STORAGE
				cr, err := h.createStateChangeListener(ws, reqid)
				if err != nil {
					log.Error("[rpc] websocket failed write message", "error", err)
				}
				go cr.startStateChangeListener()
			case "chain_subscribeFinalizedHeads":
				//subType = SUB_FINALIZED_HEAD
			}
			//params := msg["params"]
			var e1 error
			//_, e1 = h.registerSubscription(ws, mid, subType, params)
			
			if e1 != nil {
				// todo send error message to client
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

//func (h *HTTPServer) registerSubscription(conn *websocket.Conn, reqID float64, subscriptionType int, params interface{}) (uint32, error) {
//	wssub := h.serverConfig.WSSubscriptions
//	sub := uint32(len(wssub)) + 1
//	pA := params.([]interface{})
//	filter := make(map[string]bool)
//	for _, param := range pA {
//		filter[param.(string)] = true
//	}
//	wss := &WebSocketSubscription{
//		WSConnection:     conn,
//		SubscriptionType: subscriptionType,
//		Filter:           filter,
//	}
//	wssub[sub] = wss
//	h.serverConfig.WSSubscriptions = wssub
//	initRes := newSubscriptionResponseJSON()
//	initRes.Result = sub
//	initRes.ID = reqID
//
//	return sub, conn.WriteJSON(initRes)
//}
type StateChangeListener struct {
	Chan   chan *state.KeyValue
	WSConnection     *websocket.Conn
}

func (h *HTTPServer) createStateChangeListener(conn *websocket.Conn, reqID float64) (*StateChangeListener, error) {
	cr := &StateChangeListener{
		Chan: make(chan *state.KeyValue),
		WSConnection: conn,
	}
	h.stateChangeListener = append(h.stateChangeListener, cr)
	h.serverConfig.StorageAPI.RegisterStorageChangeChannel(cr.Chan)
	// TODO respond to client with subscription id
	initRes := newSubscriptionResponseJSON()
	//	initRes.Result = sub
	initRes.ID = reqID

	return cr, conn.WriteJSON(initRes)
}
func (cr *StateChangeListener) startStateChangeListener() {
	for change := range cr.Chan {
		fmt.Printf("change listener %v\n", change)
		if change != nil {
		//	for i, sub := range h.serverConfig.WSSubscriptions {
		//		if sub.SubscriptionType == SUB_STORAGE {
					// TODO check if change key is in subscription filter
					cKey := common.BytesToHex(change.Key)
		//			if len(sub.Filter) > 0 && !sub.Filter[cKey] {
		//				continue
		//			}
		//
					changeM := make(map[string]interface{})
					changeM["result"] = []string{cKey, common.BytesToHex(change.Value)}
					res := newSubcriptionBaseResponseJSON(1)  // todo handle subscription id
					res.Method = "state_storage"
					res.Params = changeM
					if cr.WSConnection != nil {
						err := cr.WSConnection.WriteJSON(res)
						if err != nil {
							log.Error("[rpc] error writing response", "error", err)
						}
					}
		//		}
		//	}
		}
	}
}

//func (h *HTTPServer)startStateChangeListener() {
//	// resister channel
//	h.storageChan = make(chan *state.KeyValue)
//	h.storageChanID, err = h.serverConfig.StorageAPI.RegisterStorageChangeChannel(h.storageChan)
//	if err != nil {
//		return err
//	}
//	go h.storageChangeListener()
//
//}

func (h *HTTPServer) blockReceivedListener() {
	if h.serverConfig.BlockAPI == nil {
		return
	}
// todo implement this
	//for block := range h.blockChan {
	//	if block != nil {
	//		for i, sub := range h.serverConfig.WSSubscriptions {
	//			if sub.SubscriptionType == SUB_NEW_HEAD {
	//				head := modules.HeaderToJSON(*block.Header)
	//				headM := make(map[string]interface{})
	//				headM["result"] = head
	//				res := newSubcriptionBaseResponseJSON(i)
	//				res.Method = "chain_newHead"
	//				res.Params = headM
	//				if sub.WSConnection != nil {
	//					err := sub.WSConnection.WriteJSON(res)
	//					if err != nil {
	//						log.Error("[rpc] error writing response", "error", err)
	//					}
	//				}
	//			}
	//		}
	//	}
	//}
}

//func (h *HTTPServer) storageChangeListener() {
//	if h.serverConfig.StorageAPI == nil {
//		return
//	}
//
//	for change := range h.storageChan {
//
//		if change != nil {
//			for i, sub := range h.serverConfig.WSSubscriptions {
//				if sub.SubscriptionType == SUB_STORAGE {
//					// check if change key is in subscription filter
//					cKey := common.BytesToHex(change.Key)
//					if len(sub.Filter) > 0 && !sub.Filter[cKey] {
//						continue
//					}
//
//					changeM := make(map[string]interface{})
//					changeM["result"] = []string{cKey, common.BytesToHex(change.Value)}
//					res := newSubcriptionBaseResponseJSON(i)
//					res.Method = "state_storage"
//					res.Params = changeM
//					if sub.WSConnection != nil {
//						err := sub.WSConnection.WriteJSON(res)
//						if err != nil {
//							log.Error("[rpc] error writing response", "error", err)
//						}
//					}
//				}
//			}
//		}
//	}
//}
