// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestNewEpochHandler(t *testing.T) {
	testHandleSlotFunc := func(epoch uint64, slot Slot, authorityIndex uint32,
		preRuntimeDigest *types.PreRuntimeDigest,
	) error {
		return nil
	}

	epochData := &epochData{
		threshold: scale.MaxUint128,
	}

	sd, err := time.ParseDuration("6s")
	require.NoError(t, err)

	testConstants := constants{
		slotDuration: sd,
		epochLength:  200,
	}

	keypair := keyring.Alice().(*sr25519.Keypair)

	epochHandler, err := newEpochHandler(1, 9999, epochData, testConstants, testHandleSlotFunc, keypair)
	require.NoError(t, err)
	require.Equal(t, 200, len(epochHandler.slotToPreRuntimeDigest))
	require.Equal(t, uint64(1), epochHandler.epochNumber)
	require.Equal(t, uint64(9999), epochHandler.firstSlot)
	require.Equal(t, testConstants, epochHandler.constants)
	require.Equal(t, epochData, epochHandler.epochData)
	require.NotNil(t, epochHandler.handleSlot)
}
