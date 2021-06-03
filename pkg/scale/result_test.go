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
