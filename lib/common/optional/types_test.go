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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBoolean(t *testing.T) {
	// Non-existent
	testBool := NewBoolean(false, false)

	if testBool.Exists() {
		t.Fatal("exist should be false")
	}
	if testBool.value {
		t.Fatal("value should be false")
	}
}

func TestBooleanSet(t *testing.T) {
	testBool := NewBoolean(false, false)

	if testBool.Exists() {
		t.Fatal("exist should be false")
	}
	if testBool.value {
		t.Fatal("value should be false")
	}

	testBool.Set(true)

	if !testBool.Exists() {
		t.Fatal("exist should be true")
	}
	if !testBool.value {
		t.Fatal("value should be true")
	}
}

func TestBooleanExists(t *testing.T) {
	// Non-existent
	testBool := NewBoolean(false, false)

	if testBool.Exists() {
		t.Fatal("exist should be false")
	}

	testBool.Set(false)
	if !testBool.Exists() {
		t.Fatal("exist should be true")
	}
}

func TestBooleanValue(t *testing.T) {
	testBool := NewBoolean(false, false)

	if testBool.Value() {
		t.Fatal("value should be false")
	}

	testBool.Set(true)
	if !testBool.Value() {
		t.Fatal("exist should be true")
	}
}

func TestBooleanEncode(t *testing.T) {
	// Non-existent
	testBool := NewBoolean(false, false)

	_, err := testBool.Encode()

	if err != nil {
		t.Fatal(err)
	}

	// Possibly redundant
	testBool.Set(true)

	_, err = testBool.Encode()
	if err != nil {
		t.Fatal(err)
	}
}

func TestBooleanDecode(t *testing.T) {
	testBool := NewBoolean(false, false)

	encoded, err := testBool.Encode()

	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	buf.Write(encoded)

	decoded, decodeError := testBool.Decode(buf)

	if decodeError != nil {
		t.Fatal(decodeError)
	}

	if decoded.Exists() {
		t.Fatal("decoded exist should be false")
	}

	if decoded.Value() {
		t.Fatal("decoded value should be false")
	}

	testBool.Set(true)
	encoded, err = testBool.Encode()

	if err != nil {
		t.Fatal(err)
	}

	buf = &bytes.Buffer{}
	buf.Write(encoded)

	decoded, decodeError = testBool.Decode(buf)

	if decodeError != nil {
		t.Fatal(decodeError)
	}

	if !decoded.Exists() {
		t.Fatal("decoded exist should be true")
	}

	if !decoded.Value() {
		t.Fatal("decoded value should be true")
	}
	testBool.Set(false)
	encoded, err = testBool.Encode()

	if err != nil {
		t.Fatal(err)
	}

	buf = &bytes.Buffer{}
	buf.Write(encoded)

	decoded, decodeError = testBool.Decode(buf)

	if decodeError != nil {
		t.Fatal(decodeError)
	}

	if !decoded.Exists() {
		t.Fatal("decoded exist should be true")
	}

	if decoded.Value() {
		t.Fatal("decoded value should be false")
	}
}

func TestDecodeBytes(t *testing.T) {
	testByteData := []byte("testData")

	testBytes := NewBytes(false, nil)

	require.False(t, testBytes.Exists(), "exist should be false")
	require.Equal(t, []byte(nil), testBytes.Value(), "value should be empty")

	testBytes.Set(true, testByteData)
	require.True(t, testBytes.Exists(), "exist should be true")
	require.Equal(t, testByteData, testBytes.Value(), "value should be Equal")

	encData, err := testBytes.Encode()
	require.NoError(t, err)
	require.NotNil(t, encData)

	newBytes, err := testBytes.DecodeBytes(encData)
	require.NoError(t, err)

	require.True(t, newBytes.Exists(), "exist should be true")
	require.Equal(t, testBytes.Value(), newBytes.Value(), "value should be Equal")

	// Invalid data
	_, err = newBytes.DecodeBytes(nil)
	require.Equal(t, err, ErrInvalidOptional)

	newBytes, err = newBytes.DecodeBytes([]byte{0})
	require.NoError(t, err)

	require.False(t, newBytes.Exists(), "exist should be false")
	require.Equal(t, []byte(nil), newBytes.Value(), "value should be empty")
}
