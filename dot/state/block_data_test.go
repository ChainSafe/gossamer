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
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func TestGetSet_ReceiptMessageQueue_Justification(t *testing.T) {
	s := newTestBlockState(t, nil)
	require.NotNil(t, s)

	var genesisHeader = &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	hash := common.NewHash([]byte{0})
	parentHash := genesisHeader.Hash()

	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	require.Nil(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.Nil(t, err)

	header := &types.Header{
		ParentHash:     parentHash,
		Number:         big.NewInt(1),
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.NewDigest(),
	}

	a := []byte("asdf")
	b := []byte("ghjkl")
	c := []byte("qwerty")
	body, err := types.NewBodyFromBytes([]byte{})
	require.NoError(t, err)

	bds := []*types.BlockData{{
		Hash:          header.Hash(),
		Header:        header,
		Body:          body,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}, {
		Hash:          hash,
		Header:        nil,
		Body:          body,
		Receipt:       &a,
		MessageQueue:  &b,
		Justification: &c,
	}}

	for _, blockdata := range bds {

		err := s.CompareAndSetBlockData(blockdata)
		require.Nil(t, err)

		// test Receipt
		if blockdata.Receipt != nil {
			receipt, err := s.GetReceipt(blockdata.Hash)
			require.Nil(t, err)
			require.Equal(t, *blockdata.Receipt, receipt)
		}

		// test MessageQueue
		if blockdata.MessageQueue != nil {
			messageQueue, err := s.GetMessageQueue(blockdata.Hash)
			require.Nil(t, err)
			require.Equal(t, *blockdata.MessageQueue, messageQueue)
		}
	}
}
