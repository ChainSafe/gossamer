package sync

import (
	"math/big"
	"testing"

	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestChainSync(t *testing.T) (*chainSync, <-chan *types.BlockData) {
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(0), types.Digest{})
	require.NoError(t, err)

	bs := new(syncmocks.MockBlockState)
	bs.On("BestBlockHeader").Return(header, nil)

	net := new(syncmocks.MockNetwork)
	net.On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*BlockRequestMessage")).Return(nil, nil)

	readyBlocks := make(chan *types.BlockData, MAX_RESPONSE_SIZE)
	return newChainSync(bs, net, readyBlocks), readyBlocks
}
