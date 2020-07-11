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
	"encoding/json"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
	"math/big"
	"net/http"
	"os"
	"strings"
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
	Subscription byte      `json:"subscription"`
}

func newSubcriptionBaseResponseJSON(sub byte) SubscriptionBaseResponseJSON {
	return SubscriptionBaseResponseJSON{
		Jsonrpc:      "2.0",
		Subscription: sub,
	}
}

// SubscriptionResponseJSON for json subscription responses
type SubscriptionResponseJSON struct {
	Jsonrpc string  `json:"jsonrpc"`
	Result  byte  `json:"result"`
	ID      float64 `json:"id"`
}

func newSubscriptionResponseJSON(subID byte, reqID float64) SubscriptionResponseJSON {
	return SubscriptionResponseJSON{
		Jsonrpc: "2.0",
		Result:  subID,
		ID: reqID,
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
			return true  // todo determine how this should check orgigin
		},
	}

	// todo create struct to hold ws connections so we can control writes to it.
	ws, err := upg.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}
	// create wsConn
	wsc := NewWSConn(ws, h.serverConfig)
	h.logger.Debug("WS UPGRADE", "ws", wsc)
	h.wsConns = append(h.wsConns, wsc)
	go wsc.handleComm()
	h.logger.Debug("STARTED HANDLE COMM");
	//rpcHost := fmt.Sprintf("http://%s:%d/", h.serverConfig.Host, h.serverConfig.RPCPort)
	//for {
	//	_, mbytes, err := ws.ReadMessage()
	//	if err != nil {
	//		h.logger.Error("websocket failed to read message", "error", err)
	//		return
	//	}
	//	h.logger.Debug("websocket received", "message", fmt.Sprintf("%s", mbytes))
	//
	//	// determine if request is for subscribe method type
	//	var msg map[string]interface{}
	//	err = json.Unmarshal(mbytes, &msg)
	//	if err != nil {
	//		h.logger.Error("websocket failed to unmarshal request message", "error", err)
	//		res := &ErrorResponseJSON{
	//			Jsonrpc: "2.0",
	//			Error: &ErrorMessageJSON{
	//				Code:    big.NewInt(-32600),
	//				Message: "Invalid request",
	//			},
	//			ID: nil,
	//		}
	//		err = ws.WriteJSON(res)
	//		if err != nil {
	//			h.logger.Error("websocket failed write message", "error", err)
	//		}
	//		continue
	//	}
	//	method := msg["method"]
	//	// if method contains subscribe, then register subscription
	//	if strings.Contains(fmt.Sprintf("%s", method), "subscribe") {
	//		reqid := msg["id"].(float64)
	//		//var subType int
	//		switch method {
	//		case "chain_subscribeNewHeads", "chain_subscribeNewHead":
	//			//subType = SUB_NEW_HEAD
	//		case "state_subscribeStorage":
	//			//subType = SUB_STORAGE
	//			cr, err := h.createStateChangeListener(ws, reqid)
	//			if err != nil {
	//				h.logger.Error("failed to create state change listener", "error", err)
	//			}
	//			go cr.startStateChangeListener()
	//		case "chain_subscribeFinalizedHeads":
	//			//subType = SUB_FINALIZED_HEAD
	//		}
	//		//params := msg["params"]
	//		var e1 error
	//		//_, e1 = h.registerSubscription(ws, mid, subType, params)
	//
	//		if e1 != nil {
	//			// todo send error message to client
	//			h.logger.Error("failed to register subscription", "error", err)
	//		}
	//		continue
	//	}
	//
	//	client := &http.Client{}
	//	buf := &bytes.Buffer{}
	//	_, err = buf.Write(mbytes)
	//	if err != nil {
	//		h.logger.Error("failed to write message to buffer", "error", err)
	//		return
	//	}
	//
	//	req, err := http.NewRequest("POST", rpcHost, buf)
	//	if err != nil {
	//		h.logger.Error("failed request to rpc service", "error", err)
	//		return
	//	}
	//
	//	req.Header.Set("Content-Type", "application/json;")
	//
	//	res, err := client.Do(req)
	//	if err != nil {
	//		h.logger.Error("websocket error calling rpc", "error", err)
	//		return
	//	}
	//
	//	body, err := ioutil.ReadAll(res.Body)
	//	if err != nil {
	//		h.logger.Error("error reading response body", "error", err)
	//		return
	//	}
	//
	//	err = res.Body.Close()
	//	if err != nil {
	//		h.logger.Error("error closing response body", "error", err)
	//		return
	//	}
	//	var wsSend interface{}
	//	err = json.Unmarshal(body, &wsSend)
	//	if err != nil {
	//		h.logger.Error("error unmarshal rpc response", "error", err)
	//		return
	//	}
	//
	//	err = ws.WriteJSON(wsSend)
	//	if err != nil {
	//		h.logger.Error("error writing json response", "error", err)
	//		return
	//	}
	//}

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
// todo consider logger?
func NewWSConn(conn *websocket.Conn, cfg *HTTPServerConfig) *WSConn {
	logger := log.New("pkg", "rpc")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))
	c := &WSConn{
		wsconn: conn,
		serverConfig: cfg,
		logger: logger,
	}
	return c
}
func (c *WSConn)handleComm()  {
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
				err = c.wsconn.WriteJSON(res)
				if err != nil {
					c.logger.Error("websocket failed write message", "error", err)
				}
				continue
			}
			method := msg["method"]
			// if method contains subscribe, then register subscription
			if strings.Contains(fmt.Sprintf("%s", method), "subscribe") {
				reqid := msg["id"].(float64)
				c.logger.Debug("Subscribe request", "r id", reqid)
				params := msg["params"]
				switch method {
				case "chain_subscribeNewHeads", "chain_subscribeNewHead":
		//			//subType = SUB_NEW_HEAD
				case "state_subscribeStorage":
		//			//subType = SUB_STORAGE
					scl, err := c.initStateChangeListener(reqid, params)
					if err != nil {
						c.logger.Error("failed to create state change listener", "error", err)
					}
					go scl.listen()
				case "chain_subscribeFinalizedHeads":
		//			//subType = SUB_FINALIZED_HEAD
				}
		//
		//		var e1 error
		//		//_, e1 = h.registerSubscription(ws, mid, subType, params)
		//
		//		if e1 != nil {
		//			// todo send error message to client
		//			h.logger.Error("failed to register subscription", "error", err)
		//		}
				continue
			}

	}
}
type StateChangeListener struct {
	channel   chan *state.KeyValue
	filter map[string]bool
	wsconn *WSConn
	subID byte
}

func (c *WSConn) initStateChangeListener(reqID float64, params interface{}) (*StateChangeListener, error){
	scl := &StateChangeListener{
		channel: make(chan *state.KeyValue),
		filter: make(map[string]bool),
		wsconn: c,
	}
	pA := params.([]interface{})
	for _, param := range pA {
		scl.filter[param.(string)] = true
	}

	subID, err := c.serverConfig.StorageAPI.RegisterStorageChangeChannel(scl.channel)
	if err != nil {
		return nil, err
	}
	scl.subID = subID

	initRes := newSubscriptionResponseJSON(subID, reqID)
	err = c.safeSend(initRes)
	if err != nil {
		return nil, err
	}
	return scl, nil
}

// make this two parts, one in init and return chan ID, second to lister/respond
func (c *StateChangeListener) listen() {

	for change := range c.channel {
		if change != nil {
			//check if change key is in subscription filter
			cKey := common.BytesToHex(change.Key)
			if len(c.filter) > 0 && !c.filter[cKey] {
				continue
			}

			changeM := make(map[string]interface{})
			changeM["result"] = []string{cKey, common.BytesToHex(change.Value)}
			res := newSubcriptionBaseResponseJSON(c.subID)
			res.Method = "state_storage"
			res.Params = changeM
			c.wsconn.safeSend(res)
		}
	}
}


func (c *WSConn) safeSend(msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.wsconn.WriteJSON(msg)
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
