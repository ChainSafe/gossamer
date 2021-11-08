// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
)

var _ workHandler = &bootstrapSyncer{}

var bootstrapRequestData = network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification

// bootstrapSyncer handles worker logic for bootstrap mode
type bootstrapSyncer struct {
	blockState BlockState
}

func newBootstrapSyncer(blockState BlockState) *bootstrapSyncer {
	return &bootstrapSyncer{
		blockState: blockState,
	}
}

func (s *bootstrapSyncer) handleNewPeerState(ps *peerState) (*worker, error) {
	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	if ps.number.Cmp(head.Number) <= 0 {
		return nil, nil
	}

	return &worker{
		startNumber:  big.NewInt(0).Add(head.Number, big.NewInt(1)),
		targetHash:   ps.hash,
		targetNumber: ps.number,
		requestData:  bootstrapRequestData,
		direction:    network.Ascending,
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

	// we've reached the target, return
	if res.targetNumber.Cmp(head.Number) <= 0 {
		return nil, nil
	}

	startNumber := big.NewInt(0).Add(head.Number, big.NewInt(1))

	// in the case we started a block producing node, we might have produced blocks
	// before fully syncing (this should probably be fixed by connecting sync into BABE)
	if errors.Is(res.err.err, errUnknownParent) {
		fin, err := s.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return nil, err
		}

		startNumber = fin.Number
	}

	return &worker{
		startHash:    common.Hash{}, // for bootstrap, just use number
		startNumber:  startNumber,
		targetHash:   res.targetHash,
		targetNumber: res.targetNumber,
		requestData:  res.requestData,
		direction:    res.direction,
	}, nil
}

func (*bootstrapSyncer) hasCurrentWorker(_ *worker, workers map[uint64]*worker) bool {
	// we're in bootstrap mode, and there already is a worker, we don't need to dispatch another
	return len(workers) != 0
}

func (*bootstrapSyncer) handleTick() ([]*worker, error) {
	return nil, nil
}
