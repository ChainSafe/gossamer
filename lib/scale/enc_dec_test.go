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
	"reflect"
	"testing"
)

func TestEncodeDecodeComplexStruct(t *testing.T) {
	type SimpleStruct struct {
		A int64
		B bool
	}

	type ComplexStruct struct {
		B   bool
		I   int
		I8  int8
		I16 int16
		I32 int32
		I64 int64
		U   uint
		U8  uint8
		U16 uint16
		U32 uint32
		U64 uint64
		Str string
		Bz  []byte
		Sub *SimpleStruct
	}

	test := &ComplexStruct{
		B:   true,
		I:   1,
		I8:  2,
		I16: 3,
		I32: 4,
		I64: 5,
		U:   6,
		U8:  7,
		U16: 8,
		U32: 9,
		U64: 10,
		Str: "choansafe",
		Bz:  []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Sub: &SimpleStruct{
			A: 99,
			B: true,
		},
	}

	enc, err := Encode(test)
	if err != nil {
		t.Fatal(err)
	}

	res := &ComplexStruct{
		Sub: &SimpleStruct{},
	}
	output, err := Decode(enc, res)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(output.(*ComplexStruct), test) {
		t.Errorf("Fail: got %+v expected %+v", output.(*ComplexStruct), test)
	}
}
