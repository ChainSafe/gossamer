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
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
)

var logger = log.New("pkg", "subscription")

// WSConn struct to hold WebSocket Connection references
type WSConn struct {
	Wsconn             *websocket.Conn
	mu                 sync.Mutex
	BlockSubChannels   map[int]byte
	StorageSubChannels map[int]byte
	qtyListeners       int
	Subscriptions      map[int]Listener
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
