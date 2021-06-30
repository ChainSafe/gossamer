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
	"time"

	. "github.com/ChainSafe/gossamer/dot/core/mocks" // nolint
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/stretchr/testify/require"
)

func TestService_ProcessBlockAnnounceMessage(t *testing.T) {
	// TODO: move to sync package
	net := new(MockNetwork) // nolint

	cfg := &Config{
		Network:  net,
		Keystore: keystore.NewGlobalKeystore(),
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.Nil(t, err)

	// simulate block sent from BABE session
	newBlock := &types.Block{
		Header: &types.Header{
			Number:     big.NewInt(1),
			ParentHash: s.blockState.BestBlockHash(),
			Digest:     types.Digest{types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()},
		},
		Body: types.NewBody([]byte{}),
	}

	expected := &network.BlockAnnounceMessage{
		ParentHash:     newBlock.Header.ParentHash,
		Number:         newBlock.Header.Number,
		StateRoot:      newBlock.Header.StateRoot,
		ExtrinsicsRoot: newBlock.Header.ExtrinsicsRoot,
		Digest:         newBlock.Header.Digest,
		BestBlock:      true,
	}

	//setup the SendMessage function
	net.On("SendMessage", expected)

	state, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = s.HandleBlockProduced(newBlock, state)
	require.NoError(t, err)

	time.Sleep(time.Second)
	net.AssertCalled(t, "SendMessage", expected)
}

func TestService_HandleTransactionMessage(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	bp := new(MockBlockProducer) // nolint
	blockC := make(chan types.Block)
	bp.On("GetBlockChannel", nil).Return(blockC)

	cfg := &Config{
		Keystore:         ks,
		TransactionState: state.NewTransactionState(),
	}

	s := NewTestService(t, cfg)
	genHash := s.blockState.GenesisHash()
	header, err := types.NewHeader(genHash, common.Hash{}, common.Hash{}, big.NewInt(1), types.NewEmptyDigest())
	require.NoError(t, err)

	// initialise block header
	err = s.rt.InitializeBlock(header)
	require.NoError(t, err)

	extBytes := CreateTestExtrinsics(t, s.rt, genHash, 0)
	//extBytes := createExtrinsics(t, s.rt, genHash, 0)
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	b, err := s.HandleTransactionMessage(msg)
	require.NoError(t, err)
	require.True(t, b)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.NotEqual(t, 0, len(pending))
	require.Equal(t, extBytes, pending[0].Extrinsic)

	extBytes = []byte(`bogus extrinsic`)
	msg = &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	b, err = s.HandleTransactionMessage(msg)
	require.NoError(t, err)
	require.False(t, b)
}
