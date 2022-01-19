// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestNewEpochHandler(t *testing.T) {
	testHandleSlotFunc := func(epoch, slotNum uint64, authorityIndex uint32, proof *VrfOutputAndProof) error {
		return nil
	}

	epochData := &epochData{
		threshold: scale.MaxUint128,
	}

	sd, err := time.ParseDuration("6s")
	require.NoError(t, err)

	constants := &constants{
		slotDuration: sd,
		epochLength:  200,
	}

	keypair := keyring.Alice().(*sr25519.Keypair)

	eh, err := newEpochHandler(context.Background(), 1, 9999, epochData, constants, testHandleSlotFunc, keypair)
	require.NoError(t, err)
	require.Equal(t, 200, len(eh.slotToProof))
	require.Equal(t, uint64(1), eh.epochNumber)
	require.Equal(t, getSlotStartTime(9999, sd), eh.startTime)
	require.Equal(t, uint64(9999), eh.firstSlot)
	require.Equal(t, constants, eh.constants)
	require.Equal(t, epochData, eh.epochData)
	require.NotNil(t, eh.handleSlot)
}

func TestEpochHandler_run(t *testing.T) {
	sd, err := time.ParseDuration("10ms")
	require.NoError(t, err)
	startSlot := getCurrentSlot(sd)

	var callsToHandleSlot, firstExecutedSlot uint64
	testHandleSlotFunc := func(epoch, slotNum uint64, authorityIndex uint32, proof *VrfOutputAndProof) error {
		require.Equal(t, uint64(1), epoch)
		if callsToHandleSlot == 0 {
			firstExecutedSlot = slotNum
		} else {
			require.Equal(t, firstExecutedSlot+callsToHandleSlot, slotNum)
		}
		require.Equal(t, uint32(0), authorityIndex)
		require.NotNil(t, proof)
		callsToHandleSlot++
		return nil
	}

	epochData := &epochData{
		threshold: scale.MaxUint128,
	}

	constants := &constants{
		slotDuration: sd,
		epochLength:  100,
	}

	keypair := keyring.Alice().(*sr25519.Keypair)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eh, err := newEpochHandler(ctx, 1, startSlot, epochData, constants, testHandleSlotFunc, keypair)
	require.NoError(t, err)
	require.Equal(t, 100, len(eh.slotToProof))

	errCh := make(chan error)
	go eh.run(errCh)
	timer := time.After(sd * 100)
	select {
	case <-timer:
		require.Equal(t, 100-(firstExecutedSlot-startSlot), callsToHandleSlot)
	case err := <-errCh:
		require.NoError(t, err)
	}
}
