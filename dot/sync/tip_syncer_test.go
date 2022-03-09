// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestTipSyncer(t *testing.T) *tipSyncer {
	finHeader, err := types.NewHeader(common.NewHash([]byte{0}),
		trie.EmptyHash, trie.EmptyHash, 200, types.NewDigest())
	require.NoError(t, err)

	bs := new(syncmocks.BlockState)
	bs.On("GetHighestFinalisedHeader").Return(finHeader, nil)
	bs.On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	readyBlocks := newBlockQueue(maxResponseSize)
	pendingBlocks := newDisjointBlockSet(pendingBlocksLimit)
	cs := &chainSync{
		blockState:    bs,
		readyBlocks:   readyBlocks,
		pendingBlocks: pendingBlocks,
	}

	return newTipSyncer(bs, pendingBlocks, readyBlocks, cs.handleReadyBlock)
}

func TestTipSyncer_handleNewPeerState(t *testing.T) {
	s := newTestTipSyncer(t)

	// peer reports state lower than our highest finalised, we should ignore
	ps := &peerState{
		number: 1,
	}

	w, err := s.handleNewPeerState(ps)
	require.NoError(t, err)
	require.Nil(t, w)

	ps = &peerState{
		number: 201,
		hash:   common.Hash{0xa, 0xb},
	}

	// otherwise, return a worker
	expected := &worker{
		startNumber:  uintPtr(ps.number),
		startHash:    ps.hash,
		targetNumber: uintPtr(ps.number),
		targetHash:   ps.hash,
		requestData:  bootstrapRequestData,
	}

	w, err = s.handleNewPeerState(ps)
	require.NoError(t, err)
	require.Equal(t, expected, w)
}

func TestTipSyncer_handleWorkerResult(t *testing.T) {
	s := newTestTipSyncer(t)

	w, err := s.handleWorkerResult(&worker{})
	require.NoError(t, err)
	require.Nil(t, w)

	w, err = s.handleWorkerResult(&worker{
		err: &workerError{
			err: errUnknownParent,
		},
	})
	require.NoError(t, err)
	require.Nil(t, w)

	// worker is for blocks lower than finalised
	w, err = s.handleWorkerResult(&worker{
		targetNumber: uintPtr(199),
	})
	require.NoError(t, err)
	require.Nil(t, w)

	w, err = s.handleWorkerResult(&worker{
		direction:   network.Descending,
		startNumber: uintPtr(199),
	})
	require.NoError(t, err)
	require.Nil(t, w)

	// worker start is lower than finalised, start should be updated
	expected := &worker{
		direction:    network.Ascending,
		startNumber:  uintPtr(201),
		targetNumber: uintPtr(300),
		requestData:  bootstrapRequestData,
	}

	w, err = s.handleWorkerResult(&worker{
		direction:    network.Ascending,
		startNumber:  uintPtr(199),
		targetNumber: uintPtr(300),
		requestData:  bootstrapRequestData,
		err:          &workerError{},
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)

	expected = &worker{
		direction:    network.Descending,
		startNumber:  uintPtr(300),
		targetNumber: uintPtr(201),
		requestData:  bootstrapRequestData,
	}

	w, err = s.handleWorkerResult(&worker{
		direction:    network.Descending,
		startNumber:  uintPtr(300),
		targetNumber: uintPtr(199),
		requestData:  bootstrapRequestData,
		err:          &workerError{},
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)

	// start and target are higher than finalised, don't modify
	expected = &worker{
		direction:    network.Descending,
		startNumber:  uintPtr(300),
		startHash:    common.Hash{0xa, 0xb},
		targetNumber: uintPtr(201),
		targetHash:   common.Hash{0xc, 0xd},
		requestData:  bootstrapRequestData,
	}

	w, err = s.handleWorkerResult(&worker{
		direction:    network.Descending,
		startNumber:  uintPtr(300),
		startHash:    common.Hash{0xa, 0xb},
		targetNumber: uintPtr(201),
		targetHash:   common.Hash{0xc, 0xd},
		requestData:  bootstrapRequestData,
		err:          &workerError{},
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)
}

func TestTipSyncer_handleTick_case1(t *testing.T) {
	s := newTestTipSyncer(t)

	w, err := s.handleTick()
	require.NoError(t, err)
	require.Nil(t, w)

	fin, _ := s.blockState.GetHighestFinalisedHeader()

	// add pending blocks w/ only hash and number, equal or lower than finalised should be removed
	s.pendingBlocks.addHashAndNumber(common.Hash{0xa}, fin.Number)
	s.pendingBlocks.addHashAndNumber(common.Hash{0xb}, fin.Number+1)

	expected := []*worker{
		{
			startHash:    common.Hash{0xb},
			startNumber:  uintPtr(fin.Number + 1),
			targetHash:   fin.Hash(),
			targetNumber: uintPtr(fin.Number),
			direction:    network.Descending,
			requestData:  bootstrapRequestData,
			pendingBlock: s.pendingBlocks.getBlock(common.Hash{0xb}),
		},
	}

	w, err = s.handleTick()
	require.NoError(t, err)
	require.Equal(t, expected, w)
	require.False(t, s.pendingBlocks.hasBlock(common.Hash{0xa}))
	require.True(t, s.pendingBlocks.hasBlock(common.Hash{0xb}))
}

func TestTipSyncer_handleTick_case2(t *testing.T) {
	s := newTestTipSyncer(t)

	fin, _ := s.blockState.GetHighestFinalisedHeader()

	// add pending blocks w/ only header
	header := &types.Header{
		Number: fin.Number + 1,
	}
	s.pendingBlocks.addHeader(header)

	expected := []*worker{
		{
			startHash:    header.Hash(),
			startNumber:  uintPtr(header.Number),
			targetHash:   header.Hash(),
			targetNumber: uintPtr(header.Number),
			direction:    network.Ascending,
			requestData:  network.RequestedDataBody + network.RequestedDataJustification,
			pendingBlock: s.pendingBlocks.getBlock(header.Hash()),
		},
	}
	w, err := s.handleTick()
	require.NoError(t, err)
	require.Equal(t, expected, w)
	require.True(t, s.pendingBlocks.hasBlock(header.Hash()))
}

func TestTipSyncer_handleTick_case3(t *testing.T) {
	s := newTestTipSyncer(t)

	fin, _ := s.blockState.GetHighestFinalisedHeader()

	// add pending block w/ full block, HasHeader will return true, so the block will be processed
	header := &types.Header{
		Number: fin.Number + 1,
	}
	block := &types.Block{
		Header: *header,
		Body:   types.Body{},
	}
	s.pendingBlocks.addBlock(block)

	w, err := s.handleTick()
	require.NoError(t, err)
	require.Equal(t, []*worker(nil), w)
	require.False(t, s.pendingBlocks.hasBlock(header.Hash()))
	require.Equal(t, block.ToBlockData(), s.readyBlocks.pop())

	// add pending block w/ full block, but block is not ready as parent is unknown
	bs := new(syncmocks.BlockState)
	bs.On("GetHighestFinalisedHeader").Return(fin, nil)
	bs.On("HasHeader", mock.AnythingOfType("common.Hash")).Return(false, nil)
	s.blockState = bs

	header = &types.Header{
		Number: fin.Number + 100,
	}
	block = &types.Block{
		Header: *header,
		Body:   types.Body{},
	}
	s.pendingBlocks.addBlock(block)

	expected := []*worker{
		{
			startHash:    header.ParentHash,
			startNumber:  uintPtr(header.Number - 1),
			targetNumber: uintPtr(fin.Number),
			direction:    network.Descending,
			requestData:  bootstrapRequestData,
			pendingBlock: s.pendingBlocks.getBlock(header.Hash()),
		},
	}

	w, err = s.handleTick()
	require.NoError(t, err)
	require.Equal(t, expected, w)
	require.True(t, s.pendingBlocks.hasBlock(header.Hash()))

	// add parent block to readyBlocks, should move block to readyBlocks
	s.readyBlocks.push(&types.BlockData{
		Hash: header.ParentHash,
	})
	w, err = s.handleTick()
	require.NoError(t, err)
	require.Equal(t, []*worker(nil), w)
	require.False(t, s.pendingBlocks.hasBlock(header.Hash()))
	s.readyBlocks.pop() // first pop will remove parent
	require.Equal(t, block.ToBlockData(), s.readyBlocks.pop())
}

func TestTipSyncer_hasCurrentWorker(t *testing.T) {
	s := newTestTipSyncer(t)
	require.False(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(0),
		targetNumber: uintPtr(0),
	}, nil))

	workers := make(map[uint64]*worker)
	workers[0] = &worker{
		startNumber:  uintPtr(1),
		targetNumber: uintPtr(128),
	}
	require.False(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(1),
		targetNumber: uintPtr(129),
	}, workers))
	require.True(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(1),
		targetNumber: uintPtr(128),
	}, workers))
	require.True(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(1),
		targetNumber: uintPtr(127),
	}, workers))

	workers[0] = &worker{
		startNumber:  uintPtr(128),
		targetNumber: uintPtr(255),
	}
	require.False(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(127),
		targetNumber: uintPtr(255),
	}, workers))
	require.True(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(128),
		targetNumber: uintPtr(255),
	}, workers))

	workers[0] = &worker{
		startNumber:  uintPtr(128),
		targetNumber: uintPtr(1),
		direction:    network.Descending,
	}
	require.False(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(129),
		targetNumber: uintPtr(1),
		direction:    network.Descending,
	}, workers))
	require.True(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(128),
		targetNumber: uintPtr(1),
		direction:    network.Descending,
	}, workers))
	require.True(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(128),
		targetNumber: uintPtr(2),
		direction:    network.Descending,
	}, workers))
	require.True(t, s.hasCurrentWorker(&worker{
		startNumber:  uintPtr(127),
		targetNumber: uintPtr(1),
		direction:    network.Descending,
	}, workers))
}
