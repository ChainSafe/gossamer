package scale_test

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

type MyStruct struct {
	Baz bool
	Bar uint32
	Foo []byte
}

func (ms MyStruct) Index() uint {
	return 1
}

type MyOtherStruct struct {
	Foo string
	Bar uint64
	Baz uint
}

func (mos MyOtherStruct) Index() uint {
	return 2
}

type MyInt16 int16

func (mi16 MyInt16) Index() uint {
	return 3
}

type MyVaryingDataType scale.VaryingDataType

func varyingDataTypeExample() {
	err := scale.RegisterVaryingDataType(MyVaryingDataType{}, MyStruct{}, MyOtherStruct{}, MyInt16(0))
	if err != nil {
		panic(err)
	}

	mvdt := MyVaryingDataType{
		MyStruct{
			Baz: true,
			Bar: 999,
			Foo: []byte{1, 2},
		},
		MyOtherStruct{
			Foo: "hello",
			Bar: 999,
			Baz: 888,
		},
		MyInt16(111),
	}
	bytes, err := scale.Marshal(mvdt)
	if err != nil {
		panic(err)
	}

	var unmarshaled MyVaryingDataType
	err = scale.Unmarshal(bytes, &unmarshaled)
	if err != nil {
		panic(err)
	}

	// [{Baz:true Bar:999 Foo:[1 2]} {Foo:hello Bar:999 Baz:888} 111]
	fmt.Printf("%+v", unmarshaled)
}
func structExample() {
	type MyStruct struct {
		Baz bool   `scale:"3"`
		Bar int32  `scale:"2"`
		Foo []byte `scale:"1"`
	}
	var ms = MyStruct{
		Baz: true,
		Bar: 999,
		Foo: []byte{1, 2},
	}
	bytes, err := scale.Marshal(ms)
	if err != nil {
		panic(err)
	}

	var unmarshaled MyStruct
	err = scale.Unmarshal(bytes, &unmarshaled)
	if err != nil {
		panic(err)
	}

	// Baz:true Bar:999 Foo:[1 2]}
	fmt.Printf("%+v", unmarshaled)
}
func TestSomething(t *testing.T) {
	// // compact length encoded uint
	// var ui uint = 999
	// bytes, err := scale.Marshal(ui)
	// if err != nil {
	// 	panic(err)
	// }

	// var unmarshaled uint
	// err = scale.Unmarshal(bytes, &unmarshaled)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Printf("%d", unmarshaled)

	// structExample()
	varyingDataTypeExample()
}
