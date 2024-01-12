// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale_test

import (
	"fmt"
	"reflect"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

type MyStruct struct {
	Baz bool
	Bar uint32
	Foo []byte
}

type MyOtherStruct struct {
	Foo string
	Bar uint64
	Baz uint
}

type MyInt16 int16

type MyVaryingDataType struct {
	inner any
}

type MyVaryingDataTypeValues interface {
	MyStruct | MyOtherStruct | MyInt16
}

func setMyVaryingDataType[Value MyVaryingDataTypeValues](mvdt *MyVaryingDataType, value Value) {
	mvdt.inner = value
}

func (mvdt *MyVaryingDataType) SetValue(value any) (err error) {
	switch value := value.(type) {
	case MyStruct:
		setMyVaryingDataType(mvdt, value)
		return
	case MyOtherStruct:
		setMyVaryingDataType(mvdt, value)
		return
	case MyInt16:
		setMyVaryingDataType(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt MyVaryingDataType) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case MyStruct:
		return 0, mvdt.inner, nil
	case MyOtherStruct:
		return 1, mvdt.inner, nil
	case MyInt16:
		return 2, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt MyVaryingDataType) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt MyVaryingDataType) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return MyStruct{}, nil
	case 1:
		return MyOtherStruct{}, nil
	case 2:
		return *new(MyInt16), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func ExampleVaryingDataType() {
	vdt := MyVaryingDataType{}

	err := vdt.SetValue(MyStruct{
		Baz: true,
		Bar: 999,
		Foo: []byte{1, 2},
	})
	if err != nil {
		panic(err)
	}

	bytes, err := scale.Marshal(vdt)
	if err != nil {
		panic(err)
	}

	dst := MyVaryingDataType{}

	err = scale.Unmarshal(bytes, &dst)
	if err != nil {
		panic(err)
	}

	fmt.Println(reflect.DeepEqual(vdt, dst))
	// Output: true
}
