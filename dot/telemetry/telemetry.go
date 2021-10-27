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
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
)

// telemetry message types
const (
	notifyFinalizedMsg    = "notify.finalized"
	blockImportMsg        = "block.import"
	systemNetworkStateMsg = "system.network_state"
	systemConnectedMsg    = "system.connected"
	systemIntervalMsg     = "system.interval"
)

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
	sync.Mutex
}

// Handler struct for holding telemetry related things
type Handler struct {
	msg                chan Message
	connections        []*telemetryConnection
	log                log.Interface
	sendMessageTimeout time.Duration
	maxRetries         int
	retryDelay         time.Duration
}

// Instance interface that telemetry handler instance needs to implement
type Instance interface {
	AddConnections(conns []*genesis.TelemetryEndpoint)
	SendMessage(msg Message) error
	startListening()
	Initialise(enabled bool)
}

var (
	once            sync.Once
	handlerInstance Instance

	enabled    = true // enabled by default
	initilised sync.Once
)

const (
	defaultMessageTimeout = time.Second
	defaultMaxRetries     = 5
	defaultRetryDelay     = time.Second * 15
)

// GetInstance singleton pattern to for accessing TelemetryHandler
func GetInstance() Instance {
	if handlerInstance == nil {
		once.Do(
			func() {
				handlerInstance = &Handler{
					msg:                make(chan Message, 256),
					log:                log.NewFromGlobal(log.AddContext("pkg", "telemetry")),
					sendMessageTimeout: defaultMessageTimeout,
					maxRetries:         defaultMaxRetries,
					retryDelay:         defaultRetryDelay,
				}
				go handlerInstance.startListening()
			})
	}
	if !enabled {
		return &NoopHandler{}
	}

	return handlerInstance
}

// Initialise function to set if telemetry is enabled
func (h *Handler) Initialise(e bool) {
	initilised.Do(
		func() {
			enabled = e
		})
}

// AddConnections adds the given telemetry endpoint as listeners that will receive telemetry data
func (h *Handler) AddConnections(conns []*genesis.TelemetryEndpoint) {
	for _, v := range conns {
		for connAttempts := 0; connAttempts < h.maxRetries; connAttempts++ {
			c, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
			if err != nil {
				h.log.Debug(fmt.Sprintf("issue adding telemetry connection: %s", err))
				time.Sleep(h.retryDelay)
				continue
			}
			h.connections = append(h.connections, &telemetryConnection{
				wsconn:    c,
				verbosity: v.Verbosity,
			})
			break
		}
	}
}

// SendMessage sends Message to connected telemetry listeners
func (h *Handler) SendMessage(msg Message) error {
	t := time.NewTicker(h.sendMessageTimeout)
	defer t.Stop()
	select {
	case h.msg <- msg:

	case <-t.C:
		return errors.New("timeout sending message")
	}
	return nil
}

func (h *Handler) startListening() {
	for {
		msg := <-h.msg
		go func() {
			msgBytes, err := h.msgToJSON(msg)
			if err != nil {
				h.log.Debug(fmt.Sprintf("issue decoding telemetry message: %s", err))
				return
			}
			for _, conn := range h.connections {
				conn.Lock()
				defer conn.Unlock()

				err = conn.wsconn.WriteMessage(websocket.TextMessage, msgBytes)
				if err != nil {
					h.log.Debug(fmt.Sprintf("issue while sending telemetry message: %s", err))
				}
			}
		}()
	}
}

func (h *Handler) msgToJSON(message Message) ([]byte, error) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	messageMap := make(map[string]interface{})
	err = json.Unmarshal(messageBytes, &messageMap)
	if err != nil {
		return nil, err
	}

	messageMap["ts"] = time.Now()

	messageMap["msg"] = message.messageType()

	fullRes, err := json.Marshal(messageMap)
	if err != nil {
		return nil, err
	}
	return fullRes, nil
}

// Message interface for Message functions
type Message interface {
	messageType() string
}

type peerInfo struct {
	Roles      byte   `json:"roles"`
	BestHash   string `json:"bestHash"`
	BestNumber uint64 `json:"bestNumber"`
}

// NoopHandler struct no op handling (ignoring) telemetry messages
type NoopHandler struct {
}

// Initialise function to set if telemetry is enabled
func (h *NoopHandler) Initialise(enabled bool) {}

func (h *NoopHandler) startListening() {}

// SendMessage no op for telemetry send message function
func (h *NoopHandler) SendMessage(msg Message) error {
	return nil
}

// AddConnections no op for telemetry add connections function
func (h *NoopHandler) AddConnections(conns []*genesis.TelemetryEndpoint) {}
