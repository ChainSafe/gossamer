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

package subscription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
)

var logger = log.New("pkg", "rpc/subscription")

// WSConn struct to hold WebSocket Connection references
type WSConn struct {
	Wsconn             *websocket.Conn
	mu                 sync.Mutex
	BlockSubChannels   map[uint]byte
	StorageSubChannels map[int]byte
	qtyListeners       uint
	Subscriptions      map[uint]Listener
	StorageAPI         modules.StorageAPI
	BlockAPI           modules.BlockAPI
	RuntimeAPI         modules.RuntimeAPI
	CoreAPI            modules.CoreAPI
	TxStateAPI         modules.TransactionStateAPI
	RPCHost            string
}

//HandleComm handles messages received on websocket connections
func (c *WSConn) HandleComm() {
	for {
		_, mbytes, err := c.Wsconn.ReadMessage()
		if err != nil {
			logger.Warn("websocket failed to read message", "error", err)
			return
		}
		logger.Trace("websocket received", "message", mbytes)

		// determine if request is for subscribe method type
		var msg map[string]interface{}
		err = json.Unmarshal(mbytes, &msg)
		if err != nil {
			logger.Warn("websocket failed to unmarshal request message", "error", err)
			c.safeSendError(0, big.NewInt(-32600), "Invalid request")
			continue
		}

		method := msg["method"]
		params := msg["params"]
		logger.Debug("ws method called", "method", method, "params", params)

		// if method contains subscribe, then register subscription
		if strings.Contains(fmt.Sprintf("%s", method), "subscribe") {
			reqid := msg["id"].(float64)
			switch method {
			case "chain_subscribeNewHeads", "chain_subscribeNewHead":
				bl, err1 := c.initBlockListener(reqid)
				if err1 != nil {
					logger.Warn("failed to create block listener", "error", err)
					continue
				}
				c.startListener(bl)
			case "state_subscribeStorage":
				_, err2 := c.initStorageChangeListener(reqid, params)
				if err2 != nil {
					logger.Warn("failed to create state change listener", "error", err2)
					continue
				}

			case "chain_subscribeFinalizedHeads":
				bfl, err3 := c.initBlockFinalizedListener(reqid)
				if err3 != nil {
					logger.Warn("failed to create block finalised", "error", err3)
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

		req, err := http.NewRequest("POST", c.RPCHost, buf)
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

func (c *WSConn) initStorageChangeListener(reqID float64, params interface{}) (uint, error) {
	if c.StorageAPI == nil {
		c.safeSendError(reqID, nil, "error StorageAPI not set")
		return 0, fmt.Errorf("error StorageAPI not set")
	}

	myObs := &StorageObserver{
		filter: make(map[string][]byte),
		wsconn: c,
	}

	pA, ok := params.([]interface{})
	if !ok {
		return 0, fmt.Errorf("unknown parameter type")
	}
	for _, param := range pA {
		switch p := param.(type) {
		case []interface{}:
			for _, pp := range param.([]interface{}) {
				data, ok := pp.(string)
				if !ok {
					return 0, fmt.Errorf("unknown parameter type")
				}
				myObs.filter[data] = []byte{}
			}
		case string:
			myObs.filter[p] = []byte{}
		default:
			return 0, fmt.Errorf("unknown parameter type")
		}
	}

	c.qtyListeners++
	myObs.id = c.qtyListeners

	c.StorageAPI.RegisterStorageObserver(myObs)

	c.Subscriptions[myObs.id] = myObs

	initRes := newSubscriptionResponseJSON(myObs.id, reqID)
	c.safeSend(initRes)

	return myObs.id, nil
}

func (c *WSConn) initBlockListener(reqID float64) (uint, error) {
	bl := &BlockListener{
		Channel: make(chan *types.Block),
		wsconn:  c,
	}

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	chanID, err := c.BlockAPI.RegisterImportedChannel(bl.Channel)
	if err != nil {
		return 0, err
	}
	bl.ChanID = chanID
	c.qtyListeners++
	bl.subID = c.qtyListeners
	c.Subscriptions[bl.subID] = bl
	c.BlockSubChannels[bl.subID] = chanID
	initRes := newSubscriptionResponseJSON(bl.subID, reqID)
	c.safeSend(initRes)

	return bl.subID, nil
}

func (c *WSConn) initBlockFinalizedListener(reqID float64) (uint, error) {
	bfl := &BlockFinalizedListener{
		channel: make(chan *types.FinalisationInfo),
		wsconn:  c,
	}

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	chanID, err := c.BlockAPI.RegisterFinalizedChannel(bfl.channel)
	if err != nil {
		return 0, err
	}
	bfl.chanID = chanID
	c.qtyListeners++
	bfl.subID = c.qtyListeners
	c.Subscriptions[bfl.subID] = bfl
	c.BlockSubChannels[bfl.subID] = chanID
	initRes := newSubscriptionResponseJSON(bfl.subID, reqID)
	c.safeSend(initRes)

	return bfl.subID, nil
}

func (c *WSConn) initExtrinsicWatch(reqID float64, params interface{}) (uint, error) {
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
		finalisedChan: make(chan *types.FinalisationInfo),
	}

	if c.BlockAPI == nil {
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	esl.importedChanID, err = c.BlockAPI.RegisterImportedChannel(esl.importedChan)
	if err != nil {
		return 0, err
	}

	esl.finalisedChanID, err = c.BlockAPI.RegisterFinalizedChannel(esl.finalisedChan)
	if err != nil {
		return 0, err
	}

	c.qtyListeners++
	esl.subID = c.qtyListeners
	c.Subscriptions[esl.subID] = esl
	c.BlockSubChannels[esl.subID] = esl.importedChanID

	err = c.CoreAPI.HandleSubmittedExtrinsic(extBytes)
	if err != nil {
		return 0, err
	}
	c.safeSend(newSubscriptionResponseJSON(esl.subID, reqID))

	// TODO (ed) since HandleSubmittedExtrinsic has been called we assume the extrinsic is in the tx queue
	//  should we add a channel to tx queue so we're notified when it's in the queue (See issue #1535)
	if c.CoreAPI.IsBlockProducer() {
		c.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, esl.subID, "ready"))
	}

	// todo (ed) determine which peer extrinsic has been broadcast to, and set status
	return esl.subID, err
}

func (c *WSConn) initRuntimeVersionListener(reqID float64) (uint, error) {
	rvl := &RuntimeVersionListener{
		wsconn: c,
	}
	if c.CoreAPI == nil {
		c.safeSendError(reqID, nil, "error CoreAPI not set")
		return 0, fmt.Errorf("error CoreAPI not set")
	}
	c.qtyListeners++
	rvl.subID = c.qtyListeners
	c.Subscriptions[rvl.subID] = rvl
	initRes := newSubscriptionResponseJSON(rvl.subID, reqID)
	c.safeSend(initRes)

	return rvl.subID, nil
}

func (c *WSConn) safeSend(msg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.Wsconn.WriteJSON(msg)
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
	err := c.Wsconn.WriteJSON(res)
	if err != nil {
		logger.Debug("error sending websocket message", "error", err)
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

func (c *WSConn) startListener(lid uint) {
	go c.Subscriptions[lid].Listen()
}
