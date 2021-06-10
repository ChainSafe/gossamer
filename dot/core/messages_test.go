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
	net := new(mockNetwork)
	newBlocks := make(chan types.Block)

	cfg := &Config{
		Network:         net,
		Keystore:        keystore.NewGlobalKeystore(),
		NewBlocks:       newBlocks,
		IsBlockProducer: false,
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.Nil(t, err)

	expected := &network.BlockAnnounceMessage{
		Number:         big.NewInt(1),
		ParentHash:     s.blockState.BestBlockHash(),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         nil,
		BestBlock:      true,
	}

	// simulate block sent from BABE session
	newBlocks <- types.Block{
		Header: &types.Header{
			Number:     big.NewInt(1),
			ParentHash: s.blockState.BestBlockHash(),
		},
		Body: types.NewBody([]byte{}),
	}

	time.Sleep(testMessageTimeout)
	require.NotNil(t, net.Message)
	require.Equal(t, network.BlockAnnounceMsgType, net.Message.(network.NotificationsMessage).Type())
	require.Equal(t, expected, net.Message)
}

func TestService_HandleTransactionMessage(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	cfg := &Config{
		Keystore:         ks,
		TransactionState: state.NewTransactionState(),
		IsBlockProducer:  true,
		BlockProducer:    &mockBlockProducer{},
	}

	s := NewTestService(t, cfg)
	genHash := s.blockState.GenesisHash()
	header, err := types.NewHeader(genHash, common.Hash{}, common.Hash{}, big.NewInt(1), types.NewEmptyDigest())
	require.NoError(t, err)

	// initialise block header
	err = s.rt.InitializeBlock(header)
	require.NoError(t, err)

	extBytes := CreateTestExtrinsics(t, s.rt, genHash, 0)

	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	err = s.HandleTransactionMessage(msg)
	require.NoError(t, err)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.NotEqual(t, 0, len(pending))
	require.Equal(t, extBytes, pending[0].Extrinsic)
}
