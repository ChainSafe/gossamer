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
	"math/big"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
)

var _ workHandler = &tipSyncer{}

// tipSyncer handles workers when syncing at the tip of the chain
type tipSyncer struct {
	blockState    BlockState
	pendingBlocks DisjointBlockSet
	readyBlocks   *blockQueue
}

func newTipSyncer(blockState BlockState, pendingBlocks DisjointBlockSet, readyBlocks *blockQueue) *tipSyncer {
	return &tipSyncer{
		blockState:    blockState,
		pendingBlocks: pendingBlocks,
		readyBlocks:   readyBlocks,
	}
}

func (s *tipSyncer) handleNewPeerState(ps *peerState) (*worker, error) {
	fin, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, err
	}

	if ps.number.Cmp(fin.Number) <= 0 {
		return nil, nil
	}

	return &worker{
		startHash:    ps.hash,
		startNumber:  ps.number,
		targetHash:   ps.hash,
		targetNumber: ps.number,
		requestData:  bootstrapRequestData,
	}, nil
}

func (s *tipSyncer) handleWorkerResult(res *worker) (*worker, error) {
	if res.err == nil {
		return nil, nil
	}

	if errors.Is(res.err.err, errUnknownParent) {
		// handleTick will handle the errUnknownParent case
		return nil, nil
	}

	fin, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, err
	}

	// don't retry if we're requesting blocks lower than finalised
	switch res.direction {
	case network.Ascending:
		if res.targetNumber.Cmp(fin.Number) <= 0 {
			return nil, nil
		}

		// if start is lower than finalised, increase it to finalised+1
		if res.startNumber.Cmp(fin.Number) <= 0 {
			res.startNumber = big.NewInt(0).Add(fin.Number, big.NewInt(1))
			res.startHash = common.Hash{}
		}
	case network.Descending:
		if res.startNumber.Cmp(fin.Number) <= 0 {
			return nil, nil
		}

		// if target is lower than finalised, increase it to finalised+1
		if res.targetNumber.Cmp(fin.Number) <= 0 {
			res.targetNumber = big.NewInt(0).Add(fin.Number, big.NewInt(1))
			res.targetHash = common.Hash{}
		}
	}

	return &worker{
		startHash:    res.startHash,
		startNumber:  res.startNumber,
		targetHash:   res.targetHash,
		targetNumber: res.targetNumber,
		direction:    res.direction,
		requestData:  res.requestData,
	}, nil
}

func (*tipSyncer) hasCurrentWorker(w *worker, workers map[uint64]*worker) bool {
	if w == nil || w.startNumber == nil || w.targetNumber == nil {
		return true
	}

	for _, curr := range workers {
		if w.direction != curr.direction || w.requestData != curr.requestData {
			continue
		}

		targetDiff := w.targetNumber.Cmp(curr.targetNumber)
		startDiff := w.startNumber.Cmp(curr.startNumber)

		switch w.direction {
		case network.Ascending:
			// worker target is greater than existing worker's target
			if targetDiff > 0 {
				continue
			}

			// worker start is less than existing worker's start
			if startDiff < 0 {
				continue
			}
		case network.Descending:
			// worker target is less than existing worker's target
			if targetDiff < 0 {
				continue
			}

			// worker start is greater than existing worker's start
			if startDiff > 0 {
				continue
			}
		}

		// worker (start, end) is within curr (start, end), if hashes are equal then the request is either
		// for the same data or some subset of data that is covered by curr
		if w.startHash.Equal(curr.startHash) || w.targetHash.Equal(curr.targetHash) {
			return true
		}
	}

	return false
}

// handleTick traverses the pending blocks set to find which forks still need to be requested
func (s *tipSyncer) handleTick() ([]*worker, error) {
	if s.pendingBlocks.size() == 0 {
		return nil, nil
	}

	fin, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, err
	}

	// cases for each block in pending set:
	// 1. only hash and number are known; in this case, request the full block (and ancestor chain)
	// 2. only header is known; in this case, request the block body
	// 3. entire block is known; in this case, check if we have become aware of the parent
	// if we have, move it to the ready blocks queue; otherwise, request the chain of ancestors

	var workers []*worker

	for _, block := range s.pendingBlocks.getBlocks() {
		if block.number.Cmp(fin.Number) <= 0 {
			// TODO: delete from pending set (this should not happen, it should have already been deleted)
			s.pendingBlocks.removeBlock(block.hash)
			continue
		}

		if block.header == nil {
			// case 1
			workers = append(workers, &worker{
				startHash:    block.hash,
				startNumber:  block.number,
				targetHash:   fin.Hash(),
				targetNumber: fin.Number,
				direction:    network.Descending,
				requestData:  bootstrapRequestData,
			})
			continue
		}

		if block.body == nil {
			// case 2
			workers = append(workers, &worker{
				startHash:    block.hash,
				startNumber:  block.number,
				targetHash:   block.hash,
				targetNumber: block.number,
				requestData:  network.RequestedDataBody + network.RequestedDataJustification,
			})
			continue
		}

		// case 3
		has, err := s.blockState.HasHeader(block.header.ParentHash)
		if err != nil {
			return nil, err
		}

		if has || s.readyBlocks.has(block.header.ParentHash) {
			// block is ready, as parent is known!
			// also, move any pendingBlocks that are descendants of this block to the ready blocks queue
			handleReadyBlock(block.toBlockData(), s.pendingBlocks, s.readyBlocks)
			continue
		}

		// request descending chain from (parent of pending block) -> (last finalised block)
		workers = append(workers, &worker{
			startHash:    block.header.ParentHash,
			startNumber:  big.NewInt(0).Sub(block.number, big.NewInt(1)),
			targetNumber: fin.Number,
			direction:    network.Descending,
			requestData:  bootstrapRequestData,
		})
	}

	return workers, nil
}
