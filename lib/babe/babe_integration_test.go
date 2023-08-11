// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
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
		BlockImportHandler: blockImportHandler,
	}

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, gen, genTrie, genHeader, nil)

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
	cfg := ServiceConfig{}
	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	bs := createTestService(t, cfg, gen, genTrie, genHeader, nil)
	err := bs.Start()
	require.NoError(t, err)
	err = bs.Stop()
	require.NoError(t, err)
}

func TestService_PauseAndResume(t *testing.T) {
	cfg := ServiceConfig{}
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, nil)
	err := babeService.Start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	wg := sync.WaitGroup{}
	wg.Add(4)

	babeService.Pause()
	_, ok := <-babeService.pauseCh
	require.False(t, ok)

	_, ok = <-babeService.doneCh
	require.False(t, ok)

	babeService.Resume()
	err = babeService.Stop()
	require.NoError(t, err)
}

func TestService_HandleSlotWithLaggingSlot(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
	}

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, nil)

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

	bestBlockHeader, err := babeService.blockState.GetHeader(bestBlockHash)
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	timestamp := time.Unix(6, 0)
	slot := getSlot(t, rt, timestamp)
	ext := runtime.NewTestExtrinsic(t, rt, parentHash, parentHash, 0, signature.TestKeyringPairAlice,
		"System.remark", []byte{0xab, 0xcd})
	block := createTestBlockWithSlot(t, babeService, bestBlockHeader, [][]byte{common.MustHexToBytes(ext)},
		testEpochIndex, epochData, slot)

	err = babeService.blockState.AddBlock(block)
	require.NoError(t, err)
	time.Sleep(babeService.constants.slotDuration)

	header, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)

	bestBlockSlotNum, err := babeService.blockState.GetSlotForBlock(header.Hash())
	require.NoError(t, err)

	slotnum := uint64(1)
	slot = Slot{
		start:    time.Now(),
		duration: babeService.constants.slotDuration * time.Millisecond,
		number:   slotnum,
	}
	preRuntimeDigest, err := types.NewBabePrimaryPreDigest(
		0, slot.number,
		[sr25519.VRFOutputLength]byte{},
		[sr25519.VRFProofLength]byte{},
	).ToPreRuntimeDigest()

	require.NoError(t, err)

	slot = Slot{
		start:    time.Now(),
		duration: babeService.constants.slotDuration * time.Millisecond,
		number:   bestBlockSlotNum - 1,
	}
	err = babeService.handleSlot(
		babeService.epochHandler.epochNumber,
		slot,
		babeService.epochHandler.epochData.authorityIndex,
		preRuntimeDigest)

	require.ErrorIs(t, err, errLaggingSlot)
}

func TestService_HandleSlotWithSameSlot(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)
	const authorityIndex = 0

	bestBlockHash := babeService.blockState.BestBlockHash()
	runtime, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, runtime, time.Unix(6, 0))
	preRuntimeDigest, err := claimSlot(testEpochIndex, slot.number, epochData, babeService.keypair)
	require.NoError(t, err)

	builder := NewBlockBuilder(
		babeService.keypair,
		babeService.transactionState,
		babeService.blockState,
		authorityIndex,
		preRuntimeDigest,
	)

	block, err := builder.buildBlock(&genesisHeader, slot, runtime)
	require.NoError(t, err)

	// Create new non authority service
	cfgBob := ServiceConfig{
		Keypair: keyring.Bob().(*sr25519.Keypair),
	}
	genBob, genTrieBob, genHeaderBob := newWestendDevGenesisWithTrieAndHeader(t)
	babeServiceBob := createTestService(t, cfgBob, genBob, genTrieBob, genHeaderBob, nil)

	// Add block created by alice to bob
	err = babeServiceBob.blockState.AddBlock(block)
	require.NoError(t, err)

	// If the slot we are claiming is the same as the slot of the best block header, test that we can
	// still claim the slot without error.
	bestBlockSlotNum, err := babeServiceBob.blockState.GetSlotForBlock(block.Header.Hash())
	require.NoError(t, err)

	slot = Slot{
		start:    time.Unix(6, 0),
		duration: babeServiceBob.constants.slotDuration * time.Millisecond,
		number:   bestBlockSlotNum,
	}
	preRuntimeDigest, err = types.NewBabePrimaryPreDigest(
		0, slot.number,
		[sr25519.VRFOutputLength]byte{},
		[sr25519.VRFProofLength]byte{},
	).ToPreRuntimeDigest()
	require.NoError(t, err)

	// slot gets occupied even if it has been occupied by a block
	// authored by someone else
	err = babeServiceBob.handleSlot(
		testEpochIndex,
		slot,
		0,
		preRuntimeDigest)
	require.NoError(t, err)
}
