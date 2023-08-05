// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var _ workHandler = &tipSyncer{}

type handleReadyBlockFunc func(*types.BlockData)

// tipSyncer handles workers when syncing at the tip of the chain
type tipSyncer struct {
	blockState       BlockState
	pendingBlocks    DisjointBlockSet
	readyBlocks      *blockQueue
	handleReadyBlock handleReadyBlockFunc
}

func newTipSyncer(blockState BlockState, pendingBlocks DisjointBlockSet, readyBlocks *blockQueue,
	handleReadyBlock handleReadyBlockFunc) *tipSyncer {
	return &tipSyncer{
		blockState:       blockState,
		pendingBlocks:    pendingBlocks,
		readyBlocks:      readyBlocks,
		handleReadyBlock: handleReadyBlock,
	}
}

func (s *tipSyncer) handleNewPeerState(ps *peerState) (*worker, error) {
	fin, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, err
	}

	if ps.number <= fin.Number {
		return nil, nil
	}

	return &worker{
		startHash:    ps.hash,
		startNumber:  uintPtr(ps.number),
		targetHash:   ps.hash,
		targetNumber: uintPtr(ps.number),
		requestData:  bootstrapRequestData,
	}, nil
}

func (s *tipSyncer) handleWorkerResult(res *worker) (
	workerToRetry *worker, err error) {
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
		if *res.targetNumber <= fin.Number {
			return nil, nil
		}

		// if start is lower than finalised, increase it to finalised+1
		if *res.startNumber <= fin.Number {
			*res.startNumber = fin.Number + 1
			res.startHash = common.Hash{}
		}
	case network.Descending:
		if *res.startNumber <= fin.Number {
			return nil, nil
		}

		// if target is lower than finalised, increase it to finalised+1
		if *res.targetNumber <= fin.Number {
			*res.targetNumber = fin.Number + 1
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

		switch w.direction {
		case network.Ascending:
			if *w.targetNumber > *curr.targetNumber ||
				*w.startNumber < *curr.startNumber {
				continue
			}
		case network.Descending:
			if *w.targetNumber < *curr.targetNumber ||
				*w.startNumber > *curr.startNumber {
				continue
			}
		}

		// worker (start, end) is within curr (start, end), if hashes are equal then the request is either
		// for the same data or some subset of data that is covered by curr
		if w.startHash == curr.startHash || w.targetHash == curr.targetHash {
			return true
		}
	}

	return false
}

// handleTick traverses the pending blocks set to find which forks still need to be requested
func (s *tipSyncer) handleTick() ([]*worker, error) {
	logger.Debugf("handling tick, we have %d pending blocks", s.pendingBlocks.size())

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
		if block.number <= fin.Number {
			// delete from pending set (this should not happen, it should have already been deleted)
			s.pendingBlocks.removeBlock(block.hash)
			continue
		}

		logger.Tracef("handling pending block number %d with hash %s", block.number, block.hash)

		if block.header == nil {
			// case 1
			workers = append(workers, &worker{
				startHash:    block.hash,
				startNumber:  uintPtr(block.number),
				targetHash:   fin.Hash(),
				targetNumber: uintPtr(fin.Number),
				direction:    network.Descending,
				requestData:  bootstrapRequestData,
				pendingBlock: block,
			})
			continue
		}

		if block.body == nil {
			// case 2
			workers = append(workers, &worker{
				startHash:    block.hash,
				startNumber:  uintPtr(block.number),
				targetHash:   block.hash,
				targetNumber: uintPtr(block.number),
				requestData:  network.RequestedDataBody + network.RequestedDataJustification,
				pendingBlock: block,
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
			s.handleReadyBlock(block.toBlockData())
			continue
		}

		// request descending chain from (parent of pending block) -> (last finalised block)
		workers = append(workers, &worker{
			startHash:    block.header.ParentHash,
			startNumber:  uintPtr(block.number - 1),
			targetNumber: uintPtr(fin.Number),
			direction:    network.Descending,
			requestData:  bootstrapRequestData,
			pendingBlock: block,
		})
	}

	return workers, nil
}
