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
				diff := cmp.Diff(vdt.value, tt.in.(VaryingDataType).value, cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
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
					diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.out).Interface(), cmpopts.IgnoreUnexported(tt.in))
				} else {
					diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.in).Interface(), cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
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
				diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.out).Interface())
			} else {
				diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.in).Interface(), cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
			}
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
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
	for _, t := range allTests {
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
				diff := cmp.Diff(vdt.value, tt.in.(VaryingDataType).value, cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
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
					diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.out).Interface(), cmpopts.IgnoreUnexported(tt.in))
				} else {
					diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.in).Interface(), cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
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
				diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.out).Interface())
			} else {
				diff = cmp.Diff(reflect.ValueOf(dst).Elem().Interface(), reflect.ValueOf(tt.in).Interface(), cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
			}
			if diff != "" {
				t.Errorf("decodeState.unmarshal() = %s", diff)
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
