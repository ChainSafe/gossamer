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

package core

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

func addTestBlocksToStateWithDigest(t *testing.T, depth int, blockState BlockState, babesession *babe.Session) {
	previousHash := blockState.BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	require.Nil(t, err)

	for i := 1; i <= depth; i++ {

		// TODO: don't hard code this, move this to BABE
		//predigest := babesession.buildBlockPreDigest(babe.Slot{Number: 2})
		preDigest, err := common.HexToBytes("0x014241424538e93dcef2efc275b72b4fa748332dc4c9f13be1125909cf90c8e9109c45da16b04bc5fdf9fe06a4f35e4ae4ed7e251ff9ee3d0d840c8237c9fb9057442dbf00f210d697a7b4959f792a81b948ff88937e30bf9709a8ab1314f71284da89a40000000000000000001100000000000000")
		require.Nil(t, err)

		nextEpochData := &babe.NextEpochDescriptor{
			Authorities: babesession.AuthorityData(),
		}

		consensusDigest := &types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              nextEpochData.Encode(),
		}

		conDigest := consensusDigest.Encode()

		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
				Digest:     [][]byte{preDigest, conDigest},
			},
			Body: &types.Body{},
		}

		previousHash = block.Header.Hash()

		err = blockState.AddBlock(block)
		require.Nil(t, err)
	}
}

// test handleBlockDigest
func TestHandleBlockDigest(t *testing.T) {
	s := newTestServiceWithBabeSession(t)
	addTestBlocksToStateWithDigest(t, 2, s.blockState, s.bs)

	s.epochNumber = uint64(2) // test preDigest item is for block in slot 17 epoch 2

	number, err := s.blockState.BestBlockNumber()
	require.Nil(t, err)

	t.Log(number)

	header, err := s.blockState.BestBlockHeader()
	require.Nil(t, err)

	err = s.handleBlockDigest(header)
	require.Nil(t, err)

	require.Equal(t, number, s.firstBlock)

	t.Log(header.Number)

	// test two blocks claiming to be first block
	err = s.handleBlockDigest(header)
	require.NotNil(t, err) // expect error: "first block already set for current epoch"

	// expect first block not to be updated
	require.Equal(t, s.firstBlock, big.NewInt(2))

	// test two blocks claiming to be first block
	// block with lower number than existing `firstBlock` should be chosen
	s.firstBlock = big.NewInt(0).Add(number, big.NewInt(1))

	err = s.handleBlockDigest(header)
	require.Nil(t, err)

	// expect first block to be updated
	require.Equal(t, s.firstBlock, number)
}

// test handleConsensusDigest
func TestHandleConsensusDigest(t *testing.T) {
	s := newTestService(t, nil)
	addTestBlocksToStateWithDigest(t, 1, s.blockState, s.bs)

	number, err := s.blockState.BestBlockNumber()
	require.Nil(t, err)

	header, err := s.blockState.BestBlockHeader()
	require.Nil(t, err)

	var item types.DigestItem

	for _, digest := range header.Digest {
		item, err = types.DecodeDigestItem(digest)
		require.Nil(t, err)
	}

	// check if digest item is consensus digest type
	if item.Type() == types.ConsensusDigestType {
		digest := item.(*types.ConsensusDigest)

		err = s.handleConsensusDigest(header, digest)
		require.Nil(t, err)
	}

	require.Equal(t, number, s.firstBlock)
}

// test setNextEpochDescriptor
func TestSetNextEpochDescriptor(t *testing.T) {
	s := newTestService(t, nil)
	addTestBlocksToStateWithDigest(t, 1, s.blockState, s.bs)

	header, err := s.blockState.BestBlockHeader()
	require.Nil(t, err)

	var item types.DigestItem

	for _, digest := range header.Digest {
		item, err = types.DecodeDigestItem(digest)
		require.Nil(t, err)
	}

	// check if digest item is consensus digest type
	if item.Type() == types.ConsensusDigestType {
		digest := item.(*types.ConsensusDigest)

		err = s.setNextEpochDescriptor(digest.Data)
		require.Nil(t, err)
	}
}
