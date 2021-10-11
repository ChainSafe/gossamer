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

	digest := types.NewDigest()
	err := digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())
	require.NoError(t, err)
	digest2 := types.NewDigest()
	err = digest2.Add(*types.NewBabeSecondaryPlainPreDigest(0, 2).ToPreRuntimeDigest())
	require.NoError(t, err)
	digest3 := types.NewDigest()
	err = digest3.Add(*types.NewBabeSecondaryPlainPreDigest(0, 200).ToPreRuntimeDigest())
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

	header2Again := types.Header{
		Number:     big.NewInt(2),
		Digest:     digest3,
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

	err = bs.AddBlock(&types.Block{
		Header: header2Again,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = bs.SetFinalisedHash(header2Again.Hash(), 0, 0)
	require.NoError(t, err)
	require.Equal(t, header2Again.Hash(), bs.lastFinalised)

	h1, err := bs.GetHeaderByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, &header1, h1)

	h2, err := bs.GetHeaderByNumber(big.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, &header2Again, h2)
}
