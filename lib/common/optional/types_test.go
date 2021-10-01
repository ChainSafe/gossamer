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

package optional

import (
	"testing"
)


func TestDecodeBytes(t *testing.T) {
	//testByteData := []byte("testData")
	//
	//testBytes := NewBytes(false, nil)
	//
	//require.False(t, testBytes.Exists(), "exist should be false")
	//require.Equal(t, []byte(nil), testBytes.Value(), "value should be empty")

	//testBytes.Set(true, testByteData)
	//require.True(t, testBytes.Exists(), "exist should be true")
	//require.Equal(t, testByteData, testBytes.Value(), "value should be Equal")

	//encData, err := testBytes.Encode()
	//require.NoError(t, err)
	//require.NotNil(t, encData)

	//newBytes, err := testBytes.DecodeBytes(encData)
	//require.NoError(t, err)
	//
	//require.True(t, newBytes.Exists(), "exist should be true")
	//require.Equal(t, testBytes.Value(), newBytes.Value(), "value should be Equal")
	//
	//// Invalid data
	//_, err = newBytes.DecodeBytes(nil)
	//require.Equal(t, err, ErrInvalidOptional)
	//
	//newBytes, err = newBytes.DecodeBytes([]byte{0})
	//require.NoError(t, err)
	//
	//require.False(t, newBytes.Exists(), "exist should be false")
	//require.Equal(t, []byte(nil), newBytes.Value(), "value should be empty")
}
