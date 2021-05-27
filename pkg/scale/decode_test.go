package scale

import (
	"math/big"
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
			var dst *big.Int
			// dst := reflect.ValueOf(tt.in).Interface()
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
	for _, tt := range stringTests {
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
			// dst := reflect.ValueOf(tt.in).Interface()
			var dst bool
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeStructManual(t *testing.T) {
	// nil case
	var dst *MyStruct = nil
	var b = []byte{0}
	var want *MyStruct = nil

	// dst = structTests[0].dst

	err := Unmarshal(b, &dst)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(dst, want) {
		t.Errorf("decodeState.unmarshal() = %v, want %v", dst, want)
	}

	// zero case MyStruct
	var dst1 *MyStruct = &MyStruct{}
	err = Unmarshal(b, &dst1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(dst1, want) {
		t.Errorf("decodeState.unmarshal() = %v, want %v", dst, want)
	}

	// zero case MyStruct
	var dst2 *MyStruct = &MyStruct{Baz: true}
	err = Unmarshal(b, &dst2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(dst2, want) {
		t.Errorf("decodeState.unmarshal() = %v, want %v", dst, want)
	}

}

func Test_decodeState_decodeStruct(t *testing.T) {
	for _, tt := range append([]test{}, structTests[0]) {
		t.Run(tt.name, func(t *testing.T) {
			// dst := reflect.ValueOf(tt.in).Interface()
			dst := reflect.New(reflect.Indirect(reflect.ValueOf(tt.in)).Type()).Elem().Interface()
			// dst := tt.dst
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			// dstv := reflect.ValueOf(dst)
			// inv := reflect.ValueOf(tt.in)

			// if dstv.Kind() != inv.Kind() {
			// 	t.Errorf("decodeState.unmarshal() differing kind = %T, want %T", dst, tt.in)
			// 	return
			// }

			// switch inv.Kind() {
			// case reflect.Ptr:
			// 	fmt.Println(dst, dstv.Interface(), tt.in, inv.Interface())
			// 	if inv.IsZero() {
			// 		fmt.Println(dst, tt.in, dstv.Interface(), inv.Interface())
			// 		if !reflect.DeepEqual(dstv.Interface(), tt.in) {
			// 			t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			// 		}
			// 	} else if inv.IsNil() {
			// 		if !reflect.DeepEqual(dst, tt.in) {
			// 			t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			// 		}
			// 	} else {
			// 		// // have to do this since reflect.DeepEqual won't come back true for different addresses
			// 		// if !reflect.DeepEqual(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.in).Elem().Interface()) {
			// 		// 	t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			// 		// }
			// 		if !reflect.DeepEqual(dst, inv) {
			// 			t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			// 		}
			// 	}
			// default:
			// 	if !reflect.DeepEqual(dst, tt.in) {
			// 		t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			// 	}
			// }
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}
func Test_decodeState_decodeArray(t *testing.T) {
	for _, tt := range arrayTests {
		t.Run(tt.name, func(t *testing.T) {
			// dst := reflect.ValueOf(tt.in).Interface()
			dst := reflect.New(reflect.Indirect(reflect.ValueOf(tt.in)).Type()).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeSlice(t *testing.T) {
	for _, tt := range sliceTests {
		t.Run(tt.name, func(t *testing.T) {
			// dst := reflect.ValueOf(tt.in).Interface()
			dst := reflect.New(reflect.Indirect(reflect.ValueOf(tt.in)).Type()).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}
