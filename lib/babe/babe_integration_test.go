// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

func TestService_SlotDuration(t *testing.T) {
	duration, err := time.ParseDuration("1000ms")
	require.NoError(t, err)

	bs := &Service{
		constants: constants{
			slotDuration: duration,
		},
	}

	dur := bs.constants.slotDuration
	require.Equal(t, dur.Milliseconds(), int64(1000))
}

func TestService_ProducesBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockImportHandler := NewMockBlockImportHandler(ctrl)
	blockImportHandler.EXPECT().HandleBlockProduced(gomock.Any(), gomock.Any()).
		Return(nil).MinTimes(2)
	cfg := ServiceConfig{
		Authority:          true,
		Lead:               true,
		BlockImportHandler: blockImportHandler,
	}

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, gen, genTrie, genHeader)

	err := babeService.Start()
	require.NoError(t, err)
	time.Sleep(babeService.constants.slotDuration * 2)
	err = babeService.Stop()
	require.NoError(t, err)
}

func TestService_GetAuthorityIndex(t *testing.T) {
	kpA, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	kpB, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	pubA := kpA.Public().(*sr25519.PublicKey)
	pubB := kpB.Public().(*sr25519.PublicKey)

	authData := []types.Authority{
		{Key: pubA, Weight: 1},
		{Key: pubB, Weight: 1},
	}

	bs := &Service{
		keypair:   kpA,
		authority: true,
	}

	idx, err := bs.getAuthorityIndex(authData)
	require.NoError(t, err)
	require.Equal(t, uint32(0), idx)

	bs = &Service{
		keypair:   kpB,
		authority: true,
	}

	idx, err = bs.getAuthorityIndex(authData)
	require.NoError(t, err)
	require.Equal(t, uint32(1), idx)
}

func TestStartAndStop(t *testing.T) {
	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	bs := createTestService(t, ServiceConfig{}, gen, genTrie, genHeader)
	err := bs.Start()
	require.NoError(t, err)
	err = bs.Stop()
	require.NoError(t, err)
}

func TestService_PauseAndResume(t *testing.T) {
	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	bs := createTestService(t, ServiceConfig{}, gen, genTrie, genHeader)
	err := bs.Start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	go func() {
		_ = bs.Pause()
	}()

	go func() {
		_ = bs.Pause()
	}()

	go func() {
		err := bs.Resume()
		require.NoError(t, err)
	}()

	go func() {
		err := bs.Resume()
		require.NoError(t, err)
	}()

	err = bs.Stop()
	require.NoError(t, err)
}

// Since multiple validator, sometimes not our turn to propose
// TODO, use westend dev
func TestService_HandleSlotWithLaggingSlot(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
		Lead:      true,
	}

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, gen, genTrie, genHeader)

	err := babeService.Start()
	require.NoError(t, err)
	defer func() {
		err = babeService.Stop()
		require.NoError(t, err)
	}()

	parentHash := babeService.blockState.GenesisHash()
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, rt, time.Now())
	ext := runtime.NewTestExtrinsic(t, rt, parentHash, parentHash, 0, signature.TestKeyringPairAlice, "System.remark", []byte{0xab, 0xcd})
	block := createTestBlockWithSlot(t, babeService, emptyHeader, [][]byte{common.MustHexToBytes(ext)}, testEpochIndex, epochData, slot)

	babeService.blockState.AddBlock(block)
	time.Sleep(babeService.constants.slotDuration)

	header, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)

	bestBlockSlotNum, err := babeService.blockState.GetSlotForBlock(header.Hash())
	require.NoError(t, err)

	slotnum := uint64(1)
	slot = Slot{
		start:    time.Now(),
		duration: time.Duration(babeService.constants.slotDuration) * time.Millisecond,
		number:   slotnum,
	}
	preRuntimeDigest, err := types.NewBabePrimaryPreDigest(
		0, slot.number,
		[sr25519.VRFOutputLength]byte{},
		[sr25519.VRFProofLength]byte{},
	).ToPreRuntimeDigest()

	require.NoError(t, err)

	err = babeService.handleSlot(
		babeService.epochHandler.epochNumber,
		bestBlockSlotNum-1,
		babeService.epochHandler.epochData.authorityIndex,
		preRuntimeDigest)

	require.ErrorIs(t, err, errLaggingSlot)
}

// TODO this test is really flaky with westend so probably should be rewritten.
// Built for 2 nodes and with westend its either 1 or 3. Consult team
func TestService_HandleSlotWithSameSlot(t *testing.T) {
	t.Skip()
	alice := keyring.Alice().(*sr25519.Keypair)
	bob := keyring.Bob().(*sr25519.Keypair)

	// Create babe service for alice
	cfgAlice := ServiceConfig{
		Authority: true,
		Lead:      true,
		Keypair:   alice,
		AuthData: []types.Authority{
			{
				Key:    alice.Public().(*sr25519.PublicKey),
				Weight: 1,
			},
			{
				Key:    bob.Public().(*sr25519.PublicKey),
				Weight: 1,
			},
		},
	}

	// Create babe service for bob
	cfgBob := ServiceConfig{
		Authority: true,
		Lead:      true,
		Keypair:   bob,
		AuthData: []types.Authority{
			{
				Key:    alice.Public().(*sr25519.PublicKey),
				Weight: 1,
			},
			{
				Key:    bob.Public().(*sr25519.PublicKey),
				Weight: 1,
			},
		},
	}

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeServiceBob := createTestService(t, cfgBob, gen, genTrie, genHeader)

	err := babeServiceBob.Start()
	require.NoError(t, err)
	defer func() {
		_ = babeServiceBob.Stop()
	}()

	// wait till bob creates a block
	time.Sleep(babeServiceBob.constants.slotDuration)
	require.NoError(t, err)

	block, err := babeServiceBob.blockState.GetBlockByNumber(1)
	require.NoError(t, err)

	err = babeServiceBob.Stop()
	require.NoError(t, err)

	time.Sleep(babeServiceBob.constants.slotDuration)

	gen, genTrie, genHeader = newWestendDevGenesisWithTrieAndHeader(t)
	babeServiceAlice := createTestService(t, cfgAlice, gen, genTrie, genHeader)

	// Add block created by Bob to Alice
	err = babeServiceAlice.blockState.AddBlock(block)
	require.NoError(t, err)

	time.Sleep(babeServiceBob.constants.slotDuration)

	bestBlockHeader, err := babeServiceAlice.blockState.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, block.Header.Hash(), bestBlockHeader.Hash())

	// If the slot we are claiming is the same as the slot of the best block header, test that we can
	// still claim the slot without error.
	bestBlockSlotNum, err := babeServiceAlice.blockState.GetSlotForBlock(bestBlockHeader.Hash())
	require.NoError(t, err)

	slot := Slot{
		start:    time.Now(),
		duration: time.Second,
		number:   bestBlockSlotNum,
	}
	preRuntimeDigest, err := types.NewBabePrimaryPreDigest(
		0, slot.number,
		[sr25519.VRFOutputLength]byte{},
		[sr25519.VRFProofLength]byte{},
	).ToPreRuntimeDigest()
	require.NoError(t, err)

	// slot gets occupied even if it has been occupied by a block
	// authored by someone else
	err = babeServiceAlice.handleSlot(
		testEpochIndex,
		bestBlockSlotNum,
		0,
		preRuntimeDigest)
	require.NoError(t, err)
}
