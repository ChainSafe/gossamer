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

	. "github.com/ChainSafe/gossamer/dot/core/mocks"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/stretchr/testify/require"
)

func TestService_ProcessBlockAnnounceMessage(t *testing.T) {
	// TODO: move to sync package
	net := new(MockNetwork)

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

	// simulate block sent from BABE session
	newBlock := types.Block{
		Header: &types.Header{
			Number:     big.NewInt(1),
			ParentHash: s.blockState.BestBlockHash(),
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
	newBlocks <- newBlock

	time.Sleep(time.Second * 2)

	net.AssertCalled(t, "SendMessage", expected)
}

func createExtrinsics(t *testing.T, rt runtime.Instance, genHash common.Hash, nonce uint64) types.Extrinsic {
	t.Helper()
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	decoded, err := scale.Decode(rawMeta, []byte{})
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = ctypes.DecodeFromBytes(decoded.([]byte), meta)
	require.NoError(t, err)

	rv, err := rt.Version()
	require.NoError(t, err)

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	bob, err := ctypes.NewAddressFromHexAccountID(keyring.Bob().Public().Hex())
	require.NoError(t, err)

	c, err := ctypes.NewCall(meta, "Balances.transfer", bob, ctypes.NewUCompactFromUInt(12345))
	require.NoError(t, err)

	// Create the extrinsic
	ext := ctypes.NewExtrinsic(c)

	o := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(genHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(rv.SpecVersion()),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion()),
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := ctypes.EncodeToHexString(ext)
	require.NoError(t, err)

	extBytes := types.Extrinsic(common.MustHexToBytes(extEnc))
	return extBytes
}

func TestService_HandleTransactionMessage(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	bp := new(MockBlockProducer)
	blockC := make(chan types.Block)
	bp.On("GetBlockChannel", nil).Return(blockC)

	cfg := &Config{
		Keystore:         ks,
		TransactionState: state.NewTransactionState(),
		IsBlockProducer:  true,
		BlockProducer:    bp,
	}

	s := NewTestService(t, cfg)
	genHash := s.blockState.GenesisHash()
	header, err := types.NewHeader(genHash, common.Hash{}, common.Hash{}, big.NewInt(1), types.NewEmptyDigest())
	require.NoError(t, err)

	// initialise block header
	err = s.rt.InitializeBlock(header)
	require.NoError(t, err)

	extBytes := createExtrinsics(t, s.rt, genHash, 0)

	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	err = s.HandleTransactionMessage(msg)
	require.NoError(t, err)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.NotEqual(t, 0, len(pending))
	require.Equal(t, extBytes, pending[0].Extrinsic)
}
