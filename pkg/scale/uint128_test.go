// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint128FromBigInt(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}
	bi := new(big.Int).SetBytes(bytes)
	u, _ := NewUint128(bi)
	res := u.Bytes(binary.BigEndian)
	require.Equal(t, bytes, res)

	bytes = []byte{1, 2}
	bi = new(big.Int).SetBytes(bytes)
	u, _ = NewUint128(bi)
	res = u.Bytes(binary.BigEndian)
	require.Equal(t, bytes, res)
}

func TestUint128FromLEBytes(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}
	u, _ := NewUint128(bytes)
	res := u.Bytes()
	require.Equal(t, bytes, res)

	bytes = []byte{1, 2}
	u, _ = NewUint128(bytes)
	res = u.Bytes()
	require.Equal(t, bytes, res)
}

func TestUint128_Cmp(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}
	u0, _ := NewUint128(bytes)
	u1, _ := NewUint128(bytes)
	require.Equal(t, 0, u0.Compare(u1))

	bytes = []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5}
	u2, _ := NewUint128(bytes)
	require.Equal(t, 1, u0.Compare(u2))
	require.Equal(t, -1, u2.Compare(u0))

	bytes = []byte{1, 2, 3}
	u3, _ := NewUint128(bytes)
	require.Equal(t, 1, u0.Compare(u3))
	require.Equal(t, -1, u3.Compare(u0))
}
