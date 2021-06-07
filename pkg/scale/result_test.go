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

type MyResult Result

func TestEncodeDecodeResult(t *testing.T) {
	err := RegisterResult(MyResult{}, MyStruct{}, false)
	if err != nil {
		t.Errorf("%v", err)
	}

	ms := MyStruct{
		Foo: []byte{0x01},
		Bar: 2,
		Baz: true,
	}
	mr := MyResult{Ok: ms}
	bytes, err := Marshal(mr)
	if err != nil {
		t.Errorf("%v", err)
	}

	if !reflect.DeepEqual([]byte{0x00, 0x04, 0x01, 0x02, 0, 0, 0, 0x01}, bytes) {
		t.Errorf("unexpected bytes: %v", bytes)
	}

	mr1 := MyResult{Err: true}
	bytes, err = Marshal(mr1)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual([]byte{0x01, 0x01}, bytes) {
		t.Errorf("unexpected bytes: %v", bytes)
	}

	mr2 := MyResult{}
	err = Unmarshal([]byte{0x00, 0x04, 0x01, 0x02, 0, 0, 0, 0x01}, &mr2)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual(MyResult{Ok: ms}, mr2) {
		t.Errorf("unexpected MyResult")
	}

	mr3 := MyResult{}
	err = Unmarshal([]byte{0x01, 0x01}, &mr3)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual(MyResult{Err: true}, mr3) {
		t.Errorf("unexpected MyResult yo")
	}
}
