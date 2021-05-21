package scale

import (
	"reflect"
	"testing"
)

func Test_decodeState_decodeFixedWidthInt(t *testing.T) {
	for _, tt := range fixedWidthIntegerTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.ValueOf(tt.in).Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeBigInt(t *testing.T) {
	for _, tt := range bigIntTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.ValueOf(tt.in).Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeBytes(t *testing.T) {
	for _, tt := range boolTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.ValueOf(tt.in).Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeBool(t *testing.T) {
	for _, tt := range boolTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.ValueOf(tt.in).Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

// func Test_decodeState_decodeBool(t *testing.T) {
// 	var b bool
// 	type args struct {
// 		data []byte
// 		dst  interface{}
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 		want    interface{}
// 	}{
// 		{
// 			args: args{
// 				data: []byte{0x01},
// 				dst:  &b,
// 			},
// 			want: true,
// 		},
// 		{
// 			args: args{
// 				data: []byte{0x00},
// 				dst:  &b,
// 			},
// 			want: false,
// 		},
// 		{
// 			name: "error case",
// 			args: args{
// 				data: []byte{0x03},
// 				dst:  &b,
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := Unmarshal(tt.args.data, tt.args.dst)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err != nil {
// 				return
// 			}
// 			got := reflect.ValueOf(tt.args.dst).Elem().Interface()
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("decodeState.unmarshal() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
