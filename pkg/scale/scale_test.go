package scale

import (
	"bytes"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func Test_encodeState_encodeFixedWidthInteger(t *testing.T) {
	type args struct {
		i interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "int(1)",
			args: args{
				i: int(1),
			},
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int(16383)",
			args: args{
				i: int(16383),
			},
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int(1073741823)",
			args: args{
				i: int(1073741823),
			},
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "int(9223372036854775807)",
			args: args{
				i: int(9223372036854775807),
			},
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
		{
			name: "uint(1)",
			args: args{
				i: int(1),
			},
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint(16383)",
			args: args{
				i: uint(16383),
			},
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint(1073741823)",
			args: args{
				i: uint(1073741823),
			},
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "uint(9223372036854775807)",
			args: args{
				i: uint(9223372036854775807),
			},
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
		{
			name: "int64(1)",
			args: args{
				i: int64(1),
			},
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int64(16383)",
			args: args{
				i: int64(16383),
			},
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "int64(1073741823)",
			args: args{
				i: int64(1073741823),
			},
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "int64(9223372036854775807)",
			args: args{
				i: int64(9223372036854775807),
			},
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
		{
			name: "uint64(1)",
			args: args{
				i: uint64(1),
			},
			want: []byte{0x01, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint64(16383)",
			args: args{
				i: uint64(16383),
			},
			want: []byte{0xff, 0x3f, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint64(1073741823)",
			args: args{
				i: uint64(1073741823),
			},
			want: []byte{0xff, 0xff, 0xff, 0x3f, 0, 0, 0, 0},
		},
		{
			name: "uint64(9223372036854775807)",
			args: args{
				i: uint64(9223372036854775807),
			},
			want: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
		{
			name: "int32(1)",
			args: args{
				i: int32(1),
			},
			want: []byte{0x01, 0, 0, 0},
		},
		{
			name: "int32(16383)",
			args: args{
				i: int32(16383),
			},
			want: []byte{0xff, 0x3f, 0, 0},
		},
		{
			name: "int32(1073741823)",
			args: args{
				i: int32(1073741823),
			},
			want: []byte{0xff, 0xff, 0xff, 0x3f},
		},
		{
			name: "uint32(1)",
			args: args{
				i: uint32(1),
			},
			want: []byte{0x01, 0, 0, 0},
		},
		{
			name: "uint32(16383)",
			args: args{
				i: uint32(16383),
			},
			want: []byte{0xff, 0x3f, 0, 0},
		},
		{
			name: "uint32(1073741823)",
			args: args{
				i: uint32(1073741823),
			},
			want: []byte{0xff, 0xff, 0xff, 0x3f},
		},
		{
			name: "int8(1)",
			args: args{
				i: int8(1),
			},
			want: []byte{0x01},
		},
		{
			name: "uint8(1)",
			args: args{
				i: uint8(1),
			},
			want: []byte{0x01},
		},
		{
			name: "int16(1)",
			args: args{
				i: int16(1),
			},
			want: []byte{0x01, 0},
		},
		{
			name: "int16(16383)",
			args: args{
				i: int16(16383),
			},
			want: []byte{0xff, 0x3f},
		},
		{
			name: "uint16(1)",
			args: args{
				i: uint16(1),
			},
			want: []byte{0x01, 0},
		},
		{
			name: "uint16(16383)",
			args: args{
				i: uint16(16383),
			},
			want: []byte{0xff, 0x3f},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeFixedWidthInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeFixedWidthInt() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeBigInteger(t *testing.T) {
	type args struct {
		i *big.Int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name:    "error nil pointer",
			wantErr: true,
		},
		{
			name: "big.NewInt(0)",
			args: args{
				i: big.NewInt(0),
			},
			want: []byte{0x00},
		},
		{
			name: "big.NewInt(1)",
			args: args{
				i: big.NewInt(1),
			},
			want: []byte{0x04},
		},
		{
			name: "big.NewInt(42)",
			args: args{
				i: big.NewInt(42),
			},
			want: []byte{0xa8},
		},
		{
			name: "big.NewInt(69)",
			args: args{
				i: big.NewInt(69),
			},
			want: []byte{0x15, 0x01},
		},
		{
			name: "big.NewInt(1000)",
			args: args{
				i: big.NewInt(1000),
			},
			want: []byte{0xa1, 0x0f},
		},
		{
			name: "big.NewInt(16383)",
			args: args{
				i: big.NewInt(16383),
			},
			want: []byte{0xfd, 0xff},
		},
		{
			name: "big.NewInt(16384)",
			args: args{
				i: big.NewInt(16384),
			},
			want: []byte{0x02, 0x00, 0x01, 0x00},
		},
		{
			name: "big.NewInt(1073741823)",
			args: args{
				i: big.NewInt(1073741823),
			},
			want: []byte{0xfe, 0xff, 0xff, 0xff},
		},
		{
			name: "big.NewInt(1073741824)",
			args: args{
				i: big.NewInt(1073741824),
			},
			want: []byte{3, 0, 0, 0, 64},
		},
		{
			name: "big.NewInt(1<<32 - 1)",
			args: args{
				i: big.NewInt(1<<32 - 1),
			},
			want: []byte{0x03, 0xff, 0xff, 0xff, 0xff},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeBigInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeBigInt() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeBytes(t *testing.T) {
	var byteArray = func(length int) []byte {
		b := make([]byte, length)
		for i := 0; i < length; i++ {
			b[i] = 0xff
		}
		return b
	}
	testString1 := "We love you! We believe in open source as wonderful form of giving."                           // n = 67
	testString2 := strings.Repeat("We need a longer string to test with. Let's multiple this several times.", 230) // n = 72 * 230 = 16560
	testString3 := "Let's test some special ASCII characters: ~  · © ÿ"                                           // n = 55 (UTF-8 encoding versus n = 51 with ASCII encoding)

	type args struct {
		b interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "[]byte{0x01}",
			args: args{
				b: []byte{0x01},
			},
			want: []byte{0x04, 0x01},
		},
		{
			name: "[]byte{0xff}",
			args: args{
				b: []byte{0xff},
			},
			want: []byte{0x04, 0xff},
		},
		{
			name: "[]byte{0x01, 0x01}",
			args: args{
				b: []byte{0x01, 0x01},
			},
			want: []byte{0x08, 0x01, 0x01},
		},
		{
			name: "byteArray(32)",
			args: args{
				b: byteArray(32),
			},
			want: append([]byte{0x80}, byteArray(32)...),
		},
		{
			name: "byteArray(64)",
			args: args{
				b: byteArray(64),
			},
			want: append([]byte{0x01, 0x01}, byteArray(64)...),
		},
		{
			name: "byteArray(16384)",
			args: args{
				b: byteArray(16384),
			},
			want: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...),
		},
		{
			name: "\"a\"",
			args: args{
				b: []byte("a"),
			},
			want: []byte{0x04, 'a'},
		},
		{
			name: "\"go-pre\"",
			args: args{
				b: []byte("go-pre"),
			},
			want: append([]byte{0x18}, string("go-pre")...),
		},
		{
			name: "testString1",
			args: args{
				b: testString1,
			},
			want: append([]byte{0x0D, 0x01}, testString1...),
		},
		{
			name: "testString2, long string",
			args: args{
				b: testString2,
			},
			want: append([]byte{0xC2, 0x02, 0x01, 0x00}, testString2...),
		},
		{
			name: "testString3, special chars",
			args: args{
				b: testString3,
			},
			want: append([]byte{0xDC}, testString3...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{}
			if err := es.marshal(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeBytes() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeBool(t *testing.T) {
	type fields struct {
		Buffer bytes.Buffer
	}
	type args struct {
		b bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "false",
			args: args{
				b: false,
			},
			want: []byte{0x00},
		},
		{
			name: "true",
			args: args{
				b: true,
			},
			want: []byte{0x01},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{
				Buffer: tt.fields.Buffer,
			}
			if err := es.marshal(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeBool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeBool() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeStruct(t *testing.T) {
	type myStruct struct {
		Foo []byte
		Bar int32
		Baz bool
	}
	var nilPtrMyStruct *myStruct
	// pointer to
	var ptrMystruct *myStruct = &myStruct{
		Foo: []byte{0x01},
		Bar: 2,
		Baz: true,
	}
	var nilPtrMyStruct2 *myStruct = &myStruct{
		Foo: []byte{0x01},
		Bar: 2,
		Baz: true,
	}
	nilPtrMyStruct2 = nil

	type args struct {
		t interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "nilPtrMyStruct",
			args: args{
				nilPtrMyStruct,
			},
			want: []byte{0},
		},
		{
			name: "ptrMystruct",
			args: args{
				ptrMystruct,
			},
			want: []byte{1, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "ptrMystruct cache hit",
			args: args{
				ptrMystruct,
			},
			want: []byte{1, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "nilPtrMyStruct2",
			args: args{
				nilPtrMyStruct2,
			},
			want: []byte{0},
		},
		{
			name: "&struct {[]byte, int32}",
			args: args{
				t: &struct {
					Foo []byte
					Bar int32
					Baz bool
				}{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},
			},
			want: []byte{1, 0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32}",
			args: args{
				t: struct {
					Foo []byte
					Bar int32
					Baz bool
				}{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32, bool}",
			args: args{
				t: struct {
					Baz bool   `scale:"3"`
					Bar int32  `scale:"2"`
					Foo []byte `scale:"1"`
				}{
					Foo: []byte{0x01},
					Bar: 2,
					Baz: true,
				},
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32, bool} with untagged attributes",
			args: args{
				t: struct {
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
			args: args{
				t: struct {
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
			args: args{
				t: struct {
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
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
		{
			name: "struct {[]byte, int32, bool} with ignored attributes",
			args: args{
				t: struct {
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
			},
			want: []byte{0x04, 0x01, 0x02, 0, 0, 0, 0x01},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{fieldScaleIndicesCache: cache}
			err := es.marshal(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeStruct() = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeStruct() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeSlice(t *testing.T) {
	type fields struct {
		Buffer bytes.Buffer
	}
	type args struct {
		t interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "[]int{1, 2, 3, 4}",
			args: args{
				[]int{1, 2, 3, 4},
			},
			want: []byte{0x10, 0x04, 0x08, 0x0c, 0x10},
		},
		{
			name: "[]int{16384, 2, 3, 4}",
			args: args{
				[]int{16384, 2, 3, 4},
			},
			want: []byte{0x10, 0x02, 0x00, 0x01, 0x00, 0x08, 0x0c, 0x10},
		},
		{
			name: "[]int{1073741824, 2, 3, 4}",
			args: args{
				[]int{1073741824, 2, 3, 4},
			},
			want: []byte{0x10, 0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
		},
		{
			name: "[]int{1 << 32, 2, 3, 1 << 32}",
			args: args{
				[]int{1 << 32, 2, 3, 1 << 32},
			},
			want: []byte{0x10, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x0c, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "[]bool{true, false, true}",
			args: args{
				[]bool{true, false, true},
			},
			want: []byte{0x0c, 0x01, 0x00, 0x01},
		},
		{
			name: "[][]int{{0, 1}, {1, 0}}",
			args: args{
				[][]int{{0, 1}, {1, 0}},
			},
			want: []byte{0x08, 0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
		},
		{
			name: "[]*big.Int{big.NewInt(0), big.NewInt(1)}",
			args: args{
				[]*big.Int{big.NewInt(0), big.NewInt(1)},
			},
			want: []byte{0x08, 0x00, 0x04},
		},
		{
			name: "[][]byte{{0x00, 0x01}, {0x01, 0x00}}",
			args: args{
				[][]byte{{0x00, 0x01}, {0x01, 0x00}},
			},
			want: []byte{0x08, 0x08, 0x00, 0x01, 0x08, 0x01, 0x00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{
				Buffer: tt.fields.Buffer,
			}
			err := es.marshal(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeStruct() = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeStruct() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}

func Test_encodeState_encodeArray(t *testing.T) {
	type fields struct {
		Buffer bytes.Buffer
	}
	type args struct {
		t interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "[4]int{1, 2, 3, 4}",
			args: args{
				[4]int{1, 2, 3, 4},
			},
			want: []byte{0x04, 0x08, 0x0c, 0x10},
		},
		{
			name: "[4]int{16384, 2, 3, 4}",
			args: args{
				[4]int{16384, 2, 3, 4},
			},
			want: []byte{0x02, 0x00, 0x01, 0x00, 0x08, 0x0c, 0x10},
		},
		{
			name: "[4]int{1073741824, 2, 3, 4}",
			args: args{
				[4]int{1073741824, 2, 3, 4},
			},
			want: []byte{0x03, 0x00, 0x00, 0x00, 0x40, 0x08, 0x0c, 0x10},
		},
		{
			name: "[4]int{1 << 32, 2, 3, 1 << 32}",
			args: args{
				[4]int{1 << 32, 2, 3, 1 << 32},
			},
			want: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x0c, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "[3]bool{true, false, true}",
			args: args{
				[3]bool{true, false, true},
			},
			want: []byte{0x01, 0x00, 0x01},
		},
		{
			name: "[2][]int{{0, 1}, {1, 0}}",
			args: args{
				[2][]int{{0, 1}, {1, 0}},
			},
			want: []byte{0x08, 0x00, 0x04, 0x08, 0x04, 0x00},
		},
		{
			name: "[2][2]int{{0, 1}, {1, 0}}",
			args: args{
				[2][2]int{{0, 1}, {1, 0}},
			},
			want: []byte{0x00, 0x04, 0x04, 0x00},
		},
		{
			name: "[2]*big.Int{big.NewInt(0), big.NewInt(1)}",
			args: args{
				[2]*big.Int{big.NewInt(0), big.NewInt(1)},
			},
			want: []byte{0x00, 0x04},
		},
		{
			name: "[2][]byte{{0x00, 0x01}, {0x01, 0x00}}",
			args: args{
				[2][]byte{{0x00, 0x01}, {0x01, 0x00}},
			},
			want: []byte{0x08, 0x00, 0x01, 0x08, 0x01, 0x00},
		},
		{
			name: "[2][2]byte{{0x00, 0x01}, {0x01, 0x00}}",
			args: args{
				[2][2]byte{{0x00, 0x01}, {0x01, 0x00}},
			},
			want: []byte{0x00, 0x01, 0x01, 0x00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &encodeState{
				Buffer: tt.fields.Buffer,
			}
			err := es.marshal(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeState.encodeStruct() = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(es.Buffer.Bytes(), tt.want) {
				t.Errorf("encodeState.encodeStruct() = %v, want %v", es.Buffer.Bytes(), tt.want)
			}
		})
	}
}
