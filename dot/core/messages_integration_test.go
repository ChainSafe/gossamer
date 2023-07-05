// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package core

import (
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func createExtrinsic(t *testing.T, rt runtime.Instance, genHash common.Hash, nonce uint64) types.Extrinsic {
	t.Helper()
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(decoded, meta)
	require.NoError(t, err)

	rv, err := rt.Version()
	require.NoError(t, err)

	c, err := ctypes.NewCall(meta, "System.remark", []byte{0xab, 0xcd})
	require.NoError(t, err)

	ext := ctypes.NewExtrinsic(c)
	options := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(genHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(rv.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion),
	}

	// Sign the transaction using Alice's key
	err = ext.Sign(signature.TestKeyringPairAlice, options)
	require.NoError(t, err)

	extEnc, err := codec.EncodeToHex(ext)
	require.NoError(t, err)

	extBytes := types.Extrinsic(common.MustHexToBytes(extEnc))
	return extBytes
}

func TestService_HandleBlockProduced(t *testing.T) {
	ctrl := gomock.NewController(t)

	net := NewMockNetwork(ctrl)
	cfg := &Config{
		Network:  net,
		Keystore: keystore.NewGlobalKeystore(),
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.NoError(t, err)

	// simulate block sent from BABE session
	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	// Used to define the state root of new block for testing
	parentHash := s.blockState.(*state.BlockState).GenesisHash()
	genesisBlock, err := s.blockState.(*state.BlockState).GetBlockByHash(parentHash)
	require.NoError(t, err)

	newBlock := types.Block{
		Header: types.Header{
			Number:     1,
			ParentHash: parentHash,
			Digest:     digest,
			StateRoot:  genesisBlock.Header.StateRoot,
		},
		Body: *types.NewBody([]types.Extrinsic{}),
	}

	onBlockImportHandlerMock := NewMockBlockImportDigestHandler(ctrl)
	onBlockImportHandlerMock.EXPECT().Handle(&newBlock.Header).Return(nil)

	s.onBlockImport = onBlockImportHandlerMock

	expected := &network.BlockAnnounceMessage{
		ParentHash:     newBlock.Header.ParentHash,
		Number:         newBlock.Header.Number,
		StateRoot:      newBlock.Header.StateRoot,
		ExtrinsicsRoot: newBlock.Header.ExtrinsicsRoot,
		Digest:         digest,
		BestBlock:      true,
	}

	net.EXPECT().GossipMessage(expected)

	state, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = s.HandleBlockProduced(&newBlock, state)
	require.NoError(t, err)

	time.Sleep(time.Second)
}

func TestService_HandleTransactionMessage(t *testing.T) {
	t.Parallel()

	const peer1 = "testPeer1"

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any())

	net := NewMockNetwork(ctrl)
	net.EXPECT().IsSynced().Return(true).Times(2)
	net.EXPECT().ReportPeer(
		gomock.AssignableToTypeOf(peerset.ReputationChange{}),
		gomock.AssignableToTypeOf(peer.ID("")),
	)

	cfg := &Config{
		Keystore:         ks,
		TransactionState: state.NewTransactionState(telemetryMock),
		Network:          net,
	}

	s := NewTestService(t, cfg)
	genHash := s.blockState.(*state.BlockState).GenesisHash()
	genHeader, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	ts, err := s.storageState.TrieState(nil)
	require.NoError(t, err)
	rt.SetContextStorage(ts)

	babeConfig, err := rt.BabeConfiguration()
	require.NoError(t, err)

	currentTimestamp := uint64(time.Now().UnixMilli())
	currentSlot := currentTimestamp / babeConfig.SlotDuration

	block := buildTestBlockWithoutExtrinsics(t, rt, genHeader, currentSlot, currentTimestamp)
	onBlockImportDigestHandlerMock := NewMockBlockImportDigestHandler(ctrl)
	onBlockImportDigestHandlerMock.EXPECT().Handle(&block.Header).Return(nil)

	s.onBlockImport = onBlockImportDigestHandlerMock

	err = s.handleBlock(block, ts)
	require.NoError(t, err)

	extBytes := createExtrinsic(t, rt, genHash, 0)
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}}
	shouldPropagate, err := s.HandleTransactionMessage(peer1, msg)
	require.NoError(t, err)
	require.True(t, shouldPropagate)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.NotEmpty(t, pending)
	require.Equal(t, extBytes, pending[0].Extrinsic)

	invalidExtBytes := types.Extrinsic{byte(1)}
	msg = &network.TransactionMessage{Extrinsics: []types.Extrinsic{invalidExtBytes}}
	shouldPropagate, err = s.HandleTransactionMessage(peer1, msg)
	require.Error(t, err)
	require.False(t, shouldPropagate)
}
