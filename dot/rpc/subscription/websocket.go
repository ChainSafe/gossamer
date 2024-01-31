// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subscription

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/gorilla/websocket"
)

type websocketMessage struct {
	ID     float64 `json:"id"`
	Method string  `json:"method"`
	Params any     `json:"params"`
}

type httpclient interface {
	Do(*http.Request) (*http.Response, error)
}

var (
	errUnexpectedType          = errors.New("unexpected type")
	errUnexpectedParamLen      = errors.New("unexpected params length")
	errCannotReadFromWebsocket = errors.New("cannot read message from websocket")
	errEmptyMethod             = errors.New("empty method")
	errStorageNotSet           = errors.New("error StorageAPI not set")
	errBlockAPINotSet          = errors.New("error BlockAPI not set")
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "rpc/subscription"))

// WSConn struct to hold WebSocket Connection references
type WSConn struct {
	UnsafeEnabled bool
	Wsconn        *websocket.Conn
	mu            sync.Mutex
	qtyListeners  uint32
	Subscriptions map[uint32]Listener
	StorageAPI    StorageAPI
	BlockAPI      BlockAPI
	CoreAPI       CoreAPI
	TxStateAPI    TransactionStateAPI
	RPCHost       string
	HTTP          httpclient
}

// readWebsocketMessage will read and parse the message data to a string->interface{} data
func (c *WSConn) readWebsocketMessage() (rawBytes []byte, wsMessage *websocketMessage, err error) {
	_, rawBytes, err = c.Wsconn.ReadMessage()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", errCannotReadFromWebsocket, err.Error())
	}

	wsMessage = new(websocketMessage)
	err = json.Unmarshal(rawBytes, wsMessage)
	if err != nil {
		return nil, nil, err
	}

	if wsMessage.Method == "" {
		return nil, nil, errEmptyMethod
	}

	return rawBytes, wsMessage, nil
}

// HandleConn handles messages received on websocket connections
func (c *WSConn) HandleConn() {
	for {
		rawBytes, wsMessage, err := c.readWebsocketMessage()
		if err != nil {
			logger.Debugf("websocket failed to read message: %s", err)
			if errors.Is(err, errCannotReadFromWebsocket) {
				return
			}

			c.safeSendError(0, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
			continue
		}

		logger.Tracef("websocket message received: %s", string(rawBytes))
		logger.Debugf("ws method %s called with params %v", wsMessage.Method, wsMessage.Params)

		if !strings.Contains(wsMessage.Method, "_unsubscribe") && !strings.Contains(wsMessage.Method, "_unwatch") {
			setupListener := c.getSetupListener(wsMessage.Method)

			if setupListener == nil {
				c.executeRPCCall(rawBytes)
				continue
			}

			listener, err := setupListener(wsMessage.ID, wsMessage.Params)
			if err != nil {
				logger.Warnf("failed to create listener (method=%s): %s", wsMessage.Method, err)
				continue
			}

			listener.Listen()
			continue
		}

		listener, err := c.getUnsubListener(wsMessage.Params)
		if err != nil {
			logger.Warnf("failed to get unsubscriber (method=%s): %s", wsMessage.Method, err)

			if errors.Is(err, errUknownParamSubscribeID) || errors.Is(err, errCannotFindUnsubsriber) {
				c.safeSendError(wsMessage.ID, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
				continue
			}

			if errors.Is(err, errCannotParseID) || errors.Is(err, errCannotFindListener) {
				c.safeSend(newBooleanResponseJSON(false, wsMessage.ID))
				continue
			}
		}

		err = listener.Stop()
		if err != nil {
			logger.Warnf("failed to stop listener goroutine (method=%s): %s", wsMessage.Method, err)
			c.safeSend(newBooleanResponseJSON(false, wsMessage.ID))
		}

		c.safeSend(newBooleanResponseJSON(true, wsMessage.ID))
		continue
	}
}

func (c *WSConn) executeRPCCall(data []byte) {
	request, err := c.prepareRequest(data)
	if err != nil {
		logger.Warnf("failed while preparing the request: %s", err)
		return
	}

	var wsresponse interface{}
	err = c.executeRequest(request, &wsresponse)
	if err != nil {
		logger.Warnf("problems while executing the request: %s", err)
		return
	}

	c.safeSend(wsresponse)
}

func (c *WSConn) initStorageChangeListener(reqID float64, params interface{}) (Listener, error) {
	if c.StorageAPI == nil {
		c.safeSendError(reqID, nil, errStorageNotSet.Error())
		return nil, errStorageNotSet
	}

	stgobs := &StorageObserver{
		filter: make(map[string][]byte),
		wsconn: c,
	}

	// the following type checking/casting is needed in order to satisfy some
	// websocket request field params eg.:
	// "params": ["0x..."] or
	// "params": [["0x...", "0x..."]]
	switch filters := params.(type) {
	case []interface{}:
		for _, interfaceKey := range filters {
			switch key := interfaceKey.(type) {
			case string:
				stgobs.filter[key] = []byte{}
			case []string:
				for _, k := range key {
					stgobs.filter[k] = []byte{}
				}
			case []interface{}:
				for _, k := range key {
					k, ok := k.(string)
					if !ok {
						return nil, fmt.Errorf("%w: %T, expected type string", errUnexpectedType, k)
					}

					stgobs.filter[k] = []byte{}
				}
			default:
				return nil, fmt.Errorf("%w: %T, expected type string, []string, []interface{}", errUnexpectedType, interfaceKey)
			}
		}
	default:
		return nil, fmt.Errorf("%w: %T, expected type []interface{}", errUnexpectedType, params)
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
		c.safeSendError(reqID, nil, errBlockAPINotSet.Error())
		return nil, errBlockAPINotSet
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
	blockFinalizedListener := &BlockFinalizedListener{
		cancel:        make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
		cancelTimeout: defaultCancelTimeout,
		wsconn:        c,
	}

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, errBlockAPINotSet.Error())
		return nil, errBlockAPINotSet
	}

	blockFinalizedListener.channel = c.BlockAPI.GetFinalisedNotifierChannel()

	c.mu.Lock()

	blockFinalizedListener.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[blockFinalizedListener.subID] = blockFinalizedListener

	c.mu.Unlock()

	initRes := NewSubscriptionResponseJSON(blockFinalizedListener.subID, reqID)
	c.safeSend(initRes)

	return blockFinalizedListener, nil
}

func (c *WSConn) initAllBlocksListerner(reqID float64, _ interface{}) (Listener, error) {
	listener := newAllBlockListener(c)

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, errBlockAPINotSet.Error())
		return nil, errBlockAPINotSet
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
	var encodedExtrinsic string

	switch encodedHex := params.(type) {
	case []string:
		if len(encodedHex) != 1 {
			return nil, fmt.Errorf("%w: expected 1 param, got: %d", errUnexpectedParamLen, len(encodedHex))
		}
		encodedExtrinsic = encodedHex[0]
	// the bellow case is needed to cover a interface{} slice containing one string
	// as `[]interface{"a"}` is not the same as `[]string{"a"}`
	case []interface{}:
		if len(encodedHex) != 1 {
			return nil, fmt.Errorf("%w: expected 1 param, got: %d", errUnexpectedParamLen, len(encodedHex))
		}

		var ok bool
		encodedExtrinsic, ok = encodedHex[0].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %T, expected type string", errUnexpectedType, encodedHex[0])
		}
	default:
		return nil, fmt.Errorf("%w: %T, expected type []string or []interface{}", errUnexpectedType, params)
	}

	// The passed parameter should be a HEX of a SCALE encoded extrinsic
	extBytes, err := common.HexToBytes(encodedExtrinsic)
	if err != nil {
		return nil, err
	}

	if c.BlockAPI == nil {
		return nil, errBlockAPINotSet
	}

	txStatusChan := c.TxStateAPI.GetStatusNotifierChannel(extBytes)
	importedChan := c.BlockAPI.GetImportedBlockNotifierChannel()
	finalizedChan := c.BlockAPI.GetFinalisedNotifierChannel()

	extSubmitListener := NewExtrinsicSubmitListener(
		c,
		extBytes,
		importedChan,
		txStatusChan,
		finalizedChan,
	)

	c.mu.Lock()
	extSubmitListener.subID = atomic.AddUint32(&c.qtyListeners, 1)
	c.Subscriptions[extSubmitListener.subID] = extSubmitListener
	c.mu.Unlock()

	err = c.CoreAPI.HandleSubmittedExtrinsic(extBytes)
	if err != nil {
		switch err.(type) {
		case runtime.InvalidTransaction,
			runtime.UnknownTransaction:
			c.safeSend(newSubscriptionResponse(authorExtrinsicUpdatesMethod, extSubmitListener.subID, "invalid"))
		default:
			c.safeSendError(reqID, nil, err.Error())
		}
		return nil, fmt.Errorf("handling submitted extrinsic: %w", err)
	}
	c.safeSend(NewSubscriptionResponseJSON(extSubmitListener.subID, reqID))

	// todo (ed) determine which peer extrinsic has been broadcast to, and set status (#1535)
	return extSubmitListener, err
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
		c.safeSendError(reqID, nil, errBlockAPINotSet.Error())
		return nil, errBlockAPINotSet
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
		logger.Debugf("error sending websocket message: %s", err)
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
		logger.Debugf("error sending websocket message: %s", err)
	}
}

func (c *WSConn) prepareRequest(b []byte) (*http.Request, error) {
	buff := &bytes.Buffer{}
	if _, err := buff.Write(b); err != nil {
		logger.Warnf("failed to write message to buffer: %s", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.RPCHost, buff)
	if err != nil {
		logger.Warnf("failed request to rpc service: %s", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json;")
	return req, nil
}

func (c *WSConn) executeRequest(r *http.Request, d interface{}) error {
	res, err := c.HTTP.Do(r)
	if err != nil {
		logger.Warnf("websocket error calling rpc: %s", err)
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Warnf("error reading response body: %s", err)
		return err
	}

	err = res.Body.Close()
	if err != nil {
		logger.Warnf("error closing response body: %s", err)
		return err
	}

	err = json.Unmarshal(body, d)

	if err != nil {
		logger.Warnf("error unmarshal rpc response: %s", err)
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
