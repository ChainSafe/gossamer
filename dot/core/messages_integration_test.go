// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/core/mocks"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func createExtrinsic(t *testing.T, rt runtime.Instance, genHash common.Hash, nonce uint64) types.Extrinsic {
	t.Helper()
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = ctypes.DecodeFromBytes(decoded, meta)
	require.NoError(t, err)

	rv, err := rt.Version()
	require.NoError(t, err)

	c, err := ctypes.NewCall(meta, "System.remark", []byte{0xab, 0xcd})
	require.NoError(t, err)

	ext := ctypes.NewExtrinsic(c)
	o := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(genHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(rv.SpecVersion()),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion()),
	}

	// Sign the transaction using Alice's key
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := ctypes.EncodeToHexString(ext)
	require.NoError(t, err)

	extBytes := types.Extrinsic(common.MustHexToBytes(extEnc))
	return extBytes
}

func TestService_HandleBlockProduced(t *testing.T) {
	net := new(mocks.Network)
	cfg := &Config{
		Network:  net,
		Keystore: keystore.NewGlobalKeystore(),
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.Nil(t, err)

	// simulate block sent from BABE session
	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	newBlock := types.Block{
		Header: types.Header{
			Number:     big.NewInt(1),
			ParentHash: s.blockState.BestBlockHash(),
			Digest:     digest,
		},
		Body: *types.NewBody([]types.Extrinsic{}),
	}

	expected := &network.BlockAnnounceMessage{
		ParentHash:     newBlock.Header.ParentHash,
		Number:         newBlock.Header.Number,
		StateRoot:      newBlock.Header.StateRoot,
		ExtrinsicsRoot: newBlock.Header.ExtrinsicsRoot,
		Digest:         digest,
		BestBlock:      true,
	}

	net.On("GossipMessage", expected)

	state, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = s.HandleBlockProduced(&newBlock, state)
	require.NoError(t, err)

	time.Sleep(time.Second)
	net.AssertCalled(t, "GossipMessage", expected)
}

func TestService_HandleTransactionMessage(t *testing.T) {
	t.Parallel()

	const peer1 = "testPeer1"

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	cfg := &Config{
		Keystore:         ks,
		TransactionState: state.NewTransactionState(),
	}

	s := NewTestService(t, cfg)
	genHash := s.blockState.GenesisHash()
	genHeader, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	rt, err := s.blockState.GetRuntime(nil)
	require.NoError(t, err)

	ts, err := s.storageState.TrieState(nil)
	require.NoError(t, err)
	rt.SetContextStorage(ts)

	block := sync.BuildBlock(t, rt, genHeader, nil)

	err = s.handleBlock(block, ts)
	require.NoError(t, err)

	extBytes := createExtrinsic(t, rt, genHash, 0)
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	b, err := s.HandleTransactionMessage(peer1, msg)
	require.NoError(t, err)
	require.True(t, b)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.NotEqual(t, 0, len(pending))
	require.Equal(t, extBytes, pending[0].Extrinsic)

	extBytes = []byte(`bogus extrinsic`)
	msg = &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	b, err = s.HandleTransactionMessage(peer1, msg)
	require.NoError(t, err)
	require.False(t, b)
}
