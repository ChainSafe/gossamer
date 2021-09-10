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
	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	if ps.number.Cmp(head.Number) <= 0 {
		return nil, nil
	}

	return &worker{
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(0).Add(head.Number, big.NewInt(1)),
		targetHash:   ps.hash,
		targetNumber: ps.number,
		direction:    DIR_ASCENDING,
	}, nil
}

func (s *bootstrapSyncer) handleWorkerResult(res *worker) (*worker, error) {
	// if there is an error, potentially retry the worker
	if res.err == nil {
		return nil, nil
	}

	// new worker should update start block and re-dispatch
	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	return &worker{
		startHash:    common.EmptyHash, // for bootstrap, just use number
		startNumber:  big.NewInt(0).Add(head.Number, big.NewInt(1)),
		targetHash:   res.targetHash,
		targetNumber: res.targetNumber,
		direction:    res.direction,
	}, nil
}

func (s *bootstrapSyncer) hasCurrentWorker(_ *worker, workers map[uint64]*worker) bool {
	// we're in bootstrap mode, and there already is a worker, we don't need to dispatch another
	return len(workers) != 0
}

func (s *bootstrapSyncer) handleTick() {}
