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
