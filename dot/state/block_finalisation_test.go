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

package state

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/stretchr/testify/require"
)

func TestHighestRoundAndSetID(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	round, setID, err := bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(0), round)
	require.Equal(t, uint64(0), setID)

	err = bs.setHighestRoundAndSetID(1, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), round)
	require.Equal(t, uint64(0), setID)

	err = bs.setHighestRoundAndSetID(10, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(10), round)
	require.Equal(t, uint64(0), setID)

	err = bs.setHighestRoundAndSetID(9, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(10), round)
	require.Equal(t, uint64(0), setID)

	err = bs.setHighestRoundAndSetID(0, 1)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(0), round)
	require.Equal(t, uint64(1), setID)

	err = bs.setHighestRoundAndSetID(100000, 0)
	require.NoError(t, err)

	round, setID, err = bs.GetHighestRoundAndSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(0), round)
	require.Equal(t, uint64(1), setID)
}

func TestBlockState_SetFinalisedHash(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	h, err := bs.GetFinalisedHash(0, 0)
	require.NoError(t, err)
	require.Equal(t, testGenesisHeader.Hash(), h)

	digest := types.NewDigest()
	di, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, di)
	err = digest.Add(*di)
	require.NoError(t, err)
	header := &types.Header{
		ParentHash: testGenesisHeader.Hash(),
		Number:     big.NewInt(1),
		Digest:     digest,
	}

	testhash := header.Hash()
	err = bs.db.Put(headerKey(testhash), []byte{})
	require.NoError(t, err)

	err = bs.AddBlock(&types.Block{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = bs.SetFinalisedHash(testhash, 1, 1)
	require.NoError(t, err)

	h, err = bs.GetFinalisedHash(1, 1)
	require.NoError(t, err)
	require.Equal(t, testhash, h)
}

func TestSetFinalisedHash_setFirstSlotOnFinalisation(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
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
		Number:     big.NewInt(1),
		Digest:     digest,
		ParentHash: testGenesisHeader.Hash(),
	}

	header2 := types.Header{
		Number:     big.NewInt(2),
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

	err = bs.SetFinalisedHash(header2.Hash(), 1, 1)
	require.NoError(t, err)
	require.Equal(t, header2.Hash(), bs.lastFinalised)

	res, err := bs.baseState.loadFirstSlot()
	require.NoError(t, err)
	require.Equal(t, firstSlot, res)
}
