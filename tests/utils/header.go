// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// headerResponseToHeader converts a *ChainBlockHeaderResponse to a *types.Header
func headerResponseToHeader(t *testing.T, header *modules.ChainBlockHeaderResponse) *types.Header {
	parentHash, err := common.HexToHash(header.ParentHash)
	require.NoError(t, err)

	nb, err := common.HexToBytes(header.Number)
	require.NoError(t, err)
	number := uint(big.NewInt(0).SetBytes(nb).Uint64())

	stateRoot, err := common.HexToHash(header.StateRoot)
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash(header.ExtrinsicsRoot)
	require.NoError(t, err)

	h, err := types.NewHeader(parentHash, stateRoot, extrinsicsRoot, number, rpcLogsToDigest(t, header.Digest.Logs))
	require.NoError(t, err)
	return h
}
