// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

func TestHighestRoundAndSetID(t *testing.T) {
	bs := newTestDefaultBlockState(t, newTriesEmpty())
	round, setID, err := bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(0), round)
	require.Equal(t, uint64(0), setID)

	err = bs.SetHighestRoundAndSetID(1, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), round)
	require.Equal(t, uint64(0), setID)

	err = bs.SetHighestRoundAndSetID(10, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(10), round)
	require.Equal(t, uint64(0), setID)

	// is possible to have a lower round number
	// in the same set ID: https://github.com/ChainSafe/gossamer/issues/3150
	err = bs.SetHighestRoundAndSetID(9, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(9), round)
	require.Equal(t, uint64(0), setID)

	err = bs.SetHighestRoundAndSetID(0, 1)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(0), round)
	require.Equal(t, uint64(1), setID)

	err = bs.SetHighestRoundAndSetID(100000, 0)
	require.ErrorIs(t, err, errSetIDLowerThanHighest)
	const expectedErrorMessage = "set id lower than highest: 0 should be greater or equal 1"
	require.EqualError(t, err, expectedErrorMessage)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(0), round)
	require.Equal(t, uint64(1), setID)
}

func TestBlockState_SetFinalisedHash(t *testing.T) {
	bs := newTestDefaultBlockState(t, newTriesEmpty())
	h, err := bs.getFinalisedHash(0, 0)
	require.NoError(t, err)
	require.Equal(t, testGenesisHeader.Hash(), h)

	digest := types.NewDigest()
	di, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, di)
	err = digest.Add(*di)
	require.NoError(t, err)

	someStateRoot := common.Hash{1, 1}
	header := &types.Header{
		ParentHash: testGenesisHeader.Hash(),
		Number:     1,
		Digest:     digest,
		StateRoot:  someStateRoot,
	}

	testhash := header.Hash()
	err = bs.SetHeader(header)
	require.NoError(t, err)

	err = bs.AddBlock(&types.Block{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	// set tries with some state root
	// TODO: check if this is needed
	//bs.tries.softSet(someStateRoot, inmemory_trie.NewEmptyTrie())

	err = bs.SetFinalisedHash(testhash, 1, 1, true)
	require.NoError(t, err)

	h, err = bs.getFinalisedHash(1, 1)
	require.NoError(t, err)
	require.Equal(t, testhash, h)
}

func TestSetFinalisedHash_retrieveBlockNumber1SlotNumber(t *testing.T) {
	bs := newTestDefaultBlockState(t, newTriesEmpty())
	firstSlot := uint64(42069)

	digest := types.NewDigest()
	di, err := types.NewBabeSecondaryPlainPreDigest(0, firstSlot).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, di)
	err = digest.Add(*di)
	require.NoError(t, err)
	digest2 := types.NewDigest()
	di, err = types.NewBabeSecondaryPlainPreDigest(0, firstSlot+100).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, di)
	err = digest2.Add(*di)
	require.NoError(t, err)

	header1 := types.Header{
		Number:     1,
		Digest:     digest,
		ParentHash: testGenesisHeader.Hash(),
	}

	header2 := types.Header{
		Number:     2,
		Digest:     digest2,
		ParentHash: header1.Hash(),
	}

	err = bs.AddBlock(&types.Block{
		Header: header1,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = bs.AddBlock(&types.Block{
		Header: header2,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = bs.SetFinalisedHash(header2.Hash(), 1, 1, true)
	require.NoError(t, err)
	require.Equal(t, header2.Hash(), bs.GetLastFinalized())

	hashes, err := bs.GetHashesByNumber(1)
	require.NoError(t, err)
	require.Len(t, hashes, 1)

	blockNumber1Header, err := bs.GetHeader(hashes[0])
	require.NoError(t, err)

	veryFirstSlot, err := blockNumber1Header.SlotNumber()
	require.NoError(t, err)

	require.Equal(t, firstSlot, veryFirstSlot)
}
