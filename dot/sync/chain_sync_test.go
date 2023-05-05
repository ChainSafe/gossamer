package sync

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestChainSync_setPeerHead(t *testing.T) {
	const randomHashString = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	randomHash := common.MustHexToHash(randomHashString)

	testcases := map[string]struct {
		newChainSync func(t *testing.T, ctrl *gomock.Controller) *chainSync
		peerID       peer.ID
		bestHash     common.Hash
		bestNumber   uint
	}{
		"set_peer_head_with_new_peer": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock)

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:     peer.ID("peer-test"),
			bestHash:   randomHash,
			bestNumber: uint(20),
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cs := tt.newChainSync(t, ctrl)
			cs.setPeerHead(tt.peerID, tt.bestHash, tt.bestNumber)
		})
	}
}

func newChainSyncTest(t *testing.T, ctrl *gomock.Controller) *chainSync {
	t.Helper()

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))

	cfg := chainSyncConfig{
		bs:            mockBlockState,
		pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
		minPeers:      1,
		maxPeers:      5,
		slotDuration:  6 * time.Second,
	}

	return newChainSync(cfg)
}
