// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// PauseBABE calls the endpoint dev_control with the params ["babe", "stop"]
func PauseBABE(t *testing.T, node *Node) error {
	_, err := PostRPC(DevControl, NewEndpoint(node.RPCPort), "[\"babe\", \"stop\"]")
	return err
}

// SlotDuration Calls dev endpoint for slot duration
func SlotDuration(t *testing.T, node *Node) time.Duration {
	slotDuration, err := PostRPC("dev_slotDuration", NewEndpoint(node.RPCPort), "[]")

	if err != nil {
		require.NoError(t, err)
	}

	slotDurationDecoded := new(string)
	err = DecodeRPC(t, slotDuration, slotDurationDecoded)
	require.NoError(t, err)

	slotDurationParsed := binary.LittleEndian.Uint64(common.MustHexToBytes(*slotDurationDecoded))
	duration, err := time.ParseDuration(strconv.Itoa(int(slotDurationParsed)) + "ms")
	require.NoError(t, err)
	return duration
}

// EpochLength Calls dev endpoint for epoch length
func EpochLength(t *testing.T, node *Node) uint64 {
	epochLength, err := PostRPC("dev_epochLength", NewEndpoint(node.RPCPort), "[]")

	if err != nil {
		require.NoError(t, err)
	}

	epochLengthDecoded := new(string)
	err = DecodeRPC(t, epochLength, epochLengthDecoded)
	require.NoError(t, err)

	epochLengthParsed := binary.LittleEndian.Uint64(common.MustHexToBytes(*epochLengthDecoded))
	return epochLengthParsed
}
