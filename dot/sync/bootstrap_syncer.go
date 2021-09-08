package sync

import (
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

var _ workHandler = &bootstrapSyncer{}

type bootstrapSyncer struct {
	blockState    BlockState
	pendingBlocks DisjointBlockSet
}

func newBootstrapSyncer(blockState BlockState, pendingBlocks DisjointBlockSet) *bootstrapSyncer {
	return &bootstrapSyncer{
		blockState:    blockState,
		pendingBlocks: pendingBlocks,
	}
}

func (s *bootstrapSyncer) handleWork(ps *peerState) (*worker, error) {
	// if the peer reports a lower or equal best block number than us,
	// check if they are on a fork or not
	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	if ps.number.Cmp(head.Number) <= 0 {
		// check if our block hash for that number is the same, if so, do nothing
		hash, err := s.blockState.GetHashByNumber(ps.number)
		if err != nil {
			return nil, err
		}

		if hash.Equal(ps.hash) {
			return nil, nil
		}

		// check if their best block is on an invalid chain, if it is,
		// potentially downscore them
		// for now, we can remove them from the syncing peers set
		fin, err := s.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return nil, err
		}

		// their block hash doesn't match ours for that number (ie. they are on a different
		// chain), and also the highest finalised block is higher than that number.
		// thus the peer is on an invalid chain
		if fin.Number.Cmp(ps.number) >= 0 {
			// TODO: downscore this peer, or temporarily don't sync from them?
			logger.Trace("peer is on an invalid fork")
			return nil, nil
		}

		// TODO: peer is on a fork, add to pendingBlocks and begin fork request
		return nil, nil
	}

	// the peer has a higher best block than us, add it to the disjoint block set
	s.pendingBlocks.addHashAndNumber(ps.hash, ps.number)

	// TODO: this is for bootstrap mode, for idle fork-sync mode
	// we may want to reverse the direction and specify start hash
	return &worker{
		id:           s.nextWorker,
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(0).Add(head.Number, big.NewInt(1)),
		targetHash:   ps.hash,
		targetNumber: ps.number,
		direction:    DIR_ASCENDING,
	}, nil
}

func (s *bootstrapSyncer) handleWorkerResult(w *worker) *worker                        {}
func (s *bootstrapSyncer) hasCurrentWorker(w *worker, workers map[uint64]*worker) bool {}
func (s *bootstrapSyncer) handleTick()                                                 {}
