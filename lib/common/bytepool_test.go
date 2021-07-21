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

package common

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytePool(t *testing.T) {
	bp := NewBytePool(5)
	require.Equal(t, 0, bp.Len())

	for i := 0; i < 5; i++ {
		err := bp.Put(generateID())
		require.NoError(t, err)
	}
	err := bp.Put(generateID())
	require.EqualError(t, err, "pool is full")
	require.Equal(t, 5, bp.Len())

	for i := 0; i < 5; i++ {
		_, err := bp.Get() // nolint
		require.NoError(t, err)
	}
	_, err = bp.Get()
	require.EqualError(t, err, "all slots used")
}

func TestBytePool256(t *testing.T) {
	bp := NewBytePool256()
	require.Equal(t, 256, bp.Len())

	for i := 0; i < 256; i++ {
		_, err := bp.Get() // nolint
		require.NoError(t, err)
	}
	_, err := bp.Get()
	require.EqualError(t, err, "all slots used")
}

func generateID() byte {
	// skipcq: GSC-G404
	id := rand.Intn(256) //nolint
	return byte(id)
}
