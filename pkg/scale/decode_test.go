// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
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

			// assert response only if we aren't expecting an error
			if !tt.wantErr {
				var diff string
				if tt.out != nil {
					diff = cmp.Diff(dst, tt.out, cmpopts.IgnoreUnexported(tt.in))
				} else {
					diff = cmp.Diff(dst, tt.in, cmpopts.IgnoreUnexported(big.Int{}, tt.in, VDTValue2{}, MyStructWithIgnore{}))
				}
				if diff != "" {
					t.Errorf("decodeState.unmarshal() = %s", diff)
				}
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

// // Rust code to encode a map of string to struct.
// let mut btree_map: BTreeMap<String, User> = BTreeMap::new();
// match btree_map.entry("string1".to_string()) {
// 	Entry::Vacant(entry) => {
// 		entry.insert(User{
// 			active: true,
// 			username: "lorem".to_string(),
// 			email: "lorem@ipsum.org".to_string(),
// 			sign_in_count: 1,
// 		 });
// 		()
// 	},
// 	Entry::Occupied(_) => (),
// }
// match btree_map.entry("string2".to_string()) {
// 	Entry::Vacant(entry) => {
// 		entry.insert(User{
// 			active: false,
// 			username: "john".to_string(),
// 			email: "jack@gmail.com".to_string(),
// 			sign_in_count: 73,
// 		 });
// 		()
// 	},
// 	Entry::Occupied(_) => (),
// }
// println!("{:?}", btree_map.encode());

type user struct {
	Active      bool
	Username    string
	Email       string
	SignInCount uint64
}

func Test_decodeState_decodeMap(t *testing.T) {
	mapTests1 := []struct {
		name           string
		input          []byte
		wantErr        bool
		expectedOutput map[int8][]byte
	}{
		{
			name:           "testing a map of int8 to a byte array 1",
			input:          []byte{4, 2, 44, 115, 111, 109, 101, 32, 115, 116, 114, 105, 110, 103},
			expectedOutput: map[int8][]byte{2: []byte("some string")},
		},
		{
			name: "testing_a_map_of_int8_to_a_byte_array_2",
			input: []byte{
				8, 2, 44, 115, 111, 109, 101, 32, 115, 116, 114, 105, 110, 103, 16, 44, 108, 111, 114, 101, 109, 32,
				105, 112, 115, 117, 109,
			},
			expectedOutput: map[int8][]byte{
				2:  []byte("some string"),
				16: []byte("lorem ipsum"),
			},
		},
	}

	for _, tt := range mapTests1 {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := make(map[int8][]byte)
			if err := Unmarshal(tt.input, &actualOutput); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(actualOutput, tt.expectedOutput) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", actualOutput, tt.expectedOutput)
			}
		})
	}

	mapTests2 := []struct {
		name           string
		input          []byte
		wantErr        bool
		expectedOutput map[string]user
	}{
		{
			name:  "testing a map of string to struct",
			input: []byte{8, 28, 115, 116, 114, 105, 110, 103, 49, 1, 20, 108, 111, 114, 101, 109, 60, 108, 111, 114, 101, 109, 64, 105, 112, 115, 117, 109, 46, 111, 114, 103, 1, 0, 0, 0, 0, 0, 0, 0, 28, 115, 116, 114, 105, 110, 103, 50, 0, 16, 106, 111, 104, 110, 56, 106, 97, 99, 107, 64, 103, 109, 97, 105, 108, 46, 99, 111, 109, 73, 0, 0, 0, 0, 0, 0, 0}, //nolint:lll
			expectedOutput: map[string]user{
				"string1": {
					Active:      true,
					Username:    "lorem",
					Email:       "lorem@ipsum.org",
					SignInCount: 1,
				},
				"string2": {
					Active:      false,
					Username:    "john",
					Email:       "jack@gmail.com",
					SignInCount: 73,
				},
			},
		},
	}

	for _, tt := range mapTests2 {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := make(map[string]user)
			if err := Unmarshal(tt.input, &actualOutput); (err != nil) != tt.wantErr {
				t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(actualOutput, tt.expectedOutput) {
				t.Errorf("decodeState.unmarshal() = %v, want %v", actualOutput, tt.expectedOutput)
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
			switch tt.in.(type) {
			// case impls.VaryingDataType:
			// 	// copy the inputted vdt cause we need the cached values
			// 	cp := in
			// 	vdt := cp
			// 	// vdt.value = nil
			// 	var dst interface{} = &vdt
			// 	if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
			// 		t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			// 		return
			// 	}
			// 	diff := cmp.Diff(
			// 		vdt.value,
			// 		tt.in.(DefaultVaryingDataType).value,
			// 		cmpopts.IgnoreUnexported(big.Int{}, VDTValue2{}, MyStructWithIgnore{}, MyStructWithPrivate{}))
			// 	if diff != "" {
			// 		t.Errorf("decodeState.unmarshal() = %s", diff)
			// 	}
			default:
				var dst interface{}

				if reflect.TypeOf(tt.in).Kind().String() == "map" {
					dst = &(map[int8][]byte{})
				} else {
					dst = reflect.New(reflect.TypeOf(tt.in)).Interface()
				}

				if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
					t.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				// assert response only if we aren't expecting an error
				if !tt.wantErr {
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

		// for error cases, we don't need to modify the input since we need it to fail
		if !t.wantErr {
			ptrTest.want = []byte{0x00}
		}

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
			name: "int64_and_[]byte",
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

func Test_decodeState_decodeUint(t *testing.T) {
	t.Parallel()
	decodeUint32Tests := tests{
		{
			name: "int(1)_mode_0",
			in:   uint32(1),
			want: []byte{0x04},
		},
		{
			name: "int(16383)_mode_1",
			in:   int(16383),
			want: []byte{0xfd, 0xff},
		},
		{
			name: "int(1073741823)_mode_2",
			in:   int(1073741823),
			want: []byte{0xfe, 0xff, 0xff, 0xff},
		},
		{
			name: "int(4294967295)_mode_3",
			in:   int(4294967295),
			want: []byte{0x3, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name: "myCustomInt(9223372036854775807)_mode_3,_64bit",
			in:   myCustomInt(9223372036854775807),
			want: []byte{19, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
		{
			name:    "uint(overload)",
			in:      int(0),
			want:    []byte{0x07, 0x08, 0x09, 0x10, 0x0, 0x40},
			wantErr: true,
		},
		{
			name: "uint(16384)_mode_2",
			in:   int(16384),
			want: []byte{0x02, 0x00, 0x01, 0x0},
		},
		{
			name:    "uint(0) mode 1, error",
			in:      int(0),
			want:    []byte{0x01, 0x00},
			wantErr: true,
		},
		{
			name:    "uint(0) mode 2, error",
			in:      int(0),
			want:    []byte{0x02, 0x00, 0x00, 0x0},
			wantErr: true,
		},
		{
			name:    "uint(0) mode 3, error",
			in:      int(0),
			want:    []byte{0x03, 0x00, 0x00, 0x0},
			wantErr: true,
		},
		{
			name:    "mode 3, 64bit, error",
			in:      int(0),
			want:    []byte{19, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			wantErr: true,
		},
		{
			name: "[]int{1_<<_32,_2,_3,_1_<<_32}",
			in:   uint(4),
			want: []byte{0x10, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x0c, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name:    "[4]int{1 << 32, 2, 3, 1 << 32}",
			in:      [4]int{0, 0, 0, 0},
			want:    []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x0c, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01},
			wantErr: true,
		},
	}

	for _, tt := range decodeUint32Tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			dstv := reflect.ValueOf(&dst)
			elem := indirect(dstv)

			ds := decodeState{
				Reader: bytes.NewBuffer(tt.want),
			}
			err := ds.decodeUint(elem)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.in, dst)
		})
	}
}

type myStruct struct {
	First  uint32
	Middle any
	Last   uint32
}

func (ms *myStruct) UnmarshalSCALE(reader io.Reader) (err error) {
	buf := make([]byte, 4)
	_, err = reader.Read(buf)
	if err != nil {
		return
	}
	ms.First = binary.LittleEndian.Uint32(buf)

	buf = make([]byte, 4)
	_, err = reader.Read(buf)
	if err != nil {
		return
	}
	ms.Middle = binary.LittleEndian.Uint32(buf)

	buf = make([]byte, 4)
	_, err = reader.Read(buf)
	if err != nil {
		return
	}
	ms.Last = binary.LittleEndian.Uint32(buf)
	return nil
}

type myStructError struct {
	First  uint32
	Middle any
	Last   uint32
}

func (mse *myStructError) UnmarshalSCALE(reader io.Reader) (err error) {
	err = fmt.Errorf("eh?")
	return err
}

var _ Unmarshaler = &myStruct{}

func Test_decodeState_Unmarshaller(t *testing.T) {
	expected := myStruct{
		First:  1,
		Middle: uint32(2),
		Last:   3,
	}
	bytes := MustMarshal(expected)
	ms := myStruct{}
	Unmarshal(bytes, &ms)
	assert.Equal(t, expected, ms)

	type myParentStruct struct {
		First  uint
		Middle myStruct
		Last   uint
	}
	expectedParent := myParentStruct{
		First:  1,
		Middle: expected,
		Last:   3,
	}
	bytes = MustMarshal(expectedParent)
	mps := myParentStruct{}
	Unmarshal(bytes, &mps)
	assert.Equal(t, expectedParent, mps)
}

func Test_decodeState_Unmarshaller_Error(t *testing.T) {
	expected := myStruct{
		First:  1,
		Middle: uint32(2),
		Last:   3,
	}
	bytes := MustMarshal(expected)
	mse := myStructError{}
	err := Unmarshal(bytes, &mse)
	assert.Error(t, err, "eh?")
}
