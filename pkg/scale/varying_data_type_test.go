package scale

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

type testVDT VaryingDataType

func init() {
	err := RegisterVaryingDataType(testVDT{}, VDTValue{}, VDTValue2{}, VDTValue1{}, VDTValue3(0))
	if err != nil {
		panic(err)
	}
}

var varyingDataTypeTests = tests{
	{
		in: testVDT{
			VDTValue1{
				O: newBigIntPtr(big.NewInt(1073741823)),
			},
		},
		want: []byte{
			4, 2,
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
		in: testVDT{
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
			VDTValue3(16383),
		},
		want: newWant(
			// length encoding of 3
			[]byte{16},
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
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
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
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			diff := cmp.Diff(dst, tt.in, cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}))
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
			}
		})
	}
}
