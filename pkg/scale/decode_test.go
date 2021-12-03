// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test_decodeState_decodeFixedWidthInt(t *testing.T) {
	for _, tt := range fixedWidthIntegerTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeVariableWidthInt(t *testing.T) {
	for _, tt := range variableWidthIntegerTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
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
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
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
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_decodeState_decodeStruct(t *testing.T) {
	for _, tt := range structTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			var diff string
			if tt.out != nil {
				diff = cmp.Diff(dst, tt.out, cmpopts.IgnoreUnexported(tt.in))
			} else {
				diff = cmp.Diff(dst, tt.in, cmpopts.IgnoreUnexported(big.Int{}, tt.in, VDTValue2{}, MyStructWithIgnore{}))
			}
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
			}
		})
	}
}
func Test_decodeState_decodeArray(t *testing.T) {
	for _, tt := range arrayTests {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
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
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_unmarshal_optionality(t *testing.T) {
	var ptrTests tests
	for _, t := range append(tests{}, allTests...) {
		ptrTest := test{
			name:    t.name,
			in:      t.in,
			wantErr: t.wantErr,
			want:    t.want,
			out:     t.out,
		}

		ptrTest.want = append([]byte{0x01}, t.want...)
		ptrTests = append(ptrTests, ptrTest)
	}
	for _, tt := range ptrTests {
		t.Run(tt.name, func(t *testing.T) {
			switch in := tt.in.(type) {
			case VaryingDataType:
				// copy the inputted vdt cause we need the cached values
				copy := in
				vdt := copy
				vdt.value = nil
				var dst interface{} = &vdt
				if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
					t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				diff := cmp.Diff(
					vdt.value,
					tt.in.(VaryingDataType).value,
					cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
				if diff != "" {
					t.Errorf("decodeState.unmarshal() = %s", diff)
				}
			default:
				dst := reflect.New(reflect.TypeOf(tt.in)).Interface()
				if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
					t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				var diff string
				if tt.out != nil {
					diff = cmp.Diff(
						reflect.ValueOf(dst).Elem().Interface(),
						reflect.ValueOf(tt.out).Interface(),
						cmpopts.IgnoreUnexported(tt.in))
				} else {
					diff = cmp.Diff(
						reflect.ValueOf(dst).Elem().Interface(),
						reflect.ValueOf(tt.in).Interface(),
						cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
				}
				if diff != "" {
					t.Errorf("decodeState.unmarshal() = %s", diff)
				}
			}
		})
	}
}

func Test_unmarshal_optionality_nil_case(t *testing.T) {
	var ptrTests tests
	for _, t := range allTests {
		ptrTest := test{
			name:    t.name,
			in:      t.in,
			wantErr: t.wantErr,
			want:    t.want,
			// ignore out, since we are testing nil case
			// out:     t.out,
		}
		ptrTest.want = []byte{0x00}

		temp := reflect.New(reflect.TypeOf(t.in))
		// create a new pointer to type of temp
		tempv := reflect.New(reflect.PtrTo(temp.Type()).Elem())
		// set zero value to elem of **temp so that is nil
		tempv.Elem().Set(reflect.Zero(tempv.Elem().Type()))
		// set test.in to *temp
		ptrTest.in = tempv.Elem().Interface()

		ptrTests = append(ptrTests, ptrTest)
	}
	for _, tt := range ptrTests {
		t.Run(tt.name, func(t *testing.T) {
			// this becomes a pointer to a zero value of the underlying value
			dst := reflect.New(reflect.TypeOf(tt.in)).Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var diff string
			if tt.out != nil {
				diff = cmp.Diff(
					reflect.ValueOf(dst).Elem().Interface(),
					reflect.ValueOf(tt.out).Interface())
			} else {
				diff = cmp.Diff(
					reflect.ValueOf(dst).Elem().Interface(),
					reflect.ValueOf(tt.in).Interface(),
					cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
			}
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
			}
		})
	}
}

func Test_Decoder_Decode(t *testing.T) {
	for _, tt := range newTests(fixedWidthIntegerTests, variableWidthIntegerTests, stringTests,
		boolTests, sliceTests, arrayTests,
	) {
		t.Run(tt.name, func(t *testing.T) {
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			wantBuf := bytes.NewBuffer(tt.want)
			d := NewDecoder(wantBuf)
			if err := d.Decode(&dst); (err != nil) != tt.wantErr {
				t.Errorf("Decoder.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(dst, tt.in) {
				t.Errorf("Decoder.Decode() = %v, want %v", dst, tt.in)
			}
		})
	}
}

func Test_Decoder_Decode_MultipleCalls(t *testing.T) {
	tests := []struct {
		name    string
		ins     []interface{}
		want    []byte
		wantErr []bool
	}{
		{
			name: "int64 and []byte",
			ins:  []interface{}{int64(9223372036854775807), []byte{0x01}},
			want: append([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, []byte{0x04, 0x01}...),
		},
		{
			name:    "eof error",
			ins:     []interface{}{int64(9223372036854775807), []byte{0x01}},
			want:    []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
			wantErr: []bool{false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.want)
			d := NewDecoder(buf)

			for i := range tt.ins {
				in := tt.ins[i]
				dst := reflect.New(reflect.TypeOf(in)).Elem().Interface()
				var wantErr bool
				if len(tt.wantErr) > i {
					wantErr = tt.wantErr[i]
				}
				if err := d.Decode(&dst); (err != nil) != wantErr {
					t.Errorf("Decoder.Decode() error = %v, wantErr %v", err, tt.wantErr[i])
					return
				}
				if !wantErr && !reflect.DeepEqual(dst, in) {
					t.Errorf("Decoder.Decode() = %v, want %v", dst, in)
					return
				}
			}
		})
	}
}
