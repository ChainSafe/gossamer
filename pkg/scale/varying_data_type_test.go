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
	"math/big"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func mustNewVaryingDataType(values ...VaryingDataTypeValue) (vdt VaryingDataType) {
	vdt, err := NewVaryingDataType(values...)
	if err != nil {
		panic(err)
	}
	return
}

func mustNewVaryingDataTypeAndSet(value VaryingDataTypeValue, values ...VaryingDataTypeValue) (vdt VaryingDataType) {
	vdt = mustNewVaryingDataType(values...)
	err := vdt.Set(value)
	if err != nil {
		panic(err)
	}
	return
}

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

func (ctrd VDTValue) Index() uint {
	return 1
}

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

func (ctrd VDTValue1) Index() uint {
	return 2
}

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

func (ctrd VDTValue2) Index() uint {
	return 3
}

type VDTValue3 int16

func (ctrd VDTValue3) Index() uint {
	return 4
}

var varyingDataTypeTests = tests{
	{
		in: mustNewVaryingDataTypeAndSet(
			VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))},
			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
		),
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
		in: mustNewVaryingDataTypeAndSet(
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
			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
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
		in: mustNewVaryingDataTypeAndSet(
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
			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
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
		in: mustNewVaryingDataTypeAndSet(
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
			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
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
		in: mustNewVaryingDataTypeAndSet(
			VDTValue3(16383),
			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
		),
		want: newWant(
			// index of VDTValue2
			[]byte{4},
			// encoding of int16
			[]byte{0xff, 0x3f},
		),
	},
}

func Test_encodeState_encodeVaryingDataType(t *testing.T) {
	for _, tt := range varyingDataTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{fieldScaleIndicesCache: cache}
			vdt := tt.in.(VaryingDataType)
			if err := es.marshal(vdt); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeStruct() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_decodeState_decodeVaryingDataType(t *testing.T) {
	for _, tt := range varyingDataTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			dst, err := NewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))
			if err != nil {
				t.Errorf("%v", err)
				return
			}
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			vdt := tt.in.(VaryingDataType)
			diff := cmp.Diff(dst.Value(), vdt.Value(), cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}))
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
			}
		})
	}
}

func TestNewVaryingDataType(t *testing.T) {
	type args struct {
		values []VaryingDataTypeValue
	}
	tests := []struct {
		name    string
		args    args
		wantVdt VaryingDataType
		wantErr bool
	}{
		{
			args: args{
				values: []VaryingDataTypeValue{},
			},
			wantErr: true,
		},
		{
			args: args{
				values: []VaryingDataTypeValue{
					VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
				},
			},
			wantVdt: VaryingDataType{
				cache: map[uint]VaryingDataTypeValue{
					VDTValue{}.Index():   VDTValue{},
					VDTValue1{}.Index():  VDTValue1{},
					VDTValue2{}.Index():  VDTValue2{},
					VDTValue3(0).Index(): VDTValue3(0),
				},
			},
		},
		{
			args: args{
				values: []VaryingDataTypeValue{
					VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0), VDTValue{},
				},
			},
			wantVdt: VaryingDataType{
				cache: map[uint]VaryingDataTypeValue{
					VDTValue{}.Index():   VDTValue{},
					VDTValue1{}.Index():  VDTValue1{},
					VDTValue2{}.Index():  VDTValue2{},
					VDTValue3(0).Index(): VDTValue3(0),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVdt, err := NewVaryingDataType(tt.args.values...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVaryingDataType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotVdt, tt.wantVdt) {
				t.Errorf("NewVaryingDataType() = %v, want %v", gotVdt, tt.wantVdt)
			}
		})
	}
}

func TestVaryingDataType_Set(t *testing.T) {
	type args struct {
		value VaryingDataTypeValue
	}
	tests := []struct {
		name    string
		vdt     VaryingDataType
		args    args
		wantErr bool
	}{
		{
			vdt: mustNewVaryingDataType(VDTValue1{}),
			args: args{
				value: VDTValue1{},
			},
		},
		{
			vdt: mustNewVaryingDataType(VDTValue1{}, VDTValue2{}),
			args: args{
				value: VDTValue1{},
			},
		},
		{
			vdt: mustNewVaryingDataType(VDTValue1{}, VDTValue2{}),
			args: args{
				value: VDTValue2{},
			},
		},
		{
			vdt: mustNewVaryingDataType(VDTValue1{}, VDTValue2{}),
			args: args{
				value: VDTValue3(0),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdt := tt.vdt
			if err := vdt.Set(tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("VaryingDataType.SetValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVaryingDataTypeSlice_Add(t *testing.T) {
	type args struct {
		values []VaryingDataTypeValue
	}
	tests := []struct {
		name       string
		vdts       VaryingDataTypeSlice
		args       args
		wantErr    bool
		wantValues []VaryingDataType
	}{
		{
			name: "happy path",
			vdts: NewVaryingDataTypeSlice(MustNewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))),
			args: args{
				values: []VaryingDataTypeValue{
					VDTValue{
						B: 1,
					},
				},
			},
			wantValues: []VaryingDataType{
				mustNewVaryingDataTypeAndSet(
					VDTValue{
						B: 1,
					},
					VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
				),
			},
		},
		{
			name: "invalid value error case",
			vdts: NewVaryingDataTypeSlice(MustNewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{})),
			args: args{
				values: []VaryingDataTypeValue{
					VDTValue3(0),
				},
			},
			wantValues: []VaryingDataType{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdts := &tt.vdts
			if err := vdts.Add(tt.args.values...); (err != nil) != tt.wantErr {
				t.Errorf("VaryingDataTypeSlice.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(vdts.Types, tt.wantValues) {
				t.Errorf("NewVaryingDataType() = %v, want %v", vdts.Types, tt.wantValues)
			}
		})
	}
}

var varyingDataTypeSliceTests = tests{
	{
		in: mustNewVaryingDataTypeSliceAndSet(
			mustNewVaryingDataType(
				VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
			),
			VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))},
		),
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
		in: mustNewVaryingDataTypeSliceAndSet(
			mustNewVaryingDataType(
				VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
			),
			VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))},
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

func Test_encodeState_encodeVaryingDataTypeSlice(t *testing.T) {
	for _, tt := range varyingDataTypeSliceTests {
		t.Run(tt.name, func(t *testing.T) {
			vdt := tt.in.(VaryingDataTypeSlice)
			b, err := Marshal(vdt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(b, tt.want) {
				t.Errorf("Marshal() = %v, want %v", b, tt.want)
			}
		})
	}
}

func Test_decodeState_decodeVaryingDataTypeSlice(t *testing.T) {
	opt := cmp.Comparer(func(x, y VaryingDataType) bool {
		return reflect.DeepEqual(x.value, y.value) && reflect.DeepEqual(x.cache, y.cache)
	})

	for _, tt := range varyingDataTypeSliceTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := tt.in.(VaryingDataTypeSlice)
			dst.Types = make([]VaryingDataType, 0)
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			vdts := tt.in.(VaryingDataTypeSlice)
			diff := cmp.Diff(dst, vdts, cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}), opt)
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
			}
		})
	}
}
