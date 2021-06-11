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
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
)

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
	sync.Mutex
}

// Message struct to hold telemetry message data
type Message struct {
	values map[string]interface{}
}

// Handler struct for holding telemetry related things
type Handler struct {
	msg         chan interface{}
	connections []*telemetryConnection
	log         log.Logger
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
					msg: make(chan interface{}, 256),
					log: log.New("pkg", "telemetry"),
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
func (h *Handler) AddConnections(conns []*genesis.TelemetryEndpoint) {
	for _, v := range conns {
		c, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
		if err != nil {
			// todo (ed) try reconnecting if there is an error connecting
			h.log.Debug("issue adding telemetry connection", "error", err)
			continue
		}
		tConn := &telemetryConnection{
			wsconn:    c,
			verbosity: v.Verbosity,
		}
		h.connections = append(h.connections, tConn)
	}
}

// SendMessage sends Message to connected telemetry listeners
func (h *Handler) SendMessage(msg interface{}) error {
	select {
	case h.msg <- msg:

	case <-time.After(time.Second * 1):
		return errors.New("timeout sending message")
	}
	return nil
}

func (h *Handler) startListening() {
	for {
		msg := <-h.msg
		go func() {
			for _, conn := range h.connections {
				conn.Lock()
				fmt.Printf("SENDING %s\n", msgToJSON(msg))
				//err := conn.wsconn.WriteMessage(websocket.TextMessage, msgToBytes(msg))
				err := conn.wsconn.WriteMessage(websocket.TextMessage, msgToJSON(msg))

				if err != nil {
					h.log.Warn("issue while sending telemetry message", "error", err)
				}
				conn.Unlock()
			}
		}()
	}
}

func msgToJSON(message interface{}) []byte {
	res, err := json.Marshal(message)
	if err != nil {
		return nil
	}

	objMap := make(map[string]interface{})
	err = json.Unmarshal(res, &objMap)
	if err != nil {
		return nil
	}

	objMap["ts"] = time.Now()
	typ := reflect.TypeOf(message)
	f, _ := typ.FieldByName("Msg")
	def := f.Tag.Get("default")
	objMap["msg"] = def

	fullRes, err := json.Marshal(objMap)
	if err != nil {
		return nil
	}
	return fullRes
}

type SystemConnectedTM struct {
	Authority bool `json:"authority"`
	Chain string `json:"chain"`
	GenesisHash string `json:"genesis_hash"`
	Implementation string `json:"implementation"`
	Msg string `default:"system.connected" json:"msg"`
	Name string `json:"name"`
	NetworkID string `json:"network_id"`
	StartupTime string `json:"startup_time"`
}