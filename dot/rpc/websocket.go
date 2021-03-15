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
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/gorilla/websocket"
)

// SubscriptionBaseResponseJSON for base json response
type SubscriptionBaseResponseJSON struct {
	Jsonrpc string             `json:"jsonrpc"`
	Method  string             `json:"method"`
	Params  SubscriptionParams `json:"params"`
}

// SubscriptionParams for json param response
type SubscriptionParams struct {
	Result         interface{} `json:"result"`
	SubscriptionID int         `json:"subscription"`
}

func newSubcriptionBaseResponseJSON() SubscriptionBaseResponseJSON {
	return SubscriptionBaseResponseJSON{
		Jsonrpc: "2.0",
	}
}

func newSubscriptionResponse(method string, subID int, result interface{}) SubscriptionBaseResponseJSON {
	return SubscriptionBaseResponseJSON{
		Jsonrpc: "2.0",
		Method:  method,
		Params: SubscriptionParams{
			Result:         result,
			SubscriptionID: subID,
		},
	}
}

// SubscriptionResponseJSON for json subscription responses
type SubscriptionResponseJSON struct {
	Jsonrpc string  `json:"jsonrpc"`
	Result  int     `json:"result"`
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
	ID      float64           `json:"id"`
}

// ErrorMessageJSON json for error messages
type ErrorMessageJSON struct {
	Code    *big.Int `json:"code"`
	Message string   `json:"message"`
}

var rpcHost string

// ServeHTTP implemented to handle WebSocket connections
func (h *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upg = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if !h.serverConfig.WSExternal {
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

	go wsc.handleComm()
}

// NewWSConn to create new WebSocket Connection struct
func NewWSConn(conn *websocket.Conn, cfg *HTTPServerConfig) *WSConn {
	rpcHost = fmt.Sprintf("http://%s:%d/", cfg.Host, cfg.RPCPort)
	c := &WSConn{
		wsconn:             conn,
		subscriptions:      make(map[int]Listener),
		blockSubChannels:   make(map[int]byte),
		storageSubChannels: make(map[int]byte),
		storageAPI:         cfg.StorageAPI,
		blockAPI:           cfg.BlockAPI,
		runtimeAPI:         cfg.RuntimeAPI,
		coreAPI:            cfg.CoreAPI,
		txStateAPI:         cfg.TransactionQueueAPI,
	}
	return c
}

func (c *WSConn) safeSend(msg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.wsconn.WriteJSON(msg)
	if err != nil {
		logger.Debug("error sending websocket message", "error", err)
	}
}
func (c *WSConn) safeSendError(reqID float64, errorCode *big.Int, message string) {
	res := &ErrorResponseJSON{
		Jsonrpc: "2.0",
		Error: &ErrorMessageJSON{
			Code:    errorCode,
			Message: message,
		},
		ID: reqID,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.wsconn.WriteJSON(res)
	if err != nil {
		logger.Debug("error sending websocket message", "error", err)
	}
}

func (c *WSConn) handleComm() {
	for {
		_, mbytes, err := c.wsconn.ReadMessage()
		if err != nil {
			logger.Warn("websocket failed to read message", "error", err)
			return
		}
		logger.Debug("websocket received", "message", fmt.Sprintf("%s", mbytes))

		// determine if request is for subscribe method type
		var msg map[string]interface{}
		err = json.Unmarshal(mbytes, &msg)
		if err != nil {
			logger.Warn("websocket failed to unmarshal request message", "error", err)
			c.safeSendError(0, big.NewInt(-32600), "Invalid request")
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
					logger.Warn("failed to create block listener", "error", err)
					continue
				}
				c.startListener(bl)
			case "state_subscribeStorage":
				scl, err2 := c.initStorageChangeListener(reqid, params)
				if err2 != nil {
					logger.Warn("failed to create state change listener", "error", err2)
					continue
				}
				c.startListener(scl)
			case "chain_subscribeFinalizedHeads":
				bfl, err3 := c.initBlockFinalizedListener(reqid)
				if err3 != nil {
					logger.Warn("failed to create block finalized", "error", err3)
					continue
				}
				c.startListener(bfl)
			case "state_subscribeRuntimeVersion":
				rvl, err4 := c.initRuntimeVersionListener(reqid)
				if err4 != nil {
					logger.Warn("failed to create runtime version listener", "error", err4)
					continue
				}
				c.startListener(rvl)
			}
			continue
		}

		if strings.Contains(fmt.Sprintf("%s", method), "submitAndWatchExtrinsic") {
			reqid := msg["id"].(float64)
			params := msg["params"]
			el, e := c.initExtrinsicWatch(reqid, params)
			if e != nil {
				c.safeSendError(reqid, nil, e.Error())
			} else {
				c.startListener(el)
			}
			continue
		}

		// handle non-subscribe calls
		client := &http.Client{}
		buf := &bytes.Buffer{}
		_, err = buf.Write(mbytes)
		if err != nil {
			logger.Warn("failed to write message to buffer", "error", err)
			return
		}

		req, err := http.NewRequest("POST", rpcHost, buf)
		if err != nil {
			logger.Warn("failed request to rpc service", "error", err)
			return
		}

		req.Header.Set("Content-Type", "application/json;")

		res, err := client.Do(req)
		if err != nil {
			logger.Warn("websocket error calling rpc", "error", err)
			return
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Warn("error reading response body", "error", err)
			return
		}

		err = res.Body.Close()
		if err != nil {
			logger.Warn("error closing response body", "error", err)
			return
		}
		var wsSend interface{}
		err = json.Unmarshal(body, &wsSend)
		if err != nil {
			logger.Warn("error unmarshal rpc response", "error", err)
			return
		}

		c.safeSend(wsSend)
	}
}
func (c *WSConn) startListener(lid int) {
	go c.subscriptions[lid].Listen()
}

// Listener interface for functions that define Listener related functions
type Listener interface {
	Listen()
}

// StorageChangeListener for listening to state change channels
type StorageChangeListener struct {
	channel chan *state.SubscriptionResult
	wsconn  *WSConn
	chanID  byte
	subID   int
}

func (c *WSConn) initStorageChangeListener(reqID float64, params interface{}) (int, error) {
	scl := &StorageChangeListener{
		channel: make(chan *state.SubscriptionResult),
		wsconn:  c,
	}
	sub := &state.StorageSubscription{
		Filter:   make(map[string]bool),
		Listener: scl.channel,
	}

	pA := params.([]interface{})
	for _, param := range pA {
		switch p := param.(type) {
		case []interface{}:
			for _, pp := range param.([]interface{}) {
				sub.Filter[pp.(string)] = true
			}
		case string:
			sub.Filter[p] = true
		default:
			return 0, fmt.Errorf("unknow parameter type")
		}
	}

	if c.storageAPI == nil {
		c.safeSendError(reqID, nil, "error StorageAPI not set")
		return 0, fmt.Errorf("error StorageAPI not set")
	}

	chanID, err := c.storageAPI.RegisterStorageChangeChannel(*sub)
	if err != nil {
		return 0, err
	}
	scl.chanID = chanID

	c.qtyListeners++
	scl.subID = c.qtyListeners
	c.subscriptions[scl.subID] = scl
	c.storageSubChannels[scl.subID] = chanID

	initRes := newSubscriptionResponseJSON(scl.subID, reqID)
	c.safeSend(initRes)

	return scl.subID, nil
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *StorageChangeListener) Listen() {
	for change := range l.channel {
		if change == nil {
			continue
		}

		result := make(map[string]interface{})
		result["block"] = change.Hash.String()
		changes := [][]string{}
		for _, v := range change.Changes {
			kv := []string{common.BytesToHex(v.Key), common.BytesToHex(v.Value)}
			changes = append(changes, kv)
		}
		result["changes"] = changes

		res := newSubcriptionBaseResponseJSON()
		res.Method = "state_storage"
		res.Params.Result = result
		res.Params.SubscriptionID = l.subID
		l.wsconn.safeSend(res)
	}
}

// BlockListener to handle listening for blocks importedChan
type BlockListener struct {
	channel chan *types.Block
	wsconn  *WSConn
	chanID  byte
	subID   int
}

func (c *WSConn) initBlockListener(reqID float64) (int, error) {
	bl := &BlockListener{
		channel: make(chan *types.Block),
		wsconn:  c,
	}

	if c.blockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	chanID, err := c.blockAPI.RegisterImportedChannel(bl.channel)
	if err != nil {
		return 0, err
	}
	bl.chanID = chanID
	c.qtyListeners++
	bl.subID = c.qtyListeners
	c.subscriptions[bl.subID] = bl
	c.blockSubChannels[bl.subID] = chanID
	initRes := newSubscriptionResponseJSON(bl.subID, reqID)
	c.safeSend(initRes)

	return bl.subID, nil
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockListener) Listen() {
	for block := range l.channel {
		if block == nil {
			continue
		}
		head, err := modules.HeaderToJSON(*block.Header)
		if err != nil {
			logger.Error("failed to convert header to JSON", "error", err)
		}

		res := newSubcriptionBaseResponseJSON()
		res.Method = "chain_newHead"
		res.Params.Result = head
		res.Params.SubscriptionID = l.subID
		l.wsconn.safeSend(res)
	}
}

// BlockFinalizedListener to handle listening for finalized blocks
type BlockFinalizedListener struct {
	channel chan *types.Header
	wsconn  *WSConn
	chanID  byte
	subID   int
}

func (c *WSConn) initBlockFinalizedListener(reqID float64) (int, error) {
	bfl := &BlockFinalizedListener{
		channel: make(chan *types.Header),
		wsconn:  c,
	}

	if c.blockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	chanID, err := c.blockAPI.RegisterFinalizedChannel(bfl.channel)
	if err != nil {
		return 0, err
	}
	bfl.chanID = chanID
	c.qtyListeners++
	bfl.subID = c.qtyListeners
	c.subscriptions[bfl.subID] = bfl
	c.blockSubChannels[bfl.subID] = chanID
	initRes := newSubscriptionResponseJSON(bfl.subID, reqID)
	c.safeSend(initRes)

	return bfl.subID, nil
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockFinalizedListener) Listen() {
	for header := range l.channel {
		if header == nil {
			continue
		}
		head, err := modules.HeaderToJSON(*header)
		if err != nil {
			logger.Error("failed to convert header to JSON", "error", err)
		}
		res := newSubcriptionBaseResponseJSON()
		res.Method = "chain_finalizedHead"
		res.Params.Result = head
		res.Params.SubscriptionID = l.subID
		l.wsconn.safeSend(res)
	}
}

// ExtrinsicSubmitListener to handle listening for extrinsic events
type ExtrinsicSubmitListener struct {
	wsconn    *WSConn
	subID     int
	extrinsic types.Extrinsic

	importedChan    chan *types.Block
	importedChanID  byte
	importedHash    common.Hash
	finalizedChan   chan *types.Header
	finalizedChanID byte
}

// AuthorExtrinsicUpdates method name
const AuthorExtrinsicUpdates = "author_extrinsicUpdate"

func (c *WSConn) initExtrinsicWatch(reqID float64, params interface{}) (int, error) {
	pA := params.([]interface{})
	extBytes, err := common.HexToBytes(pA[0].(string))
	if err != nil {
		return 0, err
	}

	// listen for built blocks
	esl := &ExtrinsicSubmitListener{
		importedChan:  make(chan *types.Block),
		wsconn:        c,
		extrinsic:     types.Extrinsic(extBytes),
		finalizedChan: make(chan *types.Header),
	}

	if c.blockAPI == nil {
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	esl.importedChanID, err = c.blockAPI.RegisterImportedChannel(esl.importedChan)
	if err != nil {
		return 0, err
	}

	esl.finalizedChanID, err = c.blockAPI.RegisterFinalizedChannel(esl.finalizedChan)
	if err != nil {
		return 0, err
	}

	c.qtyListeners++
	esl.subID = c.qtyListeners
	c.subscriptions[esl.subID] = esl
	c.blockSubChannels[esl.subID] = esl.importedChanID

	err = c.coreAPI.HandleSubmittedExtrinsic(extBytes)
	if err != nil {
		return 0, err
	}
	c.safeSend(newSubscriptionResponseJSON(esl.subID, reqID))

	// TODO (ed) since HandleSubmittedExtrinsic has been called we assume the extrinsic is in the tx queue
	//  should we add a channel to tx queue so we're notified when it's in the queue
	if c.coreAPI.IsBlockProducer() {
		c.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, esl.subID, "ready"))
	}

	// todo (ed) determine which peer extrinsic has been broadcast to, and set status
	return esl.subID, err
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *ExtrinsicSubmitListener) Listen() {
	// listen for imported blocks with extrinsic
	go func() {
		for block := range l.importedChan {
			if block == nil {
				continue
			}
			exts, err := block.Body.AsExtrinsics()
			if err != nil {
				fmt.Printf("error %v\n", err)
			}
			for _, v := range exts {
				if reflect.DeepEqual(v, l.extrinsic) {
					resM := make(map[string]interface{})
					resM["inBlock"] = block.Header.Hash().String()

					l.importedHash = block.Header.Hash()
					l.wsconn.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, l.subID, resM))
				}
			}
		}
	}()

	// listen for finalized headers
	go func() {
		for header := range l.finalizedChan {
			if reflect.DeepEqual(l.importedHash, header.Hash()) {
				resM := make(map[string]interface{})
				resM["finalized"] = header.Hash().String()
				l.wsconn.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, l.subID, resM))
			}
		}
	}()
}

// RuntimeVersionListener to handle listening for Runtime Version
type RuntimeVersionListener struct {
	wsconn *WSConn
	subID  int
}

func (c *WSConn) initRuntimeVersionListener(reqID float64) (int, error) {
	rvl := &RuntimeVersionListener{
		wsconn: c,
	}
	if c.coreAPI == nil {
		c.safeSendError(reqID, nil, "error CoreAPI not set")
		return 0, fmt.Errorf("error CoreAPI not set")
	}
	c.qtyListeners++
	rvl.subID = c.qtyListeners
	c.subscriptions[rvl.subID] = rvl
	initRes := newSubscriptionResponseJSON(rvl.subID, reqID)
	c.safeSend(initRes)

	return rvl.subID, nil
}

// Listen implementation of Listen interface to listen for runtime version changes
func (l *RuntimeVersionListener) Listen() {
	rtVersion, err := l.wsconn.coreAPI.GetRuntimeVersion(nil)
	if err != nil {
		return
	}
	ver := modules.StateRuntimeVersionResponse{}

	ver.SpecName = string(rtVersion.SpecName())
	ver.ImplName = string(rtVersion.ImplName())
	ver.AuthoringVersion = rtVersion.AuthoringVersion()
	ver.SpecVersion = rtVersion.SpecVersion()
	ver.ImplVersion = rtVersion.ImplVersion()
	ver.TransactionVersion = rtVersion.TransactionVersion()
	ver.Apis = modules.ConvertAPIs(rtVersion.APIItems())

	l.wsconn.safeSend(newSubscriptionResponse("state_runtimeVersion", l.subID, ver))
}
