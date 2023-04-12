// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"testing"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
)

func TestPaymentRPC(t *testing.T) { //nolint:tparallel
	t.SkipNow() // TODO

	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.ChainSpec = genesisPath
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	t.Run("payment_queryInfo", func(t *testing.T) {
		t.Parallel()

		var response struct{} // TODO

		fetchWithTimeout(ctx, t, "payment_queryInfo", "", &response)

		// TODO assert response
	})
}
