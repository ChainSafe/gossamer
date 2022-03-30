// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

// PauseBABE calls the endpoint dev_control with the params ["babe", "stop"]
func PauseBABE(ctx context.Context, rpcPort string) error {
	endpoint := NewEndpoint(rpcPort)
	const method = "dev_control"
	const params = `["babe", "stop"]`
	_, err := PostRPC(ctx, endpoint, method, params)
	return err
}

// SlotDuration Calls dev endpoint for slot duration
func SlotDuration(ctx context.Context, rpcPort string) (
	slotDuration time.Duration, err error) {
	endpoint := NewEndpoint(rpcPort)
	const method = "dev_slotDuration"
	const params = "[]"
	data, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		return 0, fmt.Errorf("cannot post RPC: %w", err)
	}

	var slotDurationString string
	err = DecodeRPC(data, &slotDurationString)
	if err != nil {
		return 0, fmt.Errorf("cannot decode RPC response: %w", err)
	}

	b, err := common.HexToBytes(slotDurationString)
	if err != nil {
		return 0, fmt.Errorf("malformed slot duration hex string: %w", err)
	}

	slotDurationUint64 := binary.LittleEndian.Uint64(b)

	slotDuration = time.Millisecond * time.Duration(slotDurationUint64)

	return slotDuration, nil
}

// EpochLength Calls dev endpoint for epoch length
func EpochLength(ctx context.Context, rpcPort string) (epochLength uint64, err error) {
	endpoint := NewEndpoint(rpcPort)
	const method = "dev_epochLength"
	const params = "[]"
	data, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		return 0, fmt.Errorf("cannot post RPC: %w", err)
	}

	var epochLengthHexString string
	err = DecodeRPC(data, &epochLengthHexString)
	if err != nil {
		return 0, fmt.Errorf("cannot decode RPC response: %w", err)
	}

	b, err := common.HexToBytes(epochLengthHexString)
	if err != nil {
		return 0, fmt.Errorf("malformed epoch length hex string: %w", err)
	}

	epochLength = binary.LittleEndian.Uint64(b)
	return epochLength, nil
}
