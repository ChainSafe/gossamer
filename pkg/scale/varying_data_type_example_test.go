// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale_test

// import (
// 	"fmt"
// 	"reflect"
// 	"testing"

// 	"github.com/ChainSafe/gossamer/pkg/scale"
// )

// type MyStruct struct {
// 	Baz bool
// 	Bar uint32
// 	Foo []byte
// }

// func (MyStruct) Index() uint {
// 	return 1
// }

// func (m MyStruct) String() string {
// 	return fmt.Sprintf("MyStruct{Baz: %t, Bar: %d, Foo: %x}", m.Baz, m.Bar, m.Foo)
// }

// type MyOtherStruct struct {
// 	Foo string
// 	Bar uint64
// 	Baz uint
// }

// func (MyOtherStruct) Index() uint {
// 	return 2
// }

// func (m MyOtherStruct) String() string {
// 	return fmt.Sprintf("MyOtherStruct{Foo: %s, Bar: %d, Baz: %d}", m.Foo, m.Bar, m.Baz)
// }

// type MyInt16 int16

// func (MyInt16) Index() uint {
// 	return 3
// }

// func (m MyInt16) String() string { return fmt.Sprintf("MyInt16(%d)", m) }

// func ExampleVaryingDataType() {
// 	vdt, err := scale.NewVaryingDataType(MyStruct{}, MyOtherStruct{}, MyInt16(0))
// 	if err != nil {
// 		panic(err)
// 	}

// 	err = vdt.Set(MyStruct{
// 		Baz: true,
// 		Bar: 999,
// 		Foo: []byte{1, 2},
// 	})
// 	if err != nil {
// 		panic(err)
// 	}

// 	bytes, err := scale.Marshal(vdt)
// 	if err != nil {
// 		panic(err)
// 	}

// 	vdt1, err := scale.NewVaryingDataType(MyStruct{}, MyOtherStruct{}, MyInt16(0))
// 	if err != nil {
// 		panic(err)
// 	}

// 	err = scale.Unmarshal(bytes, &vdt1)
// 	if err != nil {
// 		panic(err)
// 	}

// 	if !reflect.DeepEqual(vdt, vdt1) {
// 		panic(fmt.Errorf("uh oh: %+v %+v", vdt, vdt1))
// 	}
// }

// func ExampleVaryingDataTypeSlice() {
// 	vdt, err := scale.NewVaryingDataType(MyStruct{}, MyOtherStruct{}, MyInt16(0))
// 	if err != nil {
// 		panic(err)
// 	}

// 	vdts := scale.NewVaryingDataTypeSlice(vdt)

// 	err = vdts.Add(
// 		MyStruct{
// 			Baz: true,
// 			Bar: 999,
// 			Foo: []byte{1, 2},
// 		},
// 		MyInt16(1),
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	bytes, err := scale.Marshal(vdts)
// 	if err != nil {
// 		panic(err)
// 	}

// 	vdts1 := scale.NewVaryingDataTypeSlice(vdt)
// 	if err != nil {
// 		panic(err)
// 	}

// 	err = scale.Unmarshal(bytes, &vdts1)
// 	if err != nil {
// 		panic(err)
// 	}

// 	if !reflect.DeepEqual(vdts, vdts1) {
// 		panic(fmt.Errorf("uh oh: %+v %+v", vdts, vdts1))
// 	}
// }

// func TestExamples(_ *testing.T) {
// 	ExampleVaryingDataType()
// 	ExampleVaryingDataTypeSlice()
// }
