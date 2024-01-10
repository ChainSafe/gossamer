// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ParentVDT struct {
	inner any
}

type ParentVDTValues interface {
	ChildVDT | ChildVDT1
}

func NewParentVDT[Value ParentVDTValues](value ...Value) *ParentVDT {
	if len(value) == 0 {
		return &ParentVDT{}
	}
	return &ParentVDT{
		inner: value[0],
	}
}

func setParentVDT[Value ParentVDTValues](mvdt *ParentVDT, value Value) {
	mvdt.inner = value
}

func (mvdt *ParentVDT) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ChildVDT:
		setParentVDT[ChildVDT](mvdt, value)
		return
	case ChildVDT1:
		setParentVDT[ChildVDT1](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ParentVDT) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ChildVDT:
		return 1, any(mvdt.inner), nil
	case ChildVDT1:
		return 2, any(mvdt.inner), nil
	}
	return 0, nil, ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ParentVDT) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ParentVDT) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return ChildVDT{}, nil
	case 2:
		return ChildVDT1{}, nil
	}
	return nil, ErrUnknownVaryingDataTypeValue
}

type ChildVDT struct {
	MyVaryingDataType
}

type ChildVDTValues interface {
	VDTValue | VDTValue1 | VDTValue2 | VDTValue3
}

func NewChildVDT[Value ChildVDTValues](value ...Value) *ChildVDT {
	if len(value) == 0 {
		return &ChildVDT{}
	}
	return &ChildVDT{
		MyVaryingDataType: *NewMyVaringDataType[Value](value...),
	}
}

func (cvdt *ChildVDT) SetValue(value any) (err error) {
	return cvdt.MyVaryingDataType.SetValue(value)
}

func (cvdt ChildVDT) IndexValue() (index uint, value any, err error) {
	return cvdt.MyVaryingDataType.IndexValue()
}

func (cvdt ChildVDT) Value() (value any, err error) {
	return cvdt.MyVaryingDataType.Value()
}

func (cvdt ChildVDT) ValueAt(index uint) (value any, err error) {
	return cvdt.MyVaryingDataType.ValueAt(index)
}

type ChildVDT1 struct {
	MyVaryingDataType
}

type ChildVDT1Values interface {
	VDTValue | VDTValue1 | VDTValue2 | VDTValue3
}

func NewChildVDT1[Value ChildVDT1Values](value ...Value) *ChildVDT1 {
	if len(value) == 0 {
		return &ChildVDT1{}
	}
	return &ChildVDT1{
		MyVaryingDataType: *NewMyVaringDataType[Value](value...),
	}
}

func (cvdt *ChildVDT1) SetValue(value any) (err error) {
	return cvdt.MyVaryingDataType.SetValue(value)
}

func (cvdt ChildVDT1) IndexValue() (index uint, value any, err error) {
	return cvdt.MyVaryingDataType.IndexValue()
}

func (cvdt ChildVDT1) Value() (value any, err error) {
	return cvdt.MyVaryingDataType.Value()
}

func (cvdt ChildVDT1) ValueAt(index uint) (value any, err error) {
	return cvdt.MyVaryingDataType.ValueAt(index)
}

var (
	_ = VaryingDataType(&ParentVDT{})
	_ = VaryingDataType(&ChildVDT{})
	_ = VaryingDataType(&ChildVDT1{})
)

type constructorTest struct {
	name  string
	newIn func(t *testing.T) interface{}
	want  []byte
}

var nestedVaryingDataTypeTests = []constructorTest{
	{
		name: "ParentVDT_with_ChildVDT",
		newIn: func(t *testing.T) interface{} {
			child := NewChildVDT(VDTValue3(16383))
			parent := NewParentVDT(*child)
			return parent
		},
		want: newWant(
			// index of childVDT
			[]byte{1},
			// index of VDTValue3
			[]byte{4},
			// encoding of int16
			[]byte{0xff, 0x3f},
		),
	},
	{
		name: "ParentVDT_with_ChildVDT1",
		newIn: func(t *testing.T) interface{} {
			child1 := NewChildVDT1(VDTValue{
				A: big.NewInt(1073741823),
				B: int(1073741823),
				C: uint(1073741823),
				D: int8(1),
				E: uint8(1),
				F: int16(16383),
				G: uint16(16383),
				H: int32(1073741823),
				I: uint32(1073741823),
				J: int64(9223372036854775807),
				K: uint64(9223372036854775807),
				L: byteArray(64),
				M: testStrings[1],
				N: true,
			})
			parent := NewParentVDT(*child1)
			return parent
		},
		want: newWant(
			// index of childVDT1
			[]byte{2},
			// index of VDTValue
			[]byte{1},
			// encoding of struct
			[]byte{
				0xfe, 0xff, 0xff, 0xff,
				0xfe, 0xff, 0xff, 0xff,
				0xfe, 0xff, 0xff, 0xff,
				0x01,
				0x01,
				0xff, 0x3f,
				0xff, 0x3f,
				0xff, 0xff, 0xff, 0x3f,
				0xff, 0xff, 0xff, 0x3f,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
			},
			append([]byte{0x01, 0x01}, byteArray(64)...),
			append([]byte{0xC2, 0x02, 0x01, 0x00}, testStrings[1]...),
			[]byte{0x01},
		),
	},
}

func TestVaryingDataType_EncodeNested(t *testing.T) {
	for _, tt := range nestedVaryingDataTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			vdt := tt.newIn(t).(*ParentVDT)
			b, err := Marshal(*vdt)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, b)
		})
	}
}

func TestVaryingDataType_DecodeNested(t *testing.T) {
	for _, tt := range nestedVaryingDataTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := NewParentVDT[ChildVDT]()
			err := Unmarshal(tt.want, dst)
			assert.NoError(t, err)

			expected := tt.newIn(t).(*ParentVDT)
			assert.Equal(t, expected.inner, dst.inner)

		})
	}
}
