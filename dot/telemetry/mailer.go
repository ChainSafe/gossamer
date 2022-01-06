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

//go:generate mockgen -destination=client_mock.go -package $GOPACKAGE . Client
// Client interface holds the definitions to services send messages through telemetry
type Client interface {
	SendMessage(msg Message) error
}

var ErrTimoutMessageSending = errors.New("timeout sending telemetry message")

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
	sync.Mutex
}

// Mailer holds telemetry related attributes
type Mailer struct {
	messageQueue chan Message
	connections  []*telemetryConnection
	logger       log.LeveledLogger
	enabled      bool
}

func newMailer(enabled bool, logger log.LeveledLogger) *Mailer {
	return &Mailer{
		enabled:      enabled,
		messageQueue: make(chan Message),
		logger:       logger,
	}
}

// BootstrapMailer setup the mailer, the connections and start the async message shipment
func BootstrapMailer(ctx context.Context, conns []*genesis.TelemetryEndpoint, enabled bool, logger log.LeveledLogger) (
	mailer *Mailer, err error) {
	const retryDelay = time.Second * 15

	mailer = newMailer(enabled, logger)
	if !enabled {
		return mailer, nil
	}

	for _, v := range conns {
		const maxRetries = 5
		for connAttempts := 0; connAttempts < maxRetries; connAttempts++ {
			conn, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
			if err != nil {
				mailer.logger.Debugf("cannot dial telemetry endpoint %s (try %d of %d): %s",
					v.Endpoint, connAttempts+1, maxRetries, err)

				timer := time.NewTimer(retryDelay)

				select {
				case <-timer.C:
					continue
				case <-ctx.Done():
					mailer.logger.Debugf("bootstrap telemetry issue: %w", ctx.Err())
					if !timer.Stop() {
						<-timer.C
					}

					return nil, ctx.Err()
				}
			}

			mailer.connections = append(mailer.connections, &telemetryConnection{
				wsconn:    conn,
				verbosity: v.Verbosity,
			})
			break
		}
	}

	go mailer.asyncShipment(ctx)
	return mailer, nil
}

// SendMessage sends Message to connected telemetry listeners through messageReceiver
func (m *Mailer) SendMessage(msg Message) error {
	const messageTimeout = time.Second

	timer := time.NewTimer(messageTimeout)

	select {
	case m.messageQueue <- msg:
		if !timer.Stop() {
			<-timer.C
		}

	case <-timer.C:
		return ErrTimoutMessageSending
	}
	return nil
}

func (m *Mailer) asyncShipment(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-m.messageQueue:
			if !ok {
				return
			}

			if m.enabled {
				go m.shipTelemetryMessage(msg)
			}
		}
	}
}

func (m *Mailer) shipTelemetryMessage(msg Message) {
	msgBytes, err := msgToJSON(msg)
	if err != nil {
		m.logger.Debugf("issue encoding telemetry message: %s", err)
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
}

func msgToJSON(message Message) ([]byte, error) {
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
