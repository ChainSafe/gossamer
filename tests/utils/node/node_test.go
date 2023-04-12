//go:build endtoend

// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"context"
	"testing"
	"time"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/tests/utils/config"
)

func Test_Node_InitAndStartTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)

	tomlConfig := config.Default()

	n := New(t, tomlConfig, cfg.WestendDevChain)

	n.InitAndStartTest(ctx, t, cancel)

	cancel()
}
