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

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/gorilla/websocket"
)

type websocketMessage struct {
	ID     float64 `json:"id"`
	Method string  `json:"method"`
	Params any
}

type httpclient interface {
	Do(*http.Request) (*http.Response, error)
}

var errCannotReadFromWebsocket = errors.New("cannot read message from websocket")
var errCannotUnmarshalMessage = errors.New("cannot unmarshal webasocket message data")
var logger = log.NewFromGlobal(log.AddContext("pkg", "rpc/subscription"))

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
	HTTP          httpclient
}

// readWebsocketMessage will read and parse the message data to a string->interface{} data
func (c *WSConn) readWebsocketMessage() (bytes []byte, err error) {
	_, bytes, err = c.Wsconn.ReadMessage()
	if err != nil {
		logger.Debugf("websocket failed to read message: %s", err)
		return nil, errCannotReadFromWebsocket
	}

	logger.Tracef("websocket message received: %s", string(bytes))
	return bytes, nil
}

// HandleConn handles messages received on websocket connections
func (c *WSConn) HandleConn() {
	for {
		mbytes, err := c.readWebsocketMessage()
		if errors.Is(err, errCannotReadFromWebsocket) {
			return
		} else if errors.Is(err, errCannotUnmarshalMessage) {
			c.safeSendError(0, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
			continue
		}

		msg := new(websocketMessage)
		err = json.Unmarshal(mbytes, &msg)
		if err != nil {
			logger.Debugf("failed to unmarshal websocket request message: %s", err)
			continue
		}

		if msg.Method == "" {
			c.safeSendError(0, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
			continue
		}

		logger.Debugf("ws method %s called with params %v", msg.Method, msg.Params)

		if !strings.Contains(msg.Method, "_unsubscribe") && !strings.Contains(msg.Method, "_unwatch") {
			setupListener := c.getSetupListener(msg.Method)

			if setupListener == nil {
				c.executeRPCCall(mbytes)
				continue
			}

			listener, err := setupListener(msg.ID, msg.Params)
			if err != nil {
				logger.Warnf("failed to create listener (method=%s): %s", msg.Method, err)
				continue
			}

			listener.Listen()
			continue
		}

		listener, err := c.getUnsubListener(msg.Params)
		if err != nil {
			logger.Warnf("failed to get unsubscriber (method=%s): %s", msg.Method, err)

			if errors.Is(err, errUknownParamSubscribeID) || errors.Is(err, errCannotFindUnsubsriber) {
				c.safeSendError(msg.ID, big.NewInt(InvalidRequestCode), InvalidRequestMessage)
				continue
			}

			if errors.Is(err, errCannotParseID) || errors.Is(err, errCannotFindListener) {
				c.safeSend(newBooleanResponseJSON(false, msg.ID))
				continue
			}
		}

		err = listener.Stop()
		if err != nil {
			logger.Warnf("failed to stop listener goroutine (method=%s): %s", msg.Method, err)
			c.safeSend(newBooleanResponseJSON(false, msg.ID))
		}

		c.safeSend(newBooleanResponseJSON(true, msg.ID))
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
	blockFinalizedListener := &BlockFinalizedListener{
		cancel:        make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
		cancelTimeout: defaultCancelTimeout,
		wsconn:        c,
	}

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return nil, fmt.Errorf("error BlockAPI not set")
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

	if len(pA) != 1 {
		return nil, errors.New("expecting only one parameter")
	}

	// The passed parameter should be a HEX of a SCALE encoded extrinsic
	extBytes, err := common.HexToBytes(pA[0].(string))
	if err != nil {
		return nil, err
	}

	if c.BlockAPI == nil {
		return nil, fmt.Errorf("error BlockAPI not set")
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
	if errors.Is(err, runtime.ErrInvalidTransaction) || errors.Is(err, runtime.ErrUnknownTransaction) {
		c.safeSend(newSubscriptionResponse(authorExtrinsicUpdatesMethod, extSubmitListener.subID, "invalid"))
		return nil, err
	} else if err != nil {
		c.safeSendError(reqID, nil, err.Error())
		return nil, err
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

	req, err := http.NewRequest("POST", c.RPCHost, buff)
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
