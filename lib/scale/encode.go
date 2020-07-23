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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"reflect"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Encoder is a wrapping around io.Writer
type Encoder struct {
	Writer io.Writer
}

// Encode returns the SCALE encoding of the given interface
func Encode(in interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	se := Encoder{
		Writer: buffer,
	}
	_, err := se.Encode(in)
	output := buffer.Bytes()
	return output, err
}

// EncodeCustom checks if interface has method Encode, if so use that, otherwise use regular scale encoding
func EncodeCustom(in interface{}) ([]byte, error) {
	someType := reflect.TypeOf(in)
	_, ok := someType.MethodByName("Encode")
	if ok {
		res := reflect.ValueOf(in).MethodByName("Encode").Call([]reflect.Value{})
		val := res[0].Interface()
		err := res[1].Interface()
		if err != nil {
			return val.([]byte), err.(error)
		}
		return val.([]byte), nil
	}
	return Encode(in)
}

// Encode is the top-level function which performs SCALE encoding of b which may be of type []byte, int16, int32, int64, or bool
func (se *Encoder) Encode(b interface{}) (n int, err error) {
	switch v := b.(type) {
	case []byte:
		n, err = se.encodeByteArray(v)
	case *big.Int:
		n, err = se.encodeBigInteger(v)
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		n, err = se.encodeFixedWidthInteger(v)
	case string:
		n, err = se.encodeByteArray([]byte(v))
	case bool:
		n, err = se.encodeBool(v)
	case common.Hash:
		n, err = se.Writer.Write(v.ToBytes())
	case interface{}:
		t := reflect.TypeOf(b).Kind()
		switch t {
		case reflect.Ptr:
			n, err = se.encodeTuple(v)
		case reflect.Struct:
			n, err = se.encodeTuple(v)
		case reflect.Slice, reflect.Array:
			n, err = se.encodeArray(v)
		default:
			return 0, fmt.Errorf("unsupported type: %T", b)
		}
	default:
		return 0, fmt.Errorf("unsupported type: %T", b)
	}

	return n, err
}

// EncodeCustom checks if interface has method Encode, if so use that, otherwise return error
func (se *Encoder) EncodeCustom(in interface{}) (int, error) {
	someType := reflect.TypeOf(in)
	// TODO: if not a pointer, check if type pointer has Encode method
	_, ok := someType.MethodByName("Encode")
	if ok {
		res := reflect.ValueOf(in).MethodByName("Encode").Call([]reflect.Value{})
		val := res[0].Interface()
		err := res[1].Interface()
		if err != nil {
			return 0, err.(error)
		}
		return se.Writer.Write(val.([]byte))
	}
	return 0, fmt.Errorf("cannot call EncodeCustom")
}

// encodeCustomOrEncode tries to use EncodeCustom, if that fails, it reverts to Encode
func (se *Encoder) encodeCustomOrEncode(in interface{}) (int, error) {
	n, err := se.EncodeCustom(in)
	if err == nil {
		return n, err
	}

	return se.Encode(in)
}

// encodeByteArray performs the following:
// b -> [encodeInteger(len(b)) b]
// it writes to the buffer a byte array where the first byte is the length of b encoded with SCALE, followed by the
// byte array b itself
func (se *Encoder) encodeByteArray(b []byte) (bytesEncoded int, err error) {
	var n int
	n, err = se.encodeInteger(uint(len(b)))
	if err != nil {
		return 0, err
	}

	bytesEncoded = bytesEncoded + n
	n, err = se.Writer.Write(b)
	return bytesEncoded + n, err
}

// encodeFixedWidthInteger encodes an int with size < 2**32 by putting it into little endian byte format
func (se *Encoder) encodeFixedWidthInteger(in interface{}) (bytesEncoded int, err error) {
	switch i := in.(type) {
	case int8:
		err = binary.Write(se.Writer, binary.LittleEndian, byte(i))
		bytesEncoded = 1
	case uint8:
		err = binary.Write(se.Writer, binary.LittleEndian, i)
		bytesEncoded = 1
	case int16:
		err = binary.Write(se.Writer, binary.LittleEndian, uint16(i))
		bytesEncoded = 2
	case uint16:
		err = binary.Write(se.Writer, binary.LittleEndian, i)
		bytesEncoded = 2
	case int32:
		err = binary.Write(se.Writer, binary.LittleEndian, uint32(i))
		bytesEncoded = 4
	case uint32:
		err = binary.Write(se.Writer, binary.LittleEndian, i)
		bytesEncoded = 4
	case int64:
		err = binary.Write(se.Writer, binary.LittleEndian, uint64(i))
		bytesEncoded = 8
	case uint64:
		err = binary.Write(se.Writer, binary.LittleEndian, i)
		bytesEncoded = 8
	case int:
		err = binary.Write(se.Writer, binary.LittleEndian, int64(i))
		bytesEncoded = 8
	case uint:
		err = binary.Write(se.Writer, binary.LittleEndian, uint64(i))
		bytesEncoded = 8
	default:
		err = fmt.Errorf("could not encode fixed width int, invalid type: %T", in)
	}

	return bytesEncoded, err
}

// encodeInteger performs the following on integer i:
// i  -> i^0...i^n where n is the length in bits of i
// note that the bit representation of i is in little endian; ie i^0 is the least significant bit of i,
// and i^n is the most significant bit
// if n < 2^6 write [00 i^2...i^8 ] [ 8 bits = 1 byte encoded ]
// if 2^6 <= n < 2^14 write [01 i^2...i^16] [ 16 bits = 2 byte encoded ]
// if 2^14 <= n < 2^30 write [10 i^2...i^32] [ 32 bits = 4 byte encoded ]
// if n >= 2^30 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (se *Encoder) encodeInteger(i uint) (bytesEncoded int, err error) {

	if i < 1<<6 {
		err = binary.Write(se.Writer, binary.LittleEndian, byte(i)<<2)
		return 1, err
	} else if i < 1<<14 {
		err = binary.Write(se.Writer, binary.LittleEndian, uint16(i<<2)+1)
		return 2, err
	} else if i < 1<<30 {
		err = binary.Write(se.Writer, binary.LittleEndian, uint32(i<<2)+2)
		return 4, err
	}

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

	err = binary.Write(se.Writer, binary.LittleEndian, lengthByte)
	bytesEncoded++
	if err == nil {
		binary.LittleEndian.PutUint64(o, uint64(i))
		err = binary.Write(se.Writer, binary.LittleEndian, o[0:numBytes])
		bytesEncoded += numBytes
	}

	return bytesEncoded, err
}

// encodeBigInteger performs the same encoding as encodeInteger, except on a big.Int.
// if 2^30 <= n < 2^536 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (se *Encoder) encodeBigInteger(i *big.Int) (bytesEncoded int, err error) {
	if i.Cmp(new(big.Int).Lsh(big.NewInt(1), 6)) < 0 { // if i < 1<<6
		err = binary.Write(se.Writer, binary.LittleEndian, uint8(i.Int64()<<2))
		return 1, err
	} else if i.Cmp(new(big.Int).Lsh(big.NewInt(1), 14)) < 0 { // if i < 1<<14
		err = binary.Write(se.Writer, binary.LittleEndian, uint16(i.Int64()<<2)+1)
		return 2, err
	} else if i.Cmp(new(big.Int).Lsh(big.NewInt(1), 30)) < 0 { //if i < 1<<30
		err = binary.Write(se.Writer, binary.LittleEndian, uint32(i.Int64()<<2)+2)
		return 4, err
	}

	numBytes := len(i.Bytes())
	topSixBits := uint8(numBytes - 4)
	lengthByte := topSixBits<<2 + 3

	// write byte which encodes mode and length
	err = binary.Write(se.Writer, binary.LittleEndian, lengthByte)
	if err == nil {
		// write integer itself
		err = binary.Write(se.Writer, binary.LittleEndian, reverseBytes(i.Bytes()))
	}

	return numBytes + 1, err
}

// encodeBool performs the following:
// l = true -> write [1]
// l = false -> write [0]
func (se *Encoder) encodeBool(l bool) (bytesEncoded int, err error) {
	if l {
		bytesEncoded, err = se.Writer.Write([]byte{0x01})
		return bytesEncoded, err
	}
	bytesEncoded, err = se.Writer.Write([]byte{0x00})
	return bytesEncoded, err
}

// encodeTuple reads the number of fields in the struct and their types and writes to the buffer each of the struct fields
// encoded as their respective types
func (se *Encoder) encodeTuple(t interface{}) (bytesEncoded int, err error) {
	var v reflect.Value
	switch reflect.ValueOf(t).Kind() {
	case reflect.Ptr:
		v = reflect.ValueOf(t).Elem()
	case reflect.Slice, reflect.Array, reflect.Struct:
		v = reflect.ValueOf(t)
	}

	values := make([]interface{}, 0)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			values = append(values, v.Field(i).Interface())
		}
	}

	for _, item := range values {
		n, err := se.encodeCustomOrEncode(item)
		if err != nil {
			return bytesEncoded, err
		}

		bytesEncoded += n
	}

	return bytesEncoded, nil
}

func (se *Encoder) encodeIntegerElements(arr []int) (bytesEncoded int, err error) {
	var n int
	n, err = se.encodeInteger(uint(len(arr)))
	bytesEncoded += n

	for _, elem := range arr {
		n, err = se.encodeInteger(uint(elem))
		bytesEncoded += n
	}

	return bytesEncoded, err
}

// encodeArray encodes an interface where the underlying type is an array or slice
// it writes the encoded length of the Array to the Encoder, then encodes and writes each value in the Array
func (se *Encoder) encodeArray(t interface{}) (bytesEncoded int, err error) {
	var n int
	switch arr := t.(type) {
	case []int:
		n, err = se.encodeIntegerElements(arr)
		bytesEncoded += n
	case []*big.Int:
		n, err = se.encodeInteger(uint(len(arr)))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeBigInteger(elem)
			bytesEncoded += n
		}
	case []bool:
		n, err = se.encodeInteger(uint(len(arr)))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeBool(elem)
			bytesEncoded += n
		}
	case [][]byte:
		n, err = se.encodeInteger(uint(len(arr)))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeByteArray(elem)
			bytesEncoded += n
		}
	case [][]int:
		n, err = se.encodeInteger(uint(len(arr)))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeArray(elem)
			bytesEncoded += n
		}
	case []string:
		n, err = se.encodeInteger(uint(len(arr)))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeByteArray([]byte(elem))
			bytesEncoded += n
		}
	case []common.PeerInfo:
		n, err = se.encodeInteger(uint(len(arr)))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.Encode(elem)
			bytesEncoded += n
		}
	default:
		s := reflect.ValueOf(t)
		t := reflect.TypeOf(arr).Kind()
		switch t {
		case reflect.Slice:
			n, err = se.encodeInteger(uint(s.Len()))
			bytesEncoded += n
		case reflect.Array:
			// don't encode length
		}

		for i := 0; i < s.Len(); i++ {
			n, err = se.encodeCustomOrEncode(s.Index(i).Interface())
			bytesEncoded += n
		}
	}

	return bytesEncoded, err
}
