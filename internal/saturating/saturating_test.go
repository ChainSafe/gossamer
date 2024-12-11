// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package saturating

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSub(t *testing.T) {
	require.Equal(t, uint32(0), Sub(uint32(25), uint64(50)))
	require.Equal(t, uint32(0), Sub(uint32(25), uint64(25)))
	require.Equal(t, uint32(25), Sub(uint32(50), uint64(25)))
	require.Equal(t, uint32(0), Sub(uint32(math.MaxUint32), uint64(math.MaxUint64)))
	require.Equal(t, uint32(1), Sub(uint32(math.MaxUint32), uint64(math.MaxUint32-1)))
	require.Equal(t, uint32(25), Sub(uint32(50), uint64(25)))
	require.Equal(t, uint64(0), Sub(uint64(math.MaxUint32), uint32(math.MaxUint32)))
	require.Equal(t, uint64(0), Sub(uint64(math.MaxUint32-1), uint32(math.MaxUint32)))
	require.Equal(t, uint64(math.MaxUint64-math.MaxUint32), Sub(uint64(math.MaxUint64), uint32(math.MaxUint32)))
	require.Equal(t, uint(25), Sub(uint(50), uint(25)))
	require.Equal(t, uint8(0), Sub(uint8(math.MaxUint8), uint16(math.MaxUint16)))
	require.Equal(t, uint16(0), Sub(uint16(math.MaxUint16), uint32(math.MaxUint32)))
}

func TestInto(t *testing.T) {
	require.Equal(t, uint32(math.MaxUint32), Into[uint64, uint32](math.MaxUint64))
	require.Equal(t, uint32(math.MaxUint32), Into[uint64, uint32](math.MaxUint32))
	require.Equal(t, uint32(math.MaxUint32-1), Into[uint64, uint32](math.MaxUint32-1))
	require.Equal(t, uint32(math.MaxUint32), Into[uint32, uint32](math.MaxUint32))
	require.Equal(t, uint64(math.MaxUint32), Into[uint32, uint64](math.MaxUint32))
}
