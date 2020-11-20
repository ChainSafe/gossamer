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
)

func TestNewBoolean(t *testing.T) {
	// Non-existant
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
	// Non-existant
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
	// Non-existant
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
