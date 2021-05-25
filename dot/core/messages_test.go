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
	require.Nil(t, err)

	// TODO: load BABE authority key

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	cfg := &Config{
		Keystore:         ks,
		TransactionState: state.NewTransactionState(),
		IsBlockProducer:  true,
		BlockProducer:    &mockBlockProducer{},
	}

	s := NewTestService(t, cfg)

	parentHash := common.MustHexToHash("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	header, err := types.NewHeader(parentHash, common.Hash{}, common.Hash{}, big.NewInt(1), types.NewEmptyDigest())
	require.NoError(t, err)

	// initialise block header
	err = s.rt.InitializeBlock(header)
	require.NoError(t, err)

	ext := types.Extrinsic(common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d015a3e258da3ea20581b68fe1264a35d1f62d6a0debb1a44e836375eb9921ba33e3d0f265f2da33c9ca4e10490b03918300be902fcb229f806c9cf99af4cc10f8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670"))

	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}

	err = s.HandleTransactionMessage(msg)
	require.Nil(t, err)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.NotEqual(t, 0, len(pending))
	require.Equal(t, ext, pending[0].Extrinsic)
}
