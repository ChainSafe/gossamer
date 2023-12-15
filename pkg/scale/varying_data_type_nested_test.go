// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

// import (
// 	"fmt"
// 	"math/big"
// 	"testing"

// 	"github.com/google/go-cmp/cmp"
// 	"github.com/google/go-cmp/cmp/cmpopts"
// )

// type parentVDT VaryingDataType

// func (pvdt *parentVDT) Set(val VaryingDataTypeValue) (err error) {
// 	vdt := VaryingDataType(*pvdt)
// 	err = vdt.Set(val)
// 	if err != nil {
// 		return
// 	}
// 	*pvdt = parentVDT(vdt)
// 	return
// }

// func mustNewParentVDT() parentVDT {
// 	vdt, err := NewVaryingDataType(mustNewChildVDT(), mustNewChildVDT1())
// 	if err != nil {
// 		panic(err)
// 	}
// 	return parentVDT(vdt)
// }

// type childVDT VaryingDataType

// func (childVDT) Index() uint {
// 	return 1
// }

// func (c childVDT) String() string {
// 	if c.value == nil {
// 		return "childVDT(nil)"
// 	}
// 	return fmt.Sprintf("childVDT(%s)", c.value)
// }

// func mustNewChildVDT() childVDT {
// 	vdt, err := NewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))
// 	if err != nil {
// 		panic(err)
// 	}
// 	return childVDT(vdt)
// }

// func mustNewChildVDTAndSet(vdtv VaryingDataTypeValue) childVDT {
// 	vdt, err := NewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{}, VDTValue3(0))
// 	if err != nil {
// 		panic(err)
// 	}
// 	err = vdt.Set(vdtv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return childVDT(vdt)
// }

// type childVDT1 VaryingDataType

// func (childVDT1) Index() uint {
// 	return 2
// }

// func (c childVDT1) String() string {
// 	if c.value == nil {
// 		return "childVDT1(nil)"
// 	}
// 	return fmt.Sprintf("childVDT1(%s)", c.value)
// }

// func mustNewChildVDT1() childVDT1 {
// 	vdt, err := NewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	return childVDT1(vdt)
// }

// func mustNewChildVDT1AndSet(vdtv VaryingDataTypeValue) childVDT1 {
// 	vdt, err := NewVaryingDataType(VDTValue{}, VDTValue1{}, VDTValue2{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	err = vdt.Set(vdtv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return childVDT1(vdt)
// }

// type constructorTest struct {
// 	name    string
// 	newIn   func(t *testing.T) interface{}
// 	want    []byte
// 	wantErr bool
// }

// var nestedVaryingDataTypeTests = []constructorTest{
// 	{
// 		name: "ParentVDT_with_ChildVDT",
// 		newIn: func(t *testing.T) interface{} {
// 			pvdt := mustNewParentVDT()
// 			err := pvdt.Set(mustNewChildVDTAndSet(VDTValue3(16383)))
// 			if err != nil {
// 				t.Fatalf("%v", err)
// 			}
// 			return pvdt
// 		},
// 		want: newWant(
// 			// index of childVDT
// 			[]byte{1},
// 			// index of VDTValue3
// 			[]byte{4},
// 			// encoding of int16
// 			[]byte{0xff, 0x3f},
// 		),
// 	},
// 	{
// 		name: "ParentVDT_with_ChildVDT1",
// 		newIn: func(t *testing.T) interface{} {
// 			pvdt := mustNewParentVDT()
// 			err := pvdt.Set(mustNewChildVDT1AndSet(
// 				VDTValue{
// 					A: big.NewInt(1073741823),
// 					B: int(1073741823),
// 					C: uint(1073741823),
// 					D: int8(1),
// 					E: uint8(1),
// 					F: int16(16383),
// 					G: uint16(16383),
// 					H: int32(1073741823),
// 					I: uint32(1073741823),
// 					J: int64(9223372036854775807),
// 					K: uint64(9223372036854775807),
// 					L: byteArray(64),
// 					M: testStrings[1],
// 					N: true,
// 				},
// 			))
// 			if err != nil {
// 				t.Fatalf("%v", err)
// 			}
// 			return pvdt
// 		},
// 		want: newWant(
// 			// index of childVDT1
// 			[]byte{2},
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
// }

// func Test_encodeState_encodeCustomVaryingDataType_nested(t *testing.T) {
// 	for _, tt := range nestedVaryingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			b, err := Marshal(tt.newIn(t))
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if diff := cmp.Diff(b, tt.want); diff != "" {
// 				t.Errorf("Marshal() diff: %s", diff)
// 			}
// 		})
// 	}
// }

// func Test_decodeState_decodeCustomVaryingDataType_nested(t *testing.T) {
// 	for _, tt := range nestedVaryingDataTypeTests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			dst := mustNewParentVDT()
// 			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
// 				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			expected := tt.newIn(t)

// 			diff := cmp.Diff(dst, expected,
// 				cmp.AllowUnexported(parentVDT{}, childVDT{}, childVDT1{}),
// 				cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}),
// 			)
// 			if diff != "" {
// 				t.Errorf("decodeState.unmarshal() = %s", diff)
// 			}
// 		})
// 	}
// }
