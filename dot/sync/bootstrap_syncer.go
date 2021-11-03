// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"errors"

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
	if ps.number == nil {
		return nil, errNilPeerStateNumber
	}

	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	if *ps.number <= head.Number {
		return nil, nil
	}

	startNumber := head.Number + 1

	return &worker{
		startNumber:  &startNumber,
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

	if res.targetNumber == nil {
		return nil, errNilWorkerTargetNumber
	}

	// new worker should update start block and re-dispatch
	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	// we've reached the target, return
	if *res.targetNumber <= head.Number {
		return nil, nil
	}

	startNumber := head.Number + 1

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
		startNumber:  &startNumber,
		targetHash:   res.targetHash,
		targetNumber: res.targetNumber,
		requestData:  res.requestData,
		direction:    res.direction,
	}, nil
}

func (*bootstrapSyncer) hasCurrentWorker(_ *worker, workers map[uint64]*worker) (ok bool, _ error) {
	// we're in bootstrap mode, and there already is a worker, we don't need to dispatch another
	return len(workers) != 0, nil
}

func (*bootstrapSyncer) handleTick() ([]*worker, error) {
	return nil, nil
}
