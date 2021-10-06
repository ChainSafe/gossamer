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
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
)

type httpclient interface {
	Do(*http.Request) (*http.Response, error)
}

var errCannotReadFromWebsocket = errors.New("cannot read message from websocket")
var errCannotUnmarshalMessage = errors.New("cannot unmarshal webasocket message data")
var logger = log.New("pkg", "rpc/subscription")

// DEFAULT_BUFFER_SIZE buffer size for channels
const DEFAULT_BUFFER_SIZE = 100

// WSConn struct to hold WebSocket Connection references
type WSConn struct {
	UnsafeEnabled bool
	Wsconn        *websocket.Conn
	mu            sync.Mutex
	qtyListeners  uint32
	Subscriptions map[uint32]Listener
	StorageAPI    modules.StorageAPI
	BlockAPI      modules.BlockAPI
	CoreAPI       modules.CoreAPI
	TxStateAPI    modules.TransactionStateAPI
	RPCHost       string

	HTTP httpclient
}

// readWebsocketMessage will read and parse the message data to a string->interface{} data
func (c *WSConn) readWebsocketMessage() ([]byte, map[string]interface{}, error) {
	_, mbytes, err := c.Wsconn.ReadMessage()
	if err != nil {
		logger.Debug("websocket failed to read message", "error", err)
		return nil, nil, errCannotReadFromWebsocket
	}

	logger.Trace("websocket received", "message", mbytes)

	// determine if request is for subscribe method type
	var msg map[string]interface{}
	err = json.Unmarshal(mbytes, &msg)

	if err != nil {
		logger.Debug("websocket failed to unmarshal request message", "error", err)
		return nil, nil, errCannotUnmarshalMessage
	}

	return mbytes, msg, nil
}

//HandleComm handles messages received on websocket connections
func (c *WSConn) HandleComm() {
	for {
		mbytes, msg, err := c.readWebsocketMessage()
		if errors.Is(err, errCannotReadFromWebsocket) {
			return
		}

		if errors.Is(err, errCannotUnmarshalMessage) {
			c.safeSendError(0, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
			continue
		}

		params := msg["params"]
		reqid := msg["id"].(float64)
		method := msg["method"].(string)

		logger.Debug("ws method called", "method", method, "params", params)

		if !strings.Contains(method, "_unsubscribe") && !strings.Contains(method, "_unwatch") {
			setup := c.getSetupListener(method)

			if setup == nil {
				c.executeRPCCall(mbytes)
				continue
			}

			listener, err := setup(reqid, params) //nolint
			if err != nil {
				logger.Warn("failed to create listener", "method", method, "error", err)
				continue
			}

			listener.Listen()
			continue
		}

		listener, err := c.getUnsubListener(params) //nolint

		if err != nil {
			logger.Warn("failed to get unsubscriber", "method", method, "error", err)

			if errors.Is(err, errUknownParamSubscribeID) || errors.Is(err, errCannotFindUnsubsriber) {
				c.safeSendError(reqid, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
				continue
			}

			if errors.Is(err, errCannotParseID) || errors.Is(err, errCannotFindListener) {
				c.safeSend(newBooleanResponseJSON(false, reqid))
				continue
			}
		}

		err = listener.Stop()
		if err != nil {
			logger.Warn("failed to cancel listener goroutine", "method", method, "error", err)
			c.safeSend(newBooleanResponseJSON(false, reqid))
		}

		c.safeSend(newBooleanResponseJSON(true, reqid))
		continue
	}
}

func (c *WSConn) executeRPCCall(data []byte) {
	request, err := c.prepareRequest(data)
	if err != nil {
		logger.Warn("failed while preparing the request", "error", err)
		return
	}

	var wsresponse interface{}
	err = c.executeRequest(request, &wsresponse)
	if err != nil {
		logger.Warn("problems while executing the request", "error", err)
		return
	}

	c.safeSend(wsresponse)
}

func (c *WSConn) initStorageChangeListener(reqID float64, params interface{}) (Listener, error) {
	if c.StorageAPI == nil {
		c.safeSendError(reqID, nil, "error StorageAPI not set")
		return nil, fmt.Errorf("error StorageAPI not set")
	}

	stgobs := &StorageObserver{
		filter: make(map[string][]byte),
		wsconn: c,
	}

	pA, ok := params.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unknown parameter type")
	}
	for _, param := range pA {
		switch p := param.(type) {
		case []interface{}:
			for _, pp := range param.([]interface{}) {
				data, ok := pp.(string)
				if !ok {
					return nil, fmt.Errorf("unknown parameter type")
				}
				stgobs.filter[data] = []byte{}
			}
		case string:
			stgobs.filter[p] = []byte{}
		default:
			return nil, fmt.Errorf("unknown parameter type")
		}
	}

	c.mu.Lock()

	stgobs.id = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[stgobs.id] = stgobs

	c.mu.Unlock()

	c.StorageAPI.RegisterStorageObserver(stgobs)
	initRes := NewSubscriptionResponseJSON(stgobs.id, reqID)
	c.safeSend(initRes)

	return stgobs, nil
}

func (c *WSConn) initBlockListener(reqID float64, _ interface{}) (Listener, error) {
	bl := NewBlockListener(c)

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return nil, fmt.Errorf("error BlockAPI not set")
	}

	bl.Channel = c.BlockAPI.GetImportedBlockNotifierChannel()

	c.mu.Lock()

	bl.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[bl.subID] = bl

	c.mu.Unlock()

	c.safeSend(NewSubscriptionResponseJSON(bl.subID, reqID))

	return bl, nil
}

func (c *WSConn) initBlockFinalizedListener(reqID float64, _ interface{}) (Listener, error) {
	bfl := &BlockFinalizedListener{
		cancel:        make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
		cancelTimeout: defaultCancelTimeout,
		wsconn:        c,
	}

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return nil, fmt.Errorf("error BlockAPI not set")
	}

	bfl.channel = c.BlockAPI.GetFinalisedNotifierChannel()

	c.mu.Lock()

	bfl.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[bfl.subID] = bfl

	c.mu.Unlock()

	initRes := NewSubscriptionResponseJSON(bfl.subID, reqID)
	c.safeSend(initRes)

	return bfl, nil
}

func (c *WSConn) initAllBlocksListerner(reqID float64, _ interface{}) (Listener, error) {
	listener := newAllBlockListener(c)

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return nil, fmt.Errorf("error BlockAPI not set")
	}

	listener.importedChan = c.BlockAPI.GetImportedBlockNotifierChannel()
	listener.finalizedChan = c.BlockAPI.GetFinalisedNotifierChannel()

	c.mu.Lock()
	listener.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[listener.subID] = listener
	c.mu.Unlock()

	c.safeSend(NewSubscriptionResponseJSON(listener.subID, reqID))
	return listener, nil
}

func (c *WSConn) initExtrinsicWatch(reqID float64, params interface{}) (Listener, error) {
	pA := params.([]interface{})
	extBytes, err := common.HexToBytes(pA[0].(string))
	if err != nil {
		return nil, err
	}

	// listen for built blocks
	esl := NewExtrinsicSubmitListener(c, extBytes)

	if c.BlockAPI == nil {
		return nil, fmt.Errorf("error BlockAPI not set")
	}

	esl.importedChan = c.BlockAPI.GetImportedBlockNotifierChannel()

	esl.finalisedChan = c.BlockAPI.GetFinalisedNotifierChannel()

	c.mu.Lock()

	esl.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[esl.subID] = esl

	c.mu.Unlock()

	err = c.CoreAPI.HandleSubmittedExtrinsic(extBytes)
	if err != nil {
		c.safeSendError(reqID, nil, err.Error())
		return nil, err
	}
	c.safeSend(NewSubscriptionResponseJSON(esl.subID, reqID))

	// TODO (ed) since HandleSubmittedExtrinsic has been called we assume the extrinsic is in the tx queue
	//  should we add a channel to tx queue so we're notified when it's in the queue (See issue #1535)
	c.safeSend(newSubscriptionResponse(authorExtrinsicUpdatesMethod, esl.subID, "ready"))

	// todo (ed) determine which peer extrinsic has been broadcast to, and set status
	return esl, err
}

func (c *WSConn) initRuntimeVersionListener(reqID float64, _ interface{}) (Listener, error) {
	if c.CoreAPI == nil {
		c.safeSendError(reqID, nil, "error CoreAPI not set")
		return nil, fmt.Errorf("error CoreAPI not set")
	}

	rvl := &RuntimeVersionListener{
		wsconn:        c,
		runtimeUpdate: make(chan runtime.Version),
		coreAPI:       c.CoreAPI,
	}

	chanID, err := c.BlockAPI.RegisterRuntimeUpdatedChannel(rvl.runtimeUpdate)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()

	rvl.channelID = chanID
	c.qtyListeners++
	rvl.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[rvl.subID] = rvl

	c.mu.Unlock()

	c.safeSend(NewSubscriptionResponseJSON(rvl.subID, reqID))

	return rvl, nil
}

func (c *WSConn) initGrandpaJustificationListener(reqID float64, _ interface{}) (Listener, error) {
	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return nil, fmt.Errorf("error BlockAPI not set")
	}

	jl := &GrandpaJustificationListener{
		cancel:        make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
		wsconn:        c,
		cancelTimeout: defaultCancelTimeout,
	}

	jl.finalisedCh = c.BlockAPI.GetFinalisedNotifierChannel()

	c.mu.Lock()

	jl.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[jl.subID] = jl

	c.mu.Unlock()

	c.safeSend(NewSubscriptionResponseJSON(jl.subID, reqID))

	return jl, nil
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

func (c *WSConn) prepareRequest(b []byte) (*http.Request, error) {
	buff := &bytes.Buffer{}
	if _, err := buff.Write(b); err != nil {
		logger.Warn("failed to write message to buffer", "error", buff)
		return nil, err
	}

	req, err := http.NewRequest("POST", c.RPCHost, buff)
	if err != nil {
		logger.Warn("failed request to rpc service", "error", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json;")
	return req, nil
}

func (c *WSConn) executeRequest(r *http.Request, d interface{}) error {
	res, err := c.HTTP.Do(r)
	if err != nil {
		logger.Warn("websocket error calling rpc", "error", err)
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Warn("error reading response body", "error", err)
		return err
	}

	err = res.Body.Close()
	if err != nil {
		logger.Warn("error closing response body", "error", err)
		return err
	}

	err = json.Unmarshal(body, d)

	if err != nil {
		logger.Warn("error unmarshal rpc response", "error", err)
		return err
	}

	return nil
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
