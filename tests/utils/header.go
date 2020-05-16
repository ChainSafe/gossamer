package utils

import (
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

// HeaderResponseToHeader converts a *ChainBlockHeaderResponse to a *types.Header
func HeaderResponseToHeader(t *testing.T, header *modules.ChainBlockHeaderResponse) *types.Header {
	parentHash, err := common.HexToHash(header.ParentHash)
	require.NoError(t, err)

	nb, err := common.HexToBytes(header.Number)
	require.NoError(t, err)
	number := big.NewInt(0).SetBytes(nb)

	stateRoot, err := common.HexToHash(header.StateRoot)
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash(header.ExtrinsicsRoot)
	require.NoError(t, err)

	digest := [][]byte{}

	for _, l := range header.Digest.Logs {
		var d []byte
		d, err = common.HexToBytes(l)
		require.NoError(t, err)
		digest = append(digest, d)
	}

	h, err := types.NewHeader(parentHash, number, stateRoot, extrinsicsRoot, digest)
	require.NoError(t, err)
	return h
}