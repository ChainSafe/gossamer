// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

// Logger for the telemetry client.
type Logger interface {
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}
