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

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Handler struct for holding telemetry related things
type Handler struct {
	buf    bytes.Buffer
	wsConn []*websocket.Conn
	sync.RWMutex
}

// MyJSONFormatter struct for defining JSON Formatter
type MyJSONFormatter struct {
}

// Format function for handling JSON formatting, this overrides default logging formatter to remove
//  log level, line number and timestamp
func (f *MyJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	serialised, err := json.Marshal(entry.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}
	return append(serialised, '\n'), nil
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
				go handlerInstance.sender()
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
			continue
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
	h.Lock()
	defer h.Unlock()
	payload := log.Fields{"authority": data.Authority, "chain": data.Chain, "config": "", "genesis_hash": data.GenesisHash,
		"implementation": data.SystemName, "msg": "system.connected", "name": data.NodeName, "network_id": data.NetworkID, "startup_time": data.StartTime,
		"version": data.SystemVersion}
	telemetryLogger := log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	telemetryLogger.Print()
}

// SendBlockImport sends block imported message to telemetry connection
func (h *Handler) SendBlockImport(bestHash string, height *big.Int) {
	h.Lock()
	defer h.Unlock()
	payload := log.Fields{"best": bestHash, "height": height.Int64(), "msg": "block.import", "origin": "NetworkInitialSync"}
	telemetryLogger := log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	telemetryLogger.Print()
}

// NetworkData struct to hold network data telemetry information
type NetworkData struct {
	peers   int
	rateIn  float64
	rateOut float64
}

// NewNetworkData creates networkData struct
func NewNetworkData(peers int, rateIn, rateOut float64) *NetworkData {
	return &NetworkData{
		peers:   peers,
		rateIn:  rateIn,
		rateOut: rateOut,
	}
}

// SendNetworkData send network data system.interval message to telemetry connection
func (h *Handler) SendNetworkData(data *NetworkData) {
	h.Lock()
	defer h.Unlock()
	payload := log.Fields{"bandwidth_download": data.rateIn, "bandwidth_upload": data.rateOut, "msg": "system.interval", "peers": data.peers}
	telemetryLogger := log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	telemetryLogger.Print()
}

// BlockIntervalData struct to hold data for block system.interval message
type BlockIntervalData struct {
	BestHash           common.Hash
	BestHeight         *big.Int
	FinalizedHash      common.Hash
	FinalizedHeight    *big.Int
	TXCount            int
	UsedStateCacheSize int
}

// SendBlockIntervalData send block data system interval information to telemetry connection
func (h *Handler) SendBlockIntervalData(data *BlockIntervalData) {
	h.Lock()
	defer h.Unlock()
	payload := log.Fields{"best": data.BestHash.String(), "finalized_hash": data.FinalizedHash.String(), // nolint
		"finalized_height": data.FinalizedHeight, "height": data.BestHeight, "msg": "system.interval", "txcount": data.TXCount, // nolint
		"used_state_cache_size": data.UsedStateCacheSize}
	telemetryLogger := log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	telemetryLogger.Print()
}

func (h *Handler) sender() {
	for {
		h.RLock()
		line, err := h.buf.ReadBytes(byte(10)) // byte 10 is newline character, used as delimiter
		h.RUnlock()
		if err != nil {
			continue
		}

		for _, c := range h.wsConn {
			err := c.WriteMessage(websocket.TextMessage, line)
			if err != nil {
				// TODO (ed) determine how to handle this error
				fmt.Printf("ERROR connecting to telemetry %v\n", err)
			}
		}
	}
}
