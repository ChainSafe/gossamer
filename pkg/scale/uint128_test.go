// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
