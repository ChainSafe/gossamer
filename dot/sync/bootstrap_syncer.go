package sync

import (
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

var _ workHandler = &bootstrapSyncer{}

type bootstrapSyncer struct {
	blockState BlockState
}

func newBootstrapSyncer(blockState BlockState) *bootstrapSyncer {
	return &bootstrapSyncer{
		blockState: blockState,
	}
}

func (s *bootstrapSyncer) handleWork(ps *peerState) (*worker, error) {
	// // if the peer reports a lower or equal best block number than us,
	// // check if they are on a fork or not
	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	// TODO: this is for bootstrap mode, for idle fork-sync mode
	// we may want to reverse the direction and specify start hash
	return &worker{
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(0).Add(head.Number, big.NewInt(1)),
		targetHash:   ps.hash,
		targetNumber: ps.number,
		direction:    DIR_ASCENDING,
	}, nil
}

func (s *bootstrapSyncer) handleWorkerResult(w *worker) *worker {
	return nil
}

func (s *bootstrapSyncer) hasCurrentWorker(w *worker, workers map[uint64]*worker) bool {
	return false
}

func (s *bootstrapSyncer) handleTick() {}
