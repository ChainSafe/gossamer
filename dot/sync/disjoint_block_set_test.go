package sync

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

func TestDisjointBlockSet(t *testing.T) {
	s := newDisjointBlockSet()

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
		Header: header2,
		Body:   &types.Body{0xc, 0xd},
	}
	s.addBlock(block)
	require.Equal(t, 3, s.size())
	expected = &pendingBlock{
		hash:   header2.Hash(),
		number: header2.Number,
		header: header2,
		body:   block.Body,
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
