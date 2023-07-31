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

func TestOffchainRPC(t *testing.T) { //nolint:tparallel
	t.SkipNow() // TODO

	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.ChainSpec = genesisPath
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	t.Run("offchain_localStorageSet", func(t *testing.T) {
		t.Parallel()

		var response struct{} // TODO

		fetchWithTimeout(ctx, t, "offchain_localStorageSet", "", &response)

		// TODO assert response
	})

	t.Run("offchain_localStorageGet", func(t *testing.T) {
		t.Parallel()

		var response struct{} // TODO

		fetchWithTimeout(ctx, t, "offchain_localStorageGet", "", &response)

		// TODO assert response
	})

	t.Run("offchain_localStorageGet", func(t *testing.T) {
		t.Parallel()

		var response struct{} // TODO

		fetchWithTimeout(ctx, t, "offchain_localStorageGet", "", &response)

		// TODO assert response
	})
}
