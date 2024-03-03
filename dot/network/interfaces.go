// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"encoding/json"
	"io"
)

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

// Logger is the logger to log messages.
type Logger interface {
	Warn(s string)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// MDNS is the mDNS service interface.
type MDNS interface {
	Start() error
	io.Closer
}
