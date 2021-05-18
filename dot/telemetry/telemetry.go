// Copyright 2021 ChainSafe Systems (ON) Corp.
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

package telemetry

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
)

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
}

type telemetryMessage struct {
	values map[string]interface{}
}

// Handler struct for holding telemetry related things
type telHandler struct {
	msg         chan telemetryMessage
	connections []telemetryConnection
}

type keyValue struct {
	key   string
	value interface{}
}

var (
	once            sync.Once
	handlerInstance *telHandler
)

// GetInstance singleton pattern to for accessing TelemetryHandler
func GetInstance() *telHandler { //nolint
	if handlerInstance == nil {
		once.Do(
			func() {
				handlerInstance = &telHandler{
					msg: make(chan telemetryMessage, 3),
				}
				go handlerInstance.startListening()
			})
	}
	return handlerInstance
}

// NewTelemetryMessage builds a telemetry message
func NewTelemetryMessage(values ...keyValue) *telemetryMessage { //nolint
	mvals := make(map[string]interface{})
	for _, v := range values {
		mvals[v.key] = v.value
	}
	return &telemetryMessage{
		values: mvals,
	}
}

// NewKeyValue builds a key value pair for telemetry messages
func NewKeyValue(key string, value interface{}) keyValue { //nolint
	return keyValue{
		key:   key,
		value: value,
	}
}

func (t *telHandler) AddConnections(conns []*genesis.TelemetryEndpoint) {
	for _, v := range conns {
		c, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
		if err != nil {
			// todo (ed) try reconnecting if there is an error connecting
			fmt.Printf("Error %v\n", err)
			continue
		}
		tConn := telemetryConnection{
			wsconn:    c,
			verbosity: v.Verbosity,
		}
		t.connections = append(t.connections, tConn)
	}
}

func (t *telHandler) SendMessage(msg *telemetryMessage) {
	t.msg <- *msg
}

func (t *telHandler) startListening() {
	for {
		msg := <-t.msg
		for _, v := range t.connections {
			err := v.wsconn.WriteMessage(websocket.TextMessage, msgToBytes(msg))
			if err != nil {
				// TODO (ed) determine how to handle this error
				fmt.Printf("ERROR connecting to telemetry %v\n", err)
			}
			//fmt.Printf("Send to conn %v msg %s\n", v.wsconn.RemoteAddr(), msgToBytes(msg) )
		}
	}
}

func msgToBytes(message telemetryMessage) []byte {
	res := make(map[string]interface{})
	res["id"] = 1 // todo determine how this is used
	res["payload"] = message.values
	res["ts"] = time.Now()
	resB, err := json.Marshal(res)
	if err != nil {
		return nil
	}
	return resB
}
