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
)

var _ workHandler = &tipSyncer{}

// tipSyncer handles workers when syncing at the tip of the chain
type tipSyncer struct {
	blockState    BlockState
	pendingBlocks DisjointBlockSet
	readyBlocks   *blockQueue
	workerState   *workerState
}

func newTipSyncer(blockState BlockState, pendingBlocks DisjointBlockSet, readyBlocks *blockQueue, workerState *workerState) *tipSyncer {
	return &tipSyncer{
		blockState:    blockState,
		pendingBlocks: pendingBlocks,
		readyBlocks:   readyBlocks,
		workerState:   workerState,
	}
}

func (s *tipSyncer) handleNewPeerState(ps *peerState) (*worker, error) {
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

	if errors.Is(res.err.err, errUnknownParent) || res.err.err.Error() == "stream reset" { // TODO: use errors.Is
		// handleTick will handle this case
		return nil, nil
	}

	return &worker{
		startHash:    res.startHash,
		startNumber:  res.startNumber,
		targetHash:   res.targetHash,
		targetNumber: res.targetNumber,
		direction:    res.direction,
		requestData:  bootstrapRequestData,
	}, nil
}

func (s *tipSyncer) hasCurrentWorker(w *worker, workers map[uint64]*worker) bool {
	// TODO
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

	workers := []*worker{}

	for _, block := range s.pendingBlocks.getBlocks() {
		if block.header == nil {
			// case 1

			if block.number.Cmp(fin.Number) <= 0 {
				// TODO: delete from pending set (this should not happen, it should have already been deleted)
				continue
			}

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
		has, _ := s.blockState.HasHeader(block.header.ParentHash)
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
