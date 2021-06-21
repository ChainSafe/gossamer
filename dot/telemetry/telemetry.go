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
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	log "github.com/ChainSafe/log15"
	"github.com/gorilla/websocket"
)

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
	sync.Mutex
}

// Handler struct for holding telemetry related things
type Handler struct {
	msg         chan Message
	connections []*telemetryConnection
	log         log.Logger
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
					log: log.New("pkg", "telemetry"),
				}
				go handlerInstance.startListening()
			})
	}
	return handlerInstance
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
func (h *Handler) SendMessage(msg Message) error {
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
			msgBytes, err := h.msgToJSON(msg)
			if err != nil || len(msgBytes) == 0 {
				h.log.Debug("issue decoding telemetry message", "error", err)
				return
			}
			for _, conn := range h.connections {
				conn.Lock()
				defer conn.Unlock()

				err = conn.wsconn.WriteMessage(websocket.TextMessage, msgBytes)
				if err != nil {
					h.log.Warn("issue while sending telemetry message", "error", err)
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

// SystemConnectedTM struct to hold system connected telemetry messages
type SystemConnectedTM struct {
	Authority      bool         `json:"authority"`
	Chain          string       `json:"chain"`
	GenesisHash    *common.Hash `json:"genesis_hash"`
	Implementation string       `json:"implementation"`
	Msg            string       `json:"msg"`
	Name           string       `json:"name"`
	NetworkID      string       `json:"network_id"`
	StartupTime    string       `json:"startup_time"`
	Version        string       `json:"version"`
}

// NewSystemConnectedTM function to create new System Connected Telemetry Message
func NewSystemConnectedTM(authority bool, chain string, genesisHash *common.Hash,
	implementation, name, networkID, startupTime, version string) *SystemConnectedTM {
	return &SystemConnectedTM{
		Authority:      authority,
		Chain:          chain,
		GenesisHash:    genesisHash,
		Implementation: implementation,
		Msg:            "system.connected",
		Name:           name,
		NetworkID:      networkID,
		StartupTime:    startupTime,
		Version:        version,
	}
}
func (tm *SystemConnectedTM) messageType() string {
	return tm.Msg
}

// BlockImportTM struct to hold block import telemetry messages
type BlockImportTM struct {
	BestHash *common.Hash `json:"best"`
	Height   *big.Int     `json:"height"`
	Msg      string       `json:"msg"`
	Origin   string       `json:"origin"`
}

// NewBlockImportTM function to create new Block Import Telemetry Message
func NewBlockImportTM(bestHash *common.Hash, height *big.Int, origin string) *BlockImportTM {
	return &BlockImportTM{
		BestHash: bestHash,
		Height:   height,
		Msg:      "block.import",
		Origin:   origin,
	}
}

func (tm *BlockImportTM) messageType() string {
	return tm.Msg
}

// SystemIntervalTM struct to hold system interval telemetry messages
type SystemIntervalTM struct {
	BandwidthDownload  float64      `json:"bandwidth_download,omitempty"`
	BandwidthUpload    float64      `json:"bandwidth_upload,omitempty"`
	Msg                string       `json:"msg"`
	Peers              int          `json:"peers,omitempty"`
	BestHash           *common.Hash `json:"best,omitempty"`
	BestHeight         *big.Int     `json:"height,omitempty"`
	FinalisedHash      *common.Hash `json:"finalized_hash,omitempty"`   // nolint
	FinalisedHeight    *big.Int     `json:"finalized_height,omitempty"` // nolint
	TxCount            *big.Int     `json:"txcount,omitempty"`
	UsedStateCacheSize *big.Int     `json:"used_state_cache_size,omitempty"`
}

// NewBandwidthTM function to create new Bandwidth Telemetry Message
func NewBandwidthTM(bandwidthDownload, bandwidthUpload float64, peers int) *SystemIntervalTM {
	return &SystemIntervalTM{
		BandwidthDownload: bandwidthDownload,
		BandwidthUpload:   bandwidthUpload,
		Msg:               "system.interval",
		Peers:             peers,
	}
}

// NewBlockIntervalTM function to create new Block Interval Telemetry Message
func NewBlockIntervalTM(beshHash *common.Hash, bestHeight *big.Int, finalisedHash *common.Hash,
	finalisedHeight, txCount, usedStateCacheSize *big.Int) *SystemIntervalTM {
	return &SystemIntervalTM{
		Msg:                "system.interval",
		BestHash:           beshHash,
		BestHeight:         bestHeight,
		FinalisedHash:      finalisedHash,
		FinalisedHeight:    finalisedHeight,
		TxCount:            txCount,
		UsedStateCacheSize: usedStateCacheSize,
	}
}

func (tm *SystemIntervalTM) messageType() string {
	return tm.Msg
}

// NetworkStateTM struct to hold network state telemetry messages
type NetworkStateTM struct {
	Msg   string                 `json:"msg"`
	State map[string]interface{} `json:"state"`
}

// NewNetworkStateTM function to create new Network State Telemetry Message
func NewNetworkStateTM(state map[string]interface{}) *NetworkStateTM {
	return &NetworkStateTM{
		Msg:   "system.network_state",
		State: state,
	}
}
func (tm *NetworkStateTM) messageType() string {
	return tm.Msg
}
