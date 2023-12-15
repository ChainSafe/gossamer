// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

type VDTValue struct {
	A *big.Int
	B int
	C uint
	D int8
	E uint8
	F int16
	G uint16
	H int32
	I uint32
	J int64
	K uint64
	L []byte
	M string
	N bool
}

func (VDTValue) Index() uint {
	return 1
}

func (VDTValue) String() string { return "" }

type VDTValue1 struct {
	O  **big.Int
	P  *int
	Q  *uint
	R  *int8
	S  *uint8
	T  *int16
	U  *uint16
	V  *int32
	W  *uint32
	X  *int64
	Y  *uint64
	Z  *[]byte
	AA *string
	AB *bool
}

func (VDTValue1) Index() uint {
	return 2
}

func (VDTValue1) String() string { return "" }

type VDTValue2 struct {
	A MyStruct
	B MyStructWithIgnore
	C *MyStruct
	D *MyStructWithIgnore

	E []int
	F []bool
	G []*big.Int
	H [][]int
	I [][]byte

	J [4]int
	K [3]bool
	L [2][]int
	M [2][2]int
	N [2]*big.Int
	O [2][]byte
	P [2][2]byte
}

func (VDTValue2) Index() uint {
	return 3
}

func (VDTValue2) String() string { return "" }

type VDTValue3 int16

func (VDTValue3) Index() uint {
	return 4
}

func (VDTValue3) String() string { return "" }

type MyVaryingDataType struct {
	inner any
}
type MyVaryingDataTypeValues interface {
	VDTValue | VDTValue1 | VDTValue2 | VDTValue3
}

func NewMyVaringDataType[Value MyVaryingDataTypeValues](value ...Value) *MyVaryingDataType {
	if len(value) == 0 {
		return &MyVaryingDataType{}
	}
	return &MyVaryingDataType{
		inner: value[0],
	}
}

func setMyVaryingDataType[Value MyVaryingDataTypeValues](mvdt *MyVaryingDataType, value Value) {
	mvdt.inner = value
}

func (mvdt *MyVaryingDataType) SetValue(value any) (err error) {
	switch value := value.(type) {
	case VDTValue:
		setMyVaryingDataType[VDTValue](mvdt, value)
		return
	case VDTValue1:
		setMyVaryingDataType[VDTValue1](mvdt, value)
		return
	case VDTValue2:
		setMyVaryingDataType[VDTValue2](mvdt, value)
		return
	case VDTValue3:
		setMyVaryingDataType[VDTValue3](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt MyVaryingDataType) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case VDTValue:
		return 1, any(mvdt.inner), nil
	case VDTValue1:
		return 2, any(mvdt.inner), nil
	case VDTValue2:
		return 3, any(mvdt.inner), nil
	case VDTValue3:
		return 4, any(mvdt.inner), nil
	}
	return 0, nil, ErrUnsupportedVaryingDataTypeValue
}

func (mvdt MyVaryingDataType) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt MyVaryingDataType) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return VDTValue{}, nil
	case 2:
		return VDTValue1{}, nil
	case 3:
		return VDTValue2{}, nil
	case 4:
		return VDTValue3(0), nil
	}
	return nil, errUnknownVaryingDataTypeValue
}

var varyingDataTypeTests = tests{
	test{
		in: NewMyVaringDataType(VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}),
		want: []byte{
			2,
			0x01, 0xfe, 0xff, 0xff, 0xff,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
		},
	},
	{
		in: NewMyVaringDataType(
			VDTValue{
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
			},
		),
		want: newWant(
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
	{
		in: NewMyVaringDataType(
			VDTValue1{
				O:  newBigIntPtr(big.NewInt(1073741823)),
				P:  newIntPtr(int(1073741823)),
				Q:  newUintPtr(uint(1073741823)),
				R:  newInt8Ptr(int8(1)),
				S:  newUint8Ptr(uint8(1)),
				T:  newInt16Ptr(16383),
				U:  newUint16Ptr(16383),
				V:  newInt32Ptr(1073741823),
				W:  newUint32Ptr(1073741823),
				X:  newInt64Ptr(9223372036854775807),
				Y:  newUint64Ptr(9223372036854775807),
				Z:  newBytesPtr(byteArray(64)),
				AA: newStringPtr(testStrings[1]),
				AB: newBoolPtr(true),
			},
		),
		want: newWant(
			// index of VDTValue1
			[]byte{2},
			// encoding of struct
			[]byte{
				0x01, 0xfe, 0xff, 0xff, 0xff,
				0x01, 0xfe, 0xff, 0xff, 0xff,
				0x01, 0xfe, 0xff, 0xff, 0xff,
				0x01, 0x01,
				0x01, 0x01,
				0x01, 0xff, 0x3f,
				0x01, 0xff, 0x3f,
				0x01, 0xff, 0xff, 0xff, 0x3f,
				0x01, 0xff, 0xff, 0xff, 0x3f,
				0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
				0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
			},
			append([]byte{0x01, 0x01, 0x01}, byteArray(64)...),
			append([]byte{0x01, 0xC2, 0x02, 0x01, 0x00}, testStrings[1]...),
			[]byte{0x01, 0x01},
		),
	},
	{
		in: NewMyVaringDataType(
			VDTValue2{
				A: MyStruct{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},
				B: MyStructWithIgnore{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},
				C: &MyStruct{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},
				D: &MyStructWithIgnore{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},

				E: []int{1073741824, 2, 3, 4},
				F: []bool{true, false, true},
				G: []*big.Int{big.NewInt(0), big.NewInt(1)},
				H: [][]int{{0, 1}, {1, 0}},
				I: [][]byte{{0x00, 0x01}, {0x01, 0x00}},

				J: [4]int{1073741824, 2, 3, 4},
				K: [3]bool{true, false, true},
				L: [2][]int{{0, 1}, {1, 0}},
				M: [2][2]int{{0, 1}, {1, 0}},
				N: [2]*big.Int{big.NewInt(0), big.NewInt(1)},
				O: [2][]byte{{0x00, 0x01}, {0x01, 0x00}},
				P: [2][2]byte{{0x00, 0x01}, {0x01, 0x00}},
			},
		),
		want: newWant(
			// index of VDTValue2
			[]byte{3},
			// encoding of struct
			[]byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
			[]byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
			[]byte{0x01, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
			[]byte{0x01, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},

			[]byte{0x10, 0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
			[]byte{0x0c, 0x01, 0x00, 0x01},
			[]byte{0x08, 0x00, 0x04},
			[]byte{0x08, 0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
			[]byte{0x08, 0x08, 0x00, 0x01, 0x08, 0x01, 0x00},

			[]byte{0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
			[]byte{0x01, 0x00, 0x01},
			[]byte{0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
			[]byte{0x00, 0x04, 0x04, 0x00},
			[]byte{0x00, 0x04},
			[]byte{0x08, 0x00, 0x01, 0x08, 0x01, 0x00},
			[]byte{0x00, 0x01, 0x01, 0x00},
		),
	},
	{
		in: NewMyVaringDataType(
			VDTValue3(16383),
		),
		want: newWant(
			// index of VDTValue2
			[]byte{4},
			// encoding of int16
			[]byte{0xff, 0x3f},
		),
	},
}

func TestVaryingDataType_Encode(t *testing.T) {
	for _, tt := range varyingDataTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			vdt := tt.in.(varyingDataTypeEncode)
			bytes, err := Marshal(vdt)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, bytes)
		})
	}
}

func TestVaryingDataType_Decode(t *testing.T) {
	for _, tt := range varyingDataTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := NewMyVaringDataType[VDTValue3]()
			err := Unmarshal(tt.want, dst)
			assert.NoError(t, err)

			dstVal, err := tt.in.(VaryingDataType).Value()
			assert.NoError(t, err)

			vdtVal, err := dst.Value()
			assert.NoError(t, err)

			assert.Equal(t, vdtVal, dstVal)
		})
	}
}

var varyingDataTypeSliceTests = tests{
	{
		in: []VaryingDataType{
			NewMyVaringDataType(VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}),
		},
		want: newWant(
			[]byte{
				// length
				4,
				// index
				2,
				// value
				0x01, 0xfe, 0xff, 0xff, 0xff,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
			},
		),
	},
	{
		in: []VaryingDataType{
			NewMyVaringDataType(VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}),
			NewMyVaringDataType(VDTValue{
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
			}),
		},
		want: newWant(
			[]byte{
				// length
				8,
			},
			[]byte{
				// index
				2,
				// value
				0x01, 0xfe, 0xff, 0xff, 0xff,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
			},
			[]byte{
				// index
				1,
				// value
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

func TestVaryingDataType_EncodeSlice(t *testing.T) {
	for _, tt := range varyingDataTypeSliceTests {
		t.Run(tt.name, func(t *testing.T) {
			vdt := tt.in.([]VaryingDataType)
			b, err := Marshal(vdt)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, b)
		})
	}
}

func TestVaryingDataType_DecodeSlice(t *testing.T) {
	for _, tt := range varyingDataTypeSliceTests {
		t.Run(tt.name, func(t *testing.T) {
			var dst []MyVaryingDataType
			err := Unmarshal(tt.want, &dst)
			assert.NoError(t, err)

			dstValues := make([]any, len(dst))
			for i, vdt := range dst {
				value, err := vdt.Value()
				assert.NoError(t, err)
				dstValues[i] = value
			}

			expectedValues := make([]any, len(tt.in.([]VaryingDataType)))
			for i, vdt := range tt.in.([]VaryingDataType) {
				value, err := vdt.Value()
				assert.NoError(t, err)
				expectedValues[i] = value
			}

			assert.Equal(t, expectedValues, dstValues)
		})
	}
}

func TestVaryingDataType_EncodeArray(t *testing.T) {
	vdtval1 := VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}
	mvdt := NewMyVaringDataType[VDTValue1](vdtval1)
	_ = VaryingDataType(mvdt)
	mvdtArray := [1]VaryingDataType{
		mvdt,
	}
	expected := []byte{
		2,
		0x01, 0xfe, 0xff, 0xff, 0xff,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
	}

	bytes, err := Marshal(mvdtArray)
	if err != nil {
		t.Errorf("wtf %v", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, bytes)
}

func TestVaryingDataType_DecodeArray(t *testing.T) {
	vdtval1 := VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}
	mvdt := NewMyVaringDataType[VDTValue1](vdtval1)
	_ = VaryingDataType(mvdt)
	expected := [1]MyVaryingDataType{
		*mvdt,
	}
	var mvdtArr [1]MyVaryingDataType

	bytes := []byte{
		2,
		0x01, 0xfe, 0xff, 0xff, 0xff,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
		0x00,
	}
	err := Unmarshal(bytes, &mvdtArr)
	assert.NoError(t, err)
	assert.Equal(t, expected, mvdtArr)
}
