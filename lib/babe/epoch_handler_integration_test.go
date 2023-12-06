// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEpochHandler_run_shouldReturnAfterContextCancel(t *testing.T) {
	t.Parallel()

	const authorityIndex uint32 = 0
	aliceKeyPair := keyring.Alice().(*sr25519.Keypair)
	epochData := &epochData{
		threshold:      scale.MaxUint128,
		authorityIndex: authorityIndex,
		authorities: []types.AuthorityRaw{
			{[32]byte(aliceKeyPair.Public().Encode()), 1},
		},
	}

	const slotDuration = 6 * time.Second
	const epochLength uint64 = 100

	testConstants := constants{
		slotDuration: slotDuration,
		epochLength:  epochLength,
	}

	const expectedEpoch = 1
	startSlot := getCurrentSlot(slotDuration)
	handler := testHandleSlotFunc(t, authorityIndex, expectedEpoch, startSlot)

	epochHandler, err := newEpochHandler(1, startSlot, epochData, testConstants, handler, aliceKeyPair)
	require.NoError(t, err)
	require.Equal(t, epochLength, uint64(len(epochHandler.slotToPreRuntimeDigest)))

	timeoutCtx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(7 * time.Second)
		cancel()
	}()

	errCh := make(chan error)
	go epochHandler.run(timeoutCtx, errCh)

	err = <-errCh
	require.ErrorIs(t, err, context.Canceled)
}

func TestEpochHandler_run(t *testing.T) {
	t.Parallel()

	const authorityIndex uint32 = 0
	aliceKeyPair := keyring.Alice().(*sr25519.Keypair)
	epochData := &epochData{
		threshold:      scale.MaxUint128,
		authorityIndex: authorityIndex,
		authorities: []types.AuthorityRaw{
			{[32]byte(aliceKeyPair.Public().Encode()), 1},
		},
	}

	const slotDuration = 6 * time.Second
	const epochLength uint64 = 100

	testConstants := constants{
		slotDuration: slotDuration,
		epochLength:  epochLength,
	}

	const expectedEpoch = 1
	startSlot := getCurrentSlot(slotDuration)
	handler := testHandleSlotFunc(t, authorityIndex, expectedEpoch, startSlot)

	epochHandler, err := newEpochHandler(1, startSlot, epochData, testConstants, handler, aliceKeyPair)
	require.NoError(t, err)
	require.Equal(t, epochLength, uint64(len(epochHandler.slotToPreRuntimeDigest)))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*slotDuration)
	defer cancel()

	errCh := make(chan error)
	go epochHandler.run(timeoutCtx, errCh)

	err = <-errCh
	require.ErrorIs(t, err, context.DeadlineExceeded)

}

func testHandleSlotFunc(t *testing.T, expectedAuthorityIndex uint32,
	expectedEpoch, startSlot uint64) handleSlotFunc {
	currentSlot := startSlot

	return func(epoch uint64, slot Slot, authorityIndex uint32,
		preRuntimeDigest *types.PreRuntimeDigest) error {
		require.NotNil(t, preRuntimeDigest)
		require.Equal(t, expectedEpoch, epoch)
		require.Equal(t, expectedAuthorityIndex, authorityIndex)

		require.GreaterOrEqual(t, slot.number, currentSlot)

		// increase the slot by one so we expect the next call
		// to be exactly 1 slot greater than the previous call
		currentSlot++
		return nil
	}
}
