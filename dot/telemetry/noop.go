// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import "encoding/json"

// Noop is a no-op telemetry client implementation.
type Noop struct{}

// NewNoopMailer returns a no-op telemetry mailer implementation.
func NewNoopMailer() *Noop {
	return &Noop{}
}

// SendMessage does nothing.
func (*Noop) SendMessage(_ json.Marshaler) {}
