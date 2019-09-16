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
	"bytes"
	"testing"
)

func TestBlake2b218(t *testing.T) {
	in := []byte{0x1}
	h, err := Blake2b128(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(h)
}

func TestBlake2bHash(t *testing.T) {
	in := []byte{0x1}
	h, err := Blake2bHash(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(h)
}

func TestKeccak256(t *testing.T) {
	in := []byte{}
	h := Keccak256(in)
	expected, err := HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected[:], h[:]) {
		t.Fatalf("Fail: got %x expected %x", h, expected)
	}
}
