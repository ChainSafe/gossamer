// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"

	"github.com/stretchr/testify/require"
)

func TestGetSet_ReceiptMessageQueue_Justification(t *testing.T) {
	s := newTestBlockState(t, newTriesEmpty())

	var genesisHeader = &types.Header{
		Number:    0,
		StateRoot: inmemory_trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	hash := common.NewHash([]byte{0})
	parentHash := genesisHeader.Hash()

	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.NoError(t, err)

	header := &types.Header{
		ParentHash:     parentHash,
		Number:         1,
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
		require.NoError(t, err)

		// test Receipt
		if blockdata.Receipt != nil {
			receipt, err := s.GetReceipt(blockdata.Hash)
			require.NoError(t, err)
			require.Equal(t, *blockdata.Receipt, receipt)
		}

		// test MessageQueue
		if blockdata.MessageQueue != nil {
			messageQueue, err := s.GetMessageQueue(blockdata.Hash)
			require.NoError(t, err)
			require.Equal(t, *blockdata.MessageQueue, messageQueue)
		}
	}
}
