// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// import (
// 	"bytes"
// 	"math/big"
// 	"reflect"
// 	"testing"

// 	"github.com/google/go-cmp/cmp"
// 	"github.com/google/go-cmp/cmp/cmpopts"
// )

// func mustNewVaryingDataType(values ...VaryingDataTypeValue) (vdt *DefaultVaryingDataType) {
// 	dvdt, err := NewDefaultVaryingDataType(values...)
// 	if err != nil {
// 		panic(err)
// 	}
// 	vdt = &dvdt
// 	return
// }

// func mustNewVaryingDataTypeAndSet(value VaryingDataTypeValue, values ...VaryingDataTypeValue) (vdt *DefaultVaryingDataType) {
// 	dvdt := mustNewVaryingDataType(values...)
// 	err := vdt.SetValue(value)
// 	if err != nil {
// 		panic(err)
// 	}
// 	vdt = dvdt
// 	return
// }

// type customVDT DefaultVaryingDataType

// type customVDTWithNew DefaultVaryingDataType

// func (cvwn customVDTWithNew) New() customVDTWithNew {
// 	// return customVDTWithNew(mustNewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0)))
// 	return nil
// }

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

// var varyingDataTypeTests = tests{
// 	{
// 		in: mustNewVaryingDataTypeAndSet(
// 			VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))},
// 			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 		),
// 		want: []byte{
// 			2,
// 			0x01, 0xfe, 0xff, 0xff, 0xff,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 			0x00,
// 		},
// 	},
// 	{
// 		in: mustNewVaryingDataTypeAndSet(
// 			VDTValue{
// 				A: big.NewInt(1073741823),
// 				B: int(1073741823),
// 				C: uint(1073741823),
// 				D: int8(1),
// 				E: uint8(1),
// 				F: int16(16383),
// 				G: uint16(16383),
// 				H: int32(1073741823),
// 				I: uint32(1073741823),
// 				J: int64(9223372036854775807),
// 				K: uint64(9223372036854775807),
// 				L: byteArray(64),
// 				M: testStrings[1],
// 				N: true,
// 			},
// 			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 		),
// 		want: newWant(
// 			// index of VDTValue
// 			[]byte{1},
// 			// encoding of struct
// 			[]byte{
// 				0xfe, 0xff, 0xff, 0xff,
// 				0xfe, 0xff, 0xff, 0xff,
// 				0xfe, 0xff, 0xff, 0xff,
// 				0x01,
// 				0x01,
// 				0xff, 0x3f,
// 				0xff, 0x3f,
// 				0xff, 0xff, 0xff, 0x3f,
// 				0xff, 0xff, 0xff, 0x3f,
// 				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
// 				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
// 			},
// 			append([]byte{0x01, 0x01}, byteArray(64)...),
// 			append([]byte{0xC2, 0x02, 0x01, 0x00}, testStrings[1]...),
// 			[]byte{0x01},
// 		),
// 	},
// 	{
// 		in: mustNewVaryingDataTypeAndSet(
// 			VDTValue1{
// 				O:  newBigIntPtr(big.NewInt(1073741823)),
// 				P:  newIntPtr(int(1073741823)),
// 				Q:  newUintPtr(uint(1073741823)),
// 				R:  newInt8Ptr(int8(1)),
// 				S:  newUint8Ptr(uint8(1)),
// 				T:  newInt16Ptr(16383),
// 				U:  newUint16Ptr(16383),
// 				V:  newInt32Ptr(1073741823),
// 				W:  newUint32Ptr(1073741823),
// 				X:  newInt64Ptr(9223372036854775807),
// 				Y:  newUint64Ptr(9223372036854775807),
// 				Z:  newBytesPtr(byteArray(64)),
// 				AA: newStringPtr(testStrings[1]),
// 				AB: newBoolPtr(true),
// 			},
// 			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 		),
// 		want: newWant(
// 			// index of VDTValue1
// 			[]byte{2},
// 			// encoding of struct
// 			[]byte{
// 				0x01, 0xfe, 0xff, 0xff, 0xff,
// 				0x01, 0xfe, 0xff, 0xff, 0xff,
// 				0x01, 0xfe, 0xff, 0xff, 0xff,
// 				0x01, 0x01,
// 				0x01, 0x01,
// 				0x01, 0xff, 0x3f,
// 				0x01, 0xff, 0x3f,
// 				0x01, 0xff, 0xff, 0xff, 0x3f,
// 				0x01, 0xff, 0xff, 0xff, 0x3f,
// 				0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
// 				0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
// 			},
// 			append([]byte{0x01, 0x01, 0x01}, byteArray(64)...),
// 			append([]byte{0x01, 0xC2, 0x02, 0x01, 0x00}, testStrings[1]...),
// 			[]byte{0x01, 0x01},
// 		),
// 	},
// 	{
// 		in: mustNewVaryingDataTypeAndSet(
// 			VDTValue2{
// 				A: MyStruct{
// 					Foo: []byte{0x01},
// 					Bar: 2,
// 					Baz: true,
// 				},
// 				B: MyStructWithIgnore{
// 					Foo: []byte{0x01},
// 					Bar: 2,
// 					Baz: true,
// 				},
// 				C: &MyStruct{
// 					Foo: []byte{0x01},
// 					Bar: 2,
// 					Baz: true,
// 				},
// 				D: &MyStructWithIgnore{
// 					Foo: []byte{0x01},
// 					Bar: 2,
// 					Baz: true,
// 				},

// 				E: []int{1073741824, 2, 3, 4},
// 				F: []bool{true, false, true},
// 				G: []*big.Int{big.NewInt(0), big.NewInt(1)},
// 				H: [][]int{{0, 1}, {1, 0}},
// 				I: [][]byte{{0x00, 0x01}, {0x01, 0x00}},

// 				J: [4]int{1073741824, 2, 3, 4},
// 				K: [3]bool{true, false, true},
// 				L: [2][]int{{0, 1}, {1, 0}},
// 				M: [2][2]int{{0, 1}, {1, 0}},
// 				N: [2]*big.Int{big.NewInt(0), big.NewInt(1)},
// 				O: [2][]byte{{0x00, 0x01}, {0x01, 0x00}},
// 				P: [2][2]byte{{0x00, 0x01}, {0x01, 0x00}},
// 			},
// 			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 		),
// 		want: newWant(
// 			// index of VDTValue2
// 			[]byte{3},
// 			// encoding of struct
// 			[]byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
// 			[]byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
// 			[]byte{0x01, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
// 			[]byte{0x01, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},

// 			[]byte{0x10, 0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
// 			[]byte{0x0c, 0x01, 0x00, 0x01},
// 			[]byte{0x08, 0x00, 0x04},
// 			[]byte{0x08, 0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
// 			[]byte{0x08, 0x08, 0x00, 0x01, 0x08, 0x01, 0x00},

// 			[]byte{0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
// 			[]byte{0x01, 0x00, 0x01},
// 			[]byte{0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
// 			[]byte{0x00, 0x04, 0x04, 0x00},
// 			[]byte{0x00, 0x04},
// 			[]byte{0x08, 0x00, 0x01, 0x08, 0x01, 0x00},
// 			[]byte{0x00, 0x01, 0x01, 0x00},
// 		),
// 	},
// 	{
// 		in: mustNewVaryingDataTypeAndSet(
// 			VDTValue3(16383),
// 			VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 		),
// 		want: newWant(
// 			// index of VDTValue2
// 			[]byte{4},
// 			// encoding of int16
// 			[]byte{0xff, 0x3f},
// 		),
// 	},
// }

// func Test_encodeState_encodeVaryingDataType(t *testing.T) {
// 	for _, tt := range varyingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			buffer := bytes.NewBuffer(nil)
// 			es := &encodeState{
// 				Writer:                 buffer,
// 				fieldScaleIndicesCache: cache,
// 			}
// 			vdt := tt.in.(VaryingDataType)
// 			if err := es.marshal(vdt); (err != nil) != tt.wantErr {
// 				t.Errorf("encodeState.marshal() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if !reflect.DeepEqual(buffer.Bytes(), tt.want) {
// 				t.Errorf("encodeState.marshal() = %v, want %v", buffer.Bytes(), tt.want)
// 			}
// 		})
// 	}
// }

// func Test_decodeState_decodeVaryingDataType(t *testing.T) {
// 	for _, tt := range varyingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			dst, err := NewDefaultVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))
// 			if err != nil {
// 				t.Errorf("%v", err)
// 				return
// 			}
// 			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
// 				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			vdt := tt.in.(VaryingDataType)
// 			dstVal := dst.Value()
// 			if dstVal == nil {
// 				t.Errorf("nil vdt value")
// 				return
// 			}
// 			vdtVal := vdt.Value()
// 			if vdtVal == nil {
// 				t.Errorf("nil vdt value")
// 				return
// 			}
// 			diff := cmp.Diff(dstVal, vdtVal, cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}))
// 			if diff != "" {
// 				t.Errorf("decodeState.unmarshal() = %s", diff)
// 			}
// 		})
// 	}
// }

// func Test_encodeState_encodeCustomVaryingDataType(t *testing.T) {
// 	for _, tt := range varyingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			buffer := bytes.NewBuffer(nil)
// 			es := &encodeState{
// 				Writer:                 buffer,
// 				fieldScaleIndicesCache: cache,
// 			}
// 			vdt := tt.in.(DefaultVaryingDataType)
// 			cvdt := customVDT(vdt)
// 			if err := es.marshal(cvdt); (err != nil) != tt.wantErr {
// 				t.Errorf("encodeState.encodeStruct() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if !reflect.DeepEqual(buffer.Bytes(), tt.want) {
// 				t.Errorf("encodeState.encodeStruct() = %v, want %v", buffer.Bytes(), tt.want)
// 			}
// 		})
// 	}
// }
// func Test_decodeState_decodeCustomVaryingDataType(t *testing.T) {
// 	for _, tt := range varyingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			vdt, err := NewDefaultVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))
// 			if err != nil {
// 				t.Errorf("%v", err)
// 				return
// 			}
// 			dst := customVDT(vdt)
// 			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
// 				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}

// 			dstVDT := reflect.ValueOf(tt.in).Convert(reflect.TypeOf(DefaultVaryingDataType{})).Interface().(VaryingDataType)
// 			inVDT := reflect.ValueOf(tt.in).Convert(reflect.TypeOf(DefaultVaryingDataType{})).Interface().(VaryingDataType)
// 			dstVDTVal := dstVDT.Value()
// 			if dstVDTVal == nil {
// 				t.Errorf("nil vdt value")
// 				return
// 			}
// 			inVDTVal := inVDT.Value()
// 			if dstVDTVal == nil {
// 				t.Errorf("nil vdt value")
// 				return
// 			}
// 			diff := cmp.Diff(dstVDTVal, inVDTVal,
// 				cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}))
// 			if diff != "" {
// 				t.Errorf("decodeState.unmarshal() = %s", diff)
// 			}
// 			if reflect.TypeOf(dst) != reflect.TypeOf(customVDT{}) {
// 				t.Errorf("types mismatch dst: %v expected: %v", reflect.TypeOf(dst), reflect.TypeOf(customVDT{}))
// 			}
// 		})
// 	}
// }

// func Test_decodeState_decodeCustomVaryingDataTypeWithNew(t *testing.T) {
// 	for _, tt := range varyingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			dst := customVDTWithNew{}
// 			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
// 				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}

// 			dstVDT := reflect.ValueOf(tt.in).Convert(reflect.TypeOf(DefaultVaryingDataType{})).Interface().(VaryingDataType)
// 			inVDT := reflect.ValueOf(tt.in).Convert(reflect.TypeOf(DefaultVaryingDataType{})).Interface().(VaryingDataType)
// 			dstVDTVal := dstVDT.Value()
// 			if dstVDTVal == nil {
// 				t.Errorf("nil vdt value")
// 				return
// 			}
// 			inVDTVal := inVDT.Value()
// 			if dstVDTVal == nil {
// 				t.Errorf("nil vdt value")
// 				return
// 			}
// 			diff := cmp.Diff(dstVDTVal, inVDTVal,
// 				cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}))
// 			if diff != "" {
// 				t.Errorf("decodeState.unmarshal() = %s", diff)
// 			}
// 			if reflect.TypeOf(dst) != reflect.TypeOf(customVDTWithNew{}) {
// 				t.Errorf("types mismatch dst: %v expected: %v", reflect.TypeOf(dst), reflect.TypeOf(customVDT{}))
// 			}
// 		})
// 	}
// }

// func TestNewVaryingDataType(t *testing.T) {
// 	type args struct {
// 		values []VaryingDataTypeValue
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantVdt VaryingDataType
// 		wantErr bool
// 	}{
// 		{
// 			args: args{
// 				values: []VaryingDataTypeValue{},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			args: args{
// 				values: []VaryingDataTypeValue{
// 					VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 				},
// 			},
// 			wantVdt: &DefaultVaryingDataType{
// 				cache: map[uint]VaryingDataTypeValue{
// 					VDTValue{}.Index():   VDTValue{},
// 					VDTValue1{}.Index():  VDTValue1{},
// 					VDTValue2{}.Index():  VDTValue2{},
// 					VDTValue3(0).Index(): VDTValue3(0),
// 				},
// 			},
// 		},
// 		{
// 			args: args{
// 				values: []VaryingDataTypeValue{
// 					VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0), VDTValue{},
// 				},
// 			},
// 			wantVdt: &DefaultVaryingDataType{
// 				cache: map[uint]VaryingDataTypeValue{
// 					VDTValue{}.Index():   VDTValue{},
// 					VDTValue1{}.Index():  VDTValue1{},
// 					VDTValue2{}.Index():  VDTValue2{},
// 					VDTValue3(0).Index(): VDTValue3(0),
// 				},
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			gotVdt, err := NewDefaultVaryingDataType(tt.args.values...)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("NewVaryingDataType() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(gotVdt, tt.wantVdt) {
// 				t.Errorf("NewVaryingDataType() = %v, want %v", gotVdt, tt.wantVdt)
// 			}
// 		})
// 	}
// }

// func TestVaryingDataType_Set(t *testing.T) {
// 	type args struct {
// 		value VaryingDataTypeValue
// 	}
// 	tests := []struct {
// 		name    string
// 		vdt     VaryingDataType
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			vdt: mustNewVaryingDataType(VDTValue1{}),
// 			args: args{
// 				value: VDTValue1{},
// 			},
// 		},
// 		{
// 			vdt: mustNewVaryingDataType(VDTValue1{}, VDTValue2{}),
// 			args: args{
// 				value: VDTValue1{},
// 			},
// 		},
// 		{
// 			vdt: mustNewVaryingDataType(VDTValue1{}, VDTValue2{}),
// 			args: args{
// 				value: VDTValue2{},
// 			},
// 		},
// 		{
// 			vdt: mustNewVaryingDataType(VDTValue1{}, VDTValue2{}),
// 			args: args{
// 				value: VDTValue3(0),
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			vdt := tt.vdt
// 			if err := vdt.Set(tt.args.value); (err != nil) != tt.wantErr {
// 				t.Errorf("VaryingDataType.SetValue() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

// func TestVaryingDataTypeSlice_Add(t *testing.T) {
// 	type args struct {
// 		values []VaryingDataTypeValue
// 	}
// 	tests := []struct {
// 		name       string
// 		vdts       VaryingDataTypeSlice
// 		args       args
// 		wantErr    bool
// 		wantValues []VaryingDataType
// 	}{
// 		{
// 			name: "happy_path",
// 			vdts: NewVaryingDataTypeSlice(MustNewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))),
// 			args: args{
// 				values: []VaryingDataTypeValue{
// 					VDTValue{
// 						B: 1,
// 					},
// 				},
// 			},
// 			wantValues: []VaryingDataType{
// 				mustNewVaryingDataTypeAndSet(
// 					VDTValue{
// 						B: 1,
// 					},
// 					VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 				),
// 			},
// 		},
// 		{
// 			name: "invalid_value_error_case",
// 			vdts: NewVaryingDataTypeSlice(MustNewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{})),
// 			args: args{
// 				values: []VaryingDataTypeValue{
// 					VDTValue3(0),
// 				},
// 			},
// 			wantValues: []VaryingDataType{},
// 			wantErr:    true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := tt.vdts.Add(tt.args.values...); (err != nil) != tt.wantErr {
// 				t.Errorf("VaryingDataTypeSlice.Add() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if !reflect.DeepEqual(tt.vdts.Types, tt.wantValues) {
// 				t.Errorf("NewVaryingDataType() = %v, want %v", tt.vdts.Types, tt.wantValues)
// 			}
// 		})
// 	}
// }

// var varyingDataTypeSliceTests = tests{
// 	{
// 		in: mustNewVaryingDataTypeSliceAndSet(
// 			mustNewVaryingDataType(
// 				VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 			),
// 			VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))},
// 		),
// 		want: newWant(
// 			[]byte{
// 				// length
// 				4,
// 				// index
// 				2,
// 				// value
// 				0x01, 0xfe, 0xff, 0xff, 0xff,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 			},
// 		),
// 	},
// 	{
// 		in: mustNewVaryingDataTypeSliceAndSet(
// 			mustNewVaryingDataType(
// 				VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0),
// 			),
// 			VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))},
// 			VDTValue{
// 				A: big.NewInt(1073741823),
// 				B: int(1073741823),
// 				C: uint(1073741823),
// 				D: int8(1),
// 				E: uint8(1),
// 				F: int16(16383),
// 				G: uint16(16383),
// 				H: int32(1073741823),
// 				I: uint32(1073741823),
// 				J: int64(9223372036854775807),
// 				K: uint64(9223372036854775807),
// 				L: byteArray(64),
// 				M: testStrings[1],
// 				N: true,
// 			},
// 		),
// 		want: newWant(
// 			[]byte{
// 				// length
// 				8,
// 			},
// 			[]byte{
// 				// index
// 				2,
// 				// value
// 				0x01, 0xfe, 0xff, 0xff, 0xff,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 				0x00,
// 			},
// 			[]byte{
// 				// index
// 				1,
// 				// value
// 				0xfe, 0xff, 0xff, 0xff,
// 				0xfe, 0xff, 0xff, 0xff,
// 				0xfe, 0xff, 0xff, 0xff,
// 				0x01,
// 				0x01,
// 				0xff, 0x3f,
// 				0xff, 0x3f,
// 				0xff, 0xff, 0xff, 0x3f,
// 				0xff, 0xff, 0xff, 0x3f,
// 				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
// 				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
// 			},
// 			append([]byte{0x01, 0x01}, byteArray(64)...),
// 			append([]byte{0xC2, 0x02, 0x01, 0x00}, testStrings[1]...),
// 			[]byte{0x01},
// 		),
// 	},
// }

// func Test_encodeState_encodeVaryingDataTypeSlice(t *testing.T) {
// 	for _, tt := range varyingDataTypeSliceTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			vdt := tt.in.(VaryingDataTypeSlice)
// 			b, err := Marshal(vdt)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if !reflect.DeepEqual(b, tt.want) {
// 				t.Errorf("Marshal() = %v, want %v", b, tt.want)
// 			}
// 		})
// 	}
// }

// func Test_decodeState_decodeVaryingDataTypeSlice(t *testing.T) {
// 	opt := cmp.Comparer(func(x, y VaryingDataType) bool {
// 		return reflect.DeepEqual(x.value, y.value) && reflect.DeepEqual(x.cache, y.cache)
// 	})

// 	for _, tt := range varyingDataTypeSliceTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			dst := tt.in.(VaryingDataTypeSlice)
// 			dst.Types = make([]VaryingDataType, 0)
// 			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
// 				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			vdts := tt.in.(VaryingDataTypeSlice)
// 			diff := cmp.Diff(dst, vdts, cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}), opt)
// 			if diff != "" {
// 				t.Errorf("decodeState.unmarshal() = %s", diff)
// 			}
// 		})
// 	}
// }

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

func (mvdt *MyVaryingDataType) SetValue(value VaryingDataTypeValue) (err error) {
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

func (mvdt MyVaryingDataType) IndexValue() (index uint, value VaryingDataTypeValue, err error) {
	switch mvdt.inner.(type) {
	case VDTValue:
		return 1, VaryingDataTypeValue(mvdt.inner), nil
	case VDTValue1:
		return 2, VaryingDataTypeValue(mvdt.inner), nil
	case VDTValue2:
		return 3, VaryingDataTypeValue(mvdt.inner), nil
	case VDTValue3:
		return 4, VaryingDataTypeValue(mvdt.inner), nil
	}
	return 0, nil, ErrUnsupportedVaryingDataTypeValue
}

func (mvdt MyVaryingDataType) Value() (value VaryingDataTypeValue, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt MyVaryingDataType) ValueAt(index uint) (value VaryingDataTypeValue, err error) {
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
func (mvdt *MyVaryingDataType) String() string {
	return ""
}

func TestVaryingDataType_Encode(t *testing.T) {
	vdtval1 := VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}
	mvdt := NewMyVaringDataType(vdtval1)
	_ = VaryingDataType(mvdt)
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

	bytes, err := Marshal(mvdt)
	if err != nil {
		t.Errorf("wtf %v", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, bytes)
}

func TestVaryingDataType_Decode(t *testing.T) {
	vdtval1 := VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}
	mvdt := NewMyVaringDataType[VDTValue1]()
	_ = VaryingDataType(mvdt)

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
	err := Unmarshal(bytes, mvdt)
	assert.NoError(t, err)
	value, err := mvdt.Value()
	assert.NoError(t, err)
	assert.Equal(t, vdtval1, value)
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

func TestVaryingDataType_EncodeSlice(t *testing.T) {
	vdtval1 := VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}
	mvdt := NewMyVaringDataType[VDTValue1](vdtval1)
	_ = VaryingDataType(mvdt)
	mvdtSlice := []VaryingDataType{
		mvdt,
	}
	expected := []byte{
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
	}

	bytes, err := Marshal(mvdtSlice)
	if err != nil {
		t.Errorf("wtf %v", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, bytes)
}

func TestVaryingDataType_DecodeSlice(t *testing.T) {
	vdtval1 := VDTValue1{O: newBigIntPtr(big.NewInt(1073741823))}
	mvdt := NewMyVaringDataType[VDTValue1](vdtval1)
	_ = VaryingDataType(mvdt)
	expected := []MyVaryingDataType{
		*mvdt,
	}
	var mvdtSlice []MyVaryingDataType

	bytes := []byte{
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
	}

	err := Unmarshal(bytes, &mvdtSlice)
	assert.NoError(t, err)
	assert.Equal(t, expected, mvdtSlice)
}
