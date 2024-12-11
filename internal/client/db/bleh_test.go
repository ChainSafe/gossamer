// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"log/slog"
	"os"
	"testing"
)

func TestSomething(t *testing.T) {
	slog.Debug("Debug message")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Debug("Debug message")
}
