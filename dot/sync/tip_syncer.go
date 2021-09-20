package sync

import (
	"github.com/ChainSafe/gossamer/dot/network"
)

var _ workHandler = &tipSyncer{}

// tipSyncer handles workers when syncing at the tip of the chain
// WIP
type tipSyncer struct {
	blockState    BlockState
	pendingBlocks DisjointBlockSet
	workerState   *workerState
}

func newTipSyncer(blockState BlockState, pendingBlocks DisjointBlockSet, workerState *workerState) *tipSyncer {
	return &tipSyncer{
		blockState:    blockState,
		pendingBlocks: pendingBlocks,
		workerState:   workerState,
	}
}

func (s *tipSyncer) handleNewPeerState(ps *peerState) (*worker, error) {
	return nil, nil
}

func (s *tipSyncer) handleWorkerResult(res *worker) (*worker, error) {
	// TODO: if the worker succeeded, potentially remove some blocks from the pending block set and move them into the ready queue
	return nil, nil
}

func (s *tipSyncer) hasCurrentWorker(w *worker, workers map[uint64]*worker) bool {
	return false
}

// handleTick traverses the pending blocks set to find which forks still need to be requested
func (s *tipSyncer) handleTick() (*worker, error) {
	if s.pendingBlocks.size() == 0 {
		return nil, nil
	}

	// cases for each block in pending set:
	// 1. only hash and number are known; in this case, request the full block
	// 2. only header is known; in this case, request the block body
	// 3. entire block is known; in this case, check if we have become aware of the parent

	for _, block := range s.pendingBlocks.getBlocks() {
		if block.header == nil {
			// case 1
			return &worker{
				startHash:  block.hash,
				targetHash: block.hash,
			}, nil
		}

		if block.body == nil {
			// case 2
			return &worker{
				startHash:   block.hash,
				targetHash:  block.hash,
				requestData: network.RequestedDataBody,
			}, nil
		}

		// case 3
		// WIP
	}

	return nil, nil
}
