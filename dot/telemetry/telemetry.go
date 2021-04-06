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
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Handler struct for holding telemetry related things
type Handler struct {
	buf             bytes.Buffer
	wsConn          []*websocket.Conn
	telemetryLogger *log.Entry
}

// MyJSONFormatter struct for defining JSON Formatter
type MyJSONFormatter struct {
}

// Format function for handling JSON formatting, this overrides default logging formatter to remove
//  log level, line number and timestamp
func (f *MyJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	serialized, err := json.Marshal(entry.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}
	return append(serialized, '\n'), nil
}

var (
	once            sync.Once
	handlerInstance *Handler
)

// GetInstance singleton pattern to for accessing TelemetryHandler
func GetInstance() *Handler {
	if handlerInstance == nil {
		once.Do(
			func() {
				handlerInstance = &Handler{
					buf: bytes.Buffer{},
				}
				log.SetOutput(&handlerInstance.buf)
				log.SetFormatter(new(MyJSONFormatter))
			})
	}
	return handlerInstance
}

// AddConnections adds connections to telemetry sever
func (h *Handler) AddConnections(conns []*genesis.TelemetryEndpoint) {
	for _, v := range conns {
		c, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
		if err != nil {
			fmt.Printf("Error %v\n", err)
			return
		}
		h.wsConn = append(h.wsConn, c)
	}
}

// ConnectionData struct to hold connection data
type ConnectionData struct {
	Authority     bool
	Chain         string
	GenesisHash   string
	SystemName    string
	NodeName      string
	SystemVersion string
	NetworkID     string
	StartTime     string
}

// SendConnection sends connection request message to telemetry connection
func (h *Handler) SendConnection(data *ConnectionData) {
	payload := log.Fields{"authority": data.Authority, "chain": data.Chain, "config": "", "genesis_hash": data.GenesisHash,
		"implementation": data.SystemName, "msg": "system.connected", "name": data.NodeName, "network_id": data.NetworkID, "startup_time": data.StartTime,
		"version": data.SystemVersion}
	h.telemetryLogger = log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	h.telemetryLogger.Print()
	h.sendTelemtry()
}

// SendBlockImport sends block imported message to telemetry connection
func (h *Handler) SendBlockImport(bestHash string, height *big.Int) {
	payload := log.Fields{"best": bestHash, "height": height.Int64(), "msg": "block.import", "origin": "NetworkInitialSync"}
	h.telemetryLogger = log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	h.telemetryLogger.Print()
	h.sendTelemtry()
}

func (h *Handler) sendTelemtry() {
	for _, c := range h.wsConn {
		err := c.WriteMessage(websocket.TextMessage, h.buf.Bytes())
		if err != nil {
			// TODO (ed) determine how to handle this error
			fmt.Printf("ERROR connecting to telemetry %v\n", err)
		}
	}
	h.buf.Reset()
}
