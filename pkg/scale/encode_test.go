package scale

import (
	"math/big"
	"reflect"
	"strings"
	"testing"
)

type test struct {
	name    string
	in      interface{}
	wantErr bool
	want    []byte
}
type tests []test

func newTests(ts ...tests) (appended tests) {
	for _, t := range ts {
		appended = append(appended, t...)
	}
	return
}

var (
	intTests = tests{
		{
			name: "int(1)",
			in:   int(1),
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int(16383)",
			in:   int(16383),
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int(1073741823)",
			in:   int(1073741823),
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "int(9223372036854775807)",
			in:   int(9223372036854775807),
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
	}
	uintTests = tests{
		{
			name: "uint(1)",
			in:   int(1),
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint(16383)",
			in:   uint(16383),
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint(1073741823)",
			in:   uint(1073741823),
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "uint(9223372036854775807)",
			in:   uint(9223372036854775807),
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
	}
	int64Tests = tests{
		{
			name: "int64(1)",
			in:   int64(1),
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int64(16383)",
			in:   int64(16383),
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int64(1073741823)",
			in:   int64(1073741823),
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "int64(9223372036854775807)",
			in:   int64(9223372036854775807),
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
	}
	uint64Tests = tests{
		{
			name: "uint64(1)",
			in:   uint64(1),
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint64(16383)",
			in:   uint64(16383),
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint64(1073741823)",
			in:   uint64(1073741823),
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "uint64(9223372036854775807)",
			in:   uint64(9223372036854775807),
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
	}
	int32Tests = tests{
		{
			name: "int32(1)",
			in:   int32(1),
			want: []byte{0x01, 0, 0, 0},
		},
		{
			name: "int32(16383)",
			in:   int32(16383),
			want: []byte{0xff, 0x3f, 0, 0},
		},
		{
			name: "int32(1073741823)",
			in:   int32(1073741823),
			want: []byte{0xff, 0xff, 0xff, 0x3f},
		},
	}
	uint32Tests = tests{
		{
			name: "uint32(1)",
			in:   uint32(1),
			want: []byte{0x01, 0, 0, 0},
		},
		{
			name: "uint32(16383)",
			in:   uint32(16383),
			want: []byte{0xff, 0x3f, 0, 0},
		},
		{
			name: "uint32(1073741823)",
			in:   uint32(1073741823),
			want: []byte{0xff, 0xff, 0xff, 0x3f},
		},
	}
	int8Tests = tests{
		{
			name: "int8(1)",
			in:   int8(1),
			want: []byte{0x01},
		},
	}
	uint8Tests = tests{
		{
			name: "uint8(1)",
			in:   uint8(1),
			want: []byte{0x01},
		},
	}
	int16Tests = tests{
		{
			name: "int16(1)",
			in:   int16(1),
			want: []byte{0x01, 0},
		},
		{
			name: "int16(16383)",
			in:   int16(16383),
			want: []byte{0xff, 0x3f},
		},
	}
	uint16Tests = tests{
		{
			name: "uint16(1)",
			in:   uint16(1),
			want: []byte{0x01, 0},
		},
		{
			name: "uint16(16383)",
			in:   uint16(16383),
			want: []byte{0xff, 0x3f},
		},
	}
	fixedWidthIntegerTests = newTests(
		int8Tests, int16Tests, int32Tests, int64Tests, intTests, uint8Tests, uint16Tests, uint32Tests, uint64Tests, uintTests,
	)

	zeroValBigInt *big.Int
	bigIntTests   = tests{
		{
			name:    "error nil pointer",
			in:      zeroValBigInt,
			wantErr: true,
		},
		{
			name: "big.NewInt(0)",
			in:   big.NewInt(0),
			want: []byte{0x00},
		},
		{
			name: "big.NewInt(1)",
			in:   big.NewInt(1),
			want: []byte{0x04},
		},
		{
			name: "big.NewInt(42)",
			in:   big.NewInt(42),
			want: []byte{0xa8},
		},
		{
			name: "big.NewInt(69)",
			in:   big.NewInt(69),
			want: []byte{0x15, 0x01},
		},
		{
			name: "big.NewInt(1000)",
			in:   big.NewInt(1000),
			want: []byte{0xa1, 0x0f},
		},
		{
			name: "big.NewInt(16383)",
			in:   big.NewInt(16383),
			want: []byte{0xfd, 0xff},
		},
		{
			name: "big.NewInt(16384)",
			in:   big.NewInt(16384),
			want: []byte{0x02, 0x00, 0x01, 0x00},
		},
		{
			name: "big.NewInt(1073741823)",
			in:   big.NewInt(1073741823),
			want: []byte{0xfe, 0xff, 0xff, 0xff},
		},
		{
			name: "big.NewInt(1073741824)",
			in:   big.NewInt(1073741824),
			want: []byte{3, 0, 0, 0, 64},
		},
		{
			name: "big.NewInt(1<<32 - 1)",
			in:   big.NewInt(1<<32 - 1),
			want: []byte{0x03, 0xff, 0xff, 0xff, 0xff},
		},
	}

	testStrings = []string{
		"We love you! We believe in open source as wonderful form of giving.",                           // n = 67
		strings.Repeat("We need a longer string to test with. Let's multiple this several times.", 230), // n = 72 * 230 = 16560
		"Let's test some special ASCII characters: ~  · © ÿ",                                           // n = 55 (UTF-8 encoding versus n = 51 with ASCII encoding)
	}
	stringTests = tests{
		{
			name: "[]byte{0x01}",
			in:   []byte{0x01},

			want: []byte{0x04, 0x01},
		},
		{
			name: "[]byte{0xff}",
			in:   []byte{0xff},

			want: []byte{0x04, 0xff},
		},
		{
			name: "[]byte{0x01, 0x01}",
			in:   []byte{0x01, 0x01},

			want: []byte{0x08, 0x01, 0x01},
		},
		{
			name: "byteArray(32)",
			in:   byteArray(32),

			want: append([]byte{0x80}, byteArray(32)...),
		},
		{
			name: "byteArray(64)",
			in:   byteArray(64),

			want: append([]byte{0x01, 0x01}, byteArray(64)...),
		},
		{
			name: "byteArray(16384)",
			in:   byteArray(16384),

			want: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...),
		},
		{
			name: "\"a\"",
			in:   []byte("a"),

			want: []byte{0x04, 'a'},
		},
		{
			name: "\"go-pre\"",
			in:   []byte("go-pre"),

			want: append([]byte{0x18}, string("go-pre")...),
		},
		{
			name: "testStrings[0]",
			in:   testStrings[0],

			want: append([]byte{0x0D, 0x01}, testStrings[0]...),
		},
		{
			name: "testString[1], long string",
			in:   testStrings[1],

			want: append([]byte{0xC2, 0x02, 0x01, 0x00}, testStrings[1]...),
		},
		{
			name: "testString[2], special chars",
			in:   testStrings[2],

			want: append([]byte{0xDC}, testStrings[2]...),
		},
	}

	boolTests = tests{
		{
			name: "false",
			in:   false,
			want: []byte{0x00},
		},
		{
			name: "true",
			in:   true,
			want: []byte{0x01},
		},
	}

	nilPtrMyStruct *MyStruct
	ptrMystruct    *MyStruct = &MyStruct{
		Foo: []byte{0x01},
		Bar: 2,
		Baz: true,
	}
	nilPtrMyStruct2 *MyStruct = nil
	structTests               = tests{
		// {
		// 	name: "nilPtrMyStruct",
		// 	in:   nilPtrMyStruct2,
		// 	want: []byte{0},
		// 	dst:  &MyStruct{Baz: true},
		// },
		// {
		// 	name: "ptrMystruct",
		// 	in:   ptrMystruct,
		// 	want: []byte{1, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		// 	dst:  &MyStruct{},
		// },
		// {
		// 	name: "ptrMystruct cache hit",
		// 	in:   ptrMystruct,
		// 	want: []byte{1, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		// 	dst:  &MyStruct{},
		// },
		// {
		// 	name: "nilPtrMyStruct2",
		// 	in:   nilPtrMyStruct2,
		// 	want: []byte{0},
		// 	dst:  new(MyStruct),
		// },
		// {
		// 	name: "&struct {[]byte, int32}",
		// 	in: &MyStruct{
		// 		Foo: []byte{0x01},
		// 		Bar: 2,
		// 		Baz: true,
		// 	},
		// 	want: []byte{1, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		// 	dst:  &MyStruct{},
		// },
		{
			name: "struct {[]byte, int32}",
			in: MyStruct{
				Foo: []byte{0x01},
				Bar: 2,
				Baz: true,
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32, bool}",
			in: struct {
				Baz bool   `scale:"3,enum"`
				Bar int32  `scale:"2"`
				Foo []byte `scale:"1"`
			}{
				Foo: []byte{0x01},
				Bar: 2,
				Baz: true,
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32, bool} with untagged attributes",
			in: struct {
				Baz  bool   `scale:"3"`
				Bar  int32  `scale:"2"`
				Foo  []byte `scale:"1"`
				End1 bool
				End2 []byte
				End3 []byte
			}{
				Foo:  []byte{0x01},
				Bar:  2,
				Baz:  true,
				End1: false,
				End2: []byte{0xff},
				End3: []byte{0x06},
			},
			want: []byte{
				0x04, 0x01, 0x02, 0, 0, 0, 0x01,
				// End1: false
				0x00,
				// End2: 0xff
				0x04, 0xff,
				// End3: 0x06
				0x04, 0x06,
			},
		},
		{
			name: "struct {[]byte, int32, bool} with untagged attributes",
			in: struct {
				End1 bool
				Baz  bool `scale:"3"`
				End2 []byte
				Bar  int32 `scale:"2"`
				End3 []byte
				Foo  []byte `scale:"1"`
			}{
				Foo:  []byte{0x01},
				Bar:  2,
				Baz:  true,
				End1: false,
				End2: []byte{0xff},
				// End3: 0xff
				End3: []byte{0x06},
			},

			want: []byte{
				0x04, 0x01, 0x02, 0, 0, 0, 0x01,
				// End1: false
				0x00,
				// End2: 0xff
				0x04, 0xff,
				// End3: 0x06
				0x04, 0x06,
			},
		},
		{
			name: "struct {[]byte, int32, bool} with private attributes",
			in: struct {
				priv0 string
				Baz   bool   `scale:"3"`
				Bar   int32  `scale:"2"`
				Foo   []byte `scale:"1"`
				priv1 []byte
			}{
				priv0: "stuff",
				Foo:   []byte{0x01},
				Bar:   2,
				Baz:   true,
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32, bool} with ignored attributes",
			in: struct {
				Baz           bool   `scale:"3"`
				Bar           int32  `scale:"2"`
				Foo           []byte `scale:"1"`
				Ignore        string `scale:"-"`
				somethingElse *struct {
					fields int
				}
			}{
				Foo:    []byte{0x01},
				Bar:    2,
				Baz:    true,
				Ignore: "me",
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
	}

	sliceTests = tests{
		{
			name: "[]int{1, 2, 3, 4}",
			in:   []int{1, 2, 3, 4},
			want: []byte{0x10, 0x04, 0x08, 0x0c, 0x10},
		},
		{
			name: "[]int{16384, 2, 3, 4}",
			in:   []int{16384, 2, 3, 4},
			want: []byte{0x10, 0x02, 0x00, 0x01, 0x00, 0x08, 0x0c, 0x10},
		},
		{
			name: "[]int{1073741824, 2, 3, 4}",
			in:   []int{1073741824, 2, 3, 4},
			want: []byte{0x10, 0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
		},
		{
			name: "[]int{1 << 32, 2, 3, 1 << 32}",
			in:   []int{1 << 32, 2, 3, 1 << 32},
			want: []byte{0x10, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x0c, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "[]bool{true, false, true}",
			in:   []bool{true, false, true},
			want: []byte{0x0c, 0x01, 0x00, 0x01},
		},
		{
			name: "[][]int{{0, 1}, {1, 0}}",
			in:   [][]int{{0, 1}, {1, 0}},
			want: []byte{0x08, 0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
		},
		{
			name: "[]*big.Int{big.NewInt(0), big.NewInt(1)}",
			in:   []*big.Int{big.NewInt(0), big.NewInt(1)},
			want: []byte{0x08, 0x00, 0x04},
		},
		{
			name: "[][]byte{{0x00, 0x01}, {0x01, 0x00}}",
			in:   [][]byte{{0x00, 0x01}, {0x01, 0x00}},
			want: []byte{0x08, 0x08, 0x00, 0x01, 0x08, 0x01, 0x00},
		},
	}

	arrayTests = tests{
		{
			name: "[4]int{1, 2, 3, 4}",
			in:   [4]int{1, 2, 3, 4},
			want: []byte{0x04, 0x08, 0x0c, 0x10},
		},
		{
			name: "[4]int{16384, 2, 3, 4}",
			in:   [4]int{16384, 2, 3, 4},
			want: []byte{0x02, 0x00, 0x01, 0x00, 0x08, 0x0c, 0x10},
		},
		{
			name: "[4]int{1073741824, 2, 3, 4}",
			in:   [4]int{1073741824, 2, 3, 4},
			want: []byte{0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
		},
		{
			name: "[4]int{1 << 32, 2, 3, 1 << 32}",
			in:   [4]int{1 << 32, 2, 3, 1 << 32},
			want: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x0c, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "[3]bool{true, false, true}",
			in:   [3]bool{true, false, true},
			want: []byte{0x01, 0x00, 0x01},
		},
		{
			name: "[2][]int{{0, 1}, {1, 0}}",
			in:   [2][]int{{0, 1}, {1, 0}},
			want: []byte{0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
		},
		{
			name: "[2][2]int{{0, 1}, {1, 0}}",
			in:   [2][2]int{{0, 1}, {1, 0}},
			want: []byte{0x00, 0x04, 0x04, 0x00},
		},
		{
			name: "[2]*big.Int{big.NewInt(0), big.NewInt(1)}",
			in:   [2]*big.Int{big.NewInt(0), big.NewInt(1)},
			want: []byte{0x00, 0x04},
		},
		{
			name: "[2][]byte{{0x00, 0x01}, {0x01, 0x00}}",
			in:   [2][]byte{{0x00, 0x01}, {0x01, 0x00}},
			want: []byte{0x08, 0x00, 0x01, 0x08, 0x01, 0x00},
		},
		{
			name: "[2][2]byte{{0x00, 0x01}, {0x01, 0x00}}",
			in:   [2][2]byte{{0x00, 0x01}, {0x01, 0x00}},
			want: []byte{0x00, 0x01, 0x01, 0x00},
		},
	}

	allTests = newTests(
		fixedWidthIntegerTests, bigIntTests, stringTests,
		boolTests, structTests, sliceTests, arrayTests,
	)
)

type MyStruct struct {
	Foo []byte
	Bar int32
	Baz bool
}

func Test_encodeState_encodeFixedWidthInteger(t *testing.T) {
	for _, tt := range fixedWidthIntegerTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeFixedWidthInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeFixedWidthInt() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeBigInt(t *testing.T) {
	for _, tt := range bigIntTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeBigInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeBigInt() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeBytes(t *testing.T) {
	for _, tt := range stringTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeBytes() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeBool(t *testing.T) {
	for _, tt := range boolTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeBool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeBool() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeStruct(t *testing.T) {
	for _, tt := range structTests {
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

func Test_encodeState_encodeSlice(t *testing.T) {
	for _, tt := range sliceTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{fieldScaleIndicesCache: cache}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeSlice() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeSlice() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeArray(t *testing.T) {
	for _, tt := range arrayTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{fieldScaleIndicesCache: cache}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeArray() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeArray() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_marshal_optionality(t *testing.T) {
	var ptrTests tests
	for i := range allTests {
		t := allTests[i]
		ptrTest := test{
			name:    t.name,
			in:      &t.in,
			wantErr: t.wantErr,
			want:    t.want,
		}
		switch t.in {
		case nil:
			ptrTest.want = []byte{0x00}
		default:
			ptrTest.want = append([]byte{0x01}, t.want...)
		}
		ptrTests = append(ptrTests, ptrTest)
	}
	for _, tt := range ptrTests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{fieldScaleIndicesCache: cache}
			if err := es.marshal(tt.in); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeFixedWidthInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeFixedWidthInt() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

var byteArray = func(length int) []byte {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = 0xff
	}
	return b
}
