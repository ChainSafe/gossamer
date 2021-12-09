// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
)

var messageQueue chan Message = make(chan Message, 256)

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
	sync.Mutex
}

// Handler struct for holding telemetry related things
type mailer struct {
	messageQueue chan Message
	connections  []*telemetryConnection
	logger       log.LeveledLogger
}

func newMailer(logger log.LeveledLogger) *mailer {
	return &mailer{
		messageQueue: messageQueue,
		logger:       logger,
	}
}

// BootstrapMailer setup the mailer, the connections and start the async message shipment
func BootstrapMailer(ctx context.Context, conns []*genesis.TelemetryEndpoint, logger log.LeveledLogger) {
	const (
		maxRetries = 5
		retryDelay = time.Second * 15
	)

	mlr := newMailer(logger)

	for _, v := range conns {
		for connAttempts := 0; connAttempts < maxRetries; connAttempts++ {
			c, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
			if err != nil {
				mlr.logger.Debugf("issue adding telemetry connection: %s", err)
				time.Sleep(retryDelay)
				continue
			}

			mlr.connections = append(mlr.connections, &telemetryConnection{
				wsconn:    c,
				verbosity: v.Verbosity,
			})
			break
		}
	}

	go mlr.asyncShipment(ctx)
}

// SendMessage sends Message to connected telemetry listeners throught messageReceiver
func SendMessage(msg Message) error {
	const messageTimeout = time.Second
	t := time.NewTimer(messageTimeout)
	defer t.Stop()

	select {
	case messageQueue <- msg:
	case <-t.C:
		return errors.New("timeout sending message")
	}
	return nil
}

func (m *mailer) asyncShipment(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-m.messageQueue:
			if !ok {
				return
			}

			go func(msg Message) {
				msgBytes, err := m.msgToJSON(msg)
				if err != nil {
					m.logger.Debugf("issue decoding telemetry message: %s", err)
					return
				}

				for _, conn := range m.connections {
					conn.Lock()
					defer conn.Unlock()

					err = conn.wsconn.WriteMessage(websocket.TextMessage, msgBytes)
					if err != nil {
						m.logger.Debugf("issue while sending telemetry message: %s", err)
					}
				}
			}(msg)
		}
	}
}

func (h *mailer) msgToJSON(message Message) ([]byte, error) {
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
