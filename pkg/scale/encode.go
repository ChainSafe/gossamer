// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
)

// Marshal takes in an interface{} and attempts to marshal into []byte
func Marshal(v interface{}) (b []byte, err error) {
	es := encodeState{
		fieldScaleIndicesCache: cache,
	}
	err = es.marshal(v)
	if err != nil {
		return
	}
	b = es.Bytes()
	return
}

type encodeState struct {
	bytes.Buffer
	*fieldScaleIndicesCache
}

func (es *encodeState) marshal(in interface{}) (err error) {
	switch in := in.(type) {
	case int:
		err = es.encodeUint(uint(in))
	case uint:
		err = es.encodeUint(in)
	case int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		err = es.encodeFixedWidthInt(in)
	case *big.Int:
		err = es.encodeBigInt(in)
	case *Uint128:
		err = es.encodeUint128(in)
	case []byte:
		err = es.encodeBytes(in)
	case string:
		err = es.encodeBytes([]byte(in))
	case bool:
		err = es.encodeBool(in)
	case Result:
		err = es.encodeResult(in)
	case VaryingDataType:
		err = es.encodeVaryingDataType(in)
	case VaryingDataTypeSlice:
		err = es.encodeVaryingDataTypeSlice(in)
	default:
		switch reflect.TypeOf(in).Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
			reflect.Int32, reflect.Int64, reflect.String, reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			err = es.encodeCustomPrimitive(in)
		case reflect.Ptr:
			// Assuming that anything that is a pointer is an Option to capture {nil, T}
			elem := reflect.ValueOf(in).Elem()
			switch elem.IsValid() {
			case false:
				err = es.WriteByte(0)
			default:
				err = es.WriteByte(1)
				if err != nil {
					return
				}
				err = es.marshal(elem.Interface())
			}
		case reflect.Struct:
			err = es.encodeStruct(in)
		case reflect.Array:
			err = es.encodeArray(in)
		case reflect.Slice:
			err = es.encodeSlice(in)
		default:
			err = fmt.Errorf("unsupported type: %T", in)
		}
	}
	return
}

func (es *encodeState) encodeCustomPrimitive(in interface{}) (err error) {
	switch reflect.TypeOf(in).Kind() {
	case reflect.Bool:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(false)).Interface()
	case reflect.Int:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(int(0))).Interface()
	case reflect.Int8:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(int8(0))).Interface()
	case reflect.Int16:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(int16(0))).Interface()
	case reflect.Int32:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(int32(0))).Interface()
	case reflect.Int64:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(int64(0))).Interface()
	case reflect.String:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf("")).Interface()
	case reflect.Uint:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(uint(0))).Interface()
	case reflect.Uint8:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(uint8(0))).Interface()
	case reflect.Uint16:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(uint16(0))).Interface()
	case reflect.Uint32:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(uint32(0))).Interface()
	case reflect.Uint64:
		in = reflect.ValueOf(in).Convert(reflect.TypeOf(uint64(0))).Interface()
	default:
		err = fmt.Errorf("unsupported type for custom primitive: %T", in)
		return
	}
	err = es.marshal(in)
	return
}

func (es *encodeState) encodeResult(res Result) (err error) {
	if !res.IsSet() {
		err = fmt.Errorf("Result is not set: %+v", res)
		return
	}

	var in interface{}
	switch res.mode {
	case OK:
		err = es.WriteByte(0)
		if err != nil {
			return
		}
		in = res.ok
	case Err:
		err = es.WriteByte(1)
		if err != nil {
			return
		}
		in = res.err
	}
	switch in := in.(type) {
	case empty:
	default:
		err = es.marshal(in)
	}
	return
}

func (es *encodeState) encodeVaryingDataType(vdt VaryingDataType) (err error) {
	err = es.WriteByte(byte(vdt.value.Index()))
	if err != nil {
		return
	}
	err = es.marshal(vdt.value)
	return
}

func (es *encodeState) encodeVaryingDataTypeSlice(vdts VaryingDataTypeSlice) (err error) {
	err = es.marshal(vdts.Types)
	return
}

func (es *encodeState) encodeSlice(in interface{}) (err error) {
	v := reflect.ValueOf(in)
	err = es.encodeLength(v.Len())
	if err != nil {
		return
	}
	for i := 0; i < v.Len(); i++ {
		err = es.marshal(v.Index(i).Interface())
		if err != nil {
			return
		}
	}
	return
}

// encodeArray encodes an interface where the underlying type is an array
// it writes the encoded length of the Array to the Encoder, then encodes and writes each value in the Array
func (es *encodeState) encodeArray(in interface{}) (err error) {
	v := reflect.ValueOf(in)
	for i := 0; i < v.Len(); i++ {
		err = es.marshal(v.Index(i).Interface())
		if err != nil {
			return
		}
	}
	return
}

// encodeBigInt performs the same encoding as encodeInteger, except on a big.Int.
// if 2^30 <= n < 2^536 write
// [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (es *encodeState) encodeBigInt(i *big.Int) (err error) {
	switch {
	case i == nil:
		err = fmt.Errorf("nil *big.Int")
	case i.Cmp(new(big.Int).Lsh(big.NewInt(1), 6)) < 0:
		err = binary.Write(es, binary.LittleEndian, uint8(i.Int64()<<2))
	case i.Cmp(new(big.Int).Lsh(big.NewInt(1), 14)) < 0:
		err = binary.Write(es, binary.LittleEndian, uint16(i.Int64()<<2)+1)
	case i.Cmp(new(big.Int).Lsh(big.NewInt(1), 30)) < 0:
		err = binary.Write(es, binary.LittleEndian, uint32(i.Int64()<<2)+2)
	default:
		numBytes := len(i.Bytes())
		topSixBits := uint8(numBytes - 4)
		lengthByte := topSixBits<<2 + 3

		// write byte which encodes mode and length
		err = binary.Write(es, binary.LittleEndian, lengthByte)
		if err == nil {
			// write integer itself
			err = binary.Write(es, binary.LittleEndian, reverseBytes(i.Bytes()))
		}
	}
	return
}

// encodeBool performs the following:
// l = true -> write [1]
// l = false -> write [0]
func (es *encodeState) encodeBool(l bool) (err error) {
	switch l {
	case true:
		_, err = es.Write([]byte{0x01})
	case false:
		_, err = es.Write([]byte{0x00})
	}
	return
}

// encodeByteArray performs the following:
// b -> [encodeInteger(len(b)) b]
// it writes to the buffer a byte array where the first byte is the length of b encoded with SCALE, followed by the
// byte array b itself
func (es *encodeState) encodeBytes(b []byte) (err error) {
	err = es.encodeLength(len(b))
	if err != nil {
		return
	}

	_, err = es.Write(b)
	return
}

// encodeFixedWidthInt encodes an int with size < 2**32 by putting it into little endian byte format
func (es *encodeState) encodeFixedWidthInt(i interface{}) (err error) {
	switch i := i.(type) {
	case int8:
		err = binary.Write(es, binary.LittleEndian, byte(i))
	case uint8:
		err = binary.Write(es, binary.LittleEndian, i)
	case int16:
		err = binary.Write(es, binary.LittleEndian, uint16(i))
	case uint16:
		err = binary.Write(es, binary.LittleEndian, i)
	case int32:
		err = binary.Write(es, binary.LittleEndian, uint32(i))
	case uint32:
		err = binary.Write(es, binary.LittleEndian, i)
	case int64:
		err = binary.Write(es, binary.LittleEndian, uint64(i))
	case uint64:
		err = binary.Write(es, binary.LittleEndian, i)
	default:
		err = fmt.Errorf("could not encode fixed width integer, invalid type: %T", i)
	}
	return
}

// encodeStruct reads the number of fields in the struct and their types
// and writes to the buffer each of the struct fields encoded
// as their respective types
func (es *encodeState) encodeStruct(in interface{}) (err error) {
	v, indices, err := es.fieldScaleIndices(in)
	if err != nil {
		return
	}
	for _, i := range indices {
		field := v.Field(i.fieldIndex)
		if !field.CanInterface() {
			continue
		}
		err = es.marshal(field.Interface())
		if err != nil {
			return
		}
	}
	return
}

// encodeLength is a helper function that calls encodeUint, which is the scale length encoding
func (es *encodeState) encodeLength(l int) (err error) {
	return es.encodeUint(uint(l))
}

// encodeUint performs the following on integer i:
// i  -> i^0...i^n where n is the length in bits of i
// note that the bit representation of i is in little endian; ie i^0 is the least significant bit of i,
// and i^n is the most significant bit
// if n < 2^6 write [00 i^2...i^8 ] [ 8 bits = 1 byte encoded ]
// if 2^6 <= n < 2^14 write [01 i^2...i^16] [ 16 bits = 2 byte encoded ]
// if 2^14 <= n < 2^30 write [10 i^2...i^32] [ 32 bits = 4 byte encoded ]
// if n >= 2^30 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (es *encodeState) encodeUint(i uint) (err error) {
	switch {
	case i < 1<<6:
		err = binary.Write(es, binary.LittleEndian, byte(i)<<2)
	case i < 1<<14:
		err = binary.Write(es, binary.LittleEndian, uint16(i<<2)+1)
	case i < 1<<30:
		err = binary.Write(es, binary.LittleEndian, uint32(i<<2)+2)
	default:
		o := make([]byte, 8)
		m := i
		var numBytes int
		// calculate the number of bytes needed to store i
		// the most significant byte cannot be zero
		// each iteration, shift by 1 byte until the number is zero
		// then break and save the numBytes needed
		for numBytes = 0; numBytes < 256 && m != 0; numBytes++ {
			m = m >> 8
		}

		topSixBits := uint8(numBytes - 4)
		lengthByte := topSixBits<<2 + 3

		err = binary.Write(es, binary.LittleEndian, lengthByte)
		if err == nil {
			binary.LittleEndian.PutUint64(o, uint64(i))
			err = binary.Write(es, binary.LittleEndian, o[0:numBytes])
		}
	}
	return
}

// encodeUint128 encodes a Uint128
func (es *encodeState) encodeUint128(i *Uint128) (err error) {
	if i == nil {
		err = fmt.Errorf("uint128 is nil: %v", i)
		return
	}
	err = binary.Write(es, binary.LittleEndian, padBytes(i.Bytes(), binary.LittleEndian))
	return
}
