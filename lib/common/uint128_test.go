// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint128FromBigInt(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}
	bi := new(big.Int).SetBytes(bytes)
	u := Uint128FromBigInt(bi)
	res := u.ToBEBytes()
	require.Equal(t, bytes, res)

	bytes = []byte{1, 2}
	bi = new(big.Int).SetBytes(bytes)
	u = Uint128FromBigInt(bi)
	res = u.ToBEBytes()
	require.Equal(t, bytes, res)
}

func TestUint128FromLEBytes(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}
	u := Uint128FromLEBytes(bytes)
	res := u.ToLEBytes()
	require.Equal(t, bytes, res)

	bytes = []byte{1, 2}
	u = Uint128FromLEBytes(bytes)
	res = u.ToLEBytes()
	require.Equal(t, bytes, res)
}

func TestUint128_Cmp(t *testing.T) {
	bytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6}
	u0 := Uint128FromLEBytes(bytes)
	u1 := Uint128FromLEBytes(bytes)
	require.Equal(t, 0, u0.Cmp(u1))

	bytes = []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5}
	u2 := Uint128FromLEBytes(bytes)
	require.Equal(t, 1, u0.Cmp(u2))
	require.Equal(t, -1, u2.Cmp(u0))

	bytes = []byte{1, 2, 3}
	u3 := Uint128FromLEBytes(bytes)
	require.Equal(t, 1, u0.Cmp(u3))
	require.Equal(t, -1, u3.Cmp(u0))
}
