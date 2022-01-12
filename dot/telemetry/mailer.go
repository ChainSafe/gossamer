// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
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
	*sync.Mutex

	logger  log.LeveledLogger
	enabled bool

	connections []*telemetryConnection
}

func newMailer(enabled bool, logger log.LeveledLogger) *Mailer {
	mailer := &Mailer{
		new(sync.Mutex),
		logger,
		enabled,
		nil,
	}

	return mailer
}

// BootstrapMailer setup the mailer, the connections and start the async message shipment
func BootstrapMailer(ctx context.Context, conns []*genesis.TelemetryEndpoint, enabled bool, logger log.LeveledLogger) (
	mailer *Mailer, err error) {

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

				const retryDelay = time.Second * 15
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

	return mailer, nil
}

// SendMessage sends Message to connected telemetry listeners through messageReceiver
func (m *Mailer) SendMessage(msg Message) {
	m.Lock()
	defer m.Unlock()

	if m.enabled {
		go m.shipTelemetryMessage(msg)
	}
}

func (m *Mailer) shipTelemetryMessage(msg Message) {
	msgBytes, err := json.Marshal(msg)
	fmt.Printf(">>>>>> %s\n", string(msgBytes))
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
