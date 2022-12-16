// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
)

var ErrTimoutMessageSending = errors.New("timeout sending telemetry message")

type telemetryConnection struct {
	wsconn    *websocket.Conn
	verbosity int
	sync.Mutex
}

// Mailer can send messages to the telemetry servers.
type Mailer struct {
	mutex *sync.Mutex

	logger Logger

	connections []*telemetryConnection
}

// BootstrapMailer setup the mailer, the connections and start the async message shipment
func BootstrapMailer(ctx context.Context, conns []*genesis.TelemetryEndpoint, logger Logger) (
	mailer *Mailer, err error) {
	mailer = &Mailer{
		mutex:  new(sync.Mutex),
		logger: logger,
	}

	for _, v := range conns {
		const maxRetries = 3

		for connAttempts := 0; connAttempts < maxRetries; connAttempts++ {
			const dialTimeout = 3 * time.Second
			dialCtx, dialCancel := context.WithTimeout(ctx, dialTimeout)
			conn, response, err := websocket.DefaultDialer.DialContext(dialCtx, v.Endpoint, nil)
			dialCancel()
			if err != nil {
				mailer.logger.Debugf("cannot dial telemetry endpoint %s (try %d of %d): %s",
					v.Endpoint, connAttempts+1, maxRetries, err)

				if ctxErr := ctx.Err(); ctxErr != nil {
					return nil, ctxErr
				}

				continue
			}

			err = response.Body.Close()
			if err != nil {
				mailer.logger.Warnf("cannot close body of response from %s: %s", v.Endpoint, err)
			}

			mailer.connections = append(mailer.connections, &telemetryConnection{
				wsconn:    conn,
				verbosity: v.Verbosity,
			})
			break
		}
	}

	return mailer, nil
}

// SendMessage sends Message to connected telemetry listeners through messageReceiver
func (m *Mailer) SendMessage(msg json.Marshaler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	go m.shipTelemetryMessage(msg)
}

func (m *Mailer) shipTelemetryMessage(msg json.Marshaler) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		m.logger.Debugf("issue encoding %T telemetry message: %s", msg, err)
		return
	}

	for _, conn := range m.connections {
		conn.Lock()
		defer conn.Unlock()

		err = conn.wsconn.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			m.logger.Debugf("issue while sending %T telemetry message: %s", msg, err)
		}
	}
}
