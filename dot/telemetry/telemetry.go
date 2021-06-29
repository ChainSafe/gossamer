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
	msg                chan Message
	connections        []*telemetryConnection
	log                log.Logger
	sendMessageTimeout time.Duration
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

const defaultMessageTimeout = time.Second

// GetInstance singleton pattern to for accessing TelemetryHandler
func GetInstance() *Handler { //nolint
	if handlerInstance == nil {
		once.Do(
			func() {
				handlerInstance = &Handler{
					msg:                make(chan Message, 256),
					log:                log.New("pkg", "telemetry"),
					sendMessageTimeout: defaultMessageTimeout,
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
func (h *Handler) SendMessage(msg *Message) error {
	t := time.NewTicker(h.sendMessageTimeout)
	defer t.Stop()
	select {
	case h.msg <- *msg:

	case <-t.C:
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
				err := conn.wsconn.WriteMessage(websocket.TextMessage, msgToBytes(msg))
				if err != nil {
					h.log.Warn("issue while sending telemetry message", "error", err)
				}
				conn.Unlock()
			}
		}()
	}
}

type response struct {
	ID        int                    `json:"id"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"ts"`
}

func msgToBytes(message Message) []byte {
	res := response{
		ID:        1, // todo (ed) determine how this is used
		Payload:   message.values,
		Timestamp: time.Now(),
	}
	resB, err := json.Marshal(res)
	if err != nil {
		return nil
	}
	return resB
}
