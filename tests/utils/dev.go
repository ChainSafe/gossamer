// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// PauseBABE calls the endpoint dev_control with the params ["babe", "stop"]
func PauseBABE(ctx context.Context, rpcPort string) error {
	endpoint := NewEndpoint(rpcPort)
	const params = `["babe", "stop"]`
	_, err := PostRPC(ctx, endpoint, DevControl, params)
	return err
}

// SlotDuration Calls dev endpoint for slot duration
func SlotDuration(ctx context.Context, t *testing.T, rpcPort string) time.Duration {
	endpoint := NewEndpoint(rpcPort)
	const method = "dev_slotDuration"
	const params = "[]"
	slotDuration, err := PostRPC(ctx, endpoint, method, params)

	if err != nil {
		require.NoError(t, err)
	}

	slotDurationDecoded := new(string)
	err = DecodeRPC(slotDuration, slotDurationDecoded)
	require.NoError(t, err)

	slotDurationParsed := binary.LittleEndian.Uint64(common.MustHexToBytes(*slotDurationDecoded))
	duration, err := time.ParseDuration(strconv.Itoa(int(slotDurationParsed)) + "ms")
	require.NoError(t, err)
	return duration
}

// EpochLength Calls dev endpoint for epoch length
func EpochLength(ctx context.Context, t *testing.T, rpcPort string) uint64 {
	endpoint := NewEndpoint(rpcPort)
	const method = "dev_epochLength"
	const params = "[]"
	epochLength, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		require.NoError(t, err)
	}

	epochLengthDecoded := new(string)
	err = DecodeRPC(epochLength, epochLengthDecoded)
	require.NoError(t, err)

	epochLengthParsed := binary.LittleEndian.Uint64(common.MustHexToBytes(*epochLengthDecoded))
	return epochLengthParsed
}
