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
	"context"
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

// Message struct to hold telemetry message data
type Message struct {
	values map[string]interface{}
}

// Handler struct for holding telemetry related things
type Handler struct {
	msg         chan Message
	ctx         context.Context
	connections []telemetryConnection
	sync.Mutex
}

// KeyValue object to hold key value pairs used in telemetry messages
type KeyValue struct {
	key   string
	value interface{}
}

var (
	once            sync.Once
	handlerInstance *Handler
)

// GetInstance singleton pattern to for accessing TelemetryHandler
func GetInstance() *Handler { //nolint
	if handlerInstance == nil {
		once.Do(
			func() {
				handlerInstance = &Handler{
					msg: make(chan Message, 256),
					ctx: context.Background(),
				}
				go handlerInstance.startListening()
			})
	}
	return handlerInstance
}

// NewTelemetryMessage builds a telemetry message
func NewTelemetryMessage(values ...*KeyValue) *Message { //nolint
	mvals := make(map[string]interface{})
	for _, v := range values {
		mvals[v.key] = v.value
	}
	return &Message{
		values: mvals,
	}
}

// NewKeyValue builds a key value pair for telemetry messages
func NewKeyValue(key string, value interface{}) *KeyValue { //nolint
	return &KeyValue{
		key:   key,
		value: value,
	}
}

// AddConnections adds the given telemetry endpoint as listeners that will receive telemetry data
func (t *Handler) AddConnections(conns []*genesis.TelemetryEndpoint) {
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

// SendMessage sends Message to connected telemetry listeners
func (t *Handler) SendMessage(msg *Message) {
	t.msg <- *msg
}

func (t *Handler) startListening() {
	for {
		select {
		case msg := <-t.msg:
			go func() {
				t.Lock()
				for _, v := range t.connections {
					v.wsconn.WriteMessage(websocket.TextMessage, msgToBytes(msg)) // nolint
				}
				t.Unlock()
			}()
		case <-t.ctx.Done():
			return
		}
	}
}

func msgToBytes(message Message) []byte {
	res := make(map[string]interface{})
	res["id"] = 1 // todo (ed) determine how this is used
	res["payload"] = message.values
	res["ts"] = time.Now()
	resB, err := json.Marshal(res)
	if err != nil {
		return nil
	}
	return resB
}
