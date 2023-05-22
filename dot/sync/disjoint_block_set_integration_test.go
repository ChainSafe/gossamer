//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisjointBlockSet(t *testing.T) {
	s := newDisjointBlockSet(pendingBlocksLimit)

	hash := common.Hash{0xa, 0xb}
	const number uint = 100
	s.addHashAndNumber(hash, number)
	require.True(t, s.hasBlock(hash))
	require.Equal(t, 1, s.size())

	expected := &pendingBlock{
		hash:   hash,
		number: number,
	}
	blocks := s.getBlocks()
	require.Equal(t, 1, len(blocks))
	assert.Greater(t, blocks[0].clearAt, time.Now().Add(ttl-time.Minute))
	blocks[0].clearAt = time.Time{}
	require.Equal(t, expected, blocks[0])

	header := &types.Header{
		Number: 100,
	}
	s.addHeader(header)
	require.True(t, s.hasBlock(header.Hash()))
	require.Equal(t, 2, s.size())
	expected = &pendingBlock{
		hash:   header.Hash(),
		number: header.Number,
		header: header,
	}
	block1 := s.getBlock(header.Hash())
	assert.Greater(t, block1.clearAt, time.Now().Add(ttl-time.Minute))
	block1.clearAt = time.Time{}
	require.Equal(t, expected, block1)

	header2 := &types.Header{
		Number: 999,
	}
	s.addHashAndNumber(header2.Hash(), header2.Number)
	require.Equal(t, 3, s.size())
	s.addHeader(header2)
	require.Equal(t, 3, s.size())
	expected = &pendingBlock{
		hash:   header2.Hash(),
		number: header2.Number,
		header: header2,
	}
	block2 := s.getBlock(header2.Hash())
	assert.Greater(t, block2.clearAt, time.Now().Add(ttl-time.Minute))
	block2.clearAt = time.Time{}
	require.Equal(t, expected, block2)

	block := &types.Block{
		Header: *header2,
		Body:   types.Body{{0xa}},
	}
	s.addBlock(block)
	require.Equal(t, 3, s.size())
	expected = &pendingBlock{
		hash:   header2.Hash(),
		number: header2.Number,
		header: header2,
		body:   &block.Body,
	}
	block3 := s.getBlock(header2.Hash())
	assert.Greater(t, block3.clearAt, time.Now().Add(ttl-time.Minute))
	block3.clearAt = time.Time{}
	require.Equal(t, expected, block3)

	s.removeBlock(hash)
	require.Equal(t, 2, s.size())
	require.False(t, s.hasBlock(hash))

	s.removeLowerBlocks(998)
	require.Equal(t, 1, s.size())
	require.False(t, s.hasBlock(header.Hash()))
	require.True(t, s.hasBlock(header2.Hash()))
}

func TestPendingBlock_toBlockData(t *testing.T) {
	pb := &pendingBlock{
		hash:   common.Hash{0xa, 0xb, 0xc},
		number: 1,
		header: &types.Header{
			Number: 1,
		},
		body: &types.Body{{0x1, 0x2, 0x3}},
	}

	expected := &types.BlockData{
		Hash:   pb.hash,
		Header: pb.header,
		Body:   pb.body,
	}

	require.Equal(t, expected, pb.toBlockData())
}

func TestDisjointBlockSet_ClearBlocks(t *testing.T) {
	s := newDisjointBlockSet(pendingBlocksLimit)

	testHashA := common.Hash{0}
	testHashB := common.Hash{1}

	s.blocks[testHashA] = &pendingBlock{
		hash:    testHashA,
		clearAt: time.Unix(1000, 0),
	}
	s.blocks[testHashB] = &pendingBlock{
		hash:    testHashB,
		clearAt: time.Now().Add(ttl * 2),
	}

	s.clearBlocks()
	require.Equal(t, 1, len(s.blocks))
	_, has := s.blocks[testHashB]
	require.True(t, has)
}
