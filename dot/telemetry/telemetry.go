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
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Handler struct for holding telemetry related things
type Handler struct {
	buf             bytes.Buffer
	wsConn          *websocket.Conn
	telemetryLogger *log.Entry
}

// MyJSONFormatter struct for defining JSON Formatter
type MyJSONFormatter struct {
}

// Format function for handling JSON formatting
func (f *MyJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	serialized, err := json.Marshal(entry.Data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %w", err)
	}
	return append(serialized, '\n'), nil
}

var once sync.Once

var handlerInstance *Handler

// GetInstance signleton pattern to for accessing TelemeterHandler
func GetInstance() *Handler {
	if handlerInstance == nil {
		once.Do(
			func() {
				// TODO (ed) move this so that it can be set by CLI flag
				u := url.URL{
					Scheme: "ws",
					Host:   "127.0.0.1:8000",
					Path:   "/submit/",
				}
				c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
				if err != nil {
					fmt.Printf("Error %v\n", err)
				}

				handlerInstance = &Handler{
					buf:    bytes.Buffer{},
					wsConn: c,
				}
				log.SetOutput(&handlerInstance.buf)
				log.SetFormatter(new(MyJSONFormatter))
			})
	}
	return handlerInstance
}

// SendConnection sends connection request message to telemetry connection
func (h *Handler) SendConnection(authority bool, chaintype, genesis_hash, system_name, node_name,
	system_version, network_id, start_time string) {
	payload := log.Fields{"authority": authority, "chain": chaintype, "config": "", "genesis_hash": genesis_hash,
		"implementation": system_name, "msg": "system.connected", "name": node_name, "network_id": network_id, "startup_time": start_time,
		"version": system_version}
	h.telemetryLogger = log.WithFields(log.Fields{"id": 1, "payload": payload, "ts": time.Now()})
	h.telemetryLogger.Print()
	h.sendTelemtry()
}

func (h *Handler) sendTelemtry() {
	if h.wsConn != nil {
		err := h.wsConn.WriteMessage(websocket.TextMessage, h.buf.Bytes())
		if err != nil {
			// TODO (ed) determine how to handle this error
			fmt.Printf("ERROR connecting to telemetry %v\n", err)
		}
	}
	h.buf.Reset()
}
