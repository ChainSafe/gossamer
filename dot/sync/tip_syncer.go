package sync

var _ workHandler = &tipSyncer{}

type tipSyncer struct {
	blockState BlockState
}

func newTipSyncer(blockState BlockState) *tipSyncer {
	return &tipSyncer{
		blockState: blockState,
	}
}

func (s *tipSyncer) handleWork(ps *peerState) (*worker, error) {
	return nil, nil
}

func (s *tipSyncer) handleWorkerResult(res *worker) (*worker, error) {
	return nil, nil
}

func (s *tipSyncer) hasCurrentWorker(_ *worker, workers map[uint64]*worker) bool {
	// we're in bootstrap mode, and there already is a worker, we don't need to dispatch another
	return false
}

func (s *tipSyncer) handleTick() {}
