// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package saturating

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	require.Equal(t, uint32(75), Add(uint32(25), uint32(50)))
	require.Equal(t, uint32(math.MaxUint32), Add(uint32(math.MaxUint32), uint32(50)))
	require.Equal(t, uint32(math.MaxUint32), Add(uint32(math.MaxUint32), uint32(10)))
	require.Equal(t, uint64(math.MaxUint32+10), Add(uint64(math.MaxUint32), uint64(10)))
	require.Equal(t, uint64(0), Add(uint64(0), uint64(0)))
	require.Equal(t, uint64(50), Add(uint64(25), uint64(25)))
	require.Equal(t, uint32(50), Add(uint32(25), uint32(25)))
	require.Equal(t, uint32(math.MaxUint32), Add(uint32(math.MaxUint32), uint32(math.MaxUint32)))
	require.Equal(t, uint64(math.MaxUint64), Add(uint64(math.MaxUint64), uint64(math.MaxUint64)))
	require.Equal(t, uint64(math.MaxUint64), Add(uint64(math.MaxUint64), uint64(math.MaxUint32)))
	require.Equal(t, uint16(50), Add(uint16(25), uint16(25)))
	require.Equal(t, uint8(50), Add(uint8(25), uint8(25)))

	require.Equal(t, int(75), Add(int(25), int(50)))
	require.Equal(t, int(-75), Add(int(-25), int(-50)))
	require.Equal(t, int(-25), Add(int(25), int(-50)))
	require.Equal(t, int(math.MaxInt), Add(int(math.MaxInt), 1))
	require.Equal(t, int(math.MinInt), Add(int(math.MinInt), -1))
	require.Equal(t, int(0), Add(int(math.MaxInt), -1*int(math.MaxInt)))
}

func TestSub(t *testing.T) {
	require.Equal(t, uint32(0), Sub(uint32(25), uint32(50)))
	require.Equal(t, uint32(0), Sub(uint32(25), uint32(25)))
	require.Equal(t, uint32(25), Sub(uint32(50), uint32(25)))
	require.Equal(t, uint32(0), Sub(uint32(math.MaxUint32), uint32(math.MaxUint32)))
	require.Equal(t, uint32(1), Sub(uint32(math.MaxUint32), uint32(math.MaxUint32-1)))
	require.Equal(t, uint32(25), Sub(uint32(50), uint32(25)))
	require.Equal(t, uint64(0), Sub(uint64(math.MaxUint32), uint64(math.MaxUint32)))
	require.Equal(t, uint64(0), Sub(uint64(math.MaxUint32-1), uint64(math.MaxUint32)))
	require.Equal(t, uint64(math.MaxUint64-math.MaxUint32), Sub(uint64(math.MaxUint64), uint64(math.MaxUint32)))
	require.Equal(t, uint(25), Sub(uint(50), uint(25)))
	require.Equal(t, uint8(0), Sub(uint8(math.MaxUint8-1), uint8(math.MaxUint8)))
	require.Equal(t, uint16(0), Sub(uint16(math.MaxUint16-1), uint16(math.MaxUint16)))
}

func Test_getMinMaxSigned(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		min, max := getMinMaxSigned[int]()
		require.Equal(t, min, math.MinInt)
		require.Equal(t, max, math.MaxInt)
	})
	t.Run("int8", func(t *testing.T) {
		min, max := getMinMaxSigned[int8]()
		require.Equal(t, min, int8(math.MinInt8))
		require.Equal(t, max, int8(math.MaxInt8))
	})
	t.Run("int16", func(t *testing.T) {
		min, max := getMinMaxSigned[int16]()
		require.Equal(t, min, int16(math.MinInt16))
		require.Equal(t, max, int16(math.MaxInt16))
	})
	t.Run("int32", func(t *testing.T) {
		min, max := getMinMaxSigned[int32]()
		require.Equal(t, min, int32(math.MinInt32))
		require.Equal(t, max, int32(math.MaxInt32))
	})
	t.Run("int64", func(t *testing.T) {
		min, max := getMinMaxSigned[int64]()
		require.Equal(t, min, int64(math.MinInt64))
		require.Equal(t, max, int64(math.MaxInt64))
	})
}

func TestMul(t *testing.T) {
	require.Equal(t, int(1250), Mul(int(25), int(50)))
	require.Equal(t, int(math.MaxInt), Mul(int(25), int(math.MaxInt)))
	require.Equal(t, int(math.MinInt), Mul(int(25), int(math.MinInt)))
	require.Equal(t, int(0), Mul(int(0), int(0)))
	require.Equal(t, int(25), Mul(int(25), int(1)))
	require.Equal(t, int(25), Mul(int(1), int(25)))
	require.Equal(t, int32(1250), Mul(int32(25), int32(50)))
	require.Equal(t, int32(math.MaxInt32), Mul(int32(25), int32(math.MaxInt32)))
	require.Equal(t, int32(math.MinInt32), Mul(int32(25), int32(math.MinInt32)))
	require.Equal(t, int32(0), Mul(int32(0), int32(0)))
	require.Equal(t, int32(25), Mul(int32(25), int32(1)))
	require.Equal(t, int32(25), Mul(int32(1), int32(25)))
	require.Equal(t, int(1250), Mul(int(25), int(50)))
	require.Equal(t, int(math.MaxInt), Mul(int(2), int(math.MaxInt)))
	require.Equal(t, int(1250), Mul(int(25), int(50)))
	require.Equal(t, int(2*math.MaxUint32), Mul(int(2), int(math.MaxUint32)))
	require.Equal(t, int(-2*math.MaxUint32), Mul(int(-2), int(math.MaxUint32)))
	require.Equal(t, int64(math.MaxInt64), Mul(int64(math.MaxInt64), int64(2)))
	require.Equal(t, int64(math.MinInt64), Mul(int64(math.MinInt64), int64(2)))
	require.Equal(t, uint(1250), Mul(uint(25), uint(50)))
	require.Equal(t, uint(math.MaxUint), Mul(uint(25), uint(math.MaxInt)))
	require.Equal(t, uint(0), Mul(uint(25), uint(0)))
	require.Equal(t, uint(0), Mul(uint(0), uint(0)))
	require.Equal(t, uint(25), Mul(uint(25), uint(1)))
	require.Equal(t, uint(25), Mul(uint(1), uint(25)))
}

func TestInto(t *testing.T) {
	require.Equal(t, int64(5), Into[int, int64](5))
	require.Equal(t, int64(math.MaxInt), Into[int, int64](math.MaxInt))
	require.Equal(t, int64(math.MinInt), Into[int, int64](math.MinInt))
	require.Equal(t, uint64(0), Into[int, uint64](0))
	require.Equal(t, int32(5), Into[int64, int32](5))
	require.Equal(t, int32(math.MaxInt32), Into[int64, int32](math.MaxInt64))
	require.Equal(t, int32(math.MinInt32), Into[int64, int32](math.MinInt64))
	require.Equal(t, int32(math.MaxInt32), Into[int64, int32](math.MaxInt32))
	require.Equal(t, int32(math.MinInt32+1), Into[int64, int32](math.MinInt32+1))
	require.Equal(t, int64(math.MinInt32+1), Into[int32, int64](math.MinInt32+1))
	require.Equal(t, uint32(math.MaxUint32), Into[uint64, uint32](math.MaxUint64))
	require.Equal(t, uint32(math.MaxUint32), Into[uint64, uint32](math.MaxUint32))
	require.Equal(t, uint32(math.MaxUint32-1), Into[uint64, uint32](math.MaxUint32-1))
	require.Equal(t, uint32(math.MaxUint32), Into[uint32, uint32](math.MaxUint32))
	require.Equal(t, uint64(math.MaxUint32), Into[uint32, uint64](math.MaxUint32))
	require.Panics(t, func() {
		Into[int, uint](math.MaxInt)
	})
	require.Panics(t, func() {
		Into[uint, int](math.MaxInt)
	})
}
