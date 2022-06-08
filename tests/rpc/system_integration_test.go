// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStableNetworkRPC(t *testing.T) {
	if utils.MODE != "stable" {
		t.Skip("Integration tests are disabled, going to skip.")
	}
	t.Log("Running NetworkAPI tests with PORT=" + utils.PORT)

	networkSize, err := strconv.Atoi(utils.NETWORK_SIZE)
	if err != nil {
		networkSize = 0
	}

	endpoint := rpc.NewEndpoint(utils.PORT)

	t.Run("system_health", func(t *testing.T) {
		t.Parallel()

		var response modules.SystemHealthResponse

		fetchWithTimeoutFromEndpoint(t, endpoint, "system_health", "{}", &response)

		expectedResponse := modules.SystemHealthResponse{
			Peers:           networkSize - 1,
			IsSyncing:       true,
			ShouldHavePeers: true,
		}
		assert.Equal(t, expectedResponse, response)
	})

	t.Run("system_networkState", func(t *testing.T) {
		t.Parallel()

		var response modules.SystemNetworkStateResponse

		fetchWithTimeoutFromEndpoint(t, endpoint, "system_networkState", "{}", &response)

		// TODO assert response
	})

	t.Run("system_peers", func(t *testing.T) {
		t.Parallel()

		var response modules.SystemPeersResponse

		fetchWithTimeoutFromEndpoint(t, endpoint, "system_peers", "{}", &response)

		assert.GreaterOrEqual(t, len(response), networkSize-2)

		// TODO assert response
	})
}

func fetchWithTimeoutFromEndpoint(t *testing.T, endpoint, method,
	params string, target interface{}) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	body, err := rpc.Post(ctx, endpoint, method, params)
	cancel()
	require.NoError(t, err)

	err = rpc.Decode(body, target)
	require.NoError(t, err)
}
