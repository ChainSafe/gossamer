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
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

func TestDisjointBlockSet(t *testing.T) {
	s := newDisjointBlockSet(pendingBlocksLimit)

	hash := common.Hash{0xa, 0xb}
	number := big.NewInt(100)
	s.addHashAndNumber(hash, number)
	require.True(t, s.hasBlock(hash))
	require.Equal(t, 1, s.size())

	expected := &pendingBlock{
		hash:   hash,
		number: number,
	}
	blocks := s.getBlocks()
	require.Equal(t, 1, len(blocks))
	require.Equal(t, expected, blocks[0])

	header := &types.Header{
		Number: big.NewInt(100),
	}
	s.addHeader(header)
	require.True(t, s.hasBlock(header.Hash()))
	require.Equal(t, 2, s.size())
	expected = &pendingBlock{
		hash:   header.Hash(),
		number: header.Number,
		header: header,
	}
	require.Equal(t, expected, s.getBlock(header.Hash()))

	header2 := &types.Header{
		Number: big.NewInt(999),
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
	require.Equal(t, expected, s.getBlock(header2.Hash()))

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
	require.Equal(t, expected, s.getBlock(header2.Hash()))

	s.removeBlock(hash)
	require.Equal(t, 2, s.size())
	require.False(t, s.hasBlock(hash))

	s.removeLowerBlocks(big.NewInt(998))
	require.Equal(t, 1, s.size())
	require.False(t, s.hasBlock(header.Hash()))
	require.True(t, s.hasBlock(header2.Hash()))
}

func TestPendingBlock_toBlockData(t *testing.T) {
	pb := &pendingBlock{
		hash:   common.Hash{0xa, 0xb, 0xc},
		number: big.NewInt(1),
		header: &types.Header{
			Number: big.NewInt(1),
		},
		body: &types.Body{0x1, 0x2, 0x3},
	}

	expected := &types.BlockData{
		Hash:   pb.hash,
		Header: pb.header,
		Body:   pb.body,
	}

	require.Equal(t, expected, pb.toBlockData())
}

func TestDisjointBlockSet_getReadyDescendants(t *testing.T) {
	s := newDisjointBlockSet(pendingBlocksLimit)

	// test that descendant chain gets returned by getReadyDescendants on block 1 being ready
	header1 := &types.Header{
		Number: big.NewInt(1),
	}
	block1 := &types.Block{
		Header: *header1,
		Body:   types.Body{},
	}

	header2 := &types.Header{
		ParentHash: header1.Hash(),
		Number:     big.NewInt(2),
	}
	block2 := &types.Block{
		Header: *header2,
		Body:   types.Body{},
	}
	s.addBlock(block2)

	header3 := &types.Header{
		ParentHash: header2.Hash(),
		Number:     big.NewInt(3),
	}
	block3 := &types.Block{
		Header: *header3,
		Body:   types.Body{},
	}
	s.addBlock(block3)

	header2NotDescendant := &types.Header{
		ParentHash: common.Hash{0xff},
		Number:     big.NewInt(2),
	}
	block2NotDescendant := &types.Block{
		Header: *header2NotDescendant,
		Body:   types.Body{},
	}
	s.addBlock(block2NotDescendant)

	ready := []*types.BlockData{block1.ToBlockData()}
	ready = s.getReadyDescendants(header1.Hash(), ready)
	require.Equal(t, 3, len(ready))
	require.Equal(t, block1.ToBlockData(), ready[0])
	require.Equal(t, block2.ToBlockData(), ready[1])
	require.Equal(t, block3.ToBlockData(), ready[2])
}

func TestDisjointBlockSet_getReadyDescendants_blockNotComplete(t *testing.T) {
	s := newDisjointBlockSet(pendingBlocksLimit)

	// test that descendant chain gets returned by getReadyDescendants on block 1 being ready
	// the ready list should contain only block 1 and 2, as block 3 is incomplete (body is missing)
	header1 := &types.Header{
		Number: big.NewInt(1),
	}
	block1 := &types.Block{
		Header: *header1,
		Body:   types.Body{},
	}

	header2 := &types.Header{
		ParentHash: header1.Hash(),
		Number:     big.NewInt(2),
	}
	block2 := &types.Block{
		Header: *header2,
		Body:   types.Body{},
	}
	s.addBlock(block2)

	header3 := &types.Header{
		ParentHash: header2.Hash(),
		Number:     big.NewInt(3),
	}
	s.addHeader(header3)

	header2NotDescendant := &types.Header{
		ParentHash: common.Hash{0xff},
		Number:     big.NewInt(2),
	}
	block2NotDescendant := &types.Block{
		Header: *header2NotDescendant,
		Body:   types.Body{},
	}
	s.addBlock(block2NotDescendant)

	ready := []*types.BlockData{block1.ToBlockData()}
	ready = s.getReadyDescendants(header1.Hash(), ready)
	require.Equal(t, 2, len(ready))
	require.Equal(t, block1.ToBlockData(), ready[0])
	require.Equal(t, block2.ToBlockData(), ready[1])
}
