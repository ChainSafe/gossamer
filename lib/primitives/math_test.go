// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package primitives

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/require"
)

func TestSaturatingAdd(t *testing.T) {
	require.Equal(t, uint8(2), SaturatingAdd(uint8(1), uint8(1)))
	require.Equal(t, uint8(math.MaxUint8), SaturatingAdd(uint8(math.MaxUint8), 100))

	require.Equal(t, uint32(math.MaxUint32), SaturatingAdd(uint32(math.MaxUint32), 100))
	require.Equal(t, uint32(100), SaturatingAdd(uint32(0), 100))

	// should not be able to overflow in the opposite direction as well
	require.Equal(t, int64(math.MinInt64), SaturatingAdd(int64(math.MinInt64), -100))
	require.Equal(t, int8(127), SaturatingAdd(int8(120), 7))
	require.Equal(t, int8(127), SaturatingAdd(int8(120), 8))
}

func TestSaturatingSub(t *testing.T) {
	// -128 - 100 overflows, so it should return just -128
	require.Equal(t, int8(math.MinInt8), SaturatingSub(int8(math.MinInt8), 100))
	require.Equal(t, int8(0), SaturatingSub(int8(100), 100))

	// max - (-1) = max + 1 = overflows, so it should return just max
	require.Equal(t, int64(math.MaxInt64), SaturatingSub(int64(math.MaxInt64), -1))

	// 2 - 10 = -8 which overflows, then should return just 0
	require.Equal(t, uint32(0), SaturatingSub(uint32(2), uint32(10)))
	require.Equal(t, uint64(math.MaxUint64), SaturatingSub(uint64(math.MaxUint64), uint64(0)))
}
