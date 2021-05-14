package scale

import (
	"math/big"
	"reflect"
	"testing"
)

func Test_decodeState_decodeFixedWidthInt(t *testing.T) {
	var (
		i    int
		ui   uint
		i8   int8
		ui8  uint8
		i16  int16
		ui16 uint16
		i32  int32
		ui32 uint32
		i64  int64
		ui64 uint64
	)
	type args struct {
		data []byte
		dst  interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		// int8
		{
			args: args{
				data: []byte{0x00},
				dst:  &i8,
			},
			want: int8(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &i8,
			},
			want: int8(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &i8,
			},
			want: int8(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &i8,
			},
			want: int8(64),
		},
		// uint8
		{
			args: args{
				data: []byte{0x00},
				dst:  &ui8,
			},
			want: uint8(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &ui8,
			},
			want: uint8(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &ui8,
			},
			want: uint8(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &ui8,
			},
			want: uint8(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &ui8,
			},
			want: uint8(69),
		},
		// int
		{
			args: args{
				data: []byte{0x00},
				dst:  &i,
			},
			want: int(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &i,
			},
			want: int(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &i,
			},
			want: int(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &i,
			},
			want: int(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &i,
			},
			want: int(69),
		},
		// uint
		{
			args: args{
				data: []byte{0x00},
				dst:  &ui,
			},
			want: uint(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &ui,
			},
			want: uint(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &ui,
			},
			want: uint(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &ui,
			},
			want: uint(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &ui,
			},
			want: uint(69),
		},
		// int16
		{
			args: args{
				data: []byte{0x00},
				dst:  &i16,
			},
			want: int16(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &i16,
			},
			want: int16(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &i16,
			},
			want: int16(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &i16,
			},
			want: int16(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &i16,
			},
			want: int16(69),
		},
		{
			args: args{
				data: []byte{0xff, 0x3f},
				dst:  &i16,
			},
			want: int16(16383),
		},
		{
			args: args{
				data: []byte{0x00, 0x40},
				dst:  &i16,
			},
			want: int16(16384),
		},
		// uint16
		{
			args: args{
				data: []byte{0x00},
				dst:  &ui16,
			},
			want: uint16(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &ui16,
			},
			want: uint16(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &ui16,
			},
			want: uint16(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &ui16,
			},
			want: uint16(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &ui16,
			},
			want: uint16(69),
		},
		{
			args: args{
				data: []byte{0xff, 0x3f},
				dst:  &ui16,
			},
			want: uint16(16383),
		},
		{
			args: args{
				data: []byte{0x00, 0x40},
				dst:  &ui16,
			},
			want: uint16(16384),
		},
		// int32
		{
			args: args{
				data: []byte{0x00},
				dst:  &i32,
			},
			want: int32(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &i32,
			},
			want: int32(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &i32,
			},
			want: int32(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &i32,
			},
			want: int32(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &i32,
			},
			want: int32(69),
		},
		{
			args: args{
				data: []byte{0xff, 0x3f},
				dst:  &i32,
			},
			want: int32(16383),
		},
		{
			args: args{
				data: []byte{0x00, 0x40},
				dst:  &i32,
			},
			want: int32(16384),
		},
		{
			args: args{
				data: []byte{0xff, 0xff, 0xff, 0x3f},
				dst:  &i32,
			},
			want: int32(1073741823),
		},
		{
			args: args{
				data: []byte{0x00, 0x00, 0x00, 0x40},
				dst:  &i32,
			},
			want: int32(1073741824),
		},
		// uint32
		{
			args: args{
				data: []byte{0x00},
				dst:  &ui32,
			},
			want: uint32(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &ui32,
			},
			want: uint32(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &ui32,
			},
			want: uint32(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &ui32,
			},
			want: uint32(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &ui32,
			},
			want: uint32(69),
		},
		{
			args: args{
				data: []byte{0xff, 0x3f},
				dst:  &ui32,
			},
			want: uint32(16383),
		},
		{
			args: args{
				data: []byte{0x00, 0x40},
				dst:  &ui32,
			},
			want: uint32(16384),
		},
		{
			args: args{
				data: []byte{0xff, 0xff, 0xff, 0x3f},
				dst:  &ui32,
			},
			want: uint32(1073741823),
		},
		{
			args: args{
				data: []byte{0x00, 0x00, 0x00, 0x40},
				dst:  &ui32,
			},
			want: uint32(1073741824),
		},
		// int64
		{
			args: args{
				data: []byte{0x00},
				dst:  &i64,
			},
			want: int64(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &i64,
			},
			want: int64(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &i64,
			},
			want: int64(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &i64,
			},
			want: int64(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &i64,
			},
			want: int64(69),
		},
		{
			args: args{
				data: []byte{0xff, 0x3f},
				dst:  &i64,
			},
			want: int64(16383),
		},
		{
			args: args{
				data: []byte{0x00, 0x40},
				dst:  &i64,
			},
			want: int64(16384),
		},
		{
			args: args{
				data: []byte{0xff, 0xff, 0xff, 0x3f},
				dst:  &i64,
			},
			want: int64(1073741823),
		},
		{
			args: args{
				data: []byte{0x00, 0x00, 0x00, 0x40},
				dst:  &i64,
			},
			want: int64(1073741824),
		},
		{
			args: args{
				data: []byte{0x03, 0x00, 0x00, 0x00, 0x40},
				dst:  &i64,
			},
			want: int64(274877906947),
		},
		{
			args: args{
				data: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				dst:  &i64,
			},
			want: int64(-1),
		},
		// uint64
		{
			args: args{
				data: []byte{0x00},
				dst:  &ui64,
			},
			want: uint64(0),
		},
		{
			args: args{
				data: []byte{0x01},
				dst:  &ui64,
			},
			want: uint64(1),
		},
		{
			args: args{
				data: []byte{0x2a},
				dst:  &ui64,
			},
			want: uint64(42),
		},
		{
			args: args{
				data: []byte{0x40},
				dst:  &ui64,
			},
			want: uint64(64),
		},
		{
			args: args{
				data: []byte{0x45},
				dst:  &ui64,
			},
			want: uint64(69),
		},
		{
			args: args{
				data: []byte{0xff, 0x3f},
				dst:  &ui64,
			},
			want: uint64(16383),
		},
		{
			args: args{
				data: []byte{0x00, 0x40},
				dst:  &ui64,
			},
			want: uint64(16384),
		},
		{
			args: args{
				data: []byte{0xff, 0xff, 0xff, 0x3f},
				dst:  &ui64,
			},
			want: uint64(1073741823),
		},
		{
			args: args{
				data: []byte{0x00, 0x00, 0x00, 0x40},
				dst:  &ui64,
			},
			want: uint64(1073741824),
		},
		{
			args: args{
				data: []byte{0x03, 0x00, 0x00, 0x00, 0x40},
				dst:  &ui64,
			},
			want: uint64(274877906947),
		},
		{
			args: args{
				data: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				dst:  &ui64,
			},
			want: uint64(1<<64 - 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Unmarshal(tt.args.data, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := reflect.ValueOf(tt.args.dst).Elem().Interface()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeState_decodeBigInt(t *testing.T) {
	var (
		bi *big.Int
	)
	type args struct {
		data []byte
		dst  interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name: "error case, ensure **big.Int",
			args: args{
				data: []byte{0x00},
				dst:  bi,
			},
			wantErr: true,
		},
		{
			args: args{
				data: []byte{0x00},
				dst:  &bi,
			},
			want: big.NewInt(0),
		},
		{
			args: args{
				data: []byte{0x04},
				dst:  &bi,
			},
			want: big.NewInt(1),
		},
		{
			args: args{
				data: []byte{0xa8},
				dst:  &bi,
			},
			want: big.NewInt(42),
		},
		{
			args: args{
				data: []byte{0x01, 0x01},
				dst:  &bi,
			},
			want: big.NewInt(64),
		},
		{
			args: args{
				data: []byte{0x15, 0x01},
				dst:  &bi,
			},
			want: big.NewInt(69),
		},
		{
			args: args{
				data: []byte{0xfd, 0xff},
				dst:  &bi,
			},
			want: big.NewInt(16383),
		},
		{
			args: args{
				data: []byte{0x02, 0x00, 0x01, 0x00},
				dst:  &bi,
			},
			want: big.NewInt(16384),
		},
		{
			args: args{
				data: []byte{0xfe, 0xff, 0xff, 0xff},
				dst:  &bi,
			},
			want: big.NewInt(1073741823),
		},
		{
			args: args{
				data: []byte{0x03, 0x00, 0x00, 0x00, 0x40},
				dst:  &bi,
			},
			want: big.NewInt(1073741824),
		},
		{
			args: args{
				data: []byte{0x03, 0xff, 0xff, 0xff, 0xff},
				dst:  &bi,
			},
			want: big.NewInt(1<<32 - 1),
		},
		{
			args: args{
				data: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
				dst:  &bi,
			},
			want: big.NewInt(1 << 32),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(tt.args.data, tt.args.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			got := reflect.ValueOf(tt.args.dst).Elem().Interface()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeState_decodeBytes(t *testing.T) {
	var b []byte
	var s string
	type args struct {
		data []byte
		dst  interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			args: args{
				data: []byte{0x04, 0x01},
				dst:  &b,
			},
			want: []byte{0x01},
		},
		{
			args: args{
				data: []byte{0x04, 0xff},
				dst:  &b,
			},
			want: []byte{0xff},
		},
		{
			args: args{
				data: []byte{0x08, 0x01, 0x01},
				dst:  &b,
			},
			want: []byte{0x01, 0x01},
		},
		{
			args: args{
				data: append([]byte{0x01, 0x01}, byteArray(64)...),
				dst:  &b,
			},
			want: byteArray(64),
		},
		{
			args: args{
				data: append([]byte{0xfd, 0xff}, byteArray(16383)...),
				dst:  &b,
			},
			want: byteArray(16383),
		},
		{
			args: args{
				data: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...),
				dst:  &b,
			},
			want: byteArray(16384),
		},
		// string
		{
			args: args{
				data: []byte{0x04, 0x01},
				dst:  &s,
			},
			want: string([]byte{0x01}),
		},
		{
			args: args{
				data: []byte{0x04, 0xff},
				dst:  &s,
			},
			want: string([]byte{0xff}),
		},
		{
			args: args{
				data: []byte{0x08, 0x01, 0x01},
				dst:  &s,
			},
			want: string([]byte{0x01, 0x01}),
		},
		{
			args: args{
				data: append([]byte{0x01, 0x01}, byteArray(64)...),
				dst:  &s,
			},
			want: string(byteArray(64)),
		},
		{
			args: args{
				data: append([]byte{0xfd, 0xff}, byteArray(16383)...),
				dst:  &s,
			},
			want: string(byteArray(16383)),
		},
		{
			args: args{
				data: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...),
				dst:  &s,
			},
			want: string(byteArray(16384)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(tt.args.data, tt.args.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			got := reflect.ValueOf(tt.args.dst).Elem().Interface()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeState_decodeBool(t *testing.T) {
	var b bool
	type args struct {
		data []byte
		dst  interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			args: args{
				data: []byte{0x01},
				dst:  &b,
			},
			want: true,
		},
		{
			args: args{
				data: []byte{0x00},
				dst:  &b,
			},
			want: false,
		},
		{
			name: "error case",
			args: args{
				data: []byte{0x03},
				dst:  &b,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(tt.args.data, tt.args.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			got := reflect.ValueOf(tt.args.dst).Elem().Interface()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}
